# Deploy de produção — selene.papermoon.cloud

Runbook pra subir o Selene na VPS/LXC de produção, atrás do Cloudflare
Tunnel que já roda pros outros projetos (`papermoon`, etc.). Presume que
você tem acesso SSH ao host onde vai rodar e ao [Cloudflare Zero Trust
Dashboard](https://one.dash.cloudflare.com/) do domínio `papermoon.cloud`.

## 0. Antes de começar

- **Máquina/host**: mesma máquina do `papermoon`. A porta `3000` do host
  já está em uso pelo `papermoon-web` — o Selene publica em `3010`
  (configurável via `FRONTEND_PORT`, ver `.env.production.example`).
  Confira que `3010` também está livre: `sudo ss -tlnp | grep 3010`.
- **Backend não é público**: só o frontend Next.js é exposto ao host —
  arquitetura BFF, o browser nunca chama o backend Go diretamente. Uma
  porta a menos pra abrir no `ufw`.
- **Keycloak**: usa a instância real da prefeitura
  (`sso.rondonopolis.mt.gov.br`), a mesma já configurada em dev — não
  precisa (nem deve) subir um Keycloak novo.

## 1. No servidor: clonar e configurar

```bash
git clone https://github.com/yurythx/Projeto-selene.git
cd Projeto-selene
cp .env.production.example .env.production
# edite .env.production: DB_PASSWORD, REDIS_PASSWORD, AUTH_SECRET,
# AUTH_KEYCLOAK_SECRET (ver seção 2 abaixo pra pegar esse valor)
```

Gerar os segredos:
```bash
# DB_PASSWORD / REDIS_PASSWORD — qualquer string aleatória forte
openssl rand -base64 24

# AUTH_SECRET — especificamente pro Auth.js
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

## 2. Atualizar o client do Keycloak

O client `selene` já existe no Keycloak (criado pra dev — ver
`frontend/README.md`). **Não crie um novo** — adicione a URL de produção
às configurações existentes:

1. Acesse o Keycloak Admin Console → realm `rondonopolis` → **Clients**
   → `selene`.
2. Em **Login settings**, adicione (mantendo as de dev, se ainda forem
   usadas):
   - **Valid redirect URIs**: `https://selene.papermoon.cloud/api/auth/callback/keycloak`
   - **Valid post logout redirect URIs**: `https://selene.papermoon.cloud`
   - **Web origins**: `https://selene.papermoon.cloud`
3. Aba **Credentials** → copie o **Client secret** pro
   `AUTH_KEYCLOAK_SECRET` do `.env.production` (é o mesmo secret de
   sempre, a menos que tenha sido rotacionado).

## 3. Subir os containers

```bash
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build
docker compose -f docker-compose.prod.yml ps   # todos "healthy"/"running"
curl -s http://localhost:3010/ -o /dev/null -w "%{http_code}\n"   # 307 (redirect pro /login) é o esperado
```

Migrations e seed do banco rodam automaticamente no boot do backend
(`database.Migrate`/`database.Seed` em `cmd/api/main.go`) — não precisa
de passo manual.

## 4. Liberar a porta no firewall (ufw)

Mesmo padrão do papermoon — restringe ao IP de onde o `cloudflared` roda
(não abre pro mundo):

```bash
sudo ufw allow from <IP-da-LXC-do-cloudflared> to any port 3010 proto tcp
sudo ufw status
```

## 5. Rotear no Cloudflare Tunnel

1. [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/) →
   **Networks → Tunnels** → o túnel já usado pelos outros projetos.
2. Aba **Public Hostname** → **Add a public hostname**:
   - **Subdomain**: `selene`
   - **Domain**: `papermoon.cloud`
   - **Service**: `HTTP` → `<IP-do-host>:3010`
3. Salvar. A Cloudflare cuida do certificado TLS automaticamente — não
   precisa de Let's Encrypt/certbot aqui.

## 6. Primeiro acesso e primeiro admin

Não existe usuário nenhum até o primeiro login — a tabela `users` é
populada via *just-in-time provisioning* no primeiro token válido que
chega (ver `backend/internal/middleware/auth.go`). Todo usuário novo
começa com `is_fiscal=false, is_admin=false` (princípio do menor
privilégio) — **não há como conceder o primeiro admin pela própria UI**
(não existe admin ainda para promover alguém).

1. Acesse `https://selene.papermoon.cloud` e faça login com sua conta do
   Keycloak — isso cria seu usuário no Postgres do Selene.
2. Promova esse usuário a admin diretamente no banco:
   ```bash
   docker compose -f docker-compose.prod.yml exec postgres \
     psql -U selene -d projeto_selene -c \
     "UPDATE users SET is_admin = true, is_fiscal = true WHERE email = 'seu-email@prefeitura.gov.br';"
   ```
3. Faça logout/login de novo (a sessão tinha cacheado os flags antigos).
   A partir daqui, `/admin/usuarios` já funciona pra promover todo mundo
   sem precisar mexer no banco de novo.

## 7. Verificação final

- [ ] `https://selene.papermoon.cloud` carrega e redireciona pro login
- [ ] Login via Keycloak completa e volta autenticado
- [ ] `/kanban` mostra o quadro
- [ ] Criar um contrato de teste funciona
- [ ] `docker compose -f docker-compose.prod.yml logs -f backend` sem
      erros recorrentes
- [ ] `docker compose -f docker-compose.prod.yml logs -f frontend` sem
      `UntrustedHost` (se aparecer, confira `AUTH_TRUST_HOST=true` e que
      o Cloudflare Tunnel está repassando `X-Forwarded-Proto: https`)

## Rollback

```bash
docker compose -f docker-compose.prod.yml down
git checkout <commit-anterior>
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build
```

Dados (Postgres, storage de documentos) ficam em volumes nomeados
(`postgres_data`, `storage_data`) — sobrevivem a `down` sem `-v`.

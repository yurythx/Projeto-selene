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
# INTERNAL_API_SECRET, AUTH_KEYCLOAK_SECRET (ver seção 2 abaixo pra pegar
# esse valor)
```

Gerar os segredos:
```bash
# DB_PASSWORD / REDIS_PASSWORD — qualquer string aleatória forte
openssl rand -base64 24

# AUTH_SECRET — especificamente pro Auth.js
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"

# INTERNAL_API_SECRET — MESMO valor nos dois serviços (backend e
# frontend leem a mesma variável do .env.production compartilhado, então
# um valor só já basta). Autentica a única chamada server-to-server sem
# JWT de usuário desta API (GET /internal/keycloak-config, ver
# backend/README.md e frontend/README.md) — obrigatório, o backend não
# sobe sem isso definido.
openssl rand -hex 32
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

## Testando o compose de produção localmente, antes de um deploy real

`docker-compose.prod.yml` roda igual em qualquer máquina — não precisa
esperar chegar na VPS pra descobrir que algo quebra com as variáveis de
produção (foi assim que um crash de boot real foi encontrado: ver commit
"fix: corrige crash de boot em produção quando CORS_ALLOWED_ORIGINS=""").
Pra rodar lado a lado com o stack de dev (`docker-compose.yml`) já
rodando, sem conflito:

```bash
# .env.production local só pra este teste (nunca comitar) — Keycloak
# pode apontar pra instância real da prefeitura (só busca o JWKS
# público no boot, não faz login de verdade); PUBLIC_URL local.
cp .env.production.example .env.production
# preencha DB_PASSWORD/REDIS_PASSWORD/AUTH_SECRET (comandos na seção 1)
# e ajuste PUBLIC_URL=http://localhost:3010

# -p com nome de projeto diferente do dev evita que os dois stacks
# disputem os mesmos nomes de container/volume
docker compose -f docker-compose.prod.yml --env-file .env.production -p selene-prod up -d --build
```

Se o Jaeger do dev já estiver usando `127.0.0.1:16686`, crie um override
só local (não comitado) trocando a porta do Jaeger — `ports` é
concatenado, não substituído, entre arquivos de compose, então é preciso
`!reset` antes da porta nova:

```yaml
# docker-compose.prod.local-test.yml
services:
  jaeger:
    ports: !reset
      - "127.0.0.1:16687:16686"
```

```bash
docker compose -f docker-compose.prod.yml -f docker-compose.prod.local-test.yml --env-file .env.production -p selene-prod up -d --build
```

Sem acesso ao Keycloak real (login OIDC não é testável localmente), o
primeiro admin pode ser criado via login local direto no Postgres do
stack de teste, sem passar pelo fluxo de "promover via UI" da seção 6
(que pressupõe login via Keycloak já ter criado o usuário):

```bash
# gera um hash bcrypt (mesmo script usado durante o desenvolvimento)
docker compose -f docker-compose.prod.yml -p selene-prod exec postgres psql -U selene -d projeto_selene -c "
INSERT INTO users (id, nome, email, is_fiscal, is_admin, password_hash, must_change_password, criado_em, atualizado_em)
VALUES (gen_random_uuid(), 'Admin Teste Local', 'admin@teste.local', true, true, '<hash bcrypt>', false, now(), now());
"
```

Derruba com `docker compose -f docker-compose.prod.yml -p selene-prod down -v` quando terminar (o `-v` também apaga os volumes deste teste — não afeta o stack de dev, que usa um projeto/volumes diferentes).

## Rollback

```bash
docker compose -f docker-compose.prod.yml down
git checkout <commit-anterior>
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build
```

Dados (Postgres, storage de documentos) ficam em volumes nomeados
(`postgres_data`, `storage_data`) — sobrevivem a `down` sem `-v`.

# 🌙 Projeto Selene — Backend

API em Go (Gin + GORM + Postgres), autenticação via Keycloak (OIDC/JWT), Clean Architecture. Ver [../README.md](../README.md) para a visão geral do monorepo (backend + frontend).

---

## Arquitetura

```
cmd/api/            → ponto de entrada (wiring, rotas, graceful shutdown)
internal/
  config/            → variáveis de ambiente
  database/          → conexão, migrations versionadas (golang-migrate), seed
    migrations/       → *.sql, fonte de verdade do schema (embutidas no binário)
  models/            → structs de domínio + tags GORM
  middleware/        → auth (JWT/JWKS), RequireFiscal/RequireAdmin, CORS
  repository/        → acesso a dados (interface + implementação GORM)
  service/           → regras de negócio (checklist, Kanban, upload, etc.)
  handler/           → controllers HTTP (Gin)
  testutil/          → infraestrutura compartilhada para testes de integração
```

Fluxo de dependência: `handler → service → repository → gorm/postgres`. Cada camada só conhece a de baixo através de interfaces definidas por quem consome, não por quem implementa.

### O funil do Kanban

6 etapas fixas e lineares (`internal/database/migrations/000001_initial_schema.up.sql` semeia os nomes; `internal/service/checklist.go` define os requisitos de cada uma):

1. **Elaborar OF / Pré-Empenho** — exige OF, Pré-Empenho, Ofício de Solicitação.
2. **Tramitar Planejamento / Contabilidade** — sem checklist, tramitação externa.
3. **Emitir OS / Envio à Empresa** — exige Nota de Empenho. Ao confirmar, dispara em goroutine o envio do pacote digital à contratada.
4. **Execução e Recepção** — exige Nota Fiscal/Fatura e Ordem de Recepção.
5. **Relatório de Pagamento** — exige as certidões (CNDs, Simples Nacional, Extrato do Empenho) + o relatório assinado reanexado; contratos `SERVICO` exigem adicionalmente Planilha de Medição e Boleto DAM.
6. **Contabilidade (Liquidação e Pagamento)** — etapa final; `POST /processos/:id/concluir` marca como pago.

Toda transição é atômica com a gravação em `kanban_logs` (mesma transação SQL) — ver `internal/service/kanban_service.go`.

---

## Rodando localmente

### Opção 1 — Docker Compose (recomendado)

O `docker-compose.yml` fica na **raiz do monorepo** (orquestra backend + Postgres, e futuramente o frontend também) — rode a partir de lá, não daqui de dentro de `backend/`:

```bash
cp backend/.env.example backend/.env
# edite backend/.env: KEYCLOAK_JWKS_URL, KEYCLOAK_ISSUER, etc.
make up            # a partir da raiz do repo (atalho para `docker compose up --build`)
```

Sobe Postgres + backend. A API roda as migrations e o seed automaticamente no boot. Health check em `http://localhost:8080/health`.

### Opção 2 — Go local

Requer Go 1.25+ e um Postgres acessível. Comandos abaixo assumem `cwd = backend/`.

```bash
cd backend
cp .env.example .env   # ajuste DB_* e KEYCLOAK_JWKS_URL/KEYCLOAK_ISSUER
go run ./cmd/api
```

Ou, a partir da raiz: `make -C backend run` (ver `backend/Makefile` para todos os atalhos: `build`, `test-unit`, `test-integration`, `lint`, `migrate-new`, etc.).

---

## Testes

```bash
go test ./...
```

Três níveis:
- **Unitários** (`internal/service/*_test.go`): usam dublês em memória (fakes, ver `internal/testutil/fakes.go`) dos repositories, não tocam banco, sempre rodam.
- **Handler/HTTP** (`internal/handler/*_test.go`): montam a pilha real `handler -> service` (repositories fake) e testam via `httptest` — binding de JSON/multipart, códigos de status, mapeamento de erros. Também sempre rodam, sem banco.
- **Integração** (`internal/repository/*_test.go`, `internal/service/kanban_service_test.go`): usam Postgres real (via `internal/testutil.OpenTestDB`), aplicando as migrations de verdade. **São puladas automaticamente (`SKIP`)** se não houver um Postgres acessível — não travam quem não tem Docker instalado.

Para rodá-las de propósito:

```bash
docker run -d --name selene-test-pg -e POSTGRES_PASSWORD=selene -e POSTGRES_USER=selene -e POSTGRES_DB=projeto_selene -p 55432:5432 postgres:16
export TEST_DATABASE_URL="host=localhost port=55432 user=selene password=selene dbname=projeto_selene sslmode=disable"
go test ./... -v -p 1
```

`-p 1` é importante aqui: `internal/repository` e `internal/service` têm testes de integração contra o **mesmo** Postgres externo, e o Go roda pacotes diferentes em paralelo por padrão — dois pacotes migrando/truncando o mesmo banco ao mesmo tempo pode gerar um deadlock transitório no Postgres. `-p 1` serializa a execução por pacote e evita isso.

O CI (`.github/workflows/ci.yml`) sempre roda as duas camadas, com um serviço Postgres disponível.

---

## Migrations

O schema é versionado em `internal/database/migrations/*.sql` (aplicado via [golang-migrate](https://github.com/golang-migrate/migrate)) e embutido no binário (`go:embed`) — **não** é gerado por `AutoMigrate` do GORM. As tags GORM nos `internal/models/*.go` servem só para leitura/escrita via ORM; alterar um struct não altera o banco.

Para adicionar uma migration nova:

```bash
# instale a CLI uma vez: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate create -ext sql -dir internal/database/migrations -seq nome_da_mudanca
```

Isso cria um par `NNNNNN_nome_da_mudanca.up.sql` / `.down.sql`. `database.Migrate` (chamado no boot) aplica automaticamente tudo que estiver pendente.

---

## Variáveis de ambiente

Ver `.env.example` para a lista completa com comentários. Resumo:

| Variável | Obrigatória | Descrição |
|---|---|---|
| `APP_ENV` | não (default `development`) | `production` ativa `gin.ReleaseMode` e logs em JSON por padrão. |
| `SERVER_PORT` | não (default `8080`) | Porta HTTP. |
| `CORS_ALLOWED_ORIGINS` | não (default: nenhuma) | Origens autorizadas a chamar a API pelo navegador. |
| `TRUSTED_PROXIES` | não (default: nenhum) | IPs/CIDRs confiáveis para resolver `X-Forwarded-For`. |
| `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | **sim** | Credenciais do Postgres. |
| `DB_PORT`, `DB_SSLMODE` | não (defaults `5432`/`disable`) | |
| `KEYCLOAK_JWKS_URL` | **sim** | Endpoint JWKS do realm Keycloak. |
| `KEYCLOAK_ISSUER` | **sim** | Valor exato esperado no claim `iss` do JWT. |
| `KEYCLOAK_AUDIENCE` | não | Valor esperado no claim `aud`; vazio = não valida. |
| `STORAGE_DIR` | não (default `./storage`) | Onde os documentos anexos são gravados. |
| `SMTP_*` | não | Envio real do pacote à empresa contratada (Etapa 3, com anexos MIME de verdade); sem isso, só loga. |
| `LOG_LEVEL` | não (default `info`) | `debug`/`info`/`warn`/`error`. |
| `LOG_FORMAT` | não (default por `APP_ENV`) | `json` (produção) ou `text` (desenvolvimento). |
| `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST` | não (defaults `5`/`10`) | Limite de requisições por usuário nas rotas de escrita. |

---

## API — resumo de rotas

Contrato completo em **[`api/openapi.yaml`](api/openapi.yaml)** (OpenAPI 3.0 — abra em [editor.swagger.io](https://editor.swagger.io) ou qualquer visualizador Swagger/Redoc para explorar interativamente). Resumo abaixo.

Todas sob `/api/v1`, exigem `Authorization: Bearer <JWT>` exceto `/health` e `/metrics`. `GET /contratos` e `GET /processos` aceitam `?pagina=&tamanho=` (defaults 1/20, máximo 100 por página).

> **Nota:** as respostas JSON usam PascalCase (`NumeroContrato`, não `numero_contrato`) — os models não têm tags `json` explícitas hoje, exceto onde indicado (`CaminhoStorage`, nunca serializado). Inconsistência conhecida com os corpos de requisição (snake_case); documentada como está, não corrigida nesta rodada para não quebrar clientes que já existirem.

| Rota | Método | Permissão |
|---|---|---|
| `/health` | GET | pública |
| `/metrics` | GET | pública (restrinja por rede em produção) |
| `/me` | GET | autenticado |
| `/kanban/etapas`, `/kanban/tipos-documento` | GET | autenticado |
| `/contratos`, `/contratos/:id` | GET | autenticado |
| `/contratos` | POST | fiscal |
| `/contratos/:id` | PATCH | fiscal |
| `/contratos/:id/encerrar` | POST | fiscal |
| `/processos`, `/processos/:id` | GET | autenticado |
| `/processos` | POST | fiscal |
| `/processos/:id/avancar` | POST | fiscal (valida checklist) |
| `/processos/:id/concluir` | POST | fiscal |
| `/processos/:id/documentos` | GET | autenticado |
| `/processos/:id/documentos` | POST (multipart) | fiscal |
| `/processos/:id/relatorio` | GET | autenticado (retorna PDF) |
| `/admin/users`, `/admin/users/:id` | GET | admin |
| `/admin/users/:id` | PATCH | admin |

Rotas de escrita (grupo "fiscal") têm rate limit por usuário autenticado (`RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`).

`IsFiscal`/`IsAdmin` são flags na tabela `users`. **Não há bootstrap automático de admin** — o primeiro precisa ser promovido manualmente: `UPDATE users SET is_admin = true WHERE email = '...'`.

Um contrato `Encerrar`ado (`Ativo=false`) não aceita novos `POST /processos` — o histórico continua consultável.

---

## Checklist de produção

- [x] Testes automatizados (unitários + integração, com `-race`)
- [x] Graceful shutdown (`SIGINT`/`SIGTERM`, drena requisições em andamento)
- [x] `gin.ReleaseMode` em produção, trusted proxies restritos, CORS fail-closed
- [x] `/health` verificando a dependência real (Postgres)
- [x] Migrations versionadas e reversíveis
- [x] CI (lint + vet + testes com Postgres real + build)
- [x] Imagem Docker mínima (`scratch`, ~38MB, usuário não-root)
- [x] Validação de `iss` (obrigatório) e `aud` (opcional) do JWT, além de assinatura + expiração
- [x] Paginação em `GET /contratos` e `GET /processos`
- [x] PDF real do Relatório de Pagamento (não o layout oficial da prefeitura — ver limitações)
- [x] Anexos MIME de verdade (multipart, lidos do storage) no e-mail à empresa contratada
- [x] Logging estruturado (`slog`, JSON em produção) com request ID correlacionando os logs de uma requisição
- [x] Métricas Prometheus (`/metrics`): requisições HTTP e transições do Kanban
- [x] Rate limiting por usuário nas rotas de escrita
- [x] Testes de handler (camada HTTP: binding, status codes, multipart)
- [x] Documentação OpenAPI/Swagger (`api/openapi.yaml`)
- [x] Revisão de segurança manual das áreas sensíveis (auth, upload, SMTP, rate limit) — achou e corrigiu um path traversal real (ver `git log`, commit "fix: corrige path traversal...")
- [ ] Tracing distribuído (OpenTelemetry)
- [ ] Rate limiting compartilhado entre réplicas (hoje é em memória, por instância — ver limitações)
- [ ] `security-review` automatizado no CI (hoje depende de diff contra `origin/HEAD`, que só existe depois do primeiro push)

---

## Limitações conhecidas do domínio

- **`Contrato.ContratadaEmail`**: não estava na especificação original de campos, mas é necessário para a notificação por e-mail da Etapa 3 — adicionado como campo opcional.
- **Relatório de Pagamento**: gera um PDF funcional com todas as tags do domínio preenchidas, em layout simples — não é o modelo oficial da prefeitura (nenhum template real foi fornecido). Trocar é uma mudança isolada em `internal/service/relatorio_service.go`.
- **Rate limiting em memória**: o limite por usuário é aplicado por instância do processo. Com múltiplas réplicas atrás de um load balancer, cada réplica tem seu próprio contador — o limite efetivo por usuário escala com o número de réplicas. Um backend compartilhado (Redis) resolveria isso, mas não foi adicionado antes de haver um deploy multi-réplica de verdade.
- **`Contrato.Ativo` e `gorm:"default:true"`**: o GORM omite essa coluna do `INSERT` quando o valor Go é `false` (zero-value), deixando o Postgres aplicar o `DEFAULT true` — `Create` com `Ativo: false` explícito NÃO funciona como se espera. `ContratoService.Criar` sempre define `Ativo: true` explicitamente; para encerrar, use `Update` (via `ContratoService.Encerrar`), que não tem essa omissão. Documentado no próprio campo em `internal/models/contrato.go`.
- **Selene não controla saldo orçamentário** — isso é responsabilidade dos sistemas corporativos da prefeitura (Agile); o sistema foca 100% em compliance documental.

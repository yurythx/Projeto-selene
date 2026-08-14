# 🌙 Projeto Selene

Sistema de compliance documental e Kanban para fiscalização de contratos públicos municipais. Substitui fluxos informais por um funil rígido de 6 etapas onde cada avanço exige a validação de um checklist de documentos, específico por etapa e por tipo de contrato.

Backend em Go (Gin + GORM + Postgres), autenticação via Keycloak (OIDC/JWT), Clean Architecture.

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

```bash
cp .env.example .env
# edite KEYCLOAK_JWKS_URL com a URL real do seu realm Keycloak
docker compose up --build
```

Sobe Postgres + API. A API roda as migrations e o seed automaticamente no boot. Health check em `http://localhost:8080/health`.

### Opção 2 — Go local

Requer Go 1.25+ e um Postgres acessível.

```bash
cp .env.example .env   # ajuste DB_* e KEYCLOAK_JWKS_URL
go run ./cmd/api
```

---

## Testes

```bash
go test ./...
```

Dois níveis:
- **Unitários** (`internal/service/checklist_test.go`, `contrato_service_test.go`): usam dublês em memória (fakes) dos repositories, não tocam banco, sempre rodam.
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
| `APP_ENV` | não (default `development`) | `production` ativa `gin.ReleaseMode`. |
| `SERVER_PORT` | não (default `8080`) | Porta HTTP. |
| `CORS_ALLOWED_ORIGINS` | não (default: nenhuma) | Origens autorizadas a chamar a API pelo navegador. |
| `TRUSTED_PROXIES` | não (default: nenhum) | IPs/CIDRs confiáveis para resolver `X-Forwarded-For`. |
| `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | **sim** | Credenciais do Postgres. |
| `DB_PORT`, `DB_SSLMODE` | não (defaults `5432`/`disable`) | |
| `KEYCLOAK_JWKS_URL` | **sim** | Endpoint JWKS do realm Keycloak. |
| `STORAGE_DIR` | não (default `./storage`) | Onde os documentos anexos são gravados. |
| `SMTP_*` | não | Envio real do pacote à empresa contratada (Etapa 3); sem isso, só loga. |

---

## API — resumo de rotas

Todas sob `/api/v1`, exigem `Authorization: Bearer <JWT>` exceto `/health`.

| Rota | Método | Permissão |
|---|---|---|
| `/health` | GET | pública |
| `/me` | GET | autenticado |
| `/kanban/etapas`, `/kanban/tipos-documento` | GET | autenticado |
| `/contratos`, `/contratos/:id` | GET | autenticado |
| `/contratos` | POST | fiscal |
| `/processos`, `/processos/:id` | GET | autenticado |
| `/processos` | POST | fiscal |
| `/processos/:id/avancar` | POST | fiscal (valida checklist) |
| `/processos/:id/concluir` | POST | fiscal |
| `/processos/:id/documentos` | GET | autenticado |
| `/processos/:id/documentos` | POST (multipart) | fiscal |
| `/processos/:id/relatorio` | GET | autenticado |
| `/admin/users`, `/admin/users/:id` | GET | admin |
| `/admin/users/:id` | PATCH | admin |

`IsFiscal`/`IsAdmin` são flags na tabela `users`. **Não há bootstrap automático de admin** — o primeiro precisa ser promovido manualmente: `UPDATE users SET is_admin = true WHERE email = '...'`.

---

## Checklist de produção

- [x] Testes automatizados (unitários + integração)
- [x] Graceful shutdown (`SIGINT`/`SIGTERM`, drena requisições em andamento)
- [x] `gin.ReleaseMode` em produção, trusted proxies restritos, CORS fail-closed
- [x] `/health` verificando a dependência real (Postgres)
- [x] Migrations versionadas e reversíveis
- [x] CI (lint + vet + testes com Postgres real + build)
- [x] Imagem Docker mínima (`scratch`, ~38MB, usuário não-root)
- [ ] Autenticação: validação de `iss`/`aud` do JWT (hoje só assinatura + expiração)
- [ ] Paginação em `GET /contratos` e `GET /processos`
- [ ] PDF real do Relatório de Pagamento (hoje é HTML funcional, não o layout oficial da prefeitura)
- [ ] Anexos de verdade (multipart/MIME) no e-mail à empresa contratada (hoje é só texto com a lista)
- [ ] Logging estruturado (`slog`) no lugar dos `log.Printf` pontuais
- [ ] Observabilidade (métricas, tracing)
- [ ] Rate limiting

---

## Limitações conhecidas do domínio

- **`Contrato.ContratadaEmail`**: não estava na especificação original de campos, mas é necessário para a notificação por e-mail da Etapa 3 — adicionado como campo opcional.
- **Relatório de Pagamento**: gera um HTML funcional com todas as tags do domínio preenchidas, não o PDF no template oficial da prefeitura (nenhum modelo real foi fornecido). Trocar é uma mudança isolada em `internal/service/relatorio_service.go`.
- **Selene não controla saldo orçamentário** — isso é responsabilidade dos sistemas corporativos da prefeitura (Agile); o sistema foca 100% em compliance documental.

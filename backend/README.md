# 🌙 Projeto Selene — Backend

API em Go (Gin + GORM + Postgres), autenticação via Keycloak (OIDC/JWT) **e** login local (usuário/senha), Clean Architecture. Ver [../README.md](../README.md) para a visão geral do monorepo (backend + frontend) e a lista de módulos.

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
5. **Relatório de Pagamento** — exige as certidões (CNDs, Simples Nacional, Extrato do Empenho) + o relatório assinado reanexado; contratos `SERVICO` exigem adicionalmente Planilha de Medição e Boleto DAM; contratos com `ExigeFiscalizacaoTerceirizacao=true` exigem também os documentos trabalhistas mensais da IN SCL Nº 04/2021 Art.9º-XXXII (Comprovante de Pagamento de Salário, Protocolo GFIP, Guia GRF/GPS, Relação de Trabalhadores/SEFIP) — ver a seção [SGF-Rondonópolis](#sgf-rondonópolis-in-scl-012019-e-042021).

   Esses mesmos documentos condicionais também são **restritos** por tipo de contrato: `TipoDocumento.RestritoTipoObjeto`/`RestritoTerceirizacao` (semeados junto com `tipos_documento`, migration `000010_restricao_tipo_documento`) marcam quais tipos só se aplicam a `SERVICO`/a contratos com terceirização — `service.TipoDocumentoAplicavel` faz valer isso tanto no upload (`DocumentoService.Upload` rejeita com 400 se o tipo não se aplicar ao contrato do processo) quanto no select do frontend (que só oferece os tipos aplicáveis). Documentos sem restrição continuam disponíveis em qualquer tipo de contrato.
6. **Contabilidade (Liquidação e Pagamento)** — etapa final; `POST /processos/:id/concluir` marca como pago.

Toda transição é atômica com a gravação em `kanban_logs` (mesma transação SQL) — ver `internal/service/kanban_service.go`.

### Módulos além do núcleo

Cada um tem seu próprio trio repository/service/handler, seguindo exatamente o padrão descrito acima — nenhum altera o funil do Kanban em si, todos são extensões aditivas:

- **Radar de Alertas** (`radar_service.go`) — calcula, na hora da requisição (sem job/cron), alertas de vigência de contrato, certidão vencida/vencendo e processo parado há muito tempo na mesma etapa.
- **Gerador de Documentos Legais** (`gerador_documentos_service.go`) — Notificação de Descumprimento, Atesto (com QR code de verificação pública via `GET /verificar/:codigo`, rota sem autenticação de propósito) e Minuta de Aditivo, registrados em `documentos_emitidos`.
- **Vistorias de Campo** (`vistoria_service.go`) — registro fotográfico com geolocalização opcional, dedupe de foto por SHA-256 (mesma lógica de `DocumentoService.Upload`), Relatório de Campo em PDF.
- **Dossiê do Fornecedor** (`fornecedor_service.go`) — só leitura: agrega `Contrato`/`ProcessoPagamento`/`KanbanLog`/`DocumentoEmitido` por CNPJ, sem tabela própria.
- **Login local** (`auth_service.go`, `internal/localauth/`) — autenticação usuário/senha como alternativa ao Keycloak. Ver a seção [Autenticação](#autenticação) abaixo.
- **SGF-Rondonópolis** (`designacao_service.go`, `empenho_service.go`, `ocorrencia_service.go`, `fiscalizacao_service.go`) — adequação estrita à IN SCL Nº 01/2019 e IN SCL Nº 04/2021. Ver a seção [SGF-Rondonópolis](#sgf-rondonópolis-in-scl-012019-e-042021) abaixo.

---

## Autenticação

Dois emissores de token, tratados de forma idêntica pelo resto da API a partir daí (`middleware.NewAuthMiddleware` tenta Keycloak primeiro, cai para local se falhar — ver `internal/middleware/auth.go`):

- **Keycloak** (OIDC/JWKS) — o caminho institucional. Usuário é provisionado just-in-time (`UserService.FindOrCreateByKeycloakID`) no primeiro login.
- **Login local** (`POST /auth/login`, público, rate-limitado por IP) — usuário/senha com bcrypt, token RS256 assinado por uma chave RSA efêmera gerada em memória a cada boot do processo (`internal/localauth`). **Limitação conhecida**: sessões locais não sobrevivem a um restart do backend (Keycloak não é afetado — chaves diferentes, infra separada). Contas locais são criadas só por um admin (`POST /admin/users/local`) — sem autocadastro público — e nascem com `must_change_password=true`, forçando a troca no primeiro login (aplicado no frontend, ver `frontend/README.md`).

`models.User.KeycloakID` é `*string` (nullable) e `PasswordHash` também — exatamente um dos dois é preenchido por usuário, nunca os dois.

**Configuração de Keycloak editável em runtime**: `KEYCLOAK_JWKS_URL`/`KEYCLOAK_ISSUER`/`KEYCLOAK_AUDIENCE` são só o valor de *boot* — um admin pode salvar Client ID/Secret/Issuer/Audience pela tela Configurações → Keycloak/SSO do frontend (`PUT /admin/config/keycloak`), persistidos na tabela `keycloak_config` (singleton, migration `000012`). A troca é aplicada IMEDIATAMENTE a este backend via `middleware.AuthMiddlewareState.Reload` (sem reiniciar o processo) — o novo issuer é testado (fetch do JWKS) *antes* de salvar; se não responder, nada muda (fail-closed). `GET /internal/keycloak-config` (gated por `INTERNAL_API_SECRET`, não por JWT) expõe a config completa — incluindo o Client Secret em texto puro — só pro frontend Next.js montar o provider Keycloak do NextAuth em runtime; nunca alcançável com um token de usuário comum. Ver `internal/service/keycloak_config_service.go` e `internal/middleware/auth.go`.

**Integração com o Diário Oficial (Configurações → Diário Oficial no frontend)**: mesmo padrão de singleton editável em runtime do Keycloak acima (tabela `diario_oficial_config`, migration `000013`), mas **ESTRUTURA GENÉRICA** — decisão de escopo confirmada com o usuário, a API real do Diário Oficial da cidade ainda não está definida/documentada. `PUT /admin/config/diario-oficial` guarda URL base + chave de API (nunca devolvida); `POST /admin/config/diario-oficial/testar` faz uma requisição real contra a URL configurada e reporta o que aconteceu (qualquer resposta HTTP, mesmo 404/401, conta como "conectou" — só erro de rede conta como falha, já que não há um health-check garantido numa API de terceiro); `GET /admin/diario-oficial/contratos?nome=&cpf=&data=` proxeia a busca pra API externa e devolve o JSON decodificado sem validar contra um schema fixo. O contrato assumido de request/response (`GET {base_url}/contratos?nome=&cpf=&data=`, header `Authorization: Bearer {api_key}`) está documentado no comentário de escopo no topo de `internal/service/diario_oficial_service.go` — é o único lugar a ajustar quando a API real existir. Diferente do Keycloak, `Salvar` NÃO testa a conexão antes de persistir (uma API externa fora do ar temporariamente não deveria invalidar a configuração).

---

## Rodando localmente

### Opção 1 — Docker Compose (recomendado)

O `docker-compose.yml` fica na **raiz do monorepo** (orquestra backend + Postgres, e futuramente o frontend também) — rode a partir de lá, não daqui de dentro de `backend/`:

```bash
cp backend/.env.example backend/.env
# edite backend/.env: KEYCLOAK_JWKS_URL, KEYCLOAK_ISSUER, etc.
make up            # a partir da raiz do repo (atalho para `docker compose up --build`)
```

Sobe Postgres + Redis + Jaeger + backend. A API roda as migrations e o seed automaticamente no boot. Health check em `http://localhost:8080/health`, traces em `http://localhost:16686` (Jaeger UI).

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
| `KEYCLOAK_JWKS_URL` | **sim** | Endpoint JWKS do realm Keycloak — só o valor de BOOT; um admin pode substituir em runtime por Configurações → Keycloak/SSO (ver abaixo). |
| `KEYCLOAK_ISSUER` | **sim** | Valor exato esperado no claim `iss` do JWT — mesma ressalva acima. |
| `KEYCLOAK_AUDIENCE` | não | Valor esperado no claim `aud`; vazio = não valida. |
| `INTERNAL_API_SECRET` | **sim** | Segredo compartilhado com o frontend (mesmo valor nos dois) que autentica `GET /internal/keycloak-config` — a única rota server-to-server sem JWT de usuário desta API. Ver abaixo. |
| `STORAGE_DIR` | não (default `./storage`) | Onde os documentos anexos são gravados. |
| `SMTP_*` | não | Envio real do pacote à empresa contratada (Etapa 3, com anexos MIME de verdade); sem isso, só loga. |
| `LOG_LEVEL` | não (default `info`) | `debug`/`info`/`warn`/`error`. |
| `LOG_FORMAT` | não (default por `APP_ENV`) | `json` (produção) ou `text` (desenvolvimento). |
| `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST` | não (defaults `5`/`10`) | Limite de requisições por usuário nas rotas de escrita. |
| `REDIS_ADDR` | não (default: nenhum) | `"host:porta"` — se definido, o rate limit usa Redis compartilhado (vale entre réplicas); vazio = em memória, por instância. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | não (default: nenhum) | `"host:porta"` de um coletor OTLP/HTTP (Jaeger, Tempo); vazio = tracing desabilitado. |
| `OTEL_SERVICE_NAME` | não (default `projeto-selene-backend`) | Nome do serviço nos traces exportados. |

---

## API — resumo de rotas

Contrato completo em **[`api/openapi.yaml`](api/openapi.yaml)** (OpenAPI 3.0 — abra em [editor.swagger.io](https://editor.swagger.io) ou qualquer visualizador Swagger/Redoc para explorar interativamente). Resumo abaixo.

Todas sob `/api/v1`, exigem `Authorization: Bearer <JWT>` exceto `/health` e `/metrics`. `GET /contratos` e `GET /processos` aceitam `?pagina=&tamanho=` (defaults 1/20, máximo 100 por página).

> **Nota:** as respostas JSON usam PascalCase (`NumeroContrato`, não `numero_contrato`) — os models não têm tags `json` explícitas hoje, exceto onde indicado (`CaminhoStorage`, nunca serializado). Inconsistência conhecida com os corpos de requisição (snake_case); documentada como está, não corrigida nesta rodada para não quebrar clientes que já existirem.

| Rota | Método | Permissão |
|---|---|---|
| `/health` | GET | pública |
| `/metrics` | GET | pública (restrinja por rede em produção) |
| `/verificar/:codigo` | GET | pública (autenticidade de documento emitido, QR code) |
| `/auth/login` | POST | pública (rate-limitada por IP) |
| `/me` | GET | autenticado |
| `/auth/trocar-senha` | POST | autenticado |
| `/kanban/etapas`, `/kanban/tipos-documento` | GET | autenticado |
| `/radar` | GET | autenticado |
| `/contratos`, `/contratos/:id` | GET | autenticado |
| `/contratos` | POST | fiscal |
| `/contratos/:id` | PATCH | fiscal |
| `/contratos/:id/encerrar` | POST | fiscal |
| `/contratos/:id/notificacao`, `/contratos/:id/minuta-aditivo` | POST | fiscal (retornam PDF) |
| `/processos`, `/processos/:id` | GET | autenticado (`:id` inclui a leitura SGF decorada — ver abaixo) |
| `/processos` | POST | fiscal |
| `/processos/:id/avancar` | POST | fiscal (valida checklist **e** ocorrência aberta) |
| `/processos/:id/concluir` | POST | fiscal |
| `/processos/:id/documentos` | GET | autenticado |
| `/processos/:id/documentos` | POST (multipart) | fiscal |
| `/processos/:id/relatorio` | GET | autenticado (retorna PDF) |
| `/processos/:id/atesto` | POST | fiscal (retorna PDF) |
| `/processos/:id/vistorias` | GET | autenticado |
| `/processos/:id/vistorias` | POST | fiscal |
| `/vistorias/:id/fotos` | POST (multipart) | fiscal |
| `/vistorias/:id/relatorio` | GET | autenticado (retorna PDF) |
| `/fornecedores`, `/fornecedores/:cnpj` | GET | autenticado |
| `/contratos/:id/designacoes` | GET | autenticado |
| `/contratos/:id/designacoes` | POST | fiscal |
| `/contratos/:id/empenhos` | GET | autenticado |
| `/contratos/:id/empenhos` | POST | fiscal |
| `/empenhos/:id` | GET | autenticado (inclui saldo reconstruído) |
| `/empenhos/:id/movimentacoes` | POST | fiscal |
| `/processos/:id/ocorrencias` | GET | autenticado |
| `/processos/:id/ocorrencias` | POST | fiscal |
| `/ocorrencias/:id/notificar`, `/tratar`, `/regularizar` | POST | fiscal |
| `/servidores` | GET | autenticado (projeção mínima ID/Nome/Email — **não** é admin-only, ver abaixo) |
| `/admin/users`, `/admin/users/:id` | GET | admin |
| `/admin/users/:id` | PATCH | admin |
| `/admin/users/local` | POST | admin |

Rotas de escrita (grupo "fiscal") têm rate limit por usuário autenticado (`RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`).

`IsFiscal`/`IsAdmin` são flags na tabela `users`. **Não há bootstrap automático de admin** — o primeiro precisa ser promovido manualmente: `UPDATE users SET is_admin = true WHERE email = '...'`.

Um contrato `Encerrar`ado (`Ativo=false`) não aceita novos `POST /processos` — o histórico continua consultável.

---

## SGF-Rondonópolis (IN SCL 01/2019 e 04/2021)

Adequação estrita a duas Instruções Normativas da Prefeitura de Rondonópolis — Matriz Normativa completa (artigo por artigo) em `.claude/plans/projeto-selene-rippling-kite.md`. Extensão 100% aditiva: nenhuma tabela/coluna/rota anterior mudou de nome ou comportamento.

- **`PortariaDesignacao`** (`internal/service/designacao_service.go`) — histórico auditável de designação de fiscal/suplente/gestor/fiscal setorial por contrato (IN01 Art.4º-I/Art.6º; IN04 Art.4º-I/Art.10). `Contrato.FiscalID` continua existindo como cache de leitura rápida do fiscal ativo; esta tabela é a fonte de verdade. `GET /servidores` (`UserHandler.ListarServidores`) expõe uma projeção mínima (ID/Nome/Email) de todos os usuários, aberta a qualquer autenticado — não admin-only como `/admin/users` — pra popular o seletor de servidor do formulário "Nova designação" no frontend.
- **`Empenho` / `MovimentacaoEmpenho`** (`empenho_service.go`) — acompanhamento **paralelo/informativo** de saldo (IN01 Art.5º-VIII; IN04 Art.5º-XXII). **Não é a fonte de verdade orçamentária** — essa continua sendo exclusiva dos sistemas corporativos da prefeitura (mesma decisão já documentada para `Contrato`). Saldo sempre reconstruído do histórico de movimentações, nunca denormalizado.
- **`Ocorrencia`** (`ocorrencia_service.go`) — registro/tratativa de ocorrências (IN01 Art.3º-III/Art.5º-IV,IX; IN04 Art.3º-VIII/Art.5º-VIII,XVI), ciclo linear `REGISTRADA → NOTIFICADA → EM_TRATAMENTO → REGULARIZADA`. Uma ocorrência não regularizada **bloqueia de verdade** `POST /processos/:id/avancar` (`FiscalizacaoService.VerificarAvancoPermitido`, chamado pelo handler antes de `KanbanService.AvancarEtapa`) — regra de Camada 2 do SGF, não da norma, confirmada com o time do projeto.
- **`FiscalizacaoService`** (`fiscalizacao_service.go`) — computa, só na leitura de `GET /processos/:id` (nunca persiste), três campos extras sobre o `ProcessoPagamento` de sempre: `estado_fiscalizacao` (rótulo de Camada 2 derivado da etapa Kanban), `acao_ou_espera` (`ACAO_FISCAL`/`ESPERA_EXTERNA`) e `allowed_actions` (vocabulário fechado que o frontend usa pra decidir quais botões mostrar).
- **`Contrato.ExigeFiscalizacaoTerceirizacao`** (Camada 2, default `false`) — marca contratos de mão de obra terceirizada, sujeitos à IN04 (mais estreita que `TipoObjeto=SERVICO`: nem todo contrato de serviço é terceirização de mão de obra). Quando `true`, `checklist.go` acrescenta à Etapa 5 os documentos mensais do Art.9º-XXXII alíneas a/b.1/b.2/b.3 (`checklistCondicionalTerceirizacao`) — um subconjunto deliberadamente estreito do Anexo III (42 itens), que é um questionário de autoavaliação do fiscal (SIM/NÃO/NÃO APLICA) e não tem formato de checklist de anexo; só os itens genuinamente documentais entraram no modelo.

**Lacunas conhecidas, deixadas de propósito para uma fase futura** (ver o plano): o ramo de escalonamento da IN04 Art.5º-XVII (`ESCALADA`) não está modelado; os papéis "Unidade Administrativa de Fiscalização" e "Coordenador de Fiscal" não têm representação no domínio; os Anexos II (16 itens) e III (42 itens) completos da IN04 são questionários de autoavaliação do fiscal, estruturalmente diferentes do checklist de anexo de documentos do Selene — não foram force-fitados no modelo atual, só o subconjunto documental do Art.9º-XXXII foi incorporado a `checklist.go`.

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
- [x] Tracing distribuído (OpenTelemetry — HTTP via `otelgin`, GORM via `otelgorm`, spans manuais no `KanbanService.AvancarEtapa`), validado no Jaeger
- [x] Rate limiting compartilhado entre réplicas (Redis, GCRA via `go-redis/redis_rate`, fail-open se o Redis cair) — cai para o limiter em memória se `REDIS_ADDR` não estiver configurado
- [x] No máximo um documento anexado de cada tipo por processo (índice único com backfill seguro, migration `000011`) — reenviar um tipo já anexado exige excluir o anterior primeiro
- [x] Checklist de avanço de etapa cumulativo entre etapas (`RequisitosAcumulados`) — um documento obrigatório de uma etapa anterior, se excluído, volta a bloquear o avanço em qualquer etapa seguinte, não só a original; upload também rejeita (400) um tipo que ainda não faz parte do checklist até a etapa atual
- [x] Configuração de Keycloak editável em runtime (Configurações → Keycloak/SSO no frontend) — `AuthMiddlewareState.Reload` troca a validação de token ativa sem reiniciar o processo; novo issuer/JWKS testado antes de persistir (fail-closed)
- [x] Cache HTTP agressivo em documentos anexados (`ETag` do hash SHA-256 + `Cache-Control: immutable`, streaming direto do disco via `http.ServeContent`) — documentos são imutáveis por ID, então reabrir o mesmo arquivo não retransmite nada
- [x] `security-review` automatizado no CI — job `security-review` em `.github/workflows/ci.yml`, só em `pull_request` (a action oficial `anthropics/claude-code-security-review` faz diff contra o SHA base do PR, `fetch-depth: 2`; não depende de `origin/HEAD` já resolvido localmente como uma revisão manual antes do primeiro push dependeria). Exige o secret de repositório `CLAUDE_API_KEY` (Settings → Secrets and variables → Actions) — sem ele o job falha de propósito, em vez de pular a revisão silenciosamente.
- [x] Integração com o Diário Oficial (Configurações → Diário Oficial no frontend) — estrutura genérica (config + teste de conexão + busca por nome/CPF/data), decisão de escopo confirmada com o usuário: a API real da cidade ainda não está definida, ver `internal/service/diario_oficial_service.go`

---

## Limitações conhecidas do domínio

- **`Contrato.ContratadaEmail`**: não estava na especificação original de campos, mas é necessário para a notificação por e-mail da Etapa 3 — adicionado como campo opcional.
- **Relatório de Pagamento**: gera um PDF funcional com todas as tags do domínio preenchidas, em layout simples — não é o modelo oficial da prefeitura (nenhum template real foi fornecido). Trocar é uma mudança isolada em `internal/service/relatorio_service.go`.
- **Rate limiting em memória**: o limite por usuário é aplicado por instância do processo. Com múltiplas réplicas atrás de um load balancer, cada réplica tem seu próprio contador — o limite efetivo por usuário escala com o número de réplicas. Um backend compartilhado (Redis) resolveria isso, mas não foi adicionado antes de haver um deploy multi-réplica de verdade.
- **`Contrato.Ativo` e `gorm:"default:true"`**: o GORM omite essa coluna do `INSERT` quando o valor Go é `false` (zero-value), deixando o Postgres aplicar o `DEFAULT true` — `Create` com `Ativo: false` explícito NÃO funciona como se espera. `ContratoService.Criar` sempre define `Ativo: true` explicitamente; para encerrar, use `Update` (via `ContratoService.Encerrar`), que não tem essa omissão. Documentado no próprio campo em `internal/models/contrato.go`.
- **Selene não controla saldo orçamentário** — isso é responsabilidade dos sistemas corporativos da prefeitura (Agile); o sistema foca 100% em compliance documental. O `Empenho`/`MovimentacaoEmpenho` do SGF-Rondonópolis (ver seção acima) não muda essa decisão: é um acompanhamento paralelo/informativo alimentado manualmente pelo fiscal, nunca a fonte de verdade — em caso de divergência, o sistema corporativo prevalece.
- **Login local — sessão não sobrevive a restart do backend**: a chave RSA que assina os tokens locais é gerada em memória a cada boot (`internal/localauth`), de propósito (evita gerenciar segredo persistente pra uma funcionalidade secundária); sessões via Keycloak não são afetadas.

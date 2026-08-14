# 🌙 Projeto Selene

Sistema de compliance documental e Kanban para fiscalização de contratos públicos municipais. Substitui fluxos informais por um funil rígido de 6 etapas onde cada avanço exige a validação de um checklist de documentos, específico por etapa e por tipo de contrato.

Monorepo com dois projetos independentes:

```
projeto-selene/
├── backend/     → API Go (Gin + GORM + Postgres), autenticação Keycloak/JWT — ver backend/README.md
├── frontend/    → cliente Next.js (App Router, Auth.js + Keycloak) — ver frontend/README.md
├── docker-compose.yml      → dev: sobe o stack inteiro (Postgres, Redis, Jaeger, backend e frontend)
└── docker-compose.prod.yml → produção: só o frontend é publicado (BFF) — ver DEPLOY.md
```

## Início rápido

```bash
cp .env.example .env
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local
# edite .env / backend/.env / frontend/.env.local com os dados do seu Keycloak
# (ver frontend/README.md para o passo a passo de criação do client)
make up
```

Sobe Postgres, Redis, Jaeger, backend e frontend via `docker-compose.yml`. Health check do backend em `http://localhost:8080/health`; frontend em `http://localhost:3000`.

Veja `make help` na raiz para os atalhos de orquestração do monorepo, [`backend/README.md`](backend/README.md) para tudo sobre a API e [`frontend/README.md`](frontend/README.md) para o cliente Next.js.

## Documentação por projeto

- **[backend/README.md](backend/README.md)** — arquitetura (Clean Architecture), o funil do Kanban, como rodar/testar localmente, migrations, referência de variáveis de ambiente, rotas da API, checklist de produção e limitações conhecidas.
- **[frontend/README.md](frontend/README.md)** — stack, arquitetura BFF, setup, passo a passo do client Keycloak, testes.
- **[DEPLOY.md](DEPLOY.md)** — runbook de deploy em produção (`selene.papermoon.cloud`, atrás do Cloudflare Tunnel).

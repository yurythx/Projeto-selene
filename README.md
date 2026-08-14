# 🌙 Projeto Selene

Sistema de compliance documental e Kanban para fiscalização de contratos públicos municipais. Substitui fluxos informais por um funil rígido de 6 etapas onde cada avanço exige a validação de um checklist de documentos, específico por etapa e por tipo de contrato.

Monorepo com dois projetos independentes:

```
projeto-selene/
├── backend/     → API Go (Gin + GORM + Postgres), autenticação Keycloak/JWT — ver backend/README.md
├── frontend/    → cliente Next.js (ainda não implementado) — ver frontend/README.md
└── docker-compose.yml → sobe o stack inteiro (Postgres + backend, e futuramente o frontend)
```

## Início rápido

```bash
cp backend/.env.example backend/.env
# edite backend/.env: KEYCLOAK_JWKS_URL, KEYCLOAK_ISSUER, etc.
make up
```

Sobe Postgres + backend via `docker-compose.yml`. Health check em `http://localhost:8080/health`.

Veja `make help` na raiz para os atalhos de orquestração do monorepo, e [`backend/README.md`](backend/README.md) para tudo sobre a API: arquitetura, testes, migrations, variáveis de ambiente e rotas.

## Documentação por projeto

- **[backend/README.md](backend/README.md)** — arquitetura (Clean Architecture), o funil do Kanban, como rodar/testar localmente, migrations, referência de variáveis de ambiente, rotas da API, checklist de produção e limitações conhecidas.
- **[frontend/README.md](frontend/README.md)** — ainda não implementado.

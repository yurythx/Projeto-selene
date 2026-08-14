# Frontend — Projeto Selene

Ainda não implementado.

Pela documentação de arquitetura original, este será o cliente Next.js que consome a API do [`backend/`](../backend/README.md): autenticação via Keycloak (OIDC), quadro Kanban das 6 etapas de compliance, upload de documentos e visualização/impressão do Relatório de Pagamento.

Quando este frontend for iniciado:
- `docker-compose.yml` (na raiz) já tem um serviço `frontend` comentado, pronto para ser habilitado.
- A API já expõe CORS configurável via `CORS_ALLOWED_ORIGINS` (ver `backend/.env.example`) — inclua a origem deste app aí.
- Endpoints e contrato da API: ver a seção "API — resumo de rotas" em [`backend/README.md`](../backend/README.md).

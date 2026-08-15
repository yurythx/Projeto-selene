# 🌙 Projeto Selene

Sistema de compliance documental e Kanban para fiscalização de contratos públicos municipais. Substitui fluxos informais por um funil rígido de 6 etapas onde cada avanço exige a validação de um checklist de documentos, específico por etapa e por tipo de contrato.

## O que o sistema cobre hoje

- **Núcleo (Kanban)**: cadastro de contratos, funil de 6 etapas com checklist obrigatório por etapa/tipo de objeto, upload de documentos com deduplicação por hash, Relatório de Pagamento em PDF.
- **Autenticação dupla**: SSO institucional via Keycloak (OIDC) **e** login tradicional (e-mail/senha) como alternativa — contas locais são criadas só por um administrador, sem autocadastro público. Do ponto de vista do resto da API os dois tipos de sessão são indistinguíveis (mesmo formato de token).
- **Radar de Alertas**: badges de vigência de contrato, certidão vencida/vencendo e processo parado, calculados na hora — painel consolidado em `/radar`.
- **Gerador de Documentos Legais**: Notificação de Descumprimento, Atesto (com QR code de verificação pública) e Minuta de Aditivo, gerados em PDF a partir do histórico do contrato/processo.
- **Vistorias de Campo**: registro fotográfico com geolocalização (mobile-first) e Relatório de Campo em PDF.
- **Dossiê do Fornecedor**: consolidação por CNPJ de todos os contratos da mesma empresa, histórico de notificações e Score de Pontualidade.
- **SGF-Rondonópolis**: adequação estrita à IN SCL Nº 01/2019 e IN SCL Nº 04/2021 (Prefeitura de Rondonópolis) — histórico auditável de designação de fiscal/gestor/fiscal setorial, acompanhamento paralelo/informativo de saldo de empenho, registro/tratativa de ocorrências (que passam a bloquear de verdade o avanço de etapa do Kanban enquanto não regularizadas), e um flag de contrato (`ExigeFiscalizacaoTerceirizacao`) que acrescenta os documentos trabalhistas mensais de mão de obra terceirizada (IN04 Art.9º-XXXII) ao checklist da Etapa 5. Extensão 100% aditiva sobre o Kanban existente — nenhuma rota/campo anterior mudou de comportamento. Ver o plano completo, com a Matriz Normativa citando artigo por artigo das duas INs, em `.claude/plans/projeto-selene-rippling-kite.md`.

Monorepo com dois projetos independentes:

```
projeto-selene/
├── backend/     → API Go (Gin + GORM + Postgres), Keycloak + login local — ver backend/README.md
├── frontend/    → cliente Next.js (App Router, Auth.js + Keycloak/Credentials) — ver frontend/README.md
├── docker-compose.yml      → dev: sobe o stack inteiro (Postgres, Redis, Jaeger, backend e frontend)
└── docker-compose.prod.yml → produção: só o frontend é publicado (BFF) — ver DEPLOY.md
```

## Início rápido

```bash
cp .env.example .env
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local
# edite .env / backend/.env / frontend/.env.local com os dados do seu Keycloak
# (ver frontend/README.md para o passo a passo de criação do client — o
# login local não precisa de nenhuma configuração extra, já funciona
# out-of-the-box em paralelo ao Keycloak)
make up
```

Sobe Postgres, Redis, Jaeger, backend e frontend via `docker-compose.yml`. Health check do backend em `http://localhost:8080/health`; frontend em `http://localhost:3000`.

**Não há bootstrap automático de admin** — o primeiro usuário (local ou via Keycloak/JIT) precisa ser promovido manualmente: `UPDATE users SET is_admin = true WHERE email = '...'` direto no Postgres. A partir daí, um admin cria as contas locais seguintes por `/admin/usuarios`.

Veja `make help` na raiz para os atalhos de orquestração do monorepo, [`backend/README.md`](backend/README.md) para tudo sobre a API e [`frontend/README.md`](frontend/README.md) para o cliente Next.js.

## Documentação por projeto

- **[backend/README.md](backend/README.md)** — arquitetura (Clean Architecture), o funil do Kanban, os módulos (Radar/Gerador de Documentos/Vistorias/Dossiê/SGF), como rodar/testar localmente, migrations, referência de variáveis de ambiente, rotas da API, checklist de produção e limitações conhecidas.
- **[frontend/README.md](frontend/README.md)** — stack, arquitetura BFF, setup, passo a passo do client Keycloak, testes.
- **[DEPLOY.md](DEPLOY.md)** — runbook de deploy em produção (`selene.papermoon.cloud`, atrás do Cloudflare Tunnel).

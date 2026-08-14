# Frontend — Projeto Selene

Cliente Next.js (App Router) que consome a API do [`backend/`](../backend/README.md): autenticação via Keycloak (OIDC/Auth.js), listagem e cadastro de contratos — base pro quadro Kanban das 6 etapas de compliance.

## Stack

- **Next.js 16** (App Router, Turbopack, React 19)
- **TypeScript** + **Tailwind CSS v4** + **shadcn/ui** (biblioteca base `@base-ui/react`, não Radix — os componentes gerados usam a prop `render` em vez de `asChild`)
- **Auth.js v5** (`next-auth`) com provider Keycloak
- **TanStack Query** para mutações client-side
- **react-hook-form** + **zod** para formulários
- **Vitest** + **React Testing Library** para testes

## Arquitetura: BFF (Backend for Frontend)

O browser **nunca** chama o backend Go diretamente:

- **Leituras**: Server Components chamam o backend direto (`lib/api/client.ts`), usando o access token do usuário logado.
- **Escritas**: Client Components chamam Route Handlers do próprio Next (`app/api/.../route.ts`), que autenticam a requisição, aplicam qualquer regra server-side (ex: `fiscal_id` sempre é o usuário logado, nunca um valor vindo do client) e só então repassam pro backend.

O access token do Keycloak fica só no cookie de sessão criptografado do Auth.js — nunca é colocado no objeto `session` (que é serializado e devolvido ao browser via `useSession()`/`/api/auth/session`). Código server-side que precisa do token real usa `getAccessToken()` (`lib/auth-token.ts`), que lê o cookie diretamente via `next-auth/jwt.getToken()`. Ver os comentários em `src/auth.ts` para o detalhamento.

## Setup

```bash
npm install
cp .env.example .env.local   # preencha com os dados do seu client Keycloak
npm run generate:api          # gera src/lib/api/schema.d.ts a partir de ../backend/api/openapi.yaml
npm run dev
```

O backend precisa estar rodando (`docker compose up postgres redis jaeger backend` na raiz, ou `go run ./cmd/api` dentro de `backend/`) — ver [`backend/README.md`](../backend/README.md).

### Criando o client no Keycloak

O frontend usa a instância de Keycloak já existente (não sobe uma própria). Passo a passo pra criar o client OIDC:

1. **Clients → Create client** no realm configurado no backend (`KEYCLOAK_ISSUER`).
2. **General settings**: Client type = `OpenID Connect`; Client ID = `selene-frontend` (ou outro nome).
3. **Capability config**: `Client authentication` = **ON** (confidencial, gera secret); `Standard flow` = **ON**; `Direct access grants`/`Implicit flow`/`Service accounts` = OFF.
4. **Login settings**:
   - Root URL: `http://localhost:3000`
   - Valid redirect URIs: `http://localhost:3000/api/auth/callback/keycloak`
   - Valid post logout redirect URIs: `http://localhost:3000`
   - Web origins: `http://localhost:3000`
5. Salvar → aba **Credentials** → copiar o **Client secret**.
6. Issuer = `{url-base-do-keycloak}/realms/{realm}` — **sem barra final**, precisa ser idêntico ao `KEYCLOAK_ISSUER` do backend.

Preencha `AUTH_KEYCLOAK_ID`, `AUTH_KEYCLOAK_SECRET` e `AUTH_KEYCLOAK_ISSUER` em `.env.local` com esses valores.

## Variáveis de ambiente

Ver [`.env.example`](.env.example) para a lista completa e comentada. Duas merecem destaque:

- `API_URL` — URL do backend Go. **Sem** prefixo `NEXT_PUBLIC_` de propósito: variáveis `NEXT_PUBLIC_*` são fixadas no bundle em **build time**, o que quebraria a injeção de env em runtime do docker-compose. `API_URL` só é lida server-side (Server Components, Route Handlers), então não precisa (e não deve) ir pro bundle do browser.
- `AUTH_TRUST_HOST` — precisa ser `true` em produção (`NODE_ENV=production`, como na imagem Docker) sempre que o app roda atrás de mapeamento de porta ou proxy reverso — o que é sempre o caso em deploy real. Sem isso, o Auth.js rejeita a requisição com `UntrustedHost`.

## Testes

```bash
npm test
```

- `src/lib/api/client.test.ts` — o wrapper de fetch pro backend (headers, querystring de paginação, tratamento de erro via `ApiError`).
- `src/components/contratos/novo-contrato-dialog.test.tsx` — validação do formulário, que `fiscal_id` nunca é exposto/enviado pelo client, submissão bem e mal sucedida.

## Gerando os tipos da API

```bash
npm run generate:api
```

Roda `openapi-typescript` sobre `../backend/api/openapi.yaml` e escreve `src/lib/api/schema.d.ts`. Esse arquivo é **commitado** (não é gerado no CI) — rode o comando de novo sempre que o OpenAPI do backend mudar.

**Nota sobre nomes de campo**: respostas JSON do backend usam PascalCase (`NumeroContrato`), corpos de requisição usam snake_case (`numero_contrato`) — reflexo de uma inconsistência conhecida e documentada no backend (ver o topo de `backend/api/openapi.yaml`). Os tipos gerados refletem isso fielmente; não há camada de mapeamento.

## Docker

```bash
docker build -t projeto-selene-frontend .
```

Multi-stage (`deps` → `builder` → `runner`), usa `output: "standalone"` do Next. Roda como usuário não-root. Ver o serviço `frontend` no `docker-compose.yml` da raiz para como ele é orquestrado junto com o backend.

## Estrutura

```
src/
  app/
    (app)/            # rotas autenticadas — layout com Nav
      contratos/      # listagem (Server Component) + criação (dialog)
      kanban/          # board das 6 etapas (Server Component + client dialogs)
    api/
      auth/[...nextauth]/  # handler do Auth.js
      contratos/           # Route Handler BFF (POST, injeta fiscal_id)
      processos/            # Route Handlers BFF (criar, avançar, concluir, documentos, relatório)
    login/            # fora do grupo (app) — sem Nav
  auth.ts             # config do Auth.js (provider Keycloak, callbacks)
  proxy.ts            # checagem otimista de sessão (renomeado de middleware.ts no Next 16)
  lib/
    auth-token.ts     # getAccessToken() — lê o token real, nunca exposto ao browser
    api/
      client.ts       # wrapper de fetch tipado pro backend
      schema.d.ts     # gerado — não editar à mão
  components/
    ui/               # shadcn/ui
    contratos/        # componentes específicos da tela de contratos
    kanban/           # board, dialog de processo (documentos/avançar/concluir), novo processo
```

## Quadro Kanban

`/kanban` lista os processos de pagamento em 6 colunas (uma por etapa). Ao abrir um card: anexar documentos, avançar de etapa e baixar o Relatório de Pagamento (PDF).

**O checklist de documentos obrigatórios por etapa não é duplicado no frontend** — ele só existe no backend (`internal/service/checklist.go`), de propósito: são regras administrativas que podem mudar, e mantê-las em um único lugar evita que as duas camadas fiquem dessincronizadas. O frontend tenta avançar a etapa; se o backend responder 422 (`ChecklistIncompletoBody`), a lista de `documentos_pendentes` da própria resposta é exibida ao usuário. Isso significa que a única forma de saber o que falta é tentar avançar — aceitável para a v1, mas vale considerar expor os requisitos via API no futuro se o usuário achar o fluxo confuso.

Abrir um processo novo não tem restrição de "fiscal dono do contrato" — a regra de negócio do backend permite qualquer fiscal abrir um processo pra qualquer contrato ativo (não há checagem de propriedade em `KanbanService.CriarProcesso`).

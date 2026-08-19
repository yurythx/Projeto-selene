# Frontend — Projeto Selene

Cliente Next.js (App Router) que consome a API do [`backend/`](../backend/README.md): autenticação via Keycloak (OIDC/Auth.js) **ou** login tradicional (e-mail/senha), quadro Kanban das 6 etapas de compliance, e todos os módulos do backend (Radar de Alertas, Gerador de Documentos Legais, Vistorias de Campo, Dossiê do Fornecedor, SGF-Rondonópolis).

## Módulos cobertos

- **Kanban** (`/kanban`) — o núcleo: cards por processo de pagamento, drawer com documentos/avançar/concluir/relatório.
- **Contratos** (`/contratos`, `/contratos/[id]`) — CRUD, mais os cards SGF de Empenho e Designações (ver abaixo).
- **Radar** (`/radar`) — badges de vigência/certidão/processo parado, injetados no card e no drawer do Kanban via `lib/radar.ts`.
- **Gerador de Documentos** — Notificação de Descumprimento e Minuta de Aditivo (na página do contrato) e Atesto (no drawer do processo), todos PDF.
- **Vistorias de Campo** — dialog aninhado no drawer do Kanban (`components/kanban/vistorias-dialog.tsx`), mobile-first (geolocalização + captura de foto pela câmera).
- **Dossiê do Fornecedor** (`/fornecedores`, `/fornecedores/[cnpj]`).
- **Login local** (`/login`, `/trocar-senha`, `/admin/usuarios`) — `Credentials` provider do Auth.js ao lado do Keycloak; troca de senha obrigatória no primeiro login de conta local, aplicada por `proxy.ts`.
- **SGF-Rondonópolis** — Ocorrências (dialog aninhado no drawer do Kanban, mesmo padrão de Vistorias — bloqueia de verdade o avanço de etapa enquanto aberta) e Empenho/Designações (cards na página do contrato). Ver a seção [SGF-Rondonópolis](#sgf-rondonópolis) abaixo.

## Stack

- **Next.js 16** (App Router, Turbopack, React 19)
- **TypeScript** + **Tailwind CSS v4** + **shadcn/ui** (biblioteca base `@base-ui/react`, não Radix — os componentes gerados usam a prop `render` em vez de `asChild`)
- **Auth.js v5** (`next-auth`) com providers Keycloak **e** Credentials (login local)
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

Ver [`.env.example`](.env.example) para a lista completa e comentada. Três merecem destaque:

- `API_URL` — URL do backend Go. **Sem** prefixo `NEXT_PUBLIC_` de propósito: variáveis `NEXT_PUBLIC_*` são fixadas no bundle em **build time**, o que quebraria a injeção de env em runtime do docker-compose. `API_URL` só é lida server-side (Server Components, Route Handlers), então não precisa (e não deve) ir pro bundle do browser.
- `AUTH_TRUST_HOST` — precisa ser `true` em produção (`NODE_ENV=production`, como na imagem Docker) sempre que o app roda atrás de mapeamento de porta ou proxy reverso — o que é sempre o caso em deploy real. Sem isso, o Auth.js rejeita a requisição com `UntrustedHost`.
- `INTERNAL_API_SECRET` — segredo compartilhado com o backend (mesmo valor nos dois lados) que autentica `GET /internal/keycloak-config`. `AUTH_KEYCLOAK_ID`/`SECRET`/`ISSUER` acima são só o valor de *fallback*: um admin pode salvar uma configuração de Keycloak diferente em runtime pela tela Configurações → Keycloak/SSO (persistida no backend), e `src/auth.ts` busca a config ATIVA a cada requisição de autenticação via `lib/keycloak-config.ts` (cache de 60s) em vez de fixá-la uma vez no boot — por isso `NextAuth()` é chamado na forma "preguiçosa" (`async () => ({...})`), não com um objeto literal. Ver os comentários em `src/auth.ts` e `src/proxy.ts` (a forma preguiçosa exige um `await` extra no wrapper de middleware, achado real ao migrar).

## Testes

```bash
npm test
```

- `src/lib/api/client.test.ts` — o wrapper de fetch pro backend (headers, querystring de paginação, tratamento de erro via `ApiError`).
- `src/components/contratos/novo-contrato-dialog.test.tsx` — validação do formulário, que `fiscal_id` nunca é exposto/enviado pelo client, submissão bem e mal sucedida.
- `src/components/kanban/*.test.tsx` — documentos, e principalmente o fluxo de checklist incompleto (422 → mostra `documentos_pendentes`).
- `src/components/contratos/encerrar-contrato-button.test.tsx`, `src/components/admin/editar-usuario-dialog.test.tsx`.
- `src/lib/verify-origin.test.ts` — o helper de defesa contra CSRF.
- `src/lib/rate-limit.test.ts` — o helper de rate limit do BFF (fail-open sem Redis configurado, contagem/TTL, 429 acima do limite, fail-open em erro do Redis), com `ioredis` mockado.

### E2E (Playwright)

```bash
npx playwright install --with-deps chromium   # só na primeira vez
npm run test:e2e
```

Sobe dois servidores locais (`playwright.config.ts`): um stub do backend em memória (`e2e/mock-backend.ts` — implementa só as rotas exercitadas pelos specs, sem Go/Postgres/Keycloak reais) e a própria app rodando o artefato **standalone real** (`node .next/standalone/server.js`, o mesmo que o Dockerfile empacota — `next start` não suporta `output: "standalone"` e chegou a causar hidratação inconsistente num teste, por isso o cuidado de rodar exatamente o que vai pra produção).

A sessão do Auth.js é **injetada direto num cookie** (`e2e/fixtures/auth.ts`, via `next-auth/jwt.encode()` com o mesmo `AUTH_SECRET` do servidor de teste) em vez de logar de verdade pelo Keycloak — não faz sentido (nem seria seguro) usar credenciais reais da instância de produção da prefeitura em CI. O que os specs cobrem é o comportamento do frontend a partir de uma sessão válida; o fluxo de login em si (redirect, `client_id`, discovery document) foi validado manualmente contra o Keycloak real antes deste commit.

Specs em `e2e/*.spec.ts` (23 testes): redirecionamento de quem não tem sessão, login local (credenciais corretas/erradas, troca de senha obrigatória), CRUD de contratos, o board do Kanban (incluindo o 422 de checklist incompleto, upload de documento e vistoria de campo), Dossiê do Fornecedor, a tela de administração (visibilidade por `is_admin`), `sgf.spec.ts` (Ocorrências bloqueando/liberando o avanço de etapa, Empenho com saldo, Designações — histórico e o fluxo completo de "Nova designação"), e `csp.spec.ts` (nonce presente no header, nenhuma violação de CSP no console ao abrir Select/Dialog).

**Nota sobre `vitest.config.mts`**: `pool: "threads"` + `fileParallelism: false` não são só estilo — o pool padrão do Vitest 4 (`"forks"`, um processo filho por arquivo de teste) trava esperando os workers responderem em ambientes com poucos CPUs/contêineres (reproduzido em CI e num container Docker local simples: `Timeout waiting for worker to respond`, suíte inteira falha com "no tests found" mesmo com o código correto). `"threads"` usa `worker_threads` (sem spawn de processo) e é bem mais robusto nesse cenário.

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
    (app)/                  # rotas autenticadas — layout com Nav
      contratos/            # listagem (Server Component) + criação (dialog)
        [id]/               # detalhe, editar, encerrar + cards SGF (Empenho, Designações)
      kanban/                # board das 6 etapas (Server Component + client dialogs)
      radar/                 # painel consolidado de alertas
      fornecedores/           # Dossiê do Fornecedor, por CNPJ
        [cnpj]/
      admin/usuarios/         # só is_admin — permissões + criação de conta local
    api/                     # Route Handlers BFF — um por recurso, espelhando o backend
      auth/[...nextauth]/     # handler do Auth.js (Keycloak + Credentials)
      auth/trocar-senha/
      contratos/[id]/{designacoes,empenhos,notificacao,minuta-aditivo,encerrar}/
      processos/[id]/{ocorrencias,vistorias,atesto,avancar,concluir,documentos,relatorio}/
      ocorrencias/[id]/{notificar,tratar,regularizar}/
      empenhos/[id]/movimentacoes/
      vistorias/[id]/{fotos,relatorio}/
      admin/usuarios/[id]/, admin/usuarios/local/
      verificar/[codigo]/     # proxy PÚBLICO (sem sessão) — verificação externa de QR code
    login/                  # fora do grupo (app) — sem Nav
    trocar-senha/            # idem — troca obrigatória de senha temporária
    verificar/[codigo]/      # página pública de verificação de documento
  auth.ts                   # config do Auth.js (providers Keycloak + Credentials, callbacks)
  proxy.ts                  # checagem otimista de sessão + gate de troca de senha obrigatória
  lib/
    auth-token.ts           # getAccessToken() — lê o token real, nunca exposto ao browser
    radar.ts                 # correlação de alertas do Radar com card/drawer do Kanban
    api/
      client.ts             # wrapper de fetch tipado pro backend — 1 função por endpoint
      schema.d.ts            # gerado — não editar à mão
  components/
    ui/                     # shadcn/ui
    contratos/               # telas de contrato + cards SGF (designacoes-card, empenhos-card)
    kanban/                   # board, drawer do processo, e os dialogs aninhados
                              # (vistorias-dialog, ocorrencias-dialog)
    radar/                   # badge de nível de alerta
    admin/                    # edição de usuário, criação de conta local
    login/                    # formulário de login tradicional, troca de senha
```

## Quadro Kanban

`/kanban` lista os processos de pagamento em 6 colunas (uma por etapa). Ao abrir um card: anexar documentos, avançar de etapa e baixar o Relatório de Pagamento (PDF).

**O checklist de documentos obrigatórios por etapa não é duplicado no frontend** — ele só existe no backend (`internal/service/checklist.go`), de propósito: são regras administrativas que podem mudar, e mantê-las em um único lugar evita que as duas camadas fiquem dessincronizadas. O frontend tenta avançar a etapa; se o backend responder 422 (`ChecklistIncompletoBody`), a lista de `documentos_pendentes` da própria resposta é exibida ao usuário. Isso significa que a única forma de saber o que falta é tentar avançar — aceitável para a v1, mas vale considerar expor os requisitos via API no futuro se o usuário achar o fluxo confuso.

Abrir um processo novo não tem restrição de "fiscal dono do contrato" — a regra de negócio do backend permite qualquer fiscal abrir um processo pra qualquer contrato ativo (não há checagem de propriedade em `KanbanService.CriarProcesso`).

## SGF-Rondonópolis

Adequação às IN SCL 01/2019 e 04/2021 (ver `backend/README.md` para a Matriz Normativa e o modelo de dados) — dois pontos de entrada na UI:

- **Ocorrências** (`components/kanban/ocorrencias-dialog.tsx`) — dialog aninhado no drawer do Kanban, mesmo padrão de Vistorias. Registrar uma ocorrência **bloqueia de verdade** (não só visualmente) o botão "Avançar etapa" — o drawer refaz `GET /processos/:id` ao abrir (`processo-dialog.tsx`, query `["processo", processo.ID]`) pra ler `allowed_actions`, que substituiu os booleanos `podeAvancar`/`podeConcluir` hoje calculados a partir só de `EtapaAtualID`/`Status` como fallback (enquanto essa query não resolveu).
- **Empenho e Designações** (`components/contratos/empenhos-card.tsx`, `designacoes-card.tsx`) — cards na página de detalhe do contrato. Ambos têm CRUD completo agora: Empenho (criar, registrar reforço/anulação, saldo ao vivo) e Designações ("Nova designação" — seleciona servidor via `GET /servidores`, projeção mínima ID/Nome/Email aberta a qualquer autenticado, e papel FISCAL/FISCAL_SUPLENTE/GESTOR/FISCAL_SETORIAL; uma nova designação do mesmo papel revoga a anterior automaticamente no backend).
- **Mão de obra terceirizada** (`novo-contrato-dialog.tsx`, `editar-contrato-dialog.tsx`) — switch "Mão de obra terceirizada" no formulário de contrato, ligado a `Contrato.ExigeFiscalizacaoTerceirizacao`; quando ativo, acrescenta os documentos trabalhistas do Art.9º-XXXII da IN04 ao checklist da Etapa 5 (ver `backend/README.md`). Um badge "Mão de obra terceirizada" aparece no cabeçalho de `/contratos/[id]` quando o contrato está marcado.
- **Configurações** (`/configuracoes`) — hub admin-only com cards pras telas administrativas que não cabem em nenhuma área de negócio: Modelos de Documentos e Keycloak/SSO (editável em runtime, ver a seção Variáveis de ambiente acima). A sidebar aponta pro hub, não direto pra nenhuma das duas.
- **Pré-visualização de documento anexo** (`components/kanban/processo-page.tsx`) — o nome do documento (na lista "Documentos anexados" e nos itens satisfeitos do checklist) abre o arquivo numa aba nova (`target="_blank"`), de propósito: um visualizador nativo do navegador é mais rápido de exibir do que qualquer visualizador embutido em JS, decisão tomada depois de testar um visualizador PDF.js embutido e comparar. O backend serve com cache HTTP agressivo (ver `backend/README.md`) — reabrir o mesmo documento não retransmite nada.

## Checklist de produção

- [x] Autenticação via Auth.js v5 + Keycloak, access token nunca exposto ao browser (fica só no cookie de sessão criptografado; `getAccessToken()` lê direto via `next-auth/jwt`, nunca pelo endpoint público `/api/auth/session`)
- [x] Arquitetura BFF: browser nunca chama o backend Go direto, só os Route Handlers do próprio Next
- [x] `fiscal_id`/autorização sempre resolvidos server-side a partir da sessão — nunca aceitos do corpo da requisição do client
- [x] Defesa em profundidade contra CSRF nos 27 Route Handlers de mutação (checagem de Origin vs Host, além do SameSite=Lax do cookie de sessão) — ver `lib/verify-origin.ts`
- [x] Security headers: CSP com nonce via `src/proxy.ts` (por requisição); X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy via `next.config.ts` (estáticos, não dependem de nonce)
- [x] `loading.tsx` (streaming) e `error.tsx`/`global-error.tsx` (boundary de erro amigável) nas rotas principais
- [x] Testes automatizados (client de API, formulários, fluxo de checklist incompleto, origin check, rate limit) — 56 testes unitários/componente + 42 E2E (Playwright)
- [x] CI (lint + testes unitários + testes E2E + build + imagem Docker), mesmo pipeline do backend
- [x] Imagem Docker multi-stage, `output: standalone`, usuário não-root
- [x] Tipos gerados a partir do OpenAPI do backend (`openapi-typescript`) — sem duplicar contratos de API à mão
- [x] CSP com nonce em `script-src` (`'nonce-x' 'strict-dynamic'`, sem `'unsafe-inline'`) — gerado por requisição em `src/proxy.ts`, viável porque toda página já é dinamicamente renderizada (confirmado no output de `next build`: nenhuma rota estática). `style-src` mantém `'unsafe-inline'` de propósito — nonce não cobre o atributo `style="..."` inline que o base-ui usa pra posicionar Select/Dialog/Popover; confirmado empiricamente (`e2e/csp.spec.ts`, não só por dedução) que apertar sem isso quebra esses componentes.
- [x] Paginação de verdade na listagem de contratos (Anterior/Próxima dirigido pela URL, `components/paginacao.tsx`) e nas colunas do Kanban (botão "Carregar mais" por coluna, acumula páginas de 100 em 100 — `kanban-board.tsx`)
- [x] Rate limiting nos 27 Route Handlers de mutação (`lib/rate-limit.ts`) — Redis compartilhado com o backend (mesmo `REDIS_ADDR`/`REDIS_PASSWORD`, DB lógico separado), fixed-window por IP, fail-open se o Redis estiver indisponível ou não configurado (ex.: dev local sem o serviço `redis`). Redundante de propósito com o limite por usuário que já existe no backend Go — cobre o BFF mesmo se algum handler, por bug, nunca chegasse a acionar o limite de lá.

## Limitações conhecidas

- **Fluxo de login via Keycloak (interativo, num browser real)**: a construção da URL de autorização (issuer, client_id, redirect_uri, discovery document) foi validada via curl contra o Keycloak real; o login tradicional (Credentials) tem cobertura E2E completa em Chromium real (`e2e/auth.spec.ts`). O clique-a-clique via Keycloak real, especificamente, ainda depende de um usuário de verdade — não dá pra automatizar em CI sem credenciais reais da instância de produção da prefeitura. Um bug real relacionado foi encontrado (e corrigido) montando a suíte E2E: `getAccessToken()` decidia o nome do cookie de sessão com base em `NODE_ENV`, mas o Auth.js decide isso por requisição, com base no protocolo (`http`/`https`) — no docker-compose atual (HTTP puro, sem TLS na frente), isso fazia toda página protegida se comportar como se o usuário estivesse deslogado mesmo com uma sessão válida. `lib/auth-token.ts` agora tenta os dois nomes de cookie em vez de adivinhar.
- **Sem seletor de fiscal no cadastro de contrato/processo** — qualquer fiscal pode ser atribuído a um contrato ou abrir um processo pra qualquer contrato ativo, porque é assim que o backend autoriza hoje (sem checagem de propriedade). Documentado também no backend.
- **Confirmação de "Encerrar contrato" via `window.confirm`** — funcional, mas um `AlertDialog` dedicado (shadcn/ui já tem o primitivo) seria a versão "produção de verdade" dessa UX.
- **`POST /api/verificar/[codigo]`**: proxy público (sem checagem de sessão, de propósito — espelha a rota pública do backend) pra quem quiser verificar a autenticidade de um documento emitido programaticamente, sem carregar a página `/verificar/[codigo]` inteira. Hoje só a página usa `verificarDocumento()` diretamente; a rota BFF existe para consumidores externos.

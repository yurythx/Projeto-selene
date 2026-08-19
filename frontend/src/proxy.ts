import { NextResponse } from "next/server";
import { auth } from "@/auth";

// Next.js 16 renomeou "middleware" para "proxy" (mesma funcionalidade,
// nome novo — roda sempre em runtime nodejs, o que aliás é exigido pelo
// Auth.js). Faz só a checagem OTIMISTA de autenticação (lê o cookie de
// sessão, sem chamar o backend): redireciona pra /login quem não tem
// sessão. A checagem "de verdade" (autorização por is_admin/is_fiscal,
// dados sensíveis) acontece nos Server Components/Route Handlers via
// auth()/getAccessToken(), mais perto da fonte de dados — proxy não deve
// ser a única linha de defesa.
//
// CSP com nonce (script-src): cada requisição gera um nonce novo aqui e
// o Next.js aplica automaticamente a scripts do framework/da página (ver
// o guia oficial, "How nonces work in Next.js") — não precisa de nenhum
// <Script nonce={...}> manual porque este app não usa next/script. Só
// funciona porque TODA página aqui já é renderizada dinamicamente (todas
// chamam auth()/cookies() direta ou indiretamente, confirmado no output
// de `next build`: nenhuma rota aparece como estática) — sem isso, não
// haveria request/response pra carregar um nonce dentro. Antes desta
// mudança, o next.config.ts usava uma CSP fixa sem nonce (guia "Without
// Nonces") justamente pra não depender dessa premissa; documentado então
// como "próximo passo natural", agora resolvido.
//
// style-src continua com 'unsafe-inline' de propósito, não por omissão:
// nonce só cobre tags <style>, não o atributo `style="..."` inline — e o
// base-ui (Select/Dialog/Popover) usa esse atributo pra posicionamento
// via floating-ui, com valores calculados a cada render (não dá pra usar
// hash). Confirmado empiricamente (não só por dedução): removendo
// 'unsafe-inline' daqui, e2e/csp.spec.ts reproduz o erro real do
// Chromium ("Applying inline style violates... style-src") ao abrir um
// Select — rodado contra o mesmo artefato standalone que o Dockerfile
// empacota (ver webServer em playwright.config.ts), não um mock. Troque
// só se um dia trocar a lib de UI.
function buildCspHeader(nonce: string): string {
  const isDev = process.env.NODE_ENV === "development";
  const cspHeader = `
    default-src 'self';
    script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${isDev ? " 'unsafe-eval'" : ""};
    style-src 'self' 'unsafe-inline';
    img-src 'self' blob: data:;
    font-src 'self';
    object-src 'none';
    base-uri 'self';
    form-action 'self';
    frame-ancestors 'none';
    upgrade-insecure-requests;
  `;
  return cspHeader.replace(/\s{2,}/g, " ").trim();
}
// GUEST_ONLY_ROUTES: só fazem sentido pra quem NÃO tem sessão — um usuário
// logado que visita /login é redirecionado pra "/".
const GUEST_ONLY_ROUTES = ["/login"];

// PUBLIC_ROUTES: acessíveis com OU sem sessão, sem redirecionamento em
// nenhum sentido. /verificar (Fase 2 do roadmap) precisa disso: o QR code
// de um Atesto impresso é escaneado por quem não tem login no Selene (ex:
// auditor do TCE), mas um fiscal logado também pode querer abrir o mesmo
// link (ex: conferir o próprio QR) sem ser expulso da página.
const PUBLIC_ROUTES = ["/verificar"];

// Única rota isenta do gate de troca de senha obrigatória abaixo — sem
// isso, um usuário com mustChangePassword=true seria redirecionado de
// volta pra ela mesma (loop).
const TROCA_SENHA_ROUTE = "/trocar-senha";

// `await` aqui (top-level, suportado nativamente por módulos ES) é
// necessário desde que src/auth.ts passou a usar a forma "preguiçosa" do
// NextAuth (`NextAuth(async () => ({...}))`, ver o comentário lá — é o
// que permite trocar Client ID/Secret/Issuer do Keycloak em runtime).
// Achado real ao migrar: com config preguiçosa, next-auth/lib/index.js
// (initAuth) devolve `auth` como uma função ASYNC — chamar
// `auth((req) => {...})` (o padrão de wrapper de middleware) passa a
// devolver uma Promise que RESOLVE pro handler de verdade, em vez do
// handler em si de forma síncrona (que é o que a forma "eager"/objeto
// antiga devolvia). Sem o await, `export default` vira uma Promise, e o
// Next.js rejeita o arquivo com "must export a function". O await aqui
// só resolve isso UMA VEZ no carregamento do módulo — o handler
// resultante (a função abaixo) continua rodando por requisição
// normalmente, e a config do Keycloak dentro dele continua sendo
// buscada fresca a cada requisição (com cache curto, ver
// lib/keycloak-config.ts), não fica presa ao valor deste boot.
export default await auth((req) => {
  const { pathname } = req.nextUrl;
  const isGuestOnlyRoute = GUEST_ONLY_ROUTES.some((route) => pathname.startsWith(route));
  const isPublicRoute = PUBLIC_ROUTES.some((route) => pathname.startsWith(route));

  // Nonce novo a cada requisição — reaproveitado tanto no header de
  // resposta (é dele que o Next.js extrai o padrão 'nonce-{valor}' pra
  // aplicar aos scripts que ele mesmo injeta) quanto no header de
  // requisição x-nonce, disponível via headers() em Server
  // Components/layout caso este app um dia precise de um <Script>
  // manual (hoje não usa nenhum).
  const nonce = Buffer.from(crypto.randomUUID()).toString("base64");
  const cspHeader = buildCspHeader(nonce);

  const requestHeaders = new Headers(req.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("Content-Security-Policy", cspHeader);

  // Toda resposta (siga adiante ou redirecione) carrega a mesma CSP —
  // redirects não renderizam HTML, então o nonce neles não tem efeito
  // prático, mas manter o header presente em todo caminho evita um
  // gap silencioso se algum dia um desses branches passar a devolver
  // conteúdo em vez de só redirecionar.
  const comCsp = (response: NextResponse) => {
    response.headers.set("Content-Security-Policy", cspHeader);
    return response;
  };

  if (!req.auth?.user && !isGuestOnlyRoute && !isPublicRoute) {
    const loginUrl = new URL("/login", req.nextUrl);
    loginUrl.searchParams.set("callbackUrl", pathname);
    return comCsp(NextResponse.redirect(loginUrl));
  }

  if (req.auth?.user && isGuestOnlyRoute) {
    return comCsp(NextResponse.redirect(new URL("/", req.nextUrl)));
  }

  // Login local (usuário/senha) com senha temporária ainda não trocada —
  // "soft gate" antes de liberar qualquer outra página autenticada. Não
  // se aplica a contas Keycloak (mustChangePassword sempre false pra
  // elas, ver src/auth.ts).
  if (
    req.auth?.user?.mustChangePassword &&
    !isPublicRoute &&
    !isGuestOnlyRoute &&
    pathname !== TROCA_SENHA_ROUTE
  ) {
    return comCsp(NextResponse.redirect(new URL(TROCA_SENHA_ROUTE, req.nextUrl)));
  }

  return comCsp(NextResponse.next({ request: { headers: requestHeaders } }));
});

export const config = {
  // "icon" (gerado por app/icon.tsx, ver ACHADO DE REVISÃO ali) precisa
  // estar aqui do mesmo jeito que "favicon.ico" já estava — sem essa
  // exclusão, todo pedido do favicon do navegador batia neste middleware,
  // era tratado como rota autenticada e vinha 307 pro /login (setando
  // cookie de CSRF à toa a cada carregamento de página, sem nem servir o
  // ícone). "favicon.ico" continua na lista por segurança mesmo não
  // existindo mais o arquivo estático — não custa nada manter.
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico|icon).*)"],
};

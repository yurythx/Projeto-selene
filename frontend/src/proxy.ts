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
// GUEST_ONLY_ROUTES: só fazem sentido pra quem NÃO tem sessão — um usuário
// logado que visita /login é redirecionado pra "/".
const GUEST_ONLY_ROUTES = ["/login"];

// PUBLIC_ROUTES: acessíveis com OU sem sessão, sem redirecionamento em
// nenhum sentido. /verificar (Fase 2 do roadmap) precisa disso: o QR code
// de um Atesto impresso é escaneado por quem não tem login no Selene (ex:
// auditor do TCE), mas um fiscal logado também pode querer abrir o mesmo
// link (ex: conferir o próprio QR) sem ser expulso da página.
const PUBLIC_ROUTES = ["/verificar"];

export default auth((req) => {
  const { pathname } = req.nextUrl;
  const isGuestOnlyRoute = GUEST_ONLY_ROUTES.some((route) => pathname.startsWith(route));
  const isPublicRoute = PUBLIC_ROUTES.some((route) => pathname.startsWith(route));

  if (!req.auth?.user && !isGuestOnlyRoute && !isPublicRoute) {
    const loginUrl = new URL("/login", req.nextUrl);
    loginUrl.searchParams.set("callbackUrl", pathname);
    return NextResponse.redirect(loginUrl);
  }

  if (req.auth?.user && isGuestOnlyRoute) {
    return NextResponse.redirect(new URL("/", req.nextUrl));
  }

  return NextResponse.next();
});

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};

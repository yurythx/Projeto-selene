import { NextResponse } from "next/server";
import { signOut } from "@/auth";

/**
 * Chamada por requireApi (lib/api/client.ts) quando o backend rejeita o
 * accessToken com 401 ("token inválido ou expirado" — ex: reinício do
 * backend invalidando sessões de login local, ver a LIMITAÇÃO CONHECIDA
 * em internal/localauth/localauth.go).
 *
 * Precisa ser um Route Handler, não um redirect() direto de dentro do
 * Server Component da página: limpar o cookie de sessão exige um
 * contexto que pode mutar cookies, e Server Components de página não
 * podem. ACHADO DE REVISÃO: a primeira versão deste fix redirecionava
 * direto pra /login sem limpar o cookie — como ele continua
 * criptograficamente válido pro Auth.js (só o accessToken embutido é que
 * o backend rejeitou), proxy.ts (GUEST_ONLY_ROUTES) via sessão presente
 * mandava de volta pra "/", que redireciona pro /kanban, que falha de
 * novo com 401 e volta aqui — loop infinito, reproduzido e corrigido
 * antes do commit (ver e2e/session.spec.ts).
 */
export async function GET(request: Request) {
  await signOut({ redirect: false });
  return NextResponse.redirect(new URL("/login", request.url));
}

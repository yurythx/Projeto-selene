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
export async function GET() {
  await signOut({ redirect: false });
  // NUNCA `new URL("/login", request.url)` — no build standalone em
  // Docker, o Next monta request.url a partir de HOSTNAME/PORT internos
  // do container (HOSTNAME=0.0.0.0), não do Host real da requisição; o
  // redirect saía como "http://0.0.0.0:3000/login", que o navegador
  // rejeita (net::ERR_ADDRESS_INVALID — reproduzido ao vivo antes desta
  // correção). Mesma causa-raiz já documentada pra AUTH_URL em
  // docker-compose.yml — reaproveita a mesma env var, já configurada com
  // a origem pública correta (obrigatória, ver o comentário lá).
  return NextResponse.redirect(new URL("/login", process.env.AUTH_URL));
}

import "server-only";
import { NextResponse } from "next/server";

/**
 * Defesa em profundidade contra CSRF nas mutações do BFF (POST/PATCH).
 *
 * O cookie de sessão do Auth.js já é SameSite=Lax por padrão, o que
 * cobre o caso clássico de CSRF via <form> — cookies Lax não acompanham
 * requisições POST/PATCH iniciadas a partir de outro site (só GET de
 * navegação de topo). Esta checagem é uma camada explícita adicional, no
 * mesmo espírito do que o próprio Next.js faz internamente pra Server
 * Actions: comparar o header `Origin` da requisição com o `Host` que ela
 * mesma diz estar acessando — não deveriam divergir numa requisição
 * legítima feita pelo nosso próprio frontend.
 *
 * Retorna uma Response 403 se a origem não bate, ou `null` se está OK
 * (ou não dá pra checar — ver comentário abaixo).
 */
export function assertOrigemSegura(request: Request): NextResponse | null {
  const origin = request.headers.get("origin");
  const host = request.headers.get("host");

  // Clients HTTP diretos (testes de integração, curl) legitimamente não
  // enviam Origin — só o header Origin é sempre incluído pelo navegador
  // em fetch/XHR/form de mutação. Sem ele, não dá pra comparar; a
  // autenticação via sessão continua sendo a defesa primária.
  if (!origin || !host) return null;

  let origemHost: string;
  try {
    origemHost = new URL(origin).host;
  } catch {
    return NextResponse.json({ error: "header Origin inválido" }, { status: 403 });
  }

  if (origemHost !== host) {
    return NextResponse.json({ error: "origem da requisição não confere" }, { status: 403 });
  }

  return null;
}

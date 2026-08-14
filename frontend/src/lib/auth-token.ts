import "server-only";
import { headers } from "next/headers";
import { getToken } from "next-auth/jwt";

/**
 * Lê o access token do Keycloak diretamente do cookie de sessão do Auth.js,
 * via next-auth/jwt.getToken() — NÃO passa pelo endpoint público
 * /api/auth/session, então esse valor nunca é serializado pro browser (ver
 * o comentário em src/auth.ts sobre por que accessToken não está em
 * `session`).
 *
 * Uso: Server Components e Route Handlers que precisam autenticar uma
 * chamada ao backend Go. Client Components não podem importar isto
 * ("server-only" quebra o build se tentarem).
 *
 * Sobre `secureCookie`: o Auth.js decide o nome do cookie (com ou sem
 * prefixo `__Secure-`) por requisição, com base em `url.protocol ===
 * "https:"` — NÃO com base em NODE_ENV. Isso significa que, atrás de um
 * proxy reverso que não repassa o protocolo corretamente, ou quando o
 * app roda com NODE_ENV=production servindo HTTP puro (ex: o
 * docker-compose deste repo, sem TLS na frente), um `secureCookie`
 * fixo baseado em NODE_ENV adivinha errado e faz getToken() nunca achar
 * o cookie — a sessão existe, mas toda página protegida se comporta como
 * se o usuário estivesse deslogado. Tenta os dois nomes de cookie em vez
 * de adivinhar; o custo de tentar duas vezes é irrelevante.
 */
export async function getAccessToken(): Promise<string | null> {
  const headersList = await headers();

  for (const secureCookie of [true, false]) {
    const token = await getToken({
      req: { headers: headersList },
      secret: process.env.AUTH_SECRET,
      secureCookie,
    });
    if (token?.accessToken) {
      return token.accessToken as string;
    }
  }

  return null;
}

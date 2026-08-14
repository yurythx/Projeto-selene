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
 */
export async function getAccessToken(): Promise<string | null> {
  const token = await getToken({
    req: { headers: await headers() },
    secret: process.env.AUTH_SECRET,
    secureCookie: process.env.NODE_ENV === "production",
  });

  return (token?.accessToken as string | undefined) ?? null;
}

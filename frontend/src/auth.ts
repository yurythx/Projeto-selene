import NextAuth from "next-auth";
import Keycloak from "next-auth/providers/keycloak";
import type { JWT } from "next-auth/jwt";

/**
 * Config do Auth.js (NextAuth v5) com o provider Keycloak.
 *
 * Client ID/secret/issuer são lidos automaticamente das env vars
 * AUTH_KEYCLOAK_ID / AUTH_KEYCLOAK_SECRET / AUTH_KEYCLOAK_ISSUER (convenção
 * do Auth.js v5 — ver setEnvDefaults em @auth/core).
 *
 * Ponto crítico: os flags `is_admin`/`is_fiscal` NÃO existem no JWT do
 * Keycloak — só existem na tabela `users` do backend Go. No primeiro login
 * (quando `account` está presente), chamamos GET /api/v1/me com o access
 * token recém-emitido; isso também é o que dispara o JIT provisioning do
 * usuário no backend (a rota /me cria o usuário na primeira chamada).
 *
 * O access token e o refresh token do Keycloak ficam SÓ no `token` (JWT
 * interno do Auth.js, criptografado no cookie) — nunca são copiados para o
 * objeto `session`. Isso importa porque `session` é serializado e devolvido
 * ao browser sempre que algo chama useSession()/SessionProvider (via
 * GET /api/auth/session): qualquer campo colocado lá fica visível no
 * Network tab do navegador. Código server-side que precisa do access token
 * real (Server Components, Route Handlers) usa getAccessToken() em
 * lib/auth-token.ts, que lê o cookie diretamente via next-auth/jwt.getToken
 * — nunca passa pelo endpoint público /api/auth/session.
 */
export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [Keycloak],
  session: { strategy: "jwt" },
  callbacks: {
    async jwt({ token, account }) {
      if (account) {
        token.accessToken = account.access_token;
        token.refreshToken = account.refresh_token;
        token.expiresAt = account.expires_at;
        delete token.error;

        try {
          const res = await fetch(`${process.env.API_URL}/api/v1/me`, {
            headers: { Authorization: `Bearer ${account.access_token}` },
          });
          if (res.ok) {
            const me = await res.json();
            token.userId = me.id;
            token.isAdmin = Boolean(me.is_admin);
            token.isFiscal = Boolean(me.is_fiscal);
          }
        } catch {
          // Backend indisponível no momento do login: a sessão segue sem
          // os flags — telas que exigem is_admin/is_fiscal tratam
          // undefined como "sem permissão" (nega por padrão).
        }

        return token;
      }

      const expiresAt = typeof token.expiresAt === "number" ? token.expiresAt : 0;
      if (Date.now() < expiresAt * 1000) {
        return token;
      }

      return refreshAccessToken(token);
    },
    async session({ session, token }) {
      if (session.user) {
        session.user.id = token.userId as string;
        session.user.isAdmin = Boolean(token.isAdmin);
        session.user.isFiscal = Boolean(token.isFiscal);
      }
      session.error = token.error as string | undefined;
      return session;
    },
  },
});

async function refreshAccessToken(token: JWT): Promise<JWT> {
  try {
    const url = `${process.env.AUTH_KEYCLOAK_ISSUER}/protocol/openid-connect/token`;
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "refresh_token",
        client_id: process.env.AUTH_KEYCLOAK_ID ?? "",
        client_secret: process.env.AUTH_KEYCLOAK_SECRET ?? "",
        refresh_token: (token.refreshToken as string) ?? "",
      }),
    });

    const refreshed = await res.json();
    if (!res.ok) throw refreshed;

    return {
      ...token,
      accessToken: refreshed.access_token,
      expiresAt: Math.floor(Date.now() / 1000) + refreshed.expires_in,
      refreshToken: refreshed.refresh_token ?? token.refreshToken,
      error: undefined,
    };
  } catch {
    // Refresh falhou (refresh token expirado/revogado) — marca o erro; o
    // componente de sessão no client redireciona pro login ao ver
    // session.error === "RefreshAccessTokenError".
    return { ...token, error: "RefreshAccessTokenError" };
  }
}

import NextAuth from "next-auth";
import Keycloak from "next-auth/providers/keycloak";
import Credentials from "next-auth/providers/credentials";
import type { JWT } from "next-auth/jwt";

/**
 * Config do Auth.js (NextAuth v5) com DOIS providers: Keycloak (SSO
 * institucional) e Credentials (login tradicional e-mail/senha, contra o
 * backend Go — ver POST /api/v1/auth/login). O usuário escolhe um dos dois
 * na tela de login; o resto da aplicação trata as duas sessões de forma
 * idêntica (mesmo formato de `session`/token interno), exceto por como o
 * accessToken é renovado quando expira — ver refreshAccessToken.
 *
 * Client ID/secret/issuer do Keycloak são lidos automaticamente das env
 * vars AUTH_KEYCLOAK_ID / AUTH_KEYCLOAK_SECRET / AUTH_KEYCLOAK_ISSUER
 * (convenção do Auth.js v5 — ver setEnvDefaults em @auth/core).
 *
 * Ponto crítico: os flags `is_admin`/`is_fiscal` NÃO existem no JWT do
 * Keycloak — só existem na tabela `users` do backend Go. No primeiro login
 * via Keycloak (quando `account` está presente), chamamos GET /api/v1/me
 * com o access token recém-emitido; isso também é o que dispara o JIT
 * provisioning do usuário no backend (a rota /me cria o usuário na
 * primeira chamada). No login local esses dados já vêm prontos na própria
 * resposta de POST /auth/login (authorize() abaixo) — sem chamada extra.
 *
 * O access token e o refresh token ficam SÓ no `token` (JWT interno do
 * Auth.js, criptografado no cookie) — nunca são copiados para o objeto
 * `session`. Isso importa porque `session` é serializado e devolvido ao
 * browser sempre que algo chama useSession()/SessionProvider (via
 * GET /api/auth/session): qualquer campo colocado lá fica visível no
 * Network tab do navegador. Código server-side que precisa do access token
 * real (Server Components, Route Handlers) usa getAccessToken() em
 * lib/auth-token.ts, que lê o cookie diretamente via next-auth/jwt.getToken
 * — nunca passa pelo endpoint público /api/auth/session.
 */
export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [
    Keycloak,
    Credentials({
      id: "credentials",
      name: "Login tradicional",
      credentials: {
        email: { label: "E-mail", type: "email" },
        senha: { label: "Senha", type: "password" },
      },
      async authorize(credentials) {
        const email = credentials?.email;
        const senha = credentials?.senha;
        if (typeof email !== "string" || typeof senha !== "string" || !email || !senha) {
          return null;
        }

        const res = await fetch(`${process.env.API_URL}/api/v1/auth/login`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, senha }),
        });
        if (!res.ok) {
          // Auth.js trata authorize() retornando null como credenciais
          // inválidas — a MESMA mensagem genérica que o backend já usa
          // (ver AuthService.Login), sem repassar o corpo da resposta.
          return null;
        }

        const body = await res.json();
        // Campos extras (accessToken/isFiscal/isAdmin/mustChangePassword)
        // não fazem parte do tipo padrão `User` do Auth.js — o module
        // augmentation em src/types/next-auth.d.ts declara eles. O
        // callback jwt() abaixo os lê de volta do parâmetro `user`.
        return {
          id: body.usuario.ID,
          name: body.usuario.Nome,
          email: body.usuario.Email,
          accessToken: body.access_token as string,
          isFiscal: Boolean(body.usuario.IsFiscal),
          isAdmin: Boolean(body.usuario.IsAdmin),
          mustChangePassword: Boolean(body.usuario.MustChangePassword),
        };
      },
    }),
  ],
  session: { strategy: "jwt" },
  callbacks: {
    async jwt({ token, account, user, trigger }) {
      // Disparado explicitamente pelo client via useSession().update()
      // depois de POST /auth/trocar-senha ter sucesso (ver
      // components/trocar-senha-form.tsx) — a troca já foi validada
      // server-side pelo backend antes disso, então é seguro só atualizar
      // a claim aqui, sem round-trip nenhum.
      if (trigger === "update") {
        token.mustChangePassword = false;
        return token;
      }

      if (account?.provider === "keycloak") {
        token.provider = "keycloak";
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
            token.mustChangePassword = Boolean(me.must_change_password);
          }
        } catch {
          // Backend indisponível no momento do login: a sessão segue sem
          // os flags — telas que exigem is_admin/is_fiscal tratam
          // undefined como "sem permissão" (nega por padrão).
        }

        return token;
      }

      if (account?.provider === "credentials" && user) {
        token.provider = "credentials";
        token.accessToken = (user as { accessToken: string }).accessToken;
        token.userId = user.id;
        token.isAdmin = Boolean((user as { isAdmin?: boolean }).isAdmin);
        token.isFiscal = Boolean((user as { isFiscal?: boolean }).isFiscal);
        token.mustChangePassword = Boolean(
          (user as { mustChangePassword?: boolean }).mustChangePassword
        );
        // Sem refresh token pro login local — expiresAt só serve pra
        // saber quando parar de usar este token (ver refreshAccessToken),
        // espelhando o TTL de internal/localauth.tokenTTL no backend (8h).
        // Não há como os dois ficarem "sincronizados de verdade" sem
        // decodificar o JWT aqui — 8h é reaproveitado como constante
        // conhecida, não descoberto dinamicamente.
        token.expiresAt = Math.floor(Date.now() / 1000) + 8 * 60 * 60;
        delete token.refreshToken;
        delete token.error;

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
        session.user.mustChangePassword = Boolean(token.mustChangePassword);
      }
      session.error = token.error as string | undefined;
      return session;
    },
  },
});

async function refreshAccessToken(token: JWT): Promise<JWT> {
  // Login local não tem refresh token (ver o comentário no callback jwt)
  // — quando o token de 8h expira, o único caminho é logar de novo.
  // session.error sinaliza isso do mesmo jeito que uma falha de refresh
  // do Keycloak sinalizaria, pro resto da aplicação não precisar
  // distinguir os dois casos.
  if (token.provider === "credentials") {
    return { ...token, error: "RefreshAccessTokenError" };
  }

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

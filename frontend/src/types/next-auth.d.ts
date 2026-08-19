import type { DefaultSession } from "next-auth";
import type { JWT as DefaultJWT } from "next-auth/jwt";

// Augmenta os tipos do Auth.js com os campos extras que colocamos no token
// (jwt callback) e no que expomos em `session` (session callback) —
// ver src/auth.ts para o porquê de accessToken/refreshToken ficarem só no
// token e nunca em session.
declare module "next-auth" {
  interface Session extends DefaultSession {
    user?: DefaultSession["user"] & {
      id?: string;
      isAdmin: boolean;
      isFiscal: boolean;
      // true = conta de login local com senha temporária, ainda não
      // trocada — o frontend usa isso pra redirecionar pra
      // /trocar-senha antes de liberar o resto da navegação (ver
      // components/providers.tsx ou o gate equivalente).
      mustChangePassword: boolean;
    };
    error?: string;
  }

  // Campos extras devolvidos por authorize() do provider Credentials
  // (ver src/auth.ts) — não fazem parte do tipo padrão `User`.
  interface User {
    accessToken?: string;
    isAdmin?: boolean;
    isFiscal?: boolean;
    mustChangePassword?: boolean;
  }
}

declare module "next-auth/jwt" {
  interface JWT extends DefaultJWT {
    // "keycloak" | "credentials" — de qual provider esta sessão veio;
    // usado só pra decidir COMO renovar o accessToken quando expira (ver
    // refreshAccessToken em src/auth.ts). Login local não tem refresh
    // token de verdade.
    provider?: string;
    accessToken?: string;
    refreshToken?: string;
    expiresAt?: number;
    userId?: string;
    isAdmin?: boolean;
    isFiscal?: boolean;
    mustChangePassword?: boolean;
    error?: string;
  }
}

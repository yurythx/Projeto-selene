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
    };
    error?: string;
  }
}

declare module "next-auth/jwt" {
  interface JWT extends DefaultJWT {
    accessToken?: string;
    refreshToken?: string;
    expiresAt?: number;
    userId?: string;
    isAdmin?: boolean;
    isFiscal?: boolean;
    error?: string;
  }
}

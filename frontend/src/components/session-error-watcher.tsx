"use client";

import { useEffect } from "react";
import { useSession, signOut } from "next-auth/react";

/**
 * Sem UI própria — observa `session.error` (setado em src/auth.ts quando
 * o refresh do accessToken falha: refresh token do Keycloak
 * expirado/revogado, ou os 8h do token de login local vencendo, que não
 * tem refresh de verdade — ver o comentário em refreshAccessToken) e
 * desloga automaticamente em vez de deixar o usuário preso numa sessão
 * que o próprio Auth.js já sabe que nunca mais vai se autenticar sozinha.
 *
 * Complementa requireApi (lib/api/client.ts), que cobre o caso mais
 * comum na prática — o backend rejeitando o token direto numa chamada de
 * API (ex: reinício do backend invalidando sessões de login local
 * silenciosamente, ver a LIMITAÇÃO CONHECIDA em
 * internal/localauth/localauth.go). Este componente cobre os casos em
 * que o próprio Auth.js, no client, já sabe de antemão que o token não
 * presta mais, antes mesmo de qualquer chamada ao backend falhar.
 */
export function SessionErrorWatcher() {
  const { data: session } = useSession();

  useEffect(() => {
    if (session?.error === "RefreshAccessTokenError") {
      signOut({ redirectTo: "/login" });
    }
  }, [session?.error]);

  return null;
}

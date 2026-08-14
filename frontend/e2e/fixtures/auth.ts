import { encode } from "next-auth/jwt";
import type { BrowserContext } from "@playwright/test";
import { TEST_AUTH_SECRET } from "../env";

/**
 * Injeta uma sessão Auth.js válida direto no cookie, sem passar pelo
 * fluxo interativo de login do Keycloak — não temos (nem deveríamos ter)
 * credenciais de um usuário real da instância de produção da prefeitura
 * pra usar em CI. O fluxo de login em si (redirect, client_id, issuer,
 * discovery document) já foi validado manualmente contra o Keycloak real
 * antes deste commit; o que os testes E2E cobrem é o comportamento do
 * NOSSO app a partir de uma sessão válida.
 *
 * O token é codificado com next-auth/jwt.encode() usando o mesmo
 * AUTH_SECRET que o servidor de teste usa — o servidor decodifica esse
 * cookie exatamente como decodificaria um cookie de sessão real emitido
 * após um login de verdade.
 */
export async function injetarSessao(
  context: BrowserContext,
  usuario: { id: string; nome: string; email: string; isFiscal: boolean; isAdmin: boolean }
) {
  const secret = TEST_AUTH_SECRET;

  // Servidor de teste roda em http://localhost — sem TLS, então o Auth.js
  // usa o cookie SEM o prefixo __Secure- (ver useSecureCookies em
  // @auth/core, decidido por url.protocol, não por NODE_ENV).
  const cookieName = "authjs.session-token";

  const value = await encode({
    secret,
    salt: cookieName,
    maxAge: 30 * 24 * 60 * 60,
    token: {
      sub: usuario.id,
      name: usuario.nome,
      email: usuario.email,
      userId: usuario.id,
      isFiscal: usuario.isFiscal,
      isAdmin: usuario.isAdmin,
      accessToken: `fake-access-token-${usuario.id}`,
      refreshToken: `fake-refresh-token-${usuario.id}`,
      // Bem no futuro — os testes não exercitam o fluxo de refresh.
      expiresAt: Math.floor(Date.now() / 1000) + 60 * 60,
    },
  });

  await context.addCookies([
    {
      name: cookieName,
      value,
      domain: "localhost",
      path: "/",
      httpOnly: true,
      secure: false,
      sameSite: "Lax",
    },
  ]);
}

export const FISCAL = {
  id: "fiscal-1",
  nome: "Fiscal Teste",
  email: "fiscal@example.com",
  isFiscal: true,
  isAdmin: false,
};

export const ADMIN = {
  id: "admin-1",
  nome: "Admin Teste",
  email: "admin@example.com",
  isFiscal: false,
  isAdmin: true,
};

/**
 * Sanitiza um destino de redirecionamento pós-login (`?callbackUrl=...`)
 * pra evitar Open Redirect (CWE-601): `callbackUrl` vem de um query param,
 * ou seja, controlado por quem monta o link — um atacante pode mandar
 * `/login?callbackUrl=https://site-falso.example.com/phishing` pra uma
 * vítima, que loga de verdade na sessão dela e é redirecionada, logo em
 * seguida, pra uma página que finge ser a de sempre (session fixation /
 * phishing pós-login, o link em si parece legítimo porque aponta pro
 * domínio real do Selene).
 *
 * Só aceita caminhos relativos que começam com exatamente UMA barra:
 * - `https://evil.example.com/x` → rejeitado (tem esquema/host).
 * - `//evil.example.com/x` → rejeitado (URL "protocol-relative" — o
 *   navegador resolve isso pro host `evil.example.com`, mesmo sem
 *   `https:` explícito na frente; SEM essa checagem, isso escaparia de um
 *   regex ingênuo que só olha `startsWith("/")`).
 * - `/\evil.example.com` → rejeitado (alguns navegadores normalizam
 *   barra invertida pra barra normal antes de resolver a URL).
 * - `/kanban`, `/contratos/123` → aceitos.
 *
 * Qualquer coisa que não passe cai no default seguro ("/"), nunca lança.
 */
export function sanitizeCallbackUrl(callbackUrl: string | undefined | null): string {
  if (!callbackUrl) return "/";
  if (!/^\/(?!\/|\\)/.test(callbackUrl)) return "/";
  return callbackUrl;
}

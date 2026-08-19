import { describe, it, expect } from "vitest";
import { sanitizeCallbackUrl } from "./safe-redirect";

describe("sanitizeCallbackUrl", () => {
  it("aceita caminhos relativos simples", () => {
    expect(sanitizeCallbackUrl("/kanban")).toBe("/kanban");
    expect(sanitizeCallbackUrl("/contratos/123")).toBe("/contratos/123");
    expect(sanitizeCallbackUrl("/")).toBe("/");
  });

  it("cai no default seguro sem callbackUrl nenhum", () => {
    expect(sanitizeCallbackUrl(undefined)).toBe("/");
    expect(sanitizeCallbackUrl(null)).toBe("/");
    expect(sanitizeCallbackUrl("")).toBe("/");
  });

  it("rejeita URL absoluta pra outro domínio (open redirect direto)", () => {
    expect(sanitizeCallbackUrl("https://site-malicioso.example.com/phishing")).toBe("/");
    expect(sanitizeCallbackUrl("http://site-malicioso.example.com")).toBe("/");
  });

  it("rejeita URL protocol-relative (//host, sem esquema explícito)", () => {
    // O navegador resolve "//evil.com" pro host evil.com no mesmo
    // protocolo da página atual — um regex que só checa startsWith("/")
    // deixaria passar isso.
    expect(sanitizeCallbackUrl("//site-malicioso.example.com/phishing")).toBe("/");
  });

  it("rejeita o truque de barra invertida (alguns navegadores normalizam \\ pra /)", () => {
    expect(sanitizeCallbackUrl("/\\site-malicioso.example.com")).toBe("/");
  });

  it("rejeita esquemas não-http (javascript:, data:) por não começarem com /", () => {
    expect(sanitizeCallbackUrl("javascript:alert(1)")).toBe("/");
    expect(sanitizeCallbackUrl("data:text/html,<script>alert(1)</script>")).toBe("/");
  });
});

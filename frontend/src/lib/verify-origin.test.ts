import { describe, it, expect } from "vitest";
import { assertOrigemSegura } from "./verify-origin";

function req(headers: Record<string, string>) {
  return new Request("http://localhost:3000/api/contratos", {
    method: "POST",
    headers,
  });
}

describe("assertOrigemSegura", () => {
  it("permite quando Origin e Host batem", () => {
    expect(
      assertOrigemSegura(req({ origin: "http://localhost:3000", host: "localhost:3000" }))
    ).toBeNull();
  });

  it("permite quando não há Origin (client HTTP direto, não navegador)", () => {
    expect(assertOrigemSegura(req({ host: "localhost:3000" }))).toBeNull();
  });

  it("bloqueia com 403 quando Origin é de outro site", () => {
    const resposta = assertOrigemSegura(
      req({ origin: "https://site-malicioso.com", host: "localhost:3000" })
    );
    expect(resposta).not.toBeNull();
    expect(resposta!.status).toBe(403);
  });

  it("bloqueia com 403 quando o header Origin é inválido", () => {
    const resposta = assertOrigemSegura(req({ origin: "não-é-uma-url", host: "localhost:3000" }));
    expect(resposta).not.toBeNull();
    expect(resposta!.status).toBe(403);
  });
});

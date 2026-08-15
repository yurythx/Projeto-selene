import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

// CSP com nonce (ver src/proxy.ts): script-src é apertado
// ('nonce-x' 'strict-dynamic', sem 'unsafe-inline'), mas style-src
// mantém 'unsafe-inline' de propósito, porque nonce não cobre o
// atributo `style="..."` inline que o base-ui usa (via floating-ui)
// pra posicionar Select/Dialog/Popover — só tags <style>. Este spec
// existe pra pegar uma regressão silenciosa: se um dia alguém apertar
// style-src sem entender essa limitação, os componentes aqui
// quebrariam (popup deslocado/invisível) e o navegador reportaria a
// violação no console — verificamos os dois sinais.
test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

test.describe("Content-Security-Policy", () => {
  test("header carrega um nonce por requisição, sem 'unsafe-inline' em script-src", async ({
    page,
  }) => {
    const resposta = await page.goto("/contratos");
    const csp = resposta?.headers()["content-security-policy"];
    expect(csp).toBeTruthy();
    expect(csp).toMatch(/script-src 'self' 'nonce-[^']+' 'strict-dynamic'/);
    expect(csp).not.toContain("script-src 'self' 'unsafe-inline'");
  });

  test("abrir Select/Dialog (novo contrato, novo processo) não gera violação de CSP no console", async ({
    page,
  }) => {
    const violacoesCsp: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error" && /content security policy/i.test(msg.text())) {
        violacoesCsp.push(msg.text());
      }
    });

    await page.goto("/contratos");
    await page.getByRole("button", { name: "Novo contrato" }).click();
    await page.getByRole("combobox", { name: "Tipo de objeto" }).click();
    await page.getByText("Permanente").click();
    await page.keyboard.press("Escape");

    await page.goto("/kanban");
    await page.getByRole("button", { name: "Novo processo" }).click();
    await page.getByRole("combobox", { name: /contrato/i }).click();
    await page.getByText("1/2026 — Fornecedora Seed Ltda").click();

    expect(violacoesCsp, `violações de CSP encontradas: ${violacoesCsp.join("\n")}`).toEqual([]);
  });
});

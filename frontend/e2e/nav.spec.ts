import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

test.describe("barra de navegação", () => {
  test("alterna pra tema escuro e o preferência persiste após reload", async ({ page }) => {
    const violacoesCSP: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error" && msg.text().includes("Content Security Policy")) {
        violacoesCSP.push(msg.text());
      }
    });

    await page.goto("/kanban");
    await expect(page.locator("html")).toHaveClass(/light|dark/);

    // next-themes injeta um <script> inline no <head> que aplica o tema
    // salvo ANTES do primeiro paint — a CSP deste app não tem
    // 'unsafe-inline' em script-src (ver proxy.ts), então sem passar o
    // nonce da requisição pro ThemeProvider (app/layout.tsx →
    // components/providers.tsx) esse script seria bloqueado.
    await page.getByRole("button", { name: "Alternar tema" }).click();
    await page.getByText("Escuro", { exact: true }).click();
    await expect(page.locator("html")).toHaveClass(/dark/);

    await page.reload();
    await page.waitForLoadState("networkidle");
    await expect(page.locator("html")).toHaveClass(/dark/);

    expect(violacoesCSP).toEqual([]);
  });

  // Regressão: DropdownMenuItem é @base-ui/react/menu (não Radix) — a prop
  // que existe de verdade é onClick, não onSelect (ver MenuItemProps). O
  // botão "Sair" usava onSelect e nunca chamava signOut() de verdade;
  // achado ao testar o toggle de tema acima, que tinha o mesmo problema.
  test("'Sair' desloga de verdade e volta pro /login", async ({ page }) => {
    await page.goto("/kanban");
    await page.getByRole("button", { name: `Menu de ${FISCAL.nome}` }).click();
    await page.getByText("Sair", { exact: true }).click();

    await expect(page).toHaveURL(/\/login/);

    await page.goto("/kanban");
    await expect(page).toHaveURL(/\/login/);
  });
});

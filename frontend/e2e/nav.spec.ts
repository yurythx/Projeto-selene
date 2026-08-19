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
    //
    // Um clique só, sem dropdown — ThemeToggle troca direto pro tema
    // oposto (pedido explícito do usuário: sem menu "sistema").
    await page.getByRole("button", { name: "Mudar para tema escuro" }).click();
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

  test("recolhe a sidebar pro modo só-ícone e expande de volta", async ({ page }) => {
    await page.goto("/kanban");

    const linkKanban = page.getByRole("link", { name: "Kanban" });
    await expect(linkKanban).toBeVisible();
    await expect(linkKanban).toHaveText(/Kanban/);

    await page.getByRole("button", { name: "Recolher barra lateral" }).click();

    // O link continua no DOM (mesmo nome acessível, via aria-label) —
    // só o texto visível some, o ícone continua clicável.
    await expect(linkKanban).toBeVisible();
    await expect(linkKanban).toHaveText("");

    await page.getByRole("button", { name: "Expandir barra lateral" }).click();
    await expect(linkKanban).toHaveText(/Kanban/);
  });

  test("mobile: sidebar começa escondida, hambúrguer abre o drawer, e navegar fecha ele", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto("/kanban");

    // Escondida por padrão (fora da tela) — o link existe no DOM mas não
    // deveria estar visível pro usuário nesse viewport.
    await expect(page.getByRole("link", { name: "Contratos" })).not.toBeInViewport();

    await page.getByRole("button", { name: "Abrir menu" }).click();
    await expect(page.getByRole("link", { name: "Contratos" })).toBeInViewport();

    await page.getByRole("link", { name: "Contratos" }).click();
    await expect(page).toHaveURL(/\/contratos/);
    // Drawer fecha sozinho ao navegar — o link de Kanban (agora fora da
    // rota ativa) não deveria estar visível de novo sem reabrir o menu.
    await expect(page.getByRole("link", { name: "Kanban" })).not.toBeInViewport();
  });

  test("sino de notificações: mostra o badge, abre a lista, e marcar como lida some com o badge", async ({
    page,
    request,
  }) => {
    await request.post(`${MOCK_BACKEND_URL}/__e2e__/seed-notificacoes`);
    await page.goto("/dashboard");

    const sino = page.getByRole("button", { name: "Notificações" });
    // "2" — as duas notificações seedadas, ambas não-lidas.
    await expect(sino.getByText("2")).toBeVisible();

    await sino.click();
    await expect(page.getByText("Faltam 10 dias para o fim da vigência do contrato")).toBeVisible();
    await expect(page.getByText("CND Federal vence em 20 dias")).toBeVisible();

    await page.getByRole("button", { name: "Marcar todas como lidas" }).click();
    // O badge de contagem some quando não há mais notificação não-lida.
    await expect(sino.getByText("2")).not.toBeVisible();
  });
});

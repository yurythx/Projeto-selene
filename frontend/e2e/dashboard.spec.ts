import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

test.describe("/dashboard", () => {
  test("\"/\" redireciona pro dashboard, com os KPIs e os atalhos pras outras telas", async ({ page }) => {
    await page.goto("/");

    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
    await expect(page.getByText("Contratos ativos")).toBeVisible();
    await expect(page.getByText("Processos em andamento")).toBeVisible();
    await expect(page.getByText("Processos por etapa")).toBeVisible();

    await page.getByRole("link", { name: "Ver todos os contratos" }).click();
    await expect(page).toHaveURL(/\/contratos/);
  });

  test("item 'Dashboard' na sidebar leva de volta pra cá", async ({ page }) => {
    await page.goto("/kanban");
    await page.getByRole("link", { name: "Dashboard" }).click();
    await expect(page).toHaveURL(/\/dashboard/);
  });
});

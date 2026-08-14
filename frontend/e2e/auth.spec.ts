import { test, expect } from "@playwright/test";

test.describe("autenticação", () => {
  test("redireciona pra /login quem não tem sessão", async ({ page }) => {
    await page.goto("/kanban");
    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByRole("button", { name: "Entrar com Keycloak" })).toBeVisible();
  });

  test("/contratos e /admin/usuarios também exigem sessão", async ({ page }) => {
    await page.goto("/contratos");
    await expect(page).toHaveURL(/\/login/);

    await page.goto("/admin/usuarios");
    await expect(page).toHaveURL(/\/login/);
  });
});

import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

test.describe("/fornecedores", () => {
  test("lista o fornecedor semeado e abre o dossiê", async ({ page }) => {
    await page.goto("/fornecedores");

    await expect(page.getByRole("link", { name: "Fornecedora Seed Ltda" })).toBeVisible();

    await page.getByRole("link", { name: "Fornecedora Seed Ltda" }).click();

    await expect(page.getByRole("heading", { name: "Fornecedora Seed Ltda" })).toBeVisible();
    await expect(page.getByRole("link", { name: "1/2026" })).toBeVisible();
    await expect(page.getByText("Sem dados suficientes")).toBeVisible();
  });
});

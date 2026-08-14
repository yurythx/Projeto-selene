import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

test.describe("/contratos", () => {
  test("lista o contrato semeado e cria um novo", async ({ page }) => {
    await page.goto("/contratos");

    await expect(page.getByRole("link", { name: "1/2026" })).toBeVisible();
    await expect(page.getByText("Fornecedora Seed Ltda")).toBeVisible();

    await page.getByRole("button", { name: "Novo contrato" }).click();
    await page.getByLabel("Número do contrato").fill("2/2026");
    await page.getByLabel("Data de assinatura").fill("2026-02-01");
    await page.getByLabel("Empresa contratada").fill("Nova Fornecedora Ltda");
    await page.getByLabel("CNPJ").fill("11.111.111/0001-11");
    await page.getByRole("button", { name: "Salvar" }).click();

    await expect(page.getByRole("link", { name: "2/2026" })).toBeVisible();
    await expect(page.getByText("Nova Fornecedora Ltda")).toBeVisible();
  });

  test("abre o detalhe, edita e encerra o contrato", async ({ page }) => {
    await page.goto("/contratos");
    await page.getByRole("link", { name: "1/2026" }).click();

    await expect(page).toHaveURL(/\/contratos\/contrato-seed/);
    await expect(page.getByRole("heading", { name: "1/2026" })).toBeVisible();

    await page.getByRole("button", { name: "Editar" }).click();
    await page.getByLabel("Empresa contratada").fill("Fornecedora Renomeada Ltda");
    await page.getByRole("button", { name: "Salvar" }).click();
    await expect(page.getByText("Fornecedora Renomeada Ltda")).toBeVisible();

    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Encerrar contrato" }).click();
    await expect(page.getByText("Encerrado")).toBeVisible();
  });
});

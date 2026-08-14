import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

test.describe("/kanban", () => {
  test("mostra as 6 colunas e o processo semeado na Etapa 1", async ({ page }) => {
    await page.goto("/kanban");

    await expect(page.getByText("Elaborar OF / Pré-Empenho")).toBeVisible();
    await expect(page.getByText("Contabilidade")).toBeVisible();
    await expect(page.getByText("1/2026").first()).toBeVisible();
  });

  test("avançar com checklist incompleto mostra os documentos pendentes (422)", async ({ page }) => {
    await page.goto("/kanban");

    await page.getByText("1/2026").first().click();
    await expect(page.getByRole("heading", { name: /1\/2026/ })).toBeVisible();

    await page.getByRole("button", { name: "Avançar etapa" }).click();

    await expect(page.getByText("Checklist incompleto")).toBeVisible();
    // getByText simples ambíguo aqui: o mesmo nome de documento também
    // existe (oculto) nas opções do Select de upload, no mesmo dialog —
    // escopar pro <li> da lista de pendentes evita a violação de strict mode.
    await expect(page.getByRole("listitem").getByText("Ordem de Fornecimento (OF)")).toBeVisible();
    await expect(page.getByRole("listitem").getByText("Pré-Empenho")).toBeVisible();
    // O dialog continua aberto — não avançou de fato.
    await expect(page.getByRole("button", { name: "Avançar etapa" })).toBeVisible();
  });

  test("abre um processo novo e avança com sucesso pra Etapa 2", async ({ page }) => {
    await page.goto("/kanban");

    await page.getByRole("button", { name: "Novo processo" }).click();
    await page.getByRole("combobox", { name: /contrato/i }).click();
    await page.getByText("1/2026 — Fornecedora Seed Ltda").click();
    await page.getByLabel("Mês de referência").fill("03/2026");
    await page.getByRole("button", { name: "Abrir processo" }).click();

    await expect(page.getByText("Processo aberto na Etapa 1.")).toBeVisible();

    await page.getByText("03/2026").click();
    await page.getByRole("button", { name: "Avançar etapa" }).click();

    await expect(page.getByText("Processo avançou de etapa.")).toBeVisible();
  });

  test("anexa um documento ao processo", async ({ page }) => {
    await page.goto("/kanban");
    await page.getByText("1/2026").first().click();

    await expect(page.getByText("Nenhum documento anexado ainda.")).toBeVisible();

    await page.getByRole("combobox", { name: /tipo de documento/i }).click();
    await page.getByText("Ordem de Fornecimento (OF)").click();
    await page.setInputFiles("#arquivo", {
      name: "documento.pdf",
      mimeType: "application/pdf",
      buffer: Buffer.from("%PDF-1.4 conteúdo de teste"),
    });
    await page.getByRole("button", { name: "Anexar" }).click();

    await expect(page.getByText("Documento anexado.")).toBeVisible();
    await expect(page.getByText("arquivo-teste.pdf")).toBeVisible();
  });

  test("registra uma vistoria de campo e anexa uma foto", async ({ page }) => {
    await page.goto("/kanban");
    await page.getByText("1/2026").first().click();

    await page.getByRole("button", { name: "Vistorias de Campo" }).click();
    await expect(page.getByText("Nenhuma vistoria registrada ainda.")).toBeVisible();

    await page.getByLabel("Nova vistoria — observações").fill("Execução conforme o especificado.");
    await page.getByRole("button", { name: "Registrar vistoria" }).click();

    await expect(page.getByText("Vistoria registrada.")).toBeVisible();
    await expect(page.getByText("Execução conforme o especificado.")).toBeVisible();
    await expect(page.getByText("0 foto(s) anexada(s)")).toBeVisible();

    await page.setInputFiles('input[type="file"][accept="image/*"]', {
      name: "foto.jpg",
      mimeType: "image/jpeg",
      buffer: Buffer.from("conteúdo de imagem de teste"),
    });

    await expect(page.getByText("Foto anexada.")).toBeVisible();
    await expect(page.getByText("1 foto(s) anexada(s)")).toBeVisible();
  });
});

import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

// SGF-Rondonópolis (adequação às IN SCL 01/2019 e 04/2021) — ver o plano
// em .claude/plans/projeto-selene-rippling-kite.md. Cobertura E2E que
// faltava (achado da auditoria de revisão geral): as features novas
// tinham teste unitário/componente, mas nenhum spec exercitava o fluxo
// completo no navegador.
test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

test.describe("Ocorrências (Kanban)", () => {
  test("registrar ocorrência bloqueia o avanço de etapa; regularizar libera de novo", async ({ page }) => {
    // Processo novo (não o "processo-checklist-pendente" semeado, cujo
    // /avancar sempre responde 422 no stub — usaríamos o motivo errado
    // pra explicar por que o botão sumiu) — mesmo fluxo de
    // kanban.spec.ts "abre um processo novo e avança com sucesso".
    await page.goto("/kanban");
    await page.getByRole("button", { name: "Novo processo" }).click();
    await page.getByRole("combobox", { name: /contrato/i }).click();
    await page.getByText("1/2026 — Fornecedora Seed Ltda").click();
    await page.getByLabel("Mês de referência").fill("05/2026");
    await page.getByRole("button", { name: "Abrir processo" }).click();
    await expect(page.getByText("Processo aberto na Etapa 1.")).toBeVisible();

    await page.getByText("05/2026").click();
    await expect(page.getByRole("button", { name: "Avançar etapa" })).toBeVisible();

    await page.getByRole("button", { name: "Ocorrências" }).click();
    await expect(page.getByText("Nenhuma ocorrência registrada.")).toBeVisible();

    await page.getByLabel("Nova ocorrência").fill("Atraso na entrega dos documentos.");
    await page.getByRole("button", { name: "Registrar ocorrência" }).click();
    await expect(
      page.getByText("Ocorrência registrada. O avanço de etapa fica bloqueado até regularizar.")
    ).toBeVisible();
    await expect(page.getByText("Atraso na entrega dos documentos.")).toBeVisible();

    // Fecha o dialog de ocorrências pra conferir o efeito no drawer do
    // processo por trás — allowed_actions é recalculado (invalidação de
    // query), o botão "Avançar etapa" deve ter sumido de verdade, não só
    // visualmente por trás do dialog aberto.
    await page.keyboard.press("Escape");
    await expect(page.getByRole("button", { name: "Avançar etapa" })).not.toBeVisible();
    await expect(page.getByText("Pendência — ocorrência aberta")).toBeVisible();

    // Ciclo de vida completo: Registrada -> Notificada -> Em tratamento
    // -> Regularizada.
    await page.getByRole("button", { name: "Ocorrências" }).click();
    await page.getByRole("button", { name: "Notificar Gestor" }).click();
    await expect(page.getByText("Notificada ao Gestor").first()).toBeVisible();

    await page.getByRole("button", { name: "Iniciar tratamento" }).click();
    await expect(page.getByText("Em tratamento").first()).toBeVisible();

    await page.getByRole("button", { name: "Regularizar" }).click();
    await expect(page.getByText("Regularizada").first()).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(page.getByRole("button", { name: "Avançar etapa" })).toBeVisible();
  });
});

test.describe("Empenho (contrato)", () => {
  test("registra um empenho, um reforço e o saldo reflete os dois", async ({ page }) => {
    await page.goto("/contratos");
    await page.getByRole("link", { name: "1/2026" }).click();

    await expect(page.getByText("Nenhum empenho registrado ainda.")).toBeVisible();

    await page.getByRole("button", { name: "Novo empenho" }).click();
    await page.getByLabel("Número do empenho").fill("500/2026");
    await page.getByLabel("Data de emissão").fill("2026-01-05");
    await page.getByLabel("Valor inicial (R$)").fill("1000,00");
    await page.getByRole("button", { name: "Registrar" }).click();

    await expect(page.getByText("Empenho registrado.")).toBeVisible();
    await expect(page.getByText("Empenho nº 500/2026")).toBeVisible();
    await expect(page.getByText("Saldo: R$ 1.000,00")).toBeVisible();

    await page.getByRole("button", { name: "Reforço/Anulação" }).click();
    await page.getByRole("combobox", { name: "Tipo" }).click();
    await page.getByText("Reforço (soma ao saldo)").click();
    await page.getByLabel("Valor (R$)").fill("500,00");
    await page.getByRole("button", { name: "Registrar" }).click();

    await expect(page.getByText("Movimentação registrada.")).toBeVisible();
    await expect(page.getByText("Saldo: R$ 1.500,00")).toBeVisible();
  });
});

test.describe("Designações (contrato)", () => {
  test("card de histórico de designação renderiza sem erro", async ({ page }) => {
    await page.goto("/contratos");
    await page.getByRole("link", { name: "1/2026" }).click();

    await expect(page.getByText("Histórico de designação (SGF)")).toBeVisible();
    await expect(page.getByText("Nenhuma designação registrada ainda.")).toBeVisible();
  });
});

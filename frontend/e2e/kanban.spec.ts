import { test, expect, type Page } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

/**
 * Simula um arraste de verdade (Pointer Events, não a API nativa de HTML5
 * Drag and Drop — dnd-kit usa a primeira, então `locator.dragTo()` do
 * Playwright, que simula a segunda, não funciona aqui).
 *
 * Passos extras e pausas curtas entre eles não são só paranoia: o
 * PointerSensor do dnd-kit processa pointermove de forma assíncrona
 * (requestAnimationFrame), então uma sequência mouse.down → mouse.move
 * (1 salto) → mouse.up disparada rápido demais pode terminar o gesto
 * antes do React re-renderizar o estado "isOver" da coluna de destino —
 * flaky em CI mesmo com o drag funcionando de verdade na aplicação.
 */
interface Retangulo {
  x: number;
  y: number;
  width: number;
  height: number;
}

async function arrastarCard(page: Page, origemBox: Retangulo, destinoBox: Retangulo) {
  const origemX = origemBox.x + origemBox.width / 2;
  const origemY = origemBox.y + origemBox.height / 2;
  const destinoX = destinoBox.x + destinoBox.width / 2;
  const destinoY = destinoBox.y + Math.min(60, destinoBox.height / 2);

  await page.mouse.move(origemX, origemY);
  await page.mouse.down();
  // O PointerSensor do dnd-kit registra os listeners de pointermove/
  // pointerup (document-level) dentro do handler de pointerdown — sem
  // essa pausa, o primeiro passo do movimento abaixo pode disparar antes
  // desses listeners existirem de verdade, e o gesto inteiro é ignorado
  // silenciosamente (sem erro, só não vira um arraste).
  await page.waitForTimeout(100);

  // Trajeto em muitos passos pequenos (não 2-3 saltos grandes): cada
  // ponto intermediário é um pointermove real que o PointerSensor
  // processa via requestAnimationFrame — poucos pontos aumentam a
  // chance de o mouse.up() chegar antes do React assentar qual coluna
  // está "isOver".
  const pontosX = 12;
  for (let i = 1; i <= pontosX; i++) {
    const x = origemX + ((destinoX - origemX) * i) / pontosX;
    const y = origemY + ((destinoY - origemY) * i) / pontosX;
    await page.mouse.move(x, y);
    await page.waitForTimeout(20);
  }

  await page.waitForTimeout(200); // deixa o dnd-kit assentar "isOver" antes de soltar
  await page.mouse.up();
  await page.waitForTimeout(100); // dá tempo do onDragEnd/mutation disparar antes das asserções
}

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

  test("arrasta um card da Etapa 1 pra Etapa 2 (drag-and-drop) e avança de verdade", async ({
    page,
  }) => {
    await page.goto("/kanban");

    await page.getByRole("button", { name: "Novo processo" }).click();
    await page.getByRole("combobox", { name: /contrato/i }).click();
    await page.getByText("1/2026 — Fornecedora Seed Ltda").click();
    await page.getByLabel("Mês de referência").fill("06/2026");
    await page.getByRole("button", { name: "Abrir processo" }).click();
    await expect(page.getByText("Processo aberto na Etapa 1.")).toBeVisible();

    // O card recém-criado tem "06/2026" na badge de mês — dentro do
    // data-slot="card" do shadcn (Card não define testid próprio, mas
    // sempre marca data-slot="card").
    const card = page.locator('[data-slot="card"]', { hasText: "06/2026" });
    await expect(card).toBeVisible();
    const colunaEtapa2 = page.getByTestId("kanban-coluna-2");

    const cardBox = await card.boundingBox();
    const colunaBox = await colunaEtapa2.boundingBox();
    if (!cardBox || !colunaBox) throw new Error("bounding box não encontrado");

    await arrastarCard(page, cardBox, colunaBox);

    await expect(page.getByText("Processo avançou de etapa.")).toBeVisible();
    // O card não deveria mais aparecer na coluna de origem (Etapa 1).
    await expect(
      page.getByTestId("kanban-coluna-1").locator('[data-slot="card"]', { hasText: "06/2026" })
    ).not.toBeVisible();
    await expect(colunaEtapa2.locator('[data-slot="card"]', { hasText: "06/2026" })).toBeVisible();
  });

  test("arrastar um card pra uma coluna que não é a próxima (pular etapa) não move nada", async ({
    page,
  }) => {
    await page.goto("/kanban");

    await page.getByRole("button", { name: "Novo processo" }).click();
    await page.getByRole("combobox", { name: /contrato/i }).click();
    await page.getByText("1/2026 — Fornecedora Seed Ltda").click();
    await page.getByLabel("Mês de referência").fill("07/2026");
    await page.getByRole("button", { name: "Abrir processo" }).click();
    await expect(page.getByText("Processo aberto na Etapa 1.")).toBeVisible();

    const card = page.locator('[data-slot="card"]', { hasText: "07/2026" });
    const colunaEtapa3 = page.getByTestId("kanban-coluna-3"); // pula a Etapa 2 — inválido

    const cardBox = await card.boundingBox();
    const colunaBox = await colunaEtapa3.boundingBox();
    if (!cardBox || !colunaBox) throw new Error("bounding box não encontrado");

    await arrastarCard(page, cardBox, colunaBox);

    // Nenhum toast de sucesso, e o card continua na Etapa 1.
    await expect(page.getByText("Processo avançou de etapa.")).not.toBeVisible();
    await expect(page.getByTestId("kanban-coluna-1").locator('[data-slot="card"]', { hasText: "07/2026" })).toBeVisible();
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

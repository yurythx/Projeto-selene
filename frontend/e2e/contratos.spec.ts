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

    const comboTipo = page.getByRole("combobox", { name: "Tipo de objeto" });
    await comboTipo.click();
    await page.getByRole("option", { name: "Serviço" }).click();
    // Regressão: sem a prop `items` no Select (ver components/ui/select e
    // o comentário em contratos-filtro.tsx), o base-ui perde o rótulo
    // registrado assim que o popup fecha e o trigger passa a mostrar
    // "SERVICO" (o value cru) em vez de "Serviço" — bug real reportado em
    // produção, mesma causa do bug no select de tipo de documento.
    await expect(comboTipo).toContainText("Serviço");

    await page.getByRole("button", { name: "Salvar" }).click();

    await expect(page.getByRole("link", { name: "2/2026" })).toBeVisible();
    await expect(page.getByText("Nova Fornecedora Ltda")).toBeVisible();
  });

  test("pagina a listagem quando passa de uma página", async ({ page, request }) => {
    // 1 contrato já semeado ("1/2026") + 20 novos = 21, tamanho de
    // página 20 (ver TAMANHO_PAGINA em contratos/page.tsx) — fecha
    // exatamente em 2 páginas, sem depender de um número mágico maior.
    await request.post(`${MOCK_BACKEND_URL}/__e2e__/seed-muitos-contratos`, { data: { quantidade: 20 } });

    await page.goto("/contratos");
    await expect(page.getByText("21 contratos cadastrados.")).toBeVisible();
    await expect(page.getByText("Página 1 de 2")).toBeVisible();

    const primeiraLinha = await page.getByRole("row").nth(1).innerText();

    await page.getByRole("button", { name: "Próxima" }).click();
    await expect(page).toHaveURL(/pagina=2/);
    await expect(page.getByText("Página 2 de 2")).toBeVisible();

    // Página 2 mostra conteúdo diferente da página 1 — prova que o
    // "pagina" da URL realmente troca a fatia de dados vinda do
    // backend, não só o número no rótulo.
    const segundaLinha = await page.getByRole("row").nth(1).innerText();
    expect(segundaLinha).not.toBe(primeiraLinha);

    await page.getByRole("button", { name: "Anterior" }).click();
    await expect(page).toHaveURL(/pagina=1/);
    await expect(page.getByText("Página 1 de 2")).toBeVisible();
  });

  test("abre o detalhe, edita e encerra o contrato", async ({ page }) => {
    await page.goto("/contratos");
    await page.getByRole("link", { name: "1/2026" }).click();

    await expect(page).toHaveURL(/\/contratos\/11111111-1111-4111-8111-111111111111/);
    await expect(page.getByRole("heading", { name: "1/2026" })).toBeVisible();

    await page.getByRole("button", { name: "Editar" }).click();
    await page.getByLabel("Empresa contratada").fill("Fornecedora Renomeada Ltda");
    await page.getByRole("button", { name: "Salvar" }).click();
    await expect(page.getByText("Fornecedora Renomeada Ltda")).toBeVisible();

    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Encerrar contrato" }).click();
    // getByText("Encerrado") sem exact bate tanto no badge de situação
    // quanto no toast "Contrato encerrado." (getByText normaliza case) —
    // exact:true escopa pro badge, que é o que a asserção quer checar.
    await expect(page.getByText("Encerrado", { exact: true })).toBeVisible();
  });
});

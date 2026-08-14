import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL, ADMIN } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
});

test.describe("/admin/usuarios", () => {
  test("usuário não-admin não consegue ver a lista", async ({ page, context }) => {
    await injetarSessao(context, FISCAL);
    await page.goto("/admin/usuarios");

    await expect(page.getByText("Você não tem permissão para acessar esta página.")).toBeVisible();
    await expect(page.getByText("admin@example.com")).not.toBeVisible();
  });

  test("admin lista os usuários e edita permissões", async ({ page, context }) => {
    await injetarSessao(context, ADMIN);
    await page.goto("/admin/usuarios");

    await expect(page.getByText("fiscal@example.com")).toBeVisible();
    await expect(page.getByText("admin@example.com")).toBeVisible();

    const linhaFiscal = page.getByRole("row", { name: /Fiscal Teste/ });
    await linhaFiscal.getByRole("button", { name: "Editar" }).click();

    await page.getByRole("switch", { name: /é administrador/i }).click();
    await page.getByRole("button", { name: "Salvar" }).click();

    await expect(page.getByText("Usuário atualizado.")).toBeVisible();
  });

  // Duas sessões diferentes precisam de dois testes (contexto/página
  // novos) em vez de trocar a sessão no meio de um teste só: o
  // SessionProvider do next-auth/react cacheia a sessão no client depois
  // do primeiro fetch, então um goto() pra mesma URL depois de trocar o
  // cookie não necessariamente reflete a sessão nova a tempo — troca de
  // usuário no meio da mesma aba não é um cenário real de qualquer forma.
  test("item de nav 'Administração' não aparece pra fiscal", async ({ page, context }) => {
    await injetarSessao(context, FISCAL);
    await page.goto("/kanban");
    await expect(page.getByRole("link", { name: "Administração" })).not.toBeVisible();
  });

  test("item de nav 'Administração' aparece pra admin", async ({ page, context }) => {
    await injetarSessao(context, ADMIN);
    await page.goto("/kanban");
    await expect(page.getByRole("link", { name: "Administração" })).toBeVisible();
  });
});

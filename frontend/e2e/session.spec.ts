import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ context, request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  await injetarSessao(context, FISCAL);
});

// Regressão: um 401 do backend (token inválido/expirado — cenário real:
// reinício do backend invalida sessões de login local, ver a LIMITAÇÃO
// CONHECIDA em internal/localauth/localauth.go) borbulhava pro
// (app)/error.tsx, que mostra "backend pode estar indisponível, tente de
// novo" — mensagem enganosa (tentar de novo reenvia o mesmo token quebrado
// e cai no mesmo 401) e sem caminho de volta pro login. requireApi
// (lib/api/client.ts) trata 401 como sessão inválida e redireciona.
test("401 do backend redireciona pro /login em vez de mostrar erro genérico", async ({
  page,
  request,
}) => {
  await page.goto("/kanban");
  await expect(page.getByRole("heading", { name: "Kanban" })).toBeVisible();

  await request.post(`${MOCK_BACKEND_URL}/__e2e__/forcar-401`);

  await page.goto("/contratos");
  await expect(page).toHaveURL(/\/login/);
  await expect(page.getByText("Não foi possível carregar")).not.toBeVisible();
});

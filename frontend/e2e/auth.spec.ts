import { test, expect } from "@playwright/test";
import { MOCK_BACKEND_URL } from "./env";

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

// Login tradicional (usuário/senha) — ao contrário do resto dos specs
// (que injetam a sessão direto via fixtures/auth.ts), estes exercitam o
// fluxo de verdade: a página /login, o provider Credentials do Auth.js
// (src/auth.ts) e POST /api/v1/auth/login no mock-backend.
test.describe("login tradicional", () => {
  test.beforeEach(async ({ request }) => {
    await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
  });

  test("credenciais corretas autenticam e liberam o app", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("E-mail").fill("fiscal.local@example.com");
    await page.getByLabel("Senha").fill("senha12345");
    await page.getByRole("button", { name: "Entrar com e-mail e senha" }).click();

    // callbackUrl é "/", que por sua vez redireciona pra /dashboard
    // (app/(app)/page.tsx) — a sessão local já está válida nesse ponto.
    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  });

  test("credenciais erradas mostram mensagem sem sair da página", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("E-mail").fill("fiscal.local@example.com");
    await page.getByLabel("Senha").fill("senha-errada");
    await page.getByRole("button", { name: "Entrar com e-mail e senha" }).click();

    await expect(page.getByText("E-mail ou senha inválidos.")).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test("conta com senha temporária é redirecionada pra troca obrigatória", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("E-mail").fill("novo.local@example.com");
    await page.getByLabel("Senha").fill("temporaria123");
    await page.getByRole("button", { name: "Entrar com e-mail e senha" }).click();

    await expect(page).toHaveURL(/\/trocar-senha/);
    await expect(page.getByText("senha temporária")).toBeVisible();

    await page.getByLabel("Senha atual").fill("temporaria123");
    // exact:true — sem isso, getByLabel("Nova senha") também bate em
    // "Confirmar nova senha" (correspondência por substring do Playwright).
    await page.getByLabel("Nova senha", { exact: true }).fill("senhaNovaSegura123");
    await page.getByLabel("Confirmar nova senha").fill("senhaNovaSegura123");
    await page.getByRole("button", { name: "Trocar senha" }).click();

    // Troca de senha desliga mustChangePassword na sessão (via
    // useSession().update(), sem precisar logar de novo) — a partir daqui
    // o proxy.ts para de redirecionar pra /trocar-senha. router.push("/")
    // por sua vez cai em /dashboard (app/(app)/page.tsx redireciona).
    await expect(page).toHaveURL(/\/dashboard/);
  });
});

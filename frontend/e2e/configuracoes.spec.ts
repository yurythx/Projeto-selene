import { test, expect } from "@playwright/test";
import { injetarSessao, FISCAL, ADMIN } from "./fixtures/auth";
import { MOCK_BACKEND_URL } from "./env";

test.beforeEach(async ({ request }) => {
  await request.post(`${MOCK_BACKEND_URL}/__e2e__/reset`);
});

test.describe("/configuracoes", () => {
  test("usuário não-admin não consegue ver a lista de seções", async ({ page, context }) => {
    await injetarSessao(context, FISCAL);
    await page.goto("/configuracoes");

    await expect(page.getByText("Você não tem permissão para acessar esta página.")).toBeVisible();
    await expect(page.getByText("Keycloak / SSO")).not.toBeVisible();
  });

  test("admin vê as duas seções e navega pra Keycloak/SSO", async ({ page, context }) => {
    await injetarSessao(context, ADMIN);
    await page.goto("/configuracoes");

    await expect(page.getByText("Modelos de Documentos")).toBeVisible();
    await expect(page.getByText("Keycloak / SSO")).toBeVisible();

    await page.getByRole("link", { name: /Keycloak \/ SSO/ }).click();
    await expect(page).toHaveURL(/\/configuracoes\/keycloak$/);
    await expect(page.getByRole("heading", { name: "Keycloak / SSO" })).toBeVisible();
  });

  test("Keycloak/SSO: sem configuração salva mostra a origem de variáveis de ambiente", async ({
    page,
    context,
  }) => {
    await injetarSessao(context, ADMIN);
    await page.goto("/configuracoes/keycloak");

    await expect(page.getByText(/variáveis de ambiente do container/)).toBeVisible();
    await expect(page.getByText("Segredo (Client Secret): não configurado")).toBeVisible();
  });

  test("Keycloak/SSO: salva uma configuração nova e reflete no painel de status", async ({
    page,
    context,
  }) => {
    await injetarSessao(context, ADMIN);
    await page.goto("/configuracoes/keycloak");

    await page.getByLabel("Client ID").fill("selene-client");
    await page.getByLabel("Client Secret").fill("segredo-super-secreto");
    await page.getByLabel("Issuer URL").fill("https://sso.exemplo.gov.br/realms/selene");
    await page.getByRole("button", { name: "Salvar e aplicar" }).click();

    await expect(page.getByText("Configuração de Keycloak salva e aplicada.")).toBeVisible();
    await expect(page.getByText("configuração salva pela tela abaixo")).toBeVisible();
    await expect(page.getByText("Segredo (Client Secret): configurado")).toBeVisible();
    // O campo de segredo é limpo depois de salvar — nunca ecoa o valor
    // digitado de volta (o backend também nunca devolve o secret).
    await expect(page.getByLabel("Client Secret")).toHaveValue("");
  });

  test("Keycloak/SSO: issuer_url inválido mostra erro de validação sem enviar", async ({ page, context }) => {
    await injetarSessao(context, ADMIN);
    await page.goto("/configuracoes/keycloak");

    await page.getByLabel("Client ID").fill("selene-client");
    await page.getByLabel("Issuer URL").fill("não é uma url");
    await page.getByRole("button", { name: "Salvar e aplicar" }).click();

    await expect(page.getByText(/precisa ser uma URL válida/)).toBeVisible();
  });

  test("Diário Oficial: configura, testa a conexão, e busca contratos por nome", async ({
    page,
    context,
  }) => {
    await injetarSessao(context, ADMIN);
    await page.goto("/configuracoes");

    const linkSecao = page.getByRole("link", { name: /Diário Oficial/ });
    await expect(linkSecao).toBeVisible();
    await linkSecao.click();
    await expect(page).toHaveURL(/\/configuracoes\/diario-oficial$/);

    // Sem configuração salva ainda.
    await expect(page.getByText("nenhuma ainda")).toBeVisible();
    await expect(page.getByText("Chave de API: não configurada")).toBeVisible();

    await page.getByLabel("URL base da API").fill("https://diario.exemplo.gov.br/api");
    await page.getByLabel("Chave de API").fill("chave-super-secreta");
    await page.getByRole("button", { name: "Salvar" }).click();

    await expect(page.getByText("Configuração do Diário Oficial salva.")).toBeVisible();
    await expect(page.getByText("Chave de API: configurada")).toBeVisible();
    // Campo de chave é limpo depois de salvar — nunca ecoa de volta.
    await expect(page.getByLabel("Chave de API")).toHaveValue("");

    await page.getByRole("button", { name: "Testar conexão" }).click();
    await expect(page.getByText("O servidor respondeu", { exact: true })).toBeVisible();

    await page.getByRole("link", { name: "Ir para a busca →" }).click();
    await expect(page).toHaveURL(/\/configuracoes\/diario-oficial\/buscar$/);

    await page.getByLabel("Nome").fill("Fornecedora Teste Ltda");
    await page.getByRole("button", { name: "Buscar" }).click();

    await expect(page.getByText("1 resultado.")).toBeVisible();
    await expect(page.getByText("Fornecedora Teste Ltda")).toBeVisible();
  });
});

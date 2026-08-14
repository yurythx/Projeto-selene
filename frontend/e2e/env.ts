/**
 * Valores compartilhados entre playwright.config.ts (env do webServer) e
 * os specs/fixtures (que rodam no processo do test runner, não no do
 * webServer — env vars passadas só pro webServer não chegam aqui via
 * process.env). Importar a mesma constante dos dois lados evita
 * depender de propagação de variável de ambiente entre processos.
 */
export const APP_PORT = 3100;
export const MOCK_BACKEND_PORT = 4010;
export const MOCK_BACKEND_URL = `http://localhost:${MOCK_BACKEND_PORT}`;

// Só pra este ambiente de teste — nunca usado contra o Keycloak/backend
// reais.
export const TEST_AUTH_SECRET = "e2e-test-secret-nao-e-segredo-real-nunca-usar-em-producao";

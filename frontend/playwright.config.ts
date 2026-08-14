import { defineConfig, devices } from "@playwright/test";
import { APP_PORT, MOCK_BACKEND_PORT, TEST_AUTH_SECRET } from "./e2e/env";

// Valores fixos, só pra este ambiente de teste — nunca usados contra o
// Keycloak/backend reais. auth.ts lê AUTH_KEYCLOAK_* no import (via
// setEnvDefaults do Auth.js), então precisam existir mesmo sem serem
// usados de verdade (a sessão é injetada direto, ver e2e/fixtures/auth.ts).
const testEnv = {
  NODE_ENV: "production",
  API_URL: `http://localhost:${MOCK_BACKEND_PORT}`,
  AUTH_SECRET: TEST_AUTH_SECRET,
  AUTH_TRUST_HOST: "true",
  AUTH_KEYCLOAK_ID: "e2e-placeholder",
  AUTH_KEYCLOAK_SECRET: "e2e-placeholder",
  AUTH_KEYCLOAK_ISSUER: "http://localhost:9999/realms/e2e-placeholder",
  PORT: String(APP_PORT),
  HOSTNAME: "localhost",
};

export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*.spec.ts",
  fullyParallel: false,
  // Um worker só: os testes compartilham o mesmo mock backend em memória
  // (não há banco real pra isolar por transação) — mais simples confiar
  // em IDs únicos por teste do que introduzir paralelismo aqui.
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  // "github" anota falhas diretamente no diff do PR; "html" gera o
  // relatório em playwright-report/ que o CI sobe como artifact quando
  // algum teste falha (ver .github/workflows/ci.yml).
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: `http://localhost:${APP_PORT}`,
    trace: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: [
    {
      command: `npx tsx e2e/mock-backend.ts`,
      port: MOCK_BACKEND_PORT,
      env: { MOCK_BACKEND_PORT: String(MOCK_BACKEND_PORT) },
      reuseExistingServer: !process.env.CI,
    },
    {
      // `next start` não suporta "output: standalone" (ele mesmo avisa
      // isso) — roda o artefato standalone de verdade, o mesmo que o
      // Dockerfile empacota, pra testar contra o que de fato vai pra
      // produção. e2e/prepare-standalone.mjs copia public/ e
      // .next/static/ pra dentro de .next/standalone/, igual o Dockerfile.
      command: `npm run build && node e2e/prepare-standalone.mjs && node .next/standalone/server.js`,
      port: APP_PORT,
      env: testEnv,
      reuseExistingServer: !process.env.CI,
      timeout: 180_000,
    },
  ],
});

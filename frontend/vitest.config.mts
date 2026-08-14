import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    globals: true,
    exclude: ["node_modules", ".next"],
    // Pool "forks" (default) spawna um processo filho por worker — em
    // runners de CI com poucos CPUs (ex: GitHub Actions, 2 vCPUs) e sob
    // contêiner, isso pode travar no handshake inicial ("Timeout waiting
    // for worker to respond"), derrubando a suíte inteira com "no tests
    // found" mesmo com o código correto. Reproduzido localmente rodando
    // dentro de um container Docker simples (node:22). "threads" usa
    // worker_threads (sem spawn de processo) e é bem mais robusto nesse
    // cenário. fileParallelism:false força um único worker (equivalente
    // ao antigo poolOptions.threads.singleThread, removido no Vitest 4).
    pool: "threads",
    fileParallelism: false,
    env: {
      // lib/api/client.ts exige API_URL definida no import (falha rápido
      // se a env var real estiver faltando em runtime) — testes usam um
      // valor fixo, nunca o backend de verdade.
      API_URL: "http://backend.test",
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(dirname, "./src"),
      // "server-only" lança um erro incondicional fora do bundling
      // RSC do Next (é assim que ele impede import client-side em
      // produção) — sob Vitest/Node isso quebraria qualquer módulo
      // server-only importado por um teste. O próprio pacote já
      // publica essa variante vazia pra esse cenário.
      "server-only": path.resolve(dirname, "node_modules/server-only/empty.js"),
    },
  },
});

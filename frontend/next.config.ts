import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Necessário pro Dockerfile multi-stage: gera um build autocontido em
  // .next/standalone (só o necessário pra rodar, sem precisar copiar
  // node_modules inteiro pra imagem final).
  output: "standalone",

  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          // Content-Security-Policy NÃO está aqui: precisa de um nonce
          // novo por requisição (script-src), e o `headers()` do
          // next.config.ts roda uma vez no build/boot, sem acesso a
          // request nenhuma — por isso a CSP é montada e anexada em
          // src/proxy.ts (guia oficial do Next.js, "Adding a nonce with
          // Proxy"), a única camada que vê cada requisição individual.
          //
          // DENY: nenhuma rota deste app precisa ser enquadrada, nem por
          // ele mesmo. A pré-visualização de documento anexo
          // (components/kanban/processo-page.tsx) chegou a embutir
          // /api/processos/{id}/documentos/{docId} num <iframe> — nesse
          // meio-tempo isto foi SAMEORIGIN, senão a pré-visualização
          // ficava em branco sem erro nenhum no console (achado real,
          // não só teórico). Depois a pré-visualização passou a abrir o
          // documento numa aba nova (target="_blank", mais rápido que
          // um visualizador embutido — pedido explícito do usuário), e
          // esse caso same-origin deixou de existir — DENY voltou a ser
          // o valor correto. Páginas (não-API) têm a proteção
          // equivalente via `frame-ancestors 'none'` na CSP montada em
          // proxy.ts.
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          // Nenhuma dessas APIs é usada pelo app — nega tudo por padrão.
          {
            key: "Permissions-Policy",
            value: "camera=(), microphone=(), geolocation=()",
          },
        ],
      },
    ];
  },
};

export default nextConfig;

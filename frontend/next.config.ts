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
          // SAMEORIGIN, não DENY: a pré-visualização de documento anexo
          // (components/kanban/processo-page.tsx) embute
          // /api/processos/{id}/documentos/{docId} num <iframe> DENTRO
          // do próprio app — como este header cobre TODA rota
          // (source: "/(.*)"), incluindo as de API, "DENY" bloqueava até
          // o app se auto-enquadrar, deixando a pré-visualização em
          // branco sem erro nenhum no console (achado real, não só
          // teórico). SAMEORIGIN continua barrando qualquer site externo
          // de enquadrar o app (a proteção contra clickjacking que este
          // header existe pra dar), só permite o caso same-origin que
          // agora é legítimo. Páginas (não-API) têm a proteção
          // equivalente e mais forte via `frame-ancestors 'none'` na CSP
          // montada em proxy.ts — só as rotas de API (fora do matcher de
          // proxy.ts) dependem deste header aqui.
          { key: "X-Frame-Options", value: "SAMEORIGIN" },
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

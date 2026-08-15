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
          // Redundante com frame-ancestors 'none' da CSP montada lá, mas
          // navegadores mais antigos só entendem este header — clickjacking.
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

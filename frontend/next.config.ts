import type { NextConfig } from "next";

const isDev = process.env.NODE_ENV === "development";

// CSP sem nonce (guia oficial do Next.js, seção "Without Nonces"): mais
// simples e compatível com renderização estática, ao custo de precisar de
// 'unsafe-inline' pra estilos/scripts injetados pelo framework e pelo
// Tailwind/shadcn. A alternativa (CSP com nonce via proxy.ts) exigiria
// forçar TODAS as páginas a renderização dinâmica (connection() em cada
// page.tsx) — mudança maior, não testável em navegador real nesta sessão.
// Documentado como limitação conhecida: endurecer para nonce-based é o
// próximo passo natural se isto for pra produção de verdade.
const cspHeader = `
    default-src 'self';
    script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ""};
    style-src 'self' 'unsafe-inline';
    img-src 'self' blob: data:;
    font-src 'self';
    object-src 'none';
    base-uri 'self';
    form-action 'self';
    frame-ancestors 'none';
    upgrade-insecure-requests;
`
  .replace(/\s{2,}/g, " ")
  .trim();

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
          { key: "Content-Security-Policy", value: cspHeader },
          // Redundante com frame-ancestors 'none' da CSP acima, mas
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

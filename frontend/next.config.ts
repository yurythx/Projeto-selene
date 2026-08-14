import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Necessário pro Dockerfile multi-stage: gera um build autocontido em
  // .next/standalone (só o necessário pra rodar, sem precisar copiar
  // node_modules inteiro pra imagem final).
  output: "standalone",
};

export default nextConfig;

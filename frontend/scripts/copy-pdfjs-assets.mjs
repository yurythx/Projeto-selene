// Copia os assets estáticos do pdfjs-dist (worker, cmaps, fontes padrão,
// decodificadores de imagem em WASM) pra public/pdfjs/ — servidos
// same-origin pelo Next.js, em vez do padrão do react-pdf de buscar
// esses arquivos num CDN externo (unpkg/cdnjs). Precisa ser same-origin
// porque a CSP deste app (ver src/proxy.ts) não abre exceção nenhuma
// pra host externo: script-src/worker-src efetivo é só 'self' +
// nonce/strict-dynamic, e font-src é só 'self' — um Worker ou uma fonte
// carregada de um CDN de terceiros seria bloqueada silenciosamente pelo
// navegador (mesma classe de bug já encontrada nesta sessão com
// X-Frame-Options: DENY bloqueando o preview de documento).
//
// Roda em "prebuild" (antes de `next build`) e "predev" (antes de `next
// dev`) — ver package.json. Precisa rodar de novo sempre que a versão do
// pdfjs-dist mudar (react-pdf fixa a versão exata como dependência), mas
// como é automático a cada build/dev, isso nunca fica esquecido.
import { cpSync, mkdirSync, existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const pdfjsDir = join(__dirname, "..", "node_modules", "pdfjs-dist");
const destDir = join(__dirname, "..", "public", "pdfjs");

if (!existsSync(pdfjsDir)) {
  console.error("[copy-pdfjs-assets] node_modules/pdfjs-dist não encontrado — rode npm install primeiro.");
  process.exit(1);
}

mkdirSync(destDir, { recursive: true });

const itens = [
  { from: join(pdfjsDir, "build", "pdf.worker.min.mjs"), to: join(destDir, "pdf.worker.min.mjs") },
  { from: join(pdfjsDir, "cmaps"), to: join(destDir, "cmaps") },
  { from: join(pdfjsDir, "standard_fonts"), to: join(destDir, "standard_fonts") },
  { from: join(pdfjsDir, "wasm"), to: join(destDir, "wasm") },
  { from: join(pdfjsDir, "iccs"), to: join(destDir, "iccs") },
];

for (const { from, to } of itens) {
  if (!existsSync(from)) {
    console.warn(`[copy-pdfjs-assets] aviso: ${from} não existe (pulando) — pode ser uma versão diferente do pdfjs-dist.`);
    continue;
  }
  cpSync(from, to, { recursive: true });
}

console.log(`[copy-pdfjs-assets] assets do pdfjs-dist copiados pra ${destDir}`);

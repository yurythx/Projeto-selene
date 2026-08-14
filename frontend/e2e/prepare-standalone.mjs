// Copia public/ e .next/static/ pra dentro de .next/standalone/ — os
// mesmos dois passos que o Dockerfile faz manualmente (COPY --from=builder
// .../public e .../.next/static). O build "standalone" não inclui esses
// diretórios por padrão; sem isso, `node .next/standalone/server.js` sobe
// mas serve CSS/JS/imagens 404, o que pode causar comportamento visual
// incorreto (ou pior, mismatches de hidratação sutis) — daí valer a pena
// testar E2E contra o artefato real, não contra `next start` (que nem
// suporta "output: standalone" — ver o aviso que ele mesmo imprime).
import { cpSync, existsSync } from "node:fs";

const pares = [
  ["public", ".next/standalone/public"],
  [".next/static", ".next/standalone/.next/static"],
];

for (const [origem, destino] of pares) {
  if (!existsSync(origem)) continue;
  cpSync(origem, destino, { recursive: true });
}

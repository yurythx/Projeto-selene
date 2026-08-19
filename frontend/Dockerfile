# syntax=docker/dockerfile:1

# --- Etapa de dependências ---
# Separada do build: só refaz npm ci se package*.json mudar, não a cada
# alteração de código (mesmo princípio do Dockerfile do backend).
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# --- Etapa de build ---
FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# API_URL só precisa existir de verdade em runtime (é lida só server-side,
# nunca inlined no bundle do browser — ver next.config.ts/lib/api/client.ts).
# Um valor qualquer aqui satisfaz o "throw" em módulo do client.ts durante
# a etapa de coleta de dados de página do `next build`.
ENV API_URL=http://backend:8080
ENV AUTH_SECRET=build-placeholder-substituido-em-runtime
RUN npm run build

# --- Etapa final ---
# ACHADO DE REVISÃO (auditoria de segurança geral, Trivy): a etapa final
# antes usava node:22-alpine de novo — mas o output "standalone" só
# precisa do binário `node`, nunca invoca `npm` em runtime
# (`CMD ["node", "server.js"]`). Como COPY só ADICIONA arquivos por cima
# da imagem base, o `npm` completo (com toda a árvore de node_modules
# DELE mesmo — brace-expansion, tar, picomatch, sigstore, ip-address)
# continuava presente no filesystem da imagem final, sem nunca ser
# executado: 8 CVEs reais (7 HIGH + 1 CRITICAL) confirmados com
# `trivy image` contra a imagem de verdade já deployada, não só a base
# crua. gcr.io/distroless/nodejs22-debian12 é a imagem que o próprio
# time do Next.js recomenda pra output "standalone" — só o runtime
# Node, sem npm, sem shell, sem gerenciador de pacotes nenhum. Variante
# ":nonroot" já roda como UID/GID 65532 por padrão — mesma convenção
# que o Dockerfile do backend já usa pro stage `scratch`, então os dois
# ficam consistentes entre si.
#
# TROCA HONESTA, não uma vitória sem ressalva: a base Debian desta
# imagem carrega um libssl3 alguns patches atrás do mais recente (CVEs
# HIGH/CRITICAL reais, `trivy image` confirma) — a alpine anterior
# estava com o OpenSSL dela em dia. Mesmo assim, a troca vale a pena:
# (1) as CVEs removidas (brace-expansion/tar/picomatch/sigstore/
# ip-address do npm) eram código morto garantido — nunca executado em
# runtime; as que sobraram são no libssl3 que o Node realmente usa pra
# TLS, mas a maioria não se aplica ao nosso uso real (ex:
# CVE-2026-31789 é só em sistemas 32-bit; nosso container é amd64;
# CVE-2026-14456 é um servidor QUIC, que não rodamos — só fazemos
# chamadas HTTPS de saída pro Keycloak/SMTP/Postgres). (2) sem shell
# nem gerenciador de pacotes nesta imagem, um RCE não vira um shell
# interativo pro atacante — ganho estrutural que nenhuma versão de
# patch cobre. (3) essa defasagem do libssl3 deve se resolver sozinha
# no próximo rebuild do distroless upstream (cadência regular, fora do
# nosso controle) — não é uma regressão permanente, é um lag transitório.
FROM gcr.io/distroless/nodejs22-debian12:nonroot AS runner
WORKDIR /app

ENV NODE_ENV=production

COPY --from=builder /app/public ./public
COPY --from=builder --chown=65532:65532 /app/.next/standalone ./
COPY --from=builder --chown=65532:65532 /app/.next/static ./.next/static

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

# Sem shell nesta imagem — ENTRYPOINT já é o binário `node`
# (/nodejs/bin/node, embutido na imagem distroless); CMD só passa o
# argumento (era `["node", "server.js"]`, o "node" duplicaria o
# ENTRYPOINT).
CMD ["server.js"]

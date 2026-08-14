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
# node:22-alpine (não scratch, ao contrário do backend): o output
# "standalone" do Next ainda depende do runtime Node.
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
RUN addgroup -g 1001 -S nodejs && adduser -S nextjs -u 1001

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

CMD ["node", "server.js"]

# syntax=docker/dockerfile:1

# --- Etapa de build ---
# Patch exato (não só a major/minor) pinado de propósito — ACHADO DE
# REVISÃO (auditoria geral): govulncheck apontou 7 CVEs reais de stdlib
# (net/url, net/http, crypto/tls, encoding/xml, encoding/asn1,
# html/template, golang.org/x/net/idna via net/http) alcançáveis a
# partir do fluxo de auth (AuthMiddlewareState.Reload) e de outros
# caminhos deste binário, todos corrigidos no release de segurança
# 1.25.13 (11/08/2026). A tag flutuante "golang:1.25-alpine" já
# aponta pra 1.25.13 hoje, mas fixar o patch exato evita build
# silenciosamente "voltando" a ficar vulnerável se o cache local do
# Docker segurar uma camada antiga da tag flutuante. go.mod tem o
# mesmo floor (`toolchain go1.25.13`) — os dois precisam ficar em
# sincronia quando o Go lançar o próximo patch de segurança.
FROM golang:1.25.13-alpine AS builder

WORKDIR /src

# Camadas de dependência separadas do código-fonte: só refaz o download
# dos módulos se go.mod/go.sum mudarem, não a cada alteração de código.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produz um binário estático — necessário para rodar na
# imagem final "scratch", que não tem libc nenhuma.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# /app/storage é preparado aqui (com shell disponível) e copiado com o
# dono já correto — a imagem "scratch" final não tem mkdir/chown/shell
# nenhum, então isso precisa existir ANTES de trocar de estágio.
RUN mkdir -p /app/storage && chown -R 65532:65532 /app

# --- Etapa final ---
# scratch: sem shell, sem pacotes, sem CVEs de SO — só o binário estático
# e o essencial para TLS/DNS funcionarem (necessário para o app chamar o
# JWKS do Keycloak via HTTPS).
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder --chown=65532:65532 /app /app
COPY --from=builder /out/api /api

WORKDIR /app

# Roda como usuário não-root (UID/GID arbitrários, não precisam existir em
# /etc/passwd numa imagem scratch — o kernel só olha o número).
USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/api"]

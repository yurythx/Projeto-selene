# syntax=docker/dockerfile:1

# --- Etapa de build ---
# Usa a mesma major/minor do go.mod para evitar builds "funciona na minha
# máquina" por divergência de toolchain.
FROM golang:1.25-alpine AS builder

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

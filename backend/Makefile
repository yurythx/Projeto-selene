.PHONY: help build run test test-unit test-integration fmt vet lint tidy \
	migrate-new docker-build clean

# Postgres descartável usado pelos alvos de teste/integração locais — ver
# README.md, seção "Testes".
TEST_DB_CONTAINER := selene-test-pg
TEST_DB_PORT := 55432
TEST_DATABASE_URL := host=localhost port=$(TEST_DB_PORT) user=selene password=selene dbname=projeto_selene sslmode=disable

help: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Compila o binário da API
	go build -o bin/api ./cmd/api

run: ## Roda a API localmente (requer .env configurado)
	go run ./cmd/api

test: test-unit ## Alias de test-unit (só os testes que não exigem Postgres)

test-unit: ## Roda só os testes unitários (sem banco, sempre rodam)
	go test ./... -short

test-integration: ## Sobe um Postgres descartável e roda a suíte completa (unit + integração)
	docker run -d --name $(TEST_DB_CONTAINER) \
		-e POSTGRES_USER=selene -e POSTGRES_PASSWORD=selene -e POSTGRES_DB=projeto_selene \
		-p $(TEST_DB_PORT):5432 postgres:16 >/dev/null
	@sleep 3
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./... -v -race -p 1; \
	status=$$?; \
	docker rm -f $(TEST_DB_CONTAINER) >/dev/null; \
	exit $$status

fmt: ## Verifica formatação (gofmt)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Arquivos não formatados:"; echo "$$unformatted"; exit 1; \
	fi

vet: ## Roda go vet
	go vet ./...

lint: ## Roda golangci-lint
	golangci-lint run ./...

tidy: ## Roda go mod tidy
	go mod tidy

migrate-new: ## Cria uma nova migration: make migrate-new nome=descricao_da_mudanca
	migrate create -ext sql -dir internal/database/migrations -seq $(nome)

docker-build: ## Builda a imagem Docker do backend (sem subir o stack — ver `make up` na raiz)
	docker build -t projeto-selene-backend:local .

clean: ## Remove artefatos de build
	rm -rf bin/

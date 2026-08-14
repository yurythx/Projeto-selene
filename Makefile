.PHONY: help up down logs test lint

help: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-14s %s\n", $$1, $$2}'

up: ## Sobe o stack inteiro (Postgres + backend) via docker-compose
	docker compose up --build

down: ## Derruba o stack (removendo volumes)
	docker compose down -v

logs: ## Acompanha os logs do backend
	docker compose logs -f backend

# Atalhos que delegam para o Makefile do backend (ver backend/Makefile
# para a lista completa de alvos específicos de Go: build, run,
# test-unit, test-integration, fmt, vet, migrate-new, etc.)

test: ## Roda a suíte de testes do backend (unit + integração)
	$(MAKE) -C backend test-integration

lint: ## Roda o linter do backend
	$(MAKE) -C backend lint

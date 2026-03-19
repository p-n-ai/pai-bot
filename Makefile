.PHONY: setup dev chat-terminal nudge-terminal test test-integration test-cover lint test-all migrate migration migrate-down migrate-status migrate-version migration-create seed seed-docker build docker start stop logs analytics analytics-xlsx analytics-example ollama-pull

GOOSE_DRIVER ?= postgres
GOOSE_DSN ?= postgres://pai:pai@postgres:5432/pai?sslmode=disable
GOOSE_RUN = docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0
GOOSE_CMD = $(GOOSE_RUN) -dir /app/migrations $(GOOSE_DRIVER) "$(GOOSE_DSN)"
GOLANGCI_LINT_VERSION ?= v2.4.0
GOLANGCI_LINT_RUN = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# First-time setup
setup:
	cp -n .env.example .env 2>/dev/null || true
	go mod download
	@echo "Setup complete. Edit .env with your configuration."

# Development
dev:
	go run ./cmd/server

chat-terminal:
	docker compose run --rm --entrypoint /pai-terminal-chat app

nudge-terminal:
	docker compose run --rm --entrypoint /pai-terminal-nudge app --user-id $(USER_ID)

# Testing
test:
	go test ./...

test-v:
	go test -v ./...

test-integration:
	go test -tags=integration ./...

lint:
	$(GOLANGCI_LINT_RUN) run ./...

test-all: lint test

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Database
migrate:
	@$(GOOSE_CMD) up -allow-missing

migration: migrate

migrate-down:
	@$(GOOSE_CMD) down

migrate-status:
	@$(GOOSE_CMD) status

migrate-version:
	@$(GOOSE_CMD) version

ifndef NAME
MIGRATION_CREATE_GUARD = $(error NAME is required, e.g. make migration-create NAME=add_parent_portal)
endif

migration-create:
	$(MIGRATION_CREATE_GUARD)
	@$(GOOSE_RUN) -dir /app/migrations create $(NAME) sql

seed:
	go run ./cmd/seed

seed-docker:
	docker compose exec app /pai-seed

# Build
build:
	CGO_ENABLED=0 go build -o bin/pai-server ./cmd/server

# Docker
docker:
	docker build -f deploy/docker/Dockerfile -t pai-bot .

start:
	docker compose up -d

stop:
	docker compose down

logs:
	docker compose logs -f app

# Ollama
ollama-pull:
	docker compose --profile ollama up -d ollama
	docker exec -it $$(docker compose ps -q ollama) ollama pull llama3

# Analytics
analytics:
	./scripts/analytics.sh

analytics-xlsx:
	./scripts/analytics.sh --xlsx output/spreadsheet/pai-analytics.xlsx

analytics-example:
	./scripts/analytics.sh --example-xlsx output/spreadsheet/pai-analytics-example.xlsx

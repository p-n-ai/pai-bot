.PHONY: setup dev chat-terminal nudge-terminal test test-integration test-cover lint test-all migrate migration migrate-down migrate-version migrate-force seed seed-docker build docker start stop logs analytics analytics-xlsx analytics-example ollama-pull

MIGRATE_DSN ?= postgres://pai:pai@postgres:5432/pai?sslmode=disable
MIGRATE_RUN = docker compose --profile tools run --rm migrate -path /migrations -database "$(MIGRATE_DSN)"

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
	golangci-lint run ./...

test-all: lint test

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Database
migrate:
	@$(MIGRATE_RUN) up

migration: migrate

migrate-down:
	@$(MIGRATE_RUN) down 1

migrate-version:
	@$(MIGRATE_RUN) version

ifndef VERSION
MIGRATE_FORCE_GUARD = $(error VERSION is required, e.g. make migrate-force VERSION=2)
endif

migrate-force:
	$(MIGRATE_FORCE_GUARD)
	@$(MIGRATE_RUN) force $(VERSION)

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

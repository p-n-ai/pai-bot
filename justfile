set shell := ["bash", "-euo", "pipefail", "-c"]

alias migration := migrate

default:
  @just --list

# First-time setup
setup:
  cp -n .env.example .env 2>/dev/null || true
  go mod download
  echo "Setup complete. Edit .env with your configuration."

# Development
dev:
  go run ./cmd/server

chat-terminal:
  docker compose run --rm --entrypoint /pai-terminal-chat app

nudge-terminal:
  docker compose run --rm --entrypoint /pai-terminal-nudge app --user-id "${USER_ID:-}"

# Testing
test:
  go test ./...

test-v:
  go test -v ./...

test-integration:
  go test -tags=integration ./...

lint:
  go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"${GOLANGCI_LINT_VERSION:-v2.4.0}" run ./...

test-all: lint test

test-cover:
  go test -coverprofile=coverage.out ./...
  go tool cover -html=coverage.out -o coverage.html

# Database
migrate:
  docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir /app/migrations "${GOOSE_DRIVER:-postgres}" "${GOOSE_DSN:-postgres://pai:pai@postgres:5432/pai?sslmode=disable}" up -allow-missing

migrate-down:
  docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir /app/migrations "${GOOSE_DRIVER:-postgres}" "${GOOSE_DSN:-postgres://pai:pai@postgres:5432/pai?sslmode=disable}" down

migrate-status:
  docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir /app/migrations "${GOOSE_DRIVER:-postgres}" "${GOOSE_DSN:-postgres://pai:pai@postgres:5432/pai?sslmode=disable}" status

migrate-version:
  docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir /app/migrations "${GOOSE_DRIVER:-postgres}" "${GOOSE_DSN:-postgres://pai:pai@postgres:5432/pai?sslmode=disable}" version

migration-create name:
  docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir /app/migrations create "{{name}}" sql

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
  docker exec -it "$(docker compose ps -q ollama)" ollama pull llama3

# Analytics
analytics:
  ./scripts/analytics.sh

analytics-xlsx:
  ./scripts/analytics.sh --xlsx output/spreadsheet/pai-analytics.xlsx

analytics-example:
  ./scripts/analytics.sh --example-xlsx output/spreadsheet/pai-analytics-example.xlsx

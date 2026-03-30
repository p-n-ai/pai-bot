set shell := ["bash", "-euo", "pipefail", "-c"]

alias migration := migrate
alias backend := go
alias dev := go

default:
  @just --list

# First-time setup
setup:
  cp -n .env.example .env 2>/dev/null || true
  go mod download
  echo "Setup complete. Edit .env with your configuration."

# Development
go:
  set -a; source .env; set +a; go run ./cmd/server

frontend-deps:
  cd admin && pnpm install

frontend:
  frontend_port="${FRONTEND_PORT:-3000}"; \
  agentation_port="${AGENTATION_PORT:-4747}"; \
  if ! lsof -nP -iTCP:"$agentation_port" -sTCP:LISTEN >/dev/null 2>&1; then \
    echo "starting Agentation MCP on http://127.0.0.1:$agentation_port"; \
    cd admin && nohup pnpm exec agentation-mcp server --port "$agentation_port" >/tmp/pai-agentation.log 2>&1 & \
    disown || true; \
  fi; \
  if lsof -nP -iTCP:"$frontend_port" -sTCP:LISTEN >/dev/null 2>&1; then \
    if curl -fsS -I --max-time 5 "http://127.0.0.1:$frontend_port" >/dev/null 2>&1; then \
      echo "frontend already running on http://127.0.0.1:$frontend_port"; \
      echo "agentation mcp on http://127.0.0.1:$agentation_port"; \
      exit 0; \
    fi; \
    echo "port $frontend_port is already in use"; \
    lsof -nP -iTCP:"$frontend_port" -sTCP:LISTEN; \
    exit 1; \
  fi; \
  cd admin && NEXT_PUBLIC_AGENTATION_ENDPOINT="http://127.0.0.1:$agentation_port" pnpm dev --hostname 127.0.0.1 --port "$frontend_port"

next:
  backend_port="${BACKEND_PORT:-8080}"; \
  if curl -fsS --max-time 3 "http://127.0.0.1:$backend_port/healthz" >/dev/null 2>&1; then \
    echo "backend already running on http://127.0.0.1:$backend_port"; \
  elif lsof -nP -iTCP:"$backend_port" -sTCP:LISTEN >/dev/null 2>&1; then \
    echo "port $backend_port is already in use"; \
    lsof -nP -iTCP:"$backend_port" -sTCP:LISTEN; \
    exit 1; \
  else \
    echo "starting Go server on http://127.0.0.1:$backend_port"; \
    nohup just go >/tmp/pai-go.log 2>&1 & \
    disown || true; \
    for _ in {1..20}; do \
      if curl -fsS --max-time 3 "http://127.0.0.1:$backend_port/healthz" >/dev/null 2>&1; then \
        break; \
      fi; \
      sleep 1; \
    done; \
    if ! curl -fsS --max-time 3 "http://127.0.0.1:$backend_port/healthz" >/dev/null 2>&1; then \
      echo "backend failed to start; check /tmp/pai-go.log"; \
      exit 1; \
    fi; \
  fi; \
  just frontend

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

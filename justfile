set dotenv-load
set shell := ["bash", "-cu"]

goose_driver := env_var_or_default("GOOSE_DRIVER", "postgres")
goose_dsn := env_var_or_default("GOOSE_DSN", "postgres://pai:pai@postgres:5432/pai?sslmode=disable")
goose_run := "docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0"
goose_cmd := goose_run + " -dir /app/migrations " + goose_driver + " \"" + goose_dsn + "\""

default:
  @just --list

setup:
  cp -n .env.example .env 2>/dev/null || true
  go mod download
  npm --prefix admin install
  @echo "Setup complete. Edit .env with your configuration."

dev-backend:
  go run ./cmd/server

dev-frontend:
  npm --prefix admin run dev

dev:
  #!/usr/bin/env bash
  set -euo pipefail

  cleanup() {
    trap - EXIT INT TERM
    kill "${backend_pid:-}" "${frontend_pid:-}" 2>/dev/null || true
    wait "${backend_pid:-}" "${frontend_pid:-}" 2>/dev/null || true
  }

  trap cleanup EXIT INT TERM

  go run ./cmd/server &
  backend_pid=$!

  npm --prefix admin run dev &
  frontend_pid=$!

  exit_code=0
  while true; do
    if ! kill -0 "$backend_pid" 2>/dev/null; then
      wait "$backend_pid" || exit_code=$?
      break
    fi
    if ! kill -0 "$frontend_pid" 2>/dev/null; then
      wait "$frontend_pid" || exit_code=$?
      break
    fi
    sleep 1
  done

  exit "$exit_code"

chat-terminal:
  docker compose run --rm --entrypoint /pai-terminal-chat app

nudge-terminal user_id:
  docker compose run --rm --entrypoint /pai-terminal-nudge app --user-id {{user_id}}

test-backend:
  go test ./...

test-frontend:
  npm --prefix admin test

test:
  just test-backend
  just test-frontend

test-v:
  go test -v ./...

test-integration:
  go test -tags=integration ./...

lint:
  golangci-lint run ./...

test-all:
  just lint
  just test

test-cover:
  go test -coverprofile=coverage.out ./...
  go tool cover -html=coverage.out -o coverage.html

migrate:
  @{{goose_cmd}} up -allow-missing

migration:
  just migrate

migrate-down:
  @{{goose_cmd}} down

migrate-version:
  @{{goose_cmd}} version

migrate-status:
  @{{goose_cmd}} status

migration-create name:
  @{{goose_run}} -dir /app/migrations create {{name}} sql

seed:
  go run ./cmd/seed

seed-docker:
  docker compose exec app /pai-seed

build:
  CGO_ENABLED=0 go build -o bin/pai-server ./cmd/server

docker:
  docker build -f deploy/docker/Dockerfile -t pai-bot .

start:
  docker compose up -d

stop:
  docker compose down

logs:
  docker compose logs -f app

ollama-pull:
  docker compose --profile ollama up -d ollama
  docker exec -it "$(docker compose ps -q ollama)" ollama pull llama3

analytics:
  ./scripts/analytics.sh

analytics-xlsx:
  ./scripts/analytics.sh --xlsx output/spreadsheet/pai-analytics.xlsx

analytics-example:
  ./scripts/analytics.sh --example-xlsx output/spreadsheet/pai-analytics-example.xlsx

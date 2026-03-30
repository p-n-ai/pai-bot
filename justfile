set shell := ["bash", "-euo", "pipefail", "-c"]

alias migration := migrate
alias backend := go
alias dev := go

default:
  @just --list

# First-time setup
setup:
  cp -n .env.example .env 2>/dev/null || true
  just install-deps
  echo "Setup complete. Edit .env with your configuration."

install-deps:
  go mod download
  if [ ! -d admin/node_modules ]; then cd admin && pnpm install --frozen-lockfile; fi

install-local-runtime:
  brew_bin="$(command -v brew || true)"; \
  if [ -z "$brew_bin" ]; then \
    echo "homebrew is required for turnkey local setup"; \
    exit 1; \
  fi; \
  if ! command -v pg_isready >/dev/null 2>&1 || ! command -v psql >/dev/null 2>&1; then \
    "$brew_bin" install libpq; \
  fi; \
  if ! command -v redis-cli >/dev/null 2>&1; then \
    "$brew_bin" install redis; \
  fi

check-local-db:
  set -a; [ -f .env ] && source .env; set +a; \
  brew_bin="$(command -v brew || true)"; \
  pg_isready_bin="$(command -v pg_isready || true)"; \
  psql_bin="$(command -v psql || true)"; \
  if [ -z "$pg_isready_bin" ] && [ -n "$brew_bin" ] && [ -x "$("$brew_bin" --prefix libpq 2>/dev/null)/bin/pg_isready" ]; then \
    pg_isready_bin="$("$brew_bin" --prefix libpq)/bin/pg_isready"; \
  fi; \
  if [ -z "$psql_bin" ] && [ -n "$brew_bin" ] && [ -x "$("$brew_bin" --prefix libpq 2>/dev/null)/bin/psql" ]; then \
    psql_bin="$("$brew_bin" --prefix libpq)/bin/psql"; \
  fi; \
  if [ -z "$pg_isready_bin" ] || [ -z "$psql_bin" ]; then \
    echo "postgres client tools missing"; \
    exit 1; \
  fi; \
  db_url="${LEARN_DATABASE_URL:-postgres://pai:pai@localhost:5432/pai?sslmode=disable}"; \
  if ! "$pg_isready_bin" -d "$db_url" >/dev/null 2>&1; then \
    echo "postgres is not reachable at $db_url"; \
    echo "start it first, then retry"; \
    exit 1; \
  fi; \
  seed_state="$("$psql_bin" "$db_url" -Atqc "SELECT CASE WHEN to_regclass('public.auth_identities') IS NULL THEN 'missing_auth_identities' WHEN EXISTS (SELECT 1 FROM auth_identities WHERE identifier_normalized IN ('teacher@example.com','platform-admin@example.com')) THEN 'seeded' ELSE 'not_seeded' END")"; \
  if [ "$seed_state" != "seeded" ]; then \
    echo "database is up but demo auth data is not ready ($seed_state)"; \
    echo "run 'just seed' before 'just go' or 'just next'"; \
    exit 1; \
  fi

ensure-local-runtime:
  if ! docker info >/dev/null 2>&1; then \
    if command -v open >/dev/null 2>&1; then \
      open -a OrbStack >/dev/null 2>&1 || true; \
    fi; \
    for _ in {1..30}; do \
      if docker info >/dev/null 2>&1; then break; fi; \
      sleep 2; \
    done; \
  fi; \
  docker info >/dev/null 2>&1 || { echo "docker is required for local postgres/dragonfly"; exit 1; }; \
  docker compose up -d postgres dragonfly; \
  for service in postgres dragonfly; do \
    for _ in {1..30}; do \
      container_id="$(docker compose ps -q "$service")"; \
      health_state="running"; \
      if docker inspect "$container_id" 2>/dev/null | grep -q '"Status":"healthy"'; then health_state="healthy"; fi; \
      if docker inspect "$container_id" 2>/dev/null | grep -q '"Status":"starting"'; then health_state="starting"; fi; \
      if docker inspect "$container_id" 2>/dev/null | grep -q '"Status":"unhealthy"'; then health_state="unhealthy"; fi; \
      if [ "$health_state" = "healthy" ] || [ "$health_state" = "running" ]; then break; fi; \
      sleep 2; \
    done; \
  done

prepare-local-dev:
  just install-deps
  just install-local-runtime
  just ensure-local-runtime
  if ! just check-local-db >/dev/null 2>&1; then \
    just seed; \
  fi; \
  just check-local-db

# Development
go:
  just prepare-local-dev
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
  just prepare-local-dev; \
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

admin-lint:
  cd admin && pnpm lint

admin-test:
  cd admin && pnpm test

test-all: lint test admin-lint admin-test

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
build-backend:
  CGO_ENABLED=0 go build -o bin/pai-server ./cmd/server

admin-build:
  cd admin && pnpm build

build: build-backend admin-build

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

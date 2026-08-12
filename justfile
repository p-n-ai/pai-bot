set shell := ["bash", "-euo", "pipefail", "-c"]

alias migration := migrate
alias backend := go
alias dev := go

dev-state-dir := join(env_var_or_default("TMPDIR", "/tmp"), "pai-bot-dev")
emulate_version := "0.4.1"

default:
  @just --list

# First-time setup
setup:
  cp -n .env.example .env 2>/dev/null || true
  just install-deps
  echo "Setup complete. Edit .env with your configuration."

install-deps:
  go mod download
  if [ ! -d admin-spa/node_modules ]; then cd admin-spa && pnpm install --frozen-lockfile; fi
  if [ ! -d site/node_modules ]; then cd site && pnpm install --frozen-lockfile; fi

install-local-runtime:
  needs_postgres_tools="no"; \
  needs_redis_cli="no"; \
  brew_bin="$(command -v brew || true)"; \
  if ! command -v pg_isready >/dev/null 2>&1 || ! command -v psql >/dev/null 2>&1; then \
    needs_postgres_tools="yes"; \
  fi; \
  if ! command -v redis-cli >/dev/null 2>&1; then \
    needs_redis_cli="yes"; \
  fi; \
  if [ "$needs_postgres_tools" = "no" ] && [ "$needs_redis_cli" = "no" ]; then \
    exit 0; \
  fi; \
  if [ -z "$brew_bin" ]; then \
    if [ "$needs_postgres_tools" = "yes" ]; then \
      echo "postgres client tools missing; install pg_isready + psql, or install Homebrew and rerun"; \
      exit 1; \
    fi; \
    echo "redis-cli not found; skipping optional install because Homebrew is unavailable"; \
    exit 0; \
  fi; \
  if [ "$needs_postgres_tools" = "yes" ]; then \
    "$brew_bin" install libpq; \
  fi; \
  if [ "$needs_redis_cli" = "yes" ]; then \
    "$brew_bin" install redis; \
  fi

default-db-url:
  @printf '%s\n' "postgres://pai:pai@localhost:5432/pai?sslmode=disable"

db-url:
  @db_url=""; \
  if [ -f .env ]; then \
    set -a; \
    source .env; \
    set +a; \
    db_url="${LEARN_DATABASE_URL:-}"; \
  fi; \
  if [ -z "$db_url" ]; then \
    echo "LEARN_DATABASE_URL must be set in .env" >&2; \
    exit 1; \
  fi; \
  printf '%s\n' "$db_url"

db-url-redacted:
  @db_url="$(just db-url)"; \
  printf '%s\n' "$db_url" | sed -E 's#(postgres(ql)?://)[^/@:]+(:[^@]*)?@#\1***:***@#'

db-target-allows-auto-seed:
  @db_url="$(just db-url)"; \
  default_db_url="$(just default-db-url)"; \
  if [ "$db_url" = "$default_db_url" ]; then \
    printf 'yes\n'; \
  else \
    printf 'no\n'; \
  fi

db-seed-state:
  @db_url="$(just db-url)"; \
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
    printf 'missing_tools\n'; \
    exit 0; \
  fi; \
  if ! "$pg_isready_bin" -d "$db_url" >/dev/null 2>&1; then \
    printf 'unreachable\n'; \
    exit 0; \
  fi; \
  "$psql_bin" "$db_url" -Atqc "SELECT CASE WHEN to_regclass('public.auth_identities') IS NULL OR to_regclass('public.auth_invites') IS NULL OR to_regclass('public.auth_sessions') IS NULL OR to_regclass('public.auth_oidc_flows') IS NULL THEN 'missing_auth_schema' WHEN EXISTS (SELECT 1 FROM auth_identities WHERE identifier_normalized IN ('teacher@example.com','platform-admin@example.com')) THEN 'seeded' ELSE 'not_seeded' END"

db-migration-preflight:
  @env_goose_dsn="${GOOSE_DSN:-}"; \
  env_learn_database_url="${LEARN_DATABASE_URL:-}"; \
  if [ -f .env ]; then \
    set -a; \
    source .env; \
    set +a; \
  fi; \
  if [ -n "$env_goose_dsn" ]; then \
    GOOSE_DSN="$env_goose_dsn"; \
  fi; \
  if [ -n "$env_learn_database_url" ]; then \
    LEARN_DATABASE_URL="$env_learn_database_url"; \
  fi; \
  effective_goose_dsn="${GOOSE_DSN:-postgres://pai:pai@postgres:5432/pai?sslmode=disable}"; \
  check_local_db_url() { \
    label="$1"; \
    url="$2"; \
    if [ -z "$url" ]; then \
      return 0; \
    fi; \
    host="${url#*://}"; \
    host="${host%%/*}"; \
    host="${host##*@}"; \
    host="${host%%:*}"; \
    host="${host#[}"; \
    host="${host%]}"; \
    case "$host" in \
      ""|localhost|127.*|::1|postgres) \
        return 0; \
        ;; \
    esac; \
    redacted="$(printf '%s\n' "$url" | sed -E 's#(postgres(ql)?://)[^/@:]+(:[^@]*)?@#\1***:***@#')"; \
    echo "refusing database migration: $label points to non-local database $redacted" >&2; \
    echo "run migrations only against local Docker postgres from this checkout" >&2; \
    exit 1; \
  }; \
  check_local_db_url "GOOSE_DSN" "$effective_goose_dsn"; \
  check_local_db_url "LEARN_DATABASE_URL" "${LEARN_DATABASE_URL:-}"

conversation-identity-preflight:
  just db-migration-preflight
  @db_url="${GOOSE_DSN:-postgres://pai:pai@postgres:5432/pai?sslmode=disable}"; \
  docker compose exec -T postgres psql "$db_url" -f - < scripts/preflight-conversation-identities.sql

check-local-db:
  @db_url="$(just db-url)"; \
  db_url_redacted="$(just db-url-redacted)"; \
  db_allows_auto_seed="$(just db-target-allows-auto-seed)"; \
  seed_state="$(just db-seed-state)"; \
  case "$seed_state" in \
    seeded) exit 0 ;; \
    missing_tools) \
      echo "postgres client tools missing"; \
      exit 1; \
      ;; \
    unreachable) \
      echo "postgres is not reachable at $db_url_redacted"; \
      echo "start it first, then retry"; \
      exit 1; \
      ;; \
    missing_auth_schema) \
      echo "database schema is not ready (missing auth tables)"; \
      echo "run 'just migrate' before 'just go' or 'just admin-spa'"; \
      exit 1; \
      ;; \
    not_seeded) \
      if [ "$db_allows_auto_seed" = "yes" ]; then \
        echo "database is up but demo auth data is not ready ($seed_state)"; \
        echo "run 'just seed' before 'just go' or 'just admin-spa'"; \
        exit 1; \
      fi; \
      echo "database is reachable and migrated; skipping demo seed requirement for target $db_url_redacted"; \
      exit 0; \
      ;; \
    *) \
      echo "unexpected local database state: $seed_state"; \
      exit 1; \
      ;; \
  esac

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
  db_url="$(just db-url)"; \
  db_url_redacted="$(just db-url-redacted)"; \
  db_allows_auto_seed="$(just db-target-allows-auto-seed)"; \
  seed_state="$(just db-seed-state)"; \
  if [ "$seed_state" = "missing_auth_schema" ]; then \
    if [ "$db_allows_auto_seed" = "yes" ]; then \
      just migrate; \
      seed_state="$(just db-seed-state)"; \
    else \
      echo "database schema is incomplete and auto-migrate is disabled for target $db_url_redacted"; \
    fi; \
  fi; \
  if [ "$seed_state" = "not_seeded" ]; then \
    if [ "$db_allows_auto_seed" = "yes" ]; then \
      just seed; \
    else \
      echo "database is not seeded and auto-seed is disabled for target $db_url_redacted"; \
    fi; \
  fi; \
  just check-local-db

once-dev:
  ./scripts/once-dev.sh

once-stop:
  ./scripts/once-dev.sh stop

once-remove:
  ./scripts/once-dev.sh remove

# Development
go:
  just prepare-local-dev
  set -a; source .env; set +a; go run ./cmd/server

admin-spa-deps:
  cd admin-spa && pnpm install

admin-spa:
  #!/usr/bin/env bash
  set -euo pipefail
  state_dir="{{dev-state-dir}}"
  mkdir -p "$state_dir"
  if [ -f .env ]; then
    set -a
    source .env
    set +a
  fi
  backend_port="${BACKEND_PORT:-${LEARN_SERVER_PORT:-8080}}"
  frontend_port="${FRONTEND_PORT:-5173}"
  started_backend="no"
  backend_pid=""
  cleanup() {
    code="$?"
    rm -f "$state_dir/backend.pid"
    if [ "$started_backend" = "yes" ] && [ -n "$backend_pid" ] && kill -0 "$backend_pid" >/dev/null 2>&1; then
      kill "$backend_pid" >/dev/null 2>&1 || true
      wait "$backend_pid" >/dev/null 2>&1 || true
    fi
    exit "$code"
  }
  trap cleanup INT TERM EXIT
  if curl -fsS --max-time 3 "http://127.0.0.1:$backend_port/healthz" >/dev/null 2>&1; then
    echo "backend already running on http://127.0.0.1:$backend_port"
  elif lsof -nP -iTCP:"$backend_port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "port $backend_port is already in use"
    lsof -nP -iTCP:"$backend_port" -sTCP:LISTEN
    exit 1
  else
    echo "starting Go server on http://127.0.0.1:$backend_port"
    just go >/tmp/pai-go.log 2>&1 &
    backend_pid="$!"
    printf '%s\n' "$backend_pid" >"$state_dir/backend.pid"
    started_backend="yes"
    for _ in {1..20}; do
      if curl -fsS --max-time 3 "http://127.0.0.1:$backend_port/healthz" >/dev/null 2>&1; then
        break
      fi
      sleep 1
    done
    if ! curl -fsS --max-time 3 "http://127.0.0.1:$backend_port/healthz" >/dev/null 2>&1; then
      echo "backend failed to start; continuing with admin-spa only"
      echo "check /tmp/pai-go.log for backend boot errors"
    fi
  fi
  cd admin-spa
  pnpm dev --host 127.0.0.1 --port "$frontend_port"

emulate-auth:
  npx -y emulate@{{emulate_version}} --service google,vercel --port 4000 --seed tools/emulate/emulate.config.yaml

emulate-google:
  npx -y emulate@{{emulate_version}} --service google --port 4002 --seed tools/emulate/emulate.config.yaml

emulate-vercel:
  npx -y emulate@{{emulate_version}} --service vercel --port 4000 --seed tools/emulate/emulate.config.yaml

chat-terminal:
  docker compose run --rm --entrypoint /pai-terminal-chat app

chat-codex:
  LEARN_DEV_MODE=true LEARN_AI_CODEX_ENABLED=true LEARN_AI_DEFAULT_PROVIDER=codex go run ./cmd/terminal-chat --memory --provider codex --interactive --progress --candidate .codex/chat-codex-candidate

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
  go run ./tools/antislop cmd internal

admin-spa-check:
  cd admin-spa && pnpm check

admin-spa-test:
  cd admin-spa && pnpm test

admin-spa-e2e:
  cd admin-spa && pnpm test:e2e

admin-spa-e2e-backend:
  cd admin-spa && pnpm test:e2e:backend

test-all: lint test admin-spa-check

test-cover:
  go test -coverprofile=coverage.out ./...
  go tool cover -html=coverage.out -o coverage.html

# Database
migrate:
  just db-migration-preflight
  just conversation-identity-preflight
  docker compose --profile tools run --rm goose go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir /app/migrations "${GOOSE_DRIVER:-postgres}" "${GOOSE_DSN:-postgres://pai:pai@postgres:5432/pai?sslmode=disable}" up -allow-missing

migrate-down:
  just db-migration-preflight
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

admin-spa-build:
  cd admin-spa && pnpm build

site-dev:
  cd site && pnpm dev

site-build:
  cd site && pnpm build

build: build-backend admin-spa-build

# Docker
docker:
  docker build -f deploy/docker/Dockerfile -t pai-bot .

start:
  docker compose up -d

stop:
  just stop-local
  docker compose down

stop-local:
  #!/usr/bin/env bash
  set -euo pipefail
  state_dir="{{dev-state-dir}}"
  repo_root="$(pwd)"
  stopped_any="no"
  if [ ! -d "$state_dir" ]; then
    echo "no local dev processes found"
    exit 0
  fi
  cwd_for_pid() {
    lsof -a -p "$1" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -n1
  }
  for name in frontend agentation backend google-emulator; do
    pid_file="$state_dir/$name.pid"
    if [ ! -f "$pid_file" ]; then
      continue
    fi
    pid="$(tr -d '[:space:]' <"$pid_file")"
    rm -f "$pid_file"
    if [ -z "$pid" ] || ! kill -0 "$pid" >/dev/null 2>&1; then
      continue
    fi
    proc_cwd="$(cwd_for_pid "$pid")"
    case "$proc_cwd" in
      "$repo_root"|"$repo_root"/*)
        echo "stopping $name ($pid)"
        kill "$pid" >/dev/null 2>&1 || true
        stopped_any="yes"
        ;;
    esac
  done
  if [ "$stopped_any" = "no" ]; then
    echo "no local dev processes found"
  fi

logs:
  docker compose logs -f app

# Ollama
ollama-pull:
  docker compose --profile ollama up -d ollama
  docker exec -it "$(docker compose ps -q ollama)" ollama pull qwen3

# Analytics
analytics:
  ./scripts/analytics.sh

analytics-xlsx:
  ./scripts/analytics.sh --xlsx output/spreadsheet/pai-analytics.xlsx

analytics-example:
  ./scripts/analytics.sh --example-xlsx output/spreadsheet/pai-analytics-example.xlsx

#!/usr/bin/env bash
# Copyright 2026 the P&AI authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
# First-time setup wizard for P&AI Bot.
# Run from the repository root: ./scripts/setup.sh

set -euo pipefail

# ── Helpers ────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}!${NC} $*"; }
fail()  { echo -e "${RED}✗${NC} $*"; exit 1; }

# ── Repo root check ───────────────────────────────────────────────────────

if [ ! -f "go.mod" ] || [ ! -f ".env.example" ]; then
    fail "Please run this script from the repository root (where go.mod lives)."
fi

echo ""
echo "  ╔══════════════════════════════╗"
echo "  ║      P&AI Bot Setup          ║"
echo "  ╚══════════════════════════════╝"
echo ""

# ── Prerequisites ──────────────────────────────────────────────────────────

echo "Checking prerequisites..."

check_cmd() {
    local cmd="$1"
    local install_hint="${2:-}"
    if ! command -v "$cmd" &> /dev/null; then
        if [ -n "$install_hint" ]; then
            fail "$cmd is required but not installed. $install_hint"
        else
            fail "$cmd is required but not installed."
        fi
    fi
    info "$cmd found"
}

check_cmd go "Install from https://go.dev/dl/"
check_cmd docker "Install from https://docs.docker.com/get-docker/"

# Docker Compose is a subcommand, not a separate binary.
if ! docker compose version &> /dev/null; then
    fail "Docker Compose v2 is required. Install from https://docs.docker.com/compose/install/"
fi
info "docker compose found"

# Node.js and pnpm are optional for backend-only setup.
if command -v node &> /dev/null; then
    info "node found (needed for admin panel)"
else
    warn "node not found — admin panel (just next) won't work, but the backend will."
fi
echo ""

# Verify Docker daemon is running.
if ! docker info &> /dev/null; then
    fail "Docker is installed but the daemon is not running. Please start Docker and re-run."
fi
info "Docker daemon is running"
echo ""

# ── Environment ────────────────────────────────────────────────────────────

if [ ! -f .env ]; then
    cp .env.example .env
    info "Created .env from .env.example"
    echo ""
    echo "  Please edit .env and configure at minimum:"
    echo ""
    echo "    1. LEARN_TELEGRAM_BOT_TOKEN    (get from @BotFather on Telegram)"
    echo "    2. At least one AI provider API key, for example:"
    echo "       LEARN_AI_OPENAI_API_KEY=sk-..."
    echo "       or set LEARN_AI_OLLAMA_ENABLED=true for free self-hosted AI"
    echo ""
    read -rp "  Press Enter after editing .env... "
    echo ""
else
    info ".env already exists, skipping copy"
fi

# Load .env so that Go commands can read config (e.g., LEARN_DATABASE_URL).
# Use grep+eval to handle values with special characters (e.g., P&AI in names).
while IFS= read -r line; do
    # Skip comments and blank lines.
    [[ -z "$line" || "$line" == \#* ]] && continue
    # Export each KEY=VALUE, quoting the value to handle &, spaces, etc.
    key="${line%%=*}"
    value="${line#*=}"
    export "$key=$value"
done < .env

echo ""

# ── Submodules (curriculum data) ───────────────────────────────────────────

echo "Initializing Git submodules (curriculum data)..."
git submodule update --init --recursive
info "Submodules initialized"
echo ""

# ── Go dependencies ───────────────────────────────────────────────────────

echo "Downloading Go dependencies..."
go mod download
info "Go dependencies downloaded"
echo ""

# ── Infrastructure ─────────────────────────────────────────────────────────

echo "Starting infrastructure (PostgreSQL, Dragonfly, NATS)..."
docker compose up -d postgres dragonfly nats

# Wait for Postgres to be ready.
echo "Waiting for PostgreSQL to accept connections..."
retries=0
max_retries=30
while ! docker compose exec -T postgres pg_isready -U pai -q 2>/dev/null; do
    retries=$((retries + 1))
    if [ "$retries" -ge "$max_retries" ]; then
        fail "PostgreSQL did not become ready after ${max_retries}s. Check: docker compose logs postgres"
    fi
    printf "."
    sleep 1
done
echo ""
info "PostgreSQL is ready"
echo ""

# ── Migrations ─────────────────────────────────────────────────────────────

echo "Running database migrations..."
if command -v just &> /dev/null; then
    just migrate
else
    warn "'just' not found — running goose via Docker"
    docker compose --profile tools run --rm goose \
        go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 \
        -dir /app/migrations postgres \
        "postgres://pai:pai@postgres:5432/pai?sslmode=disable" up -allow-missing
fi
info "Migrations applied"
echo ""

# ── Seed (optional) ───────────────────────────────────────────────────────

echo "Would you like to seed demo data? (recommended for first-time setup)"
read -rp "  Seed demo data? [Y/n] " seed_answer
seed_answer="${seed_answer:-Y}"
if [[ "$seed_answer" =~ ^[Yy] ]]; then
    echo "Seeding demo data..."
    go run ./cmd/seed
    info "Demo data seeded"
else
    info "Skipping seed"
fi
echo ""

# ── Build ──────────────────────────────────────────────────────────────────

echo "Building server binary..."
mkdir -p bin
go build -o bin/pai-server ./cmd/server
info "Binary built at bin/pai-server"
echo ""

# ── Done ───────────────────────────────────────────────────────────────────

echo "  ╔══════════════════════════════╗"
echo "  ║      Setup Complete!         ║"
echo "  ╚══════════════════════════════╝"
echo ""
echo "  Start the bot:"
echo "    ./bin/pai-server           Run the compiled binary"
echo "    just go                    Or use the task runner"
echo "    docker compose up -d       Or run everything in Docker"
echo ""
echo "  Start the admin panel:"
echo "    just next                  Backend + admin on :3000"
echo ""
echo "  Chat with your bot:"
echo "    Open Telegram and send /start to your bot"
echo ""

---
title: "Setup Guide"
summary: "Local setup guide for running P&AI Bot with Go, Docker Compose, pnpm, just, migrations, server, and admin panel."
read_when:
  - You are setting up pai-bot locally for the first time
  - You are debugging local server, infrastructure, migration, or admin-panel startup
  - You need the canonical local development commands
---

# Setup Guide

Get P&AI Bot running on your machine in under 10 minutes.

## Prerequisites

| Tool | Minimum Version | Check |
|------|----------------|-------|
| Go | 1.22+ | `go version` |
| Docker + Compose | 24+ / v2+ | `docker compose version` |
| Node.js | 20+ | `node --version` |
| pnpm | 9+ | `pnpm --version` |
| just | 1.0+ | `just --version` |

**Optional:**
- [Air](https://github.com/air-verse/air) for hot reload (`go install github.com/air-verse/air@latest`)
- golangci-lint is run via `go run` in `just lint`, so a system install is not required

## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/p-n-ai/pai-bot.git
cd pai-bot

# 2. First-time setup (copies .env.example → .env, downloads Go modules)
just setup

# 3. Edit .env — add your Telegram bot token and at least one AI provider key
#    LEARN_TELEGRAM_BOT_TOKEN=<your-token>    (get from @BotFather)
#    LEARN_AI_OPENAI_API_KEY=<key>             (or any other provider)

# 4. Start everything (infra + migrations + server)
just go

# 5. Verify health
curl http://localhost:8080/healthz   # → {"status":"ok"}
```

The bot is now running and listening for Telegram messages.

## What `just go` Does

1. Installs Go modules and frontend packages (if missing)
2. Starts infrastructure: PostgreSQL, Dragonfly (cache), NATS (messaging)
3. Runs database migrations via `goose`
4. Seeds the default tenant (in single-tenant mode)
5. Starts the Go server on `:8080`

## Admin Panel

To also start the Next.js admin panel:

```bash
just next    # Starts backend (if needed) + admin panel on :3000
```

Or for frontend-only development:

```bash
just frontend    # Starts only Next.js admin on :3000
```

## Environment Variables

All backend variables use the `LEARN_` prefix. Auth variables use `PAI_AUTH_`.

Copy `.env.example` for the full list. The key ones:

| Variable | Required | Description |
|----------|----------|-------------|
| `LEARN_TELEGRAM_BOT_TOKEN` | Yes | Telegram bot token from @BotFather |
| `LEARN_AI_DEFAULT_PROVIDER` | No | Preferred provider to try first (`openai`, `anthropic`, `deepseek`, `google`, `ollama`, `openrouter`) |
| `LEARN_AI_OPENAI_API_KEY` | No* | OpenAI API key |
| `LEARN_AI_OPENAI_MODEL` | No | Default OpenAI model when no request-specific model is set |
| `LEARN_AI_ANTHROPIC_API_KEY` | No* | Anthropic API key |
| `LEARN_AI_ANTHROPIC_MODEL` | No | Default Anthropic model when no request-specific model is set |
| `LEARN_AI_DEEPSEEK_API_KEY` | No* | DeepSeek API key |
| `LEARN_AI_DEEPSEEK_MODEL` | No | Default DeepSeek model when no request-specific model is set |
| `LEARN_AI_GOOGLE_API_KEY` | No* | Google Gemini API key |
| `LEARN_AI_GOOGLE_MODEL` | No | Default Google model when no request-specific model is set |
| `LEARN_AI_OPENROUTER_API_KEY` | No* | OpenRouter API key (100+ models) |
| `LEARN_AI_OPENROUTER_MODEL` | No | Default OpenRouter model when no request-specific model is set |
| `LEARN_AI_OLLAMA_ENABLED` | No* | Enable self-hosted Ollama |
| `LEARN_AI_OLLAMA_MODEL` | No | Default Ollama model when no request-specific model is set |
| `PAI_AUTH_SECRET` | No | JWT signing secret (default: `change-me-in-production`) |
| `LEARN_TENANT_MODE` | No | `single` (default) or `multi` |

*At least one AI provider must be configured.

## Docker AI Provider Selection

For Docker Compose deploys, the `app` service already loads `.env` via `env_file`, so school admins can choose provider and model with normal env vars.

Example:

```env
LEARN_AI_DEFAULT_PROVIDER=openrouter
LEARN_AI_OPENROUTER_API_KEY=sk-or-v1-...
LEARN_AI_OPENROUTER_MODEL=qwen/qwen3-max
```

Ollama example:

```env
LEARN_AI_DEFAULT_PROVIDER=ollama
LEARN_AI_OLLAMA_ENABLED=true
LEARN_AI_OLLAMA_MODEL=qwen3
```

The app container overrides `LEARN_AI_OLLAMA_URL` to `http://ollama:11434`, so Compose users do not need to point Ollama at `localhost`.

For Google Gemini, note that preview model IDs can have different or tighter rate limits than stable IDs. If you want a steadier production default, set `LEARN_AI_GOOGLE_MODEL` to a non-preview model name such as `gemini-2.5-flash`.

## Running Tests

```bash
just test              # Unit tests only
just test-integration  # Integration tests (requires Docker — uses testcontainers)
just lint              # golangci-lint
just test-all          # All tests + lint
just admin-e2e         # Admin Playwright smoke tests
```

For direct admin E2E runs:

```bash
cd admin
pnpm test:e2e
```

## Playwright E2E Setup (Admin)

If this is your first Playwright run on a machine:

```bash
cd admin
pnpm install
pnpm exec playwright install --with-deps chromium
pnpm test:e2e
```

Authenticated E2E tests are opt-in and only run when all of these are set:
- `E2E_BACKEND_ENABLED=true`
- `E2E_AUTH_ENABLED=true`
- `E2E_ADMIN_EMAIL`
- `E2E_ADMIN_PASSWORD`

Example:

```bash
cd admin
E2E_BACKEND_ENABLED=true E2E_AUTH_ENABLED=true E2E_ADMIN_EMAIL=platform-admin@example.com E2E_ADMIN_PASSWORD=demo-password pnpm test:e2e
```

PowerShell equivalent:

```powershell
$env:E2E_BACKEND_ENABLED="true"; $env:E2E_AUTH_ENABLED="true"; $env:E2E_ADMIN_EMAIL="platform-admin@example.com"; $env:E2E_ADMIN_PASSWORD="demo-password"; pnpm test:e2e
```

You can place these in `admin/.env`, `admin/.env.local`, or repo-root `.env`; Playwright now reads those files before test startup.
By default (`E2E_BACKEND_ENABLED` unset), Playwright skips tests tagged with `@backend`.
The same `E2E_*` keys are listed in `.env.example` for copy/paste onboarding.

Useful variants:

```bash
cd admin
pnpm test:e2e:headed   # run with visible browser
pnpm test:e2e:ui       # open Playwright UI mode
```

Common issues:

- `Cannot find module '@playwright/test'`:
  - Run `cd admin && pnpm install` so `@playwright/test` exists in local `node_modules`.
- PowerShell policy blocks `pnpm` scripts:
  - Use `pnpm.cmd ...` from PowerShell, or run commands from a shell where `pnpm` is enabled.
- Browser executable missing:
  - Run `cd admin && pnpm exec playwright install --with-deps chromium`.

## Common Issues

**Port 8080 already in use:**
Set `LEARN_SERVER_PORT=9090` in `.env` or stop the conflicting process.

**Docker services won't start:**
Check Docker is running: `docker info`. On Linux, ensure your user is in the `docker` group.

**Migrations fail:**
Ensure PostgreSQL is healthy: `docker compose ps`. Wait a few seconds after starting infra and retry with `just migrate`.

**No AI responses:**
Verify at least one `LEARN_AI_*_API_KEY` is set in `.env` and the key is valid.

## Self-Hosted Ollama (Free AI)

For a fully free setup using local AI models:

```bash
# Start with Ollama profile
docker compose --profile ollama up -d

# Pull a model
just ollama-pull    # Downloads qwen3

# Enable in .env
LEARN_AI_OLLAMA_ENABLED=true
LEARN_AI_OLLAMA_URL=http://ollama:11434
```

Note: Local models are used as the last fallback in the provider chain. Response quality is lower than paid providers.

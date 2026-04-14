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
| `LEARN_AI_OPENAI_API_KEY` | No* | OpenAI API key |
| `LEARN_AI_ANTHROPIC_API_KEY` | No* | Anthropic API key |
| `LEARN_AI_DEEPSEEK_API_KEY` | No* | DeepSeek API key |
| `LEARN_AI_GOOGLE_API_KEY` | No* | Google Gemini API key |
| `LEARN_AI_OPENROUTER_API_KEY` | No* | OpenRouter API key (100+ models) |
| `LEARN_AI_OLLAMA_ENABLED` | No* | Enable self-hosted Ollama |
| `PAI_AUTH_SECRET` | No | JWT signing secret (default: `change-me-in-production`) |
| `LEARN_TENANT_MODE` | No | `single` (default) or `multi` |

*At least one AI provider must be configured.

## Running Tests

```bash
just test              # Unit tests only
just test-integration  # Integration tests (requires Docker — uses testcontainers)
just lint              # golangci-lint
just test-all          # All tests + lint
```

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
just ollama-pull    # Downloads llama3

# Enable in .env
LEARN_AI_OLLAMA_ENABLED=true
LEARN_AI_OLLAMA_URL=http://ollama:11434
```

Note: Local models are used as the last fallback in the provider chain. Response quality is lower than paid providers.

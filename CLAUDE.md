# CLAUDE.md — P&AI Bot

This file provides context for Claude Code (and other AI coding assistants) working on this project.

## Project Overview

P&AI Bot is a **proactive AI learning agent** that teaches students through chat (Telegram, WhatsApp, Web). It doesn't wait for students to ask — it initiates study sessions, tracks mastery via spaced repetition, and motivates through battles, streaks, leaderboards, and goals.

Built by the [Pandai](https://pandai.app) team. Licensed under Apache 2.0.

**Core philosophy:** Content is commodity. Motivation is the moat.

## Status

This project is in the **planning/early development phase**. The repository currently contains documentation only (README, business plan, technical plan, development timeline). No application code has been written yet.

The first curriculum target is **KSSM Matematik (Form 1, 2, 3)** — specifically Algebra topics first.

## Tech Stack

### Backend (Go)
- **Language:** Go 1.22+ (stdlib `net/http` for routing — no framework)
- **Database:** PostgreSQL 17 via `pgx/v5`
- **Cache:** Dragonfly (Redis-compatible) via `go-redis/v9`
- **Message Queue:** NATS + JetStream via `nats-io/nats.go`
- **WebSocket:** `nhooyr.io/websocket`
- **Auth:** JWT via `golang-jwt/jwt/v5` (short-lived access + refresh tokens)
- **Logging:** `log/slog` (structured JSON)
- **Telemetry:** OpenTelemetry SDK
- **Migrations:** `golang-migrate/migrate/v4`
- **Linting:** `golangci-lint`
- **Testing:** stdlib `testing` + `testcontainers-go` for integration tests

### Frontend (Admin Panel)
- **Framework:** Next.js 14 (App Router) + TypeScript
- **Admin framework:** Refine v4+
- **UI:** shadcn/ui + Tailwind CSS 3
- **Charts:** Recharts or Tremor
- **State:** TanStack Query v5
- **Validation:** Zod

### Infrastructure
- Docker Compose (dev/single-server) and Helm (Kubernetes)
- Terraform for cloud IaC
- GitHub Actions CI → ArgoCD CD
- Grafana stack (Prometheus, Loki, Tempo) for observability

## Architecture

**Modular monolith** — single Go binary with clean domain boundaries. Can split into microservices later if needed.

Key domains:
- `internal/ai/` — AI Gateway: provider-agnostic interface, model routing, token budget tracking
- `internal/agent/` — Agent Engine: conversation state machine, proactive scheduler, pedagogical prompts, quiz engine, peer challenges
- `internal/chat/` — Chat Gateway: unified interface for Telegram, WhatsApp, WebSocket
- `internal/curriculum/` — Curriculum Service: loads YAML from OSS repository
- `internal/progress/` — Progress Tracker: mastery scoring, SM-2 spaced repetition, streaks/XP
- `internal/auth/` — Authentication: JWT, RBAC middleware
- `internal/tenant/` — Multi-tenancy: tenant isolation
- `internal/platform/` — Shared infra: config, database, cache, messaging, storage, telemetry, health
- `admin/` — Next.js admin panel (teacher dashboard, parent view, school admin)

## Project Structure

```
pai-bot/
├── cmd/server/main.go          # Application entrypoint
├── internal/
│   ├── ai/                     # AI Gateway (providers, routing, budget)
│   ├── agent/                  # Agent Engine (state machine, scheduler, prompts, quiz, challenges)
│   ├── chat/                   # Chat Gateway (telegram, whatsapp, websocket)
│   ├── curriculum/             # Curriculum loader (YAML from OSS)
│   ├── progress/               # Mastery scoring, SM-2, streaks/XP
│   ├── auth/                   # JWT + RBAC middleware
│   ├── tenant/                 # Multi-tenancy isolation
│   └── platform/               # Shared: config, database, cache, messaging, storage, telemetry, health
├── admin/                      # Next.js admin panel
├── migrations/                 # SQL migrations (golang-migrate)
├── deploy/
│   ├── docker/                 # Dockerfiles
│   └── helm/pai/               # Helm chart
├── terraform/                  # Infrastructure as Code
├── scripts/                    # setup.sh, deploy.sh, analytics.sh
├── docker-compose.yml          # Local dev
├── docker-compose.prod.yml     # Production single-server
├── Makefile                    # Dev shortcuts
└── .env.example                # All config documented
```

## Key Conventions

### Go Code
- All config via environment variables with `LEARN_` prefix
- Use Go stdlib `net/http` for routing (Go 1.22+ pattern-based routing)
- Table-driven tests
- Structured logging with `slog`
- No external web framework — stdlib only
- Domain code in `internal/` — nothing exported outside the module
- Each AI provider implements the `AIProvider` interface
- Each chat channel implements the `ChatChannel` interface

### Database
- All tables include `tenant_id` for multi-tenancy
- UUID primary keys (`gen_random_uuid()`)
- Timestamps as `TIMESTAMPTZ`
- JSONB for flexible/config fields
- Migration files: `NNN_description.up.sql` / `NNN_description.down.sql`
- Parameterized queries only — never interpolate user input

### Admin Panel (Next.js)
- App Router (not Pages Router)
- Refine for CRUD/data management
- shadcn/ui components (copied into codebase, not imported as dependency)
- Tailwind for styling
- Zod for validation schemas

### AI Gateway
- Provider-agnostic: all AI calls go through the gateway interface
- Task-based routing: teaching → best model, grading → cheapest, nudges → any
- Automatic fallback chain: paid providers → self-hosted Ollama
- Token budget enforcement per tenant/student
- Never cut off a student — degrade to free models when budget runs out

### Security
- JWT with 15-min access tokens + 7-day refresh tokens
- RBAC roles: student, teacher, parent, admin, platform_admin
- Row-level security via tenant_id
- No raw user input in AI system prompts — use structured templates
- Rate limiting per user (Dragonfly) and per tenant (ingress)
- Never store API keys in code or env files in production — use secrets management

## Common Commands

```bash
make setup          # First-time setup
make dev            # Start Go server with hot reload
make test           # Run Go unit tests
make test-integration  # Integration tests (testcontainers)
make lint           # golangci-lint
make test-all       # All tests + lint
make migrate        # Run database migrations
make build          # Build Go binary + admin static
make docker         # Build Docker image
make start          # docker compose up -d
make stop           # docker compose down
make logs           # Tail logs
make analytics      # Quick metrics
make ollama-pull    # Download free AI model
```

## Environment Variables

All prefixed with `LEARN_`. Key ones:

| Variable | Required | Description |
|----------|----------|-------------|
| `LEARN_TELEGRAM_BOT_TOKEN` | Yes | Telegram bot token |
| `LEARN_DATABASE_URL` | No | PostgreSQL connection string |
| `LEARN_CACHE_URL` | No | Dragonfly/Redis connection |
| `LEARN_NATS_URL` | No | NATS server URL |
| `LEARN_AI_OPENAI_API_KEY` | No* | OpenAI API key |
| `LEARN_AI_ANTHROPIC_API_KEY` | No* | Anthropic API key |
| `LEARN_AI_OLLAMA_ENABLED` | No* | Enable self-hosted Ollama |
| `LEARN_AUTH_JWT_SECRET` | No | JWT signing secret |
| `LEARN_TENANT_MODE` | No | `single` or `multi` |

*At least one AI provider must be configured.

## Related Repositories

- [p-n-ai/oss](https://github.com/p-n-ai/oss) — Open School Syllabus: structured curriculum YAML consumed as Git submodule
- [p-n-ai/oss-bot](https://github.com/p-n-ai/oss-bot) — GitHub bot + CLI for contributing to OSS

## Key Algorithms

- **SM-2 (SuperMemo 2):** Spaced repetition scheduling in `internal/progress/spaced_rep.go`
- **Mastery Scoring:** Weighted accuracy/consistency/recency, threshold 0.75 for mastery
- **Token Budget:** Real-time tracking in Dragonfly, periodic PostgreSQL sync
- **Model Routing:** Cost-aware with circuit breaker, automatic fallback chain

## Documentation

- [README.md](README.md) — Project overview, quick start, features, deployment
- [docs/technical-plan.md](docs/technical-plan.md) — Detailed architecture, schema, infra, security
- [docs/business-plan.md](docs/business-plan.md) — Business strategy, metrics, competitive landscape
- [docs/development-timeline.md](docs/development-timeline.md) — Day-by-day 6-week development plan

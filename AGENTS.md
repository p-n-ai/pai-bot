# AGENTS.md — P&AI Bot

This file is the **single source of truth** for any coding agent (Claude Code, Cursor, Copilot, etc.) working on this repository. `CLAUDE.md` symlinks here.

## Project Overview

P&AI Bot is a **proactive AI learning agent** that teaches students through chat (Telegram, WhatsApp, Web). It doesn't wait for students to ask — it initiates study sessions, tracks mastery via spaced repetition, and motivates through battles, streaks, leaderboards, and goals.

Built by the [Pandai](https://pandai.org) team. Licensed under Apache 2.0.

**Core philosophy:** Content is commodity. Motivation is the moat.

## Status

**Day 0 foundation is complete.** The repository has a working Go backend with health endpoints, configuration system, AI gateway (6 providers), database/cache clients, Docker infrastructure, CI pipeline, and full test suite.

Development follows a 30-day timeline. See [docs/development-timeline.md](docs/development-timeline.md) for current progress.

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

Frontend working note:
- When doing frontend/UI work, use `pnpm` for package management and scripts.
- Do not spam `pnpm build`; run it only when build verification is needed for the task or handoff gate.
- For visual tuning loops on motion/color-heavy UI (for example Aurora backgrounds), temporary DialKit usage is OK for local adjustment, but do not commit the DialKit library/dependency changes.
- When building frontend/UI work, install and enable Agentation in dev and prefer running Agentation MCP alongside it for visual critique/annotation workflows.
- Keep the toolbar dev-only. If Agentation is missing in the admin app, add `<Agentation />` in the root layout and wire an explicit local MCP endpoint so annotations auto-sync in development.
- If the user asks whether you can "see the annotation", interpret that as MCP/automation visibility by default, not browser-visible markers. Only use browser inspection for annotations when the user explicitly asks for browser/UI verification.

### Infrastructure
- Docker Compose (dev/single-server) and Helm (Kubernetes)
- Terraform for cloud IaC
- GitHub Actions CI → ArgoCD CD
- Grafana stack (Prometheus, Loki, Tempo) for observability

## Architecture

**Modular monolith** — single Go binary with clean domain boundaries. Can split into microservices later if needed.

Key domains:
- `internal/ai/` — AI Gateway: provider-agnostic interface, model routing, token budget tracking
- `internal/agent/` — Agent Engine: conversation state machine, proactive scheduler, pedagogical prompts (dual-loop problem solving, adaptive explanation depth, curriculum citations), quiz engine (static + dynamic question generation with exam-style mimicry), peer challenges
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
├── Makefile                    # Legacy task runner parity
├── justfile                    # Preferred task runner
└── .env.example                # All config documented
```

## Source of Truth

Use these files as primary references:

1. `README.md` for positioning, scope, and contribution framing
2. `docs/technical-plan.md` for architecture and implementation plan
3. `docs/business-plan.md` for product strategy and metrics intent
4. `docs/development-timeline.md` for phased execution plan and task assignments
5. `docs/implementation-guide.md` for code templates, test specs, and exit criteria
6. `docs/admin-panel.md` for admin panel features, roles, and API specification
7. `docs/admin-panel-uiux.md` for admin panel UI/UX wireframes and design system

If you change one doc and it affects others, update all impacted docs in the same task.

Before changing curriculum contracts, shared API shapes, bot workflows, or OSS sync behavior, also inspect the relevant `p-n-ai` sibling/core repos listed in [Related Repositories](#related-repositories). Reuse existing patterns and data contracts when they already exist there instead of inventing local variants.

## Agent Rules of Engagement

### 0) MANDATORY: Read both implementation documents before any daily task

**Before starting ANY daily implementation task, you MUST read and cross-reference BOTH:**

1. **[docs/implementation-guide.md](docs/implementation-guide.md)** — Code templates, function signatures, test specifications, file-by-file implementation details, and exit criteria for each day
2. **[docs/development-timeline.md](docs/development-timeline.md)** — Task assignments, dependencies between tasks, engineer allocation, and day-by-day execution order

**Why both?** The implementation guide tells you **what** to build and **how** (exact code patterns, test cases, API contracts). The development timeline tells you **when** and **in what order** (task dependencies, parallelization, which tasks block others). Using only one will lead to missed dependencies or divergent implementations.

**The workflow for each day:**
1. Read the day's section in `docs/development-timeline.md` — identify task IDs, dependencies, and assignments
2. Read the day's section in `docs/implementation-guide.md` — identify code templates, test specs, and exit criteria
3. Follow TDD (Rule 5 below)
4. Verify ALL exit criteria from the implementation guide before marking any day complete

### 1) Be explicit about present vs planned state

When writing or editing docs:
- Clearly distinguish **Current** (exists in repo) vs **Planned** (intended to be built)
- Do not describe planned files/modules as already implemented

### 2) Keep architecture consistent across docs

Core architecture is a modular monolith with these domains: `internal/ai`, `internal/agent`, `internal/chat`, `internal/curriculum`, `internal/progress`, `internal/auth`, `internal/tenant`, `internal/platform`, `admin/`

If one doc changes these boundaries, propagate the same model everywhere.

### 3) Preserve key technical conventions

- Backend: Go 1.22+, stdlib `net/http`
- DB: PostgreSQL, Cache: Dragonfly, Messaging: NATS + JetStream
- Auth: JWT (access + refresh), Logging: `log/slog`, Telemetry: OpenTelemetry
- Migrations: `golang-migrate`
- Admin: Next.js + TypeScript + Refine + shadcn/ui + Tailwind

### 4) Maintain security and tenancy assumptions

- Multi-tenant design (`tenant_id` isolation)
- Role-based access (`student`, `teacher`, `parent`, `admin`, `platform_admin`)
- Parameterized SQL only
- No hardcoded secrets
- Budget-aware AI routing with graceful fallback

### 5) Test-first development (TDD) — mandatory

**MANDATORY: After finishing ANY implementation, always run `just test-all` to verify nothing is broken. Never skip this step. Never consider a task done until the full test suite passes.**

**MANDATORY: After completing ANY task from the development timeline, update `docs/development-timeline.md` using the task table format `| Task ID | Task | Owner | Status | Remark |`. Set `Status` to `✅` when completed (otherwise `⬜`), and put any notes in the `Remark` column. Never consider a task done until the timeline is updated.**

**Every implementation task must follow this cycle. No exceptions.**

1. **Write tests first** — Define expected behavior as unit tests before writing implementation
2. **Run tests, confirm RED** — Verify tests fail for the right reason (missing implementation)
3. **Write the minimum implementation** to make tests pass
4. **Run tests for the new feature** — Confirm the new tests pass
5. **Run `just test-all`** — Run the FULL test suite to ensure no earlier code is broken
6. **Fix any regressions** before moving on — if anything broke, fix it now
7. **Refactor** if needed, re-run `just test-all` to confirm nothing breaks
8. **Commit only when `just test-all` is fully green**

**Go backend testing rules:**
- Every `.go` file gets a corresponding `_test.go` in the same package
- Use table-driven tests for all tests with multiple cases
- Use `testcontainers-go` for integration tests needing real PostgreSQL/Dragonfly/NATS
- All external dependencies (AI providers, chat channels, database) behind interfaces for mocking
- `just test` = unit tests, `just test-integration` = integration tests, `just test-all` = everything + lint

**Admin panel (Next.js) testing rules:**
- Jest + React Testing Library for component tests
- Test data provider integrations and auth flows

**When adding a new feature:**
```
1. Write _test.go with test cases       → defines the contract
2. Run `just test` → confirm RED         → tests fail (not yet implemented)
3. Write implementation .go              → make tests pass
4. Run `just test` → confirm GREEN       → new tests pass
5. Run `just test-all` → full suite      → ALL tests pass, no regressions
6. If anything broke → fix it now, re-run `just test-all`
7. Commit only when `just test-all` is fully green
```

**When fixing a bug:**
```
1. Write a test that reproduces the bug  → proves the bug exists
2. Run `just test` → confirm RED         → test fails, bug confirmed
3. Fix the bug
4. Run `just test` → confirm GREEN       → bug is fixed
5. Run `just test-all` → full suite      → ALL tests pass, no regressions
6. Commit only when `just test-all` is fully green
```

### 6) Prefer incremental bootstrap

If asked to start implementation, scaffold in this order unless user specifies otherwise:

1. `go.mod` + `cmd/server/main.go`
2. `internal/platform/{config,database,cache,messaging,health}`
3. `internal/ai` provider interface + router
4. `internal/chat` Telegram adapter skeleton
5. `internal/agent` engine skeleton
6. `migrations/` + `docker-compose.yml` + `justfile`
7. `admin/` Next.js scaffold

Keep commits/doc changes small and verifiable.

## Key Conventions

### Go Code
- All config via environment variables with `LEARN_` prefix
- Use Go stdlib `net/http` for routing (Go 1.22+ pattern-based routing)
- Table-driven tests with `_test.go` files alongside every implementation file
- Structured logging with `slog`
- No external web framework — stdlib only
- Domain code in `internal/` — nothing exported outside the module
- Each AI provider implements the `Provider` interface (in `internal/ai/gateway.go`)
- Each chat channel implements the `Channel` interface (in `internal/chat/`)
- All external dependencies behind interfaces for testability

### Database
- All tables include `tenant_id` for multi-tenancy
- UUID primary keys (`gen_random_uuid()`)
- Timestamps as `TIMESTAMPTZ`
- JSONB for flexible/config fields
- Migration files: `NNN_description.up.sql` / `NNN_description.down.sql`
- Parameterized queries only — never interpolate user input

### Admin Panel (Next.js)
- App Router (not Pages Router)
- Use `pnpm` for install, dev, build, test, and lint commands
- Avoid redundant `pnpm build` runs during implementation; reserve builds for verification checkpoints
- Refine for CRUD/data management
- shadcn/ui components (copied into codebase, not imported as dependency)
- Tailwind for styling
- Zod for validation schemas

### AI Gateway
- Provider-agnostic: all AI calls go through the gateway interface
- **6 providers:** OpenAI (+ DeepSeek via configurable base URL), Anthropic, Google Gemini, Ollama, OpenRouter
- DeepSeek uses OpenAI-compatible API — same `provider_openai.go` with different base URL and API key
- Google Gemini has its own provider file (`provider_google.go`) — different API format
- Qwen, Kimi, and other models accessible via OpenRouter or self-hosted via Ollama
- Task-based routing: teaching → best model (Claude Sonnet, GPT-4o, Gemini Pro), grading/question generation → cheapest (DeepSeek V3, GPT-4o-mini, Gemini Flash), nudges → any
- Automatic fallback chain: paid providers → self-hosted Ollama
- Token budget enforcement per tenant/student
- Never cut off a student — degrade to free models when budget runs out

### Pedagogical Prompt Patterns
Inspired by [DeepTutor](https://github.com/HKUDS/DeepTutor)'s multi-agent reasoning architecture, adapted for chat-based K-12 tutoring as prompt-level patterns (no extra infrastructure).
- **Dual-loop problem solving:** Every math question follows Understand → Plan → Solve → Verify → Connect. Implemented as system prompt instructions in `internal/agent/prompts.go`
- **Curriculum citations:** Every explanation must reference the curriculum source path (e.g., "KSSM Form 1 > Algebra > Linear Equations"). The prompt builder injects `{syllabus} > {subject} > {topic}` into the system prompt
- **Adaptive explanation depth:** System prompt adjusts based on mastery level — beginner (<0.3): simple language, more examples; developing (0.3–0.6): standard with gradual notation; proficient (>0.6): concise, edge cases, cross-topic connections
- **Dynamic question generation:** When assessments.yaml has <5 questions for a topic, quiz engine generates additional questions from teaching notes via `CompleteJSON` (cheap model)
- **Exam-style mimicry:** AI-generated questions use 2–3 real UASA/SPM exemplar questions as style references to match real exam format and difficulty

### Security
- JWT with 15-min access tokens + 7-day refresh tokens
- RBAC roles: student, teacher, parent, admin, platform_admin
- Row-level security via tenant_id
- No raw user input in AI system prompts — use structured templates
- Rate limiting per user (Dragonfly) and per tenant (ingress)
- Never store API keys in code or env files in production — use secrets management

## Common Commands

Prefer `just <recipe>` over `make <target>`. Keep `Makefile` only for parity/compatibility.
If the user explicitly says "use just", prefer repo `just` recipes over raw Docker commands for local lifecycle work.

```bash
just setup             # First-time setup
just dev               # Start Go server with .env loaded
just backend           # Same as just dev
just frontend          # Start Next.js admin on :3000 and boot Agentation MCP
just test              # Run Go unit tests
just test-integration  # Integration tests (testcontainers)
just lint              # golangci-lint
just test-all          # All tests + lint
just migrate           # Run database migrations
just build             # Build Go binary + admin static
just docker            # Build Docker image
just start             # docker compose up -d
just stop              # docker compose down
just logs              # Tail logs
just analytics         # Quick metrics
just ollama-pull       # Download free AI model
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
| `LEARN_AI_DEEPSEEK_API_KEY` | No* | DeepSeek API key (OpenAI-compatible) |
| `LEARN_AI_GOOGLE_API_KEY` | No* | Google Gemini API key |
| `LEARN_AI_OPENROUTER_API_KEY` | No* | OpenRouter API key (access to 100+ models) |
| `LEARN_AI_OLLAMA_ENABLED` | No* | Enable self-hosted Ollama |
| `LEARN_AUTH_JWT_SECRET` | No | JWT signing secret |
| `LEARN_TENANT_MODE` | No | `single` or `multi` |

*At least one AI provider must be configured.

## Key Algorithms

- **SM-2 (SuperMemo 2):** Spaced repetition scheduling in `internal/progress/spaced_rep.go`
- **Mastery Scoring:** Weighted accuracy/consistency/recency, threshold 0.75 for mastery
- **Token Budget:** Real-time tracking in Dragonfly, periodic PostgreSQL sync
- **Model Routing:** Cost-aware with circuit breaker, automatic fallback chain
- **Dual-Loop Problem Solving:** Structured prompt pattern (Understand → Plan → Solve → Verify → Connect) in `internal/agent/prompts.go`
- **Adaptive Explanation Depth:** Mastery-based prompt adjustment (beginner/developing/proficient) in `internal/agent/prompts.go`
- **Dynamic Question Generation:** AI generates quiz questions from teaching notes with exam-style mimicry in `internal/agent/quiz.go`

## Documentation Quality Checklist

Before finishing any documentation change, verify:
- The file/dir map matches real repo contents
- Planned items are labeled as planned
- No contradictory claims between README and docs
- Dates/phase references are coherent
- Commands shown are either available now or explicitly marked as planned

## Non-Goals (for now)

- Only claim endpoints/features are operational if they exist in the codebase and tests pass
- Do not claim future-day features are present
- Verify claims against actual code, not just documentation

## Related Repositories

- [p-n-ai/oss](https://github.com/p-n-ai/oss) — Open School Syllabus: structured curriculum YAML consumed as Git submodule
- [p-n-ai/oss-bot](https://github.com/p-n-ai/oss-bot) — GitHub bot + CLI for contributing to OSS

Agent note:
- Treat sibling `p-n-ai` repos as context sources when work touches curriculum content, content ingestion, shared contracts, contribution flows, or sync tooling.
- If this repo depends on behavior or data shape from another `p-n-ai` repo, verify that repo before changing code/docs here and keep naming/contracts aligned.

## Documentation

- [README.md](README.md) — Project overview, quick start, features, deployment
- [docs/technical-plan.md](docs/technical-plan.md) — Detailed architecture, schema, infra, security
- [docs/business-plan.md](docs/business-plan.md) — Business strategy, metrics, competitive landscape
- [docs/development-timeline.md](docs/development-timeline.md) — Day-by-day 6-week development plan
- [docs/implementation-guide.md](docs/implementation-guide.md) — Detailed code templates, test specs, and exit criteria for each day
- [docs/admin-panel.md](docs/admin-panel.md) — Admin panel features, roles, routes, and API specification
- [docs/admin-panel-uiux.md](docs/admin-panel-uiux.md) — Admin panel UI/UX wireframes, design system, and interaction patterns

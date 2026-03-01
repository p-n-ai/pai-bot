# AGENTS.md — P&AI Bot

This file is the working guide for **any coding agent** operating in this repository.

## Project Status (Current Reality)

**Day 0 foundation is complete** (as of March 2026). The repository has working application code:

- `cmd/server/main.go` — HTTP server with `/healthz` and `/readyz` endpoints
- `internal/platform/config/` — Configuration from `LEARN_` env vars
- `internal/platform/database/` — PostgreSQL client (pgxpool)
- `internal/platform/cache/` — Dragonfly/Redis client (go-redis)
- `internal/ai/` — AI Gateway with 6 providers (OpenAI, Anthropic, Google, Ollama, OpenRouter + DeepSeek via OpenAI), router with fallback chain, budget tracker
- `migrations/001_initial.{up,down}.sql` — Database schema (tenants, users, conversations, messages, learning_progress, events)
- `deploy/docker/Dockerfile` — Multi-stage Docker build
- `docker-compose.yml` — PostgreSQL, Dragonfly, NATS, Ollama, app
- `.github/workflows/ci.yml` — CI pipeline
- `.env.example`, `Makefile` — Developer tooling

All unit tests pass (`make test-all`). See `docs/development-timeline.md` for current progress.

## Mission and Product Context

P&AI Bot is a proactive AI learning companion focused on motivation-first learning:

- Proactive study nudges (not only reactive Q&A)
- Mastery tracking + spaced repetition
- Gamified engagement (streaks, XP, challenges, leaderboards)
- Chat-first UX (Telegram first, then WhatsApp/Web)

Initial curriculum target:

- Malaysia KSSM Matematik (Form 1, Form 2, Form 3), Algebra-first

## Source of Truth

Use these files as primary references:

1. `README.md` for positioning, scope, and contribution framing
2. `docs/technical-plan.md` for architecture and implementation plan
3. `docs/business-plan.md` for product strategy and metrics intent
4. `docs/development-timeline.md` for phased execution plan and task assignments
5. `docs/implementation-guide.md` for code templates, test specs, and exit criteria

If you change one doc and it affects others, update all impacted docs in the same task.

## Agent Rules of Engagement

### 0) MANDATORY: Read both implementation documents before any daily task

**Before starting ANY daily implementation task, you MUST read and cross-reference BOTH:**

1. **`docs/implementation-guide.md`** — Contains exact code templates, function signatures, test specifications, file-by-file details, and exit criteria for each day
2. **`docs/development-timeline.md`** — Contains task IDs, dependencies between tasks, engineer allocation, and execution order

**Do NOT implement from one document alone.** The implementation guide defines **what** and **how**; the timeline defines **when** and **in what order**. Skipping either will lead to missed dependencies or divergent implementations.

**For each day's work:**
1. Read the day's section in `docs/development-timeline.md` — identify task IDs, dependencies, and assignments
2. Read the day's section in `docs/implementation-guide.md` — identify code templates, test specs, and exit criteria
3. Follow TDD (Rule 5 below)
4. Verify ALL exit criteria from the implementation guide before marking any day complete

### 1) Be explicit about present vs planned state

When writing or editing docs:

- Clearly distinguish:
  - **Current:** what exists in this repo now
  - **Planned:** what is intended to be built
- Do not describe planned files/modules as already implemented.

### 2) Keep architecture consistent across docs

Core planned architecture is a modular monolith with these planned domains:

- `internal/ai`
- `internal/agent`
- `internal/chat`
- `internal/curriculum`
- `internal/progress`
- `internal/auth`
- `internal/tenant`
- `internal/platform`
- `admin/` (Next.js panel)

If one doc changes these boundaries, propagate the same model everywhere.

### 3) Preserve key technical conventions (planned implementation)

- Backend: Go 1.22+, stdlib `net/http`
- DB: PostgreSQL
- Cache: Dragonfly (Redis-compatible)
- Messaging: NATS + JetStream
- Auth: JWT (access + refresh)
- Logging: `log/slog`
- Telemetry: OpenTelemetry
- Migrations: `golang-migrate`
- Admin: Next.js + TypeScript + Refine + shadcn/ui + Tailwind

### 4) Maintain security and tenancy assumptions

Keep these constraints intact in documentation and future code scaffolding:

- Multi-tenant design (`tenant_id` isolation)
- Role-based access (`student`, `teacher`, `parent`, `admin`, `platform_admin`)
- Parameterized SQL only
- No hardcoded secrets
- Budget-aware AI routing with graceful fallback

### 5) Test-first development (TDD) — mandatory

**Every implementation task must follow this cycle. No exceptions.**

1. **Write tests first** — Define expected behavior as unit tests before writing implementation
2. **Run tests, confirm RED** — Verify tests fail for the right reason (missing implementation)
3. **Write the minimum implementation** to make tests pass
4. **Run tests for the new feature** — Confirm the new tests pass
5. **Run `make test-all`** — Run the FULL test suite to ensure no earlier code is broken
6. **Fix any regressions** before moving on — if anything broke, fix it now
7. **Refactor** if needed, re-run `make test-all` to confirm nothing breaks
8. **Commit only when `make test-all` is fully green**

**Go backend testing rules:**
- Every `.go` file gets a corresponding `_test.go` in the same package
- Use table-driven tests for all tests with multiple cases
- Use `testcontainers-go` for integration tests needing real PostgreSQL/Dragonfly/NATS
- All external dependencies (AI providers, chat channels, database) behind interfaces for mocking
- `make test` = unit tests, `make test-integration` = integration tests, `make test-all` = everything + lint

**Admin panel (Next.js) testing rules:**
- Jest + React Testing Library for component tests
- Test data provider integrations and auth flows

**Bug fix workflow:**
1. Write a test that reproduces the bug
2. Confirm test fails (bug exists)
3. Fix the bug
4. Run `make test-all` — full suite green, no regressions
5. Commit

### 6) Prefer incremental bootstrap when code work is requested

If asked to start implementation, scaffold in this order unless user specifies otherwise:

1. `go.mod` + `cmd/server/main.go`
2. `internal/platform/{config,database,cache,messaging,health}`
3. `internal/ai` provider interface + router
4. `internal/chat` Telegram adapter skeleton
5. `internal/agent` engine skeleton
6. `migrations/` + `docker-compose.yml` + `Makefile`
7. `admin/` Next.js scaffold

Keep commits/doc changes small and verifiable.

## Documentation Quality Checklist

Before finishing any documentation change, verify:

- The file/dir map matches real repo contents
- Planned items are labeled as planned
- No contradictory claims between README and docs
- Dates/phase references are coherent
- Commands shown are either available now or explicitly marked as planned

## Non-Goals (for now)

Avoid pretending unbuilt features are validated:

- Only claim endpoints/features are operational if they exist in the codebase and tests pass
- Do not claim future-day features are present (e.g., Day 3 features during Day 1 work)
- Verify claims against actual code, not just documentation

## Related Repositories

- `p-n-ai/oss` — curriculum content source (planned integration)
- `p-n-ai/oss-bot` — curriculum contribution tooling and automation

# AGENTS.md — P&AI Bot

This file is the working guide for **any coding agent** operating in this repository.

## Project Status (Current Reality)

P&AI Bot is currently in a **planning/docs phase**.

As of February 2026, this repo contains documentation only:

- `README.md`
- `docs/technical-plan.md`
- `docs/business-plan.md`
- `docs/development-timeline.md`
- `AGENTS.md`
- `CLAUDE.md`
- `LICENSE`

There is **no application source code yet** (`cmd/`, `internal/`, `admin/`, `migrations/`, `deploy/`, `terraform/`, `.env.example`, etc. are planned but not present).

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
4. `docs/development-timeline.md` for phased execution plan

If you change one doc and it affects others, update all impacted docs in the same task.

## Agent Rules of Engagement

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

### 5) Prefer incremental bootstrap when code work is requested

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

Until code exists, avoid pretending runtime behavior is validated:

- No claims that APIs/endpoints are operational
- No claims that Docker/Helm/Terraform artifacts are present
- No claims that tests/lint pipelines are passing

## Related Repositories

- `p-n-ai/oss` — curriculum content source (planned integration)
- `p-n-ai/oss-bot` — curriculum contribution tooling and automation

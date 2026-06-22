# P&AI BOT

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

K-12 proactive learning agent for Telegram, WhatsApp, WebSocket/embed chat, and admin dashboards. Go modular monolith with Vite/Next/Astro frontends; current primary scope is backend Go and `admin-spa/`.

## STRUCTURE

```
pai-bot/
├── cmd/          # entrypoint binaries (AGENTS.md)
├── internal/     # backend domains (AGENTS.md)
├── admin-spa/    # Vite + TanStack Router admin SPA (AGENTS.md)
├── admin/        # Next admin app
├── site/         # Astro site
├── migrations/   # SQL migrations
├── deploy/       # Docker image/runtime assets
├── terraform/    # infrastructure
├── scripts/      # dev/ops helpers
├── tools/        # repo tooling
└── oss/          # curriculum/content mirror
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Server startup/routes | `internal/server`, `cmd/server/main.go` entrypoint wrapper, `internal/adminapi`, `internal/apidocs` |
| Tutor turn behavior | `internal/agent/engine.go`, `internal/agent/turn.go`, `internal/agent/prompt_builder.go` |
| Bot/chat channels | `internal/chat`, `internal/agent/dev_commands.go` |
| AI providers/routing | `internal/ai`, `internal/platform/airouter`, `internal/platform/config` |
| Auth/session/RBAC | `internal/auth`, `internal/adminapi`, migrations touching auth tables |
| Progress/mastery | `internal/progress`, `internal/agent/quiz_progress.go` |
| Curriculum/retrieval | `internal/curriculum`, `internal/retrieval`, `oss/` |
| Admin SPA API/client | `admin-spa/src/lib/admin-api.ts`, `admin-spa/src/lib/*-types.ts` |
| Admin SPA routes | `admin-spa/src/routes`, `admin-spa/src/routeTree.gen.ts` |
| Admin SPA UI | `admin-spa/src/components`, `admin-spa/src/components/ui` |
| Local seed/demo data | `cmd/seed`, `internal/platform/seed` |

## CONVENTIONS

- `cmd/*` parses flags/env and wires dependencies; reusable behavior belongs in `internal/*`.
- `cmd/server` stays a thin entrypoint; `internal/server` owns HTTP lifecycle, handlers, middleware, chat HTTP mounts, and server adapters.
- Backend I/O paths take `context.Context` first: DB, cache, AI, HTTP-ish orchestration.
- Postgres code uses `*_postgres.go`; integration tests are explicit with `_integration_test.go` naming/build tags where used.
- Tenant data paths preserve `tenant_id`; platform-admin/global access is explicit, not fallback.
- AI decisions route through `internal/ai`; tutor/chat packages stay provider-neutral.
- Admin SPA mirrors backend JSON shapes in `src/lib/*-types.ts` and keeps route files thin.

## ANTI-PATTERNS

- No imports from `cmd/` into `internal/`.
- No provider-specific branches in `internal/agent` or `internal/chat`.
- No tenant-blind SQL on product tables.
- No broad env/secret dumps in logs, tests, or CLI debug output.
- No free-form quiz JSON parsing; use structured helpers in `internal/ai`.
- No hand-editing generated TanStack router artifacts without matching route changes.

## COMMANDS

```bash
just setup
just prepare-local-dev
just go
just admin-spa
just frontend
just test
just test-integration
just test-all
just build-backend
just migration-create <name>
just migrate
just seed
just db-url-redacted
just db-seed-state
cd admin-spa && pnpm test
cd admin-spa && pnpm typecheck
cd admin-spa && pnpm build
```

## KEY CONFIGS

| Tool | Entry | Notes |
|------|-------|-------|
| Go module | `go.mod` | Go 1.25 module `github.com/p-n-ai/pai-bot` |
| Recipes | `justfile` | Preferred local workflow surface |
| Env template | `.env.example` | `LEARN_*`; auth root secret `PAI_AUTH_SECRET` |
| Local runtime | `docker-compose.yml` | Postgres + Dragonfly |
| Production runtime | `docker-compose.prod.yml`, `deploy/docker/Dockerfile` | Container packaging |
| Admin SPA | `admin-spa/package.json`, `admin-spa/vite.config.ts` | Vite, TanStack Router, React, pnpm |
| Migrations | `migrations/` | Create with `just migration-create <name>` |

## NOTES

- Local runtime should stay local: never run migrations with `GOOSE_DSN`/`LEARN_DATABASE_URL` aimed at remote DBs.
- `just frontend` starts the Next admin plus Agentation MCP; keep Agentation wiring dev-only.
- `docs/**` may describe plans; verify current behavior against code before relying on docs.

## Agent skills

### Issue tracker

Issues are private local markdown files under `.issues/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Triage uses the default five `Status:` values. See `docs/agents/triage-labels.md`.

### Domain docs

Domain docs use a single-context layout. See `docs/agents/domain.md`.

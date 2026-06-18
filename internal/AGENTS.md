# BACKEND INTERNAL PACKAGES

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Runtime product code for the Go modular monolith. Commands wire these packages; packages own behavior.

## STRUCTURE

```
internal/
├── agent/          # tutor engine, quizzes, nudges, challenges (AGENTS.md)
├── ai/             # provider gateway, router, budget, structured output (AGENTS.md)
├── chat/           # Telegram/WhatsApp/WebSocket/embed adapters (AGENTS.md)
├── auth/           # JWT, cookies, Google OIDC, guest/password auth (AGENTS.md)
├── adminapi/       # admin service helpers (AGENTS.md)
├── curriculum/     # OSS YAML loader/prerequisites (AGENTS.md)
├── progress/       # mastery, XP, streaks, SM-2 (AGENTS.md)
├── retrieval/      # curriculum search/index facade (AGENTS.md)
├── tenant/         # tenant bootstrap
├── platform/       # config/db/cache/AI router/mailer/seed (AGENTS.md)
└── terminal*/      # local CLI runtime helpers
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Wire one learner turn | `agent/engine.go`, `agent/turn.go`, `chat/gateway.go` |
| Add slash command | `chat/commands.go`, then `agent/*command*.go` |
| Add AI provider/model | `ai/provider_*.go`, `ai/router.go`, `platform/config`, `platform/airouter` |
| Add admin API behavior | `adminapi/service.go`, specific `adminapi/*.go`, then `cmd/server` route |
| Add persistence | nearest `*_postgres.go` plus integration test |
| Add local/dev runtime behavior | `terminalchat/`, `terminalnudge/`, or `cmd/*` wrapper |
| Add curriculum source behavior | `curriculum/`, `retrieval/`, `oss/` contract checks |

## CONVENTIONS

- External systems behind interfaces; unit tests use fakes unless DB/provider integration is the point.
- Postgres implementations suffix `*_postgres.go`; in-memory stores are named explicitly.
- Context first for I/O, AI calls, DB, cache, and runtime orchestration.
- HTTP route registration belongs at server/adminapi/apidocs surfaces, not deep domain files.
- Product rows stay tenant-scoped even in single-tenant mode.

## ANTI-PATTERNS

- No imports from `cmd/` into `internal/`.
- No raw user input in system prompts; use prompt builders/context packets.
- No tenant-blind SQL.
- No provider-specific branching in tutor/chat code; route through `internal/ai`.
- No command-only business logic.

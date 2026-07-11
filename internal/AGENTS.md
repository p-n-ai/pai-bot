# BACKEND INTERNAL PACKAGES

**Generated:** 2026-07-11
**Commit:** bdd0c16

Runtime product code for the Go modular monolith. Commands wire these packages; packages own behavior.

## STRUCTURE

```
internal/
├── agent/          # tutor engine, quizzes, nudges, challenges (AGENTS.md)
├── ai/             # provider gateway, router, budget, structured output (AGENTS.md)
├── llm/            # provider protocol, registry, streaming adapters (AGENTS.md)
├── chat/           # Telegram/WhatsApp/WebSocket/embed adapters (AGENTS.md)
├── auth/           # JWT, cookies, Google OIDC, guest/password auth (AGENTS.md)
├── adminapi/       # admin service helpers (AGENTS.md)
├── curriculum/     # OSS YAML loader/prerequisites (AGENTS.md)
├── progress/       # mastery, XP, streaks, SM-2 (AGENTS.md)
├── retrieval/      # curriculum search/index facade (AGENTS.md)
├── tenant/         # tenant bootstrap
├── platform/       # config/db/cache/AI router/mailer/seed (AGENTS.md)
├── server/         # HTTP lifecycle, mux, security, admin/chat mounts (AGENTS.md)
└── terminal*/      # local CLI runtime helpers
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Wire one learner turn | `agent/engine.go`, `agent/turn.go`, `chat/gateway.go` |
| Add slash command | `chat/commands.go`, then `agent/*command*.go` |
| Add product AI provider/model | `ai/provider_*.go`, `ai/router.go`, `platform/config`, `platform/airouter` |
| Change low-level OpenRouter/LLM protocol | `llm/` |
| Add admin API behavior | `adminapi/service.go`, specific `adminapi/*.go`, then `server/handler.go` |
| Change HTTP lifecycle/routes | `server/run.go`, `server/handler.go`, `server/security.go` |
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

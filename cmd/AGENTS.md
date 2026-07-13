# COMMAND ENTRYPOINTS

**Generated:** 2026-07-11
**Commit:** bdd0c16

Thin binaries for server, seed, local chat, nudges, quality harnesses, and analytics export.

## STRUCTURE

```
cmd/
├── server/                 # production HTTP/chat server (AGENTS.md)
├── seed/                   # demo/token-budget seed modes
├── terminal-chat/          # local tutor CLI or WS client
├── terminal-nudge/         # one-shot due-review nudge check
├── conversation-harness/   # YAML AI behavior harness
└── analyticsxlsx/          # workbook export CLI
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Server dependency wiring | `server/main.go` |
| HTTP routes/admin embedding | `internal/server/handler.go` |
| Security headers/origins | `internal/server/security.go` |
| Demo/auth/budget seed flags | `seed/main.go` |
| Local tutor session | `terminal-chat/main.go`, `internal/terminalchat` |
| Nudge debug CLI | `terminal-nudge/main.go`, `internal/terminalnudge` |
| AI quality scripts | `conversation-harness/main.go` |
| Analytics workbook CLI | `analyticsxlsx/main.go`, `internal/analyticsxlsx` |

## CONVENTIONS

- Parse flags/env here; delegate behavior to `internal/*`.
- Return testable errors below `main`; keep `os.Exit` at command edge.
- Command tests cover wiring, flags, output mode, and failure exits.
- Server can be orchestration-heavy; domain behavior still lives in `internal/*`.

## ANTI-PATTERNS

- No reusable business logic in `cmd/`.
- No command package importing another command package.
- No provider-specific AI setup outside platform/router helpers.
- No destructive seed mode without explicit flag and tests.

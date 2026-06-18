# TERMINAL CHAT COMMAND

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Local tutor CLI and WebSocket client for manual QA, multi-user simulation, and conversation trace dumps.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| CLI flags and mode selection | `main.go`, `main_test.go` |
| Runner/state behavior | `internal/terminalchat` |

## CONVENTIONS

- `--ws` switches to pure WebSocket client; local state path is separate.
- `--progress` opt-in enables mastery/streak/XP side effects.
- Trace dump flags should preserve latest-N limit semantics.

## ANTI-PATTERNS

- No production-only assumptions in local CLI state.
- No noisy logs unless `--verbose` requests diagnostics.

## NOTES

- Useful for quick tutor prompt regression before full harness runs.
- Multi-user prefixes exercise peer challenge flows.
- Dump files can contain sensitive prompts/responses; keep out of git.
- Keep one-shot WS mode stable for scripts.
- Mirror server AI router setup unless deliberately testing a local override.

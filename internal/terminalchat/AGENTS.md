# TERMINAL CHAT RUNTIME

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Local CLI chat state and runner used by `cmd/terminal-chat` and conversation harnesses.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Runner loop | `runner.go`, `runner_multi.go` |
| State construction | `state.go`, `state_test.go` |
| Multi-user simulation | `runner_multi_test.go` |

## CONVENTIONS

- Default terminal sessions avoid progress side effects unless flags enable them.
- Keep CLI-only presentation here; tutor decisions remain in `internal/agent`.
- State builder should mirror server dependencies closely enough for local parity.

## ANTI-PATTERNS

- No terminal-only behavior changes to production agent contracts.
- No printing secrets/config dumps in verbose mode.

## NOTES

- Multi-user mode is useful for challenge/group flows.
- WebSocket client mode bypasses local agent state entirely.
- Conversation dumps may include model-facing prompts; treat as sensitive locally.
- Keep cleanup callbacks idempotent for tests and Ctrl-C exits.

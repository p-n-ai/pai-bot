# TERMINAL NUDGE RUNTIME

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Local due-review nudge runner and capture channel used by `cmd/terminal-nudge`.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Capture output channel | `channel.go` |
| Nudge runner | `runner.go`, `runner_test.go` |

## CONVENTIONS

- Match production scheduler behavior; only output transport differs.
- Tests should assert capture payloads, not terminal formatting noise.

## ANTI-PATTERNS

- No independent nudge selection algorithm here.
- No cache/database setup in this package; command wires dependencies.

## NOTES

- Capture channel makes assertions easier than scraping stdout.
- Keep output payload close to real channel sends.
- Runner should be reusable from command tests without sleeping.
- Nudge eligibility belongs in `internal/agent` scheduler/tracker paths.
- User ID filtering is owned by command/runtime input.

# TERMINAL NUDGE COMMAND

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Local command to check due-review nudges for one user.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| CLI/dependency wiring | `main.go` |
| Nudge runtime | `internal/terminalnudge`, `internal/terminalchat` |

## CONVENTIONS

- `--user-id` is required.
- Cache mirrors server behavior: warn and continue when unavailable.
- AI provider presence is required for generated nudges.

## ANTI-PATTERNS

- No background scheduler here; one-shot check only.
- No separate terminal-only nudge copy.

## NOTES

- Designed for targeted due-review debugging.
- Cache warnings should not prevent local diagnosis.
- User-facing copy comes through agent/nudge runtime.
- Keep dependencies close to server setup for parity.
- Add tests around required flags before adding more modes.

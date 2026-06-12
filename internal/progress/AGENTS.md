# LEARNER PROGRESS

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Mastery tracking, SM-2 review scheduling, streaks, XP, and progress display helpers.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Mastery tracker | `tracker.go`, `tracker_postgres.go`, `tracker_test.go` |
| Spaced repetition | `spaced_rep.go`, `spaced_rep_test.go` |
| Streaks/XP | `streaks.go`, `xp.go`, related tests |
| User-facing copy/display | `display.go`, `display_test.go` |

## CONVENTIONS

- Algorithms stay deterministic with injectable/current time in tests.
- DB tracker preserves tenant/user/topic boundaries.
- Display helpers are presentation-only; no persistence side effects.
- Agent package decides when an activity affects progress; this package decides how.

## ANTI-PATTERNS

- No progress mutation from read-only dashboard/report paths.
- No hardcoded timezone assumptions without tests.
- No XP/streak copy drift between Telegram and admin displays.

## NOTES

- Integration coverage belongs around Postgres tracker changes.

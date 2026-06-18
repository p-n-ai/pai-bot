# SEED COMMAND

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Database seeding CLI for demo data and token budget windows.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| CLI flags/modes | `main.go`, `main_test.go` |
| Seed routines | `internal/platform/seed` |

## CONVENTIONS

- Modes are explicit strings; invalid mode exits non-zero.
- Budget windows parse RFC3339 or default to UTC month boundaries.
- Command owns config/db connection; seed package owns SQL behavior.

## ANTI-PATTERNS

- No seed SQL in `cmd/seed`.
- No destructive reseed behavior without an explicit flag and tests.

## NOTES

- Demo seed is for local/dev bootstrap, not migrations.
- Token-budget seed helps AI budget testing without manual SQL.
- Keep logs structured enough for `just` output and CI diagnosis.
- Prefer idempotent inserts/upserts in seed routines.
- Validate tenant slug before writing budget rows.

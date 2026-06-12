# ANALYTICS XLSX EXPORT

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Workbook generation library used by analytics export CLI paths.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Workbook generation | `workbook.go`, `workbook_test.go` |

## CONVENTIONS

- Keep workbook logic pure; CLI parsing lives in `cmd/analyticsxlsx`.
- Tests should inspect workbook contents, not just file existence.

## ANTI-PATTERNS

- No direct database reads here.
- No local filesystem writes from library code.

## NOTES

- `cmd/analyticsxlsx` is intentionally tiny; add behavior here first.
- Preserve stable sheet names/columns once exports reach users.
- Keep input structs decoupled from DB rows so API/CLI callers can share them.
- Prefer deterministic workbook timestamps in tests.
- If adding styles, assert the data path separately from visual formatting.

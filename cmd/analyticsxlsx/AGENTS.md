# ANALYTICS XLSX COMMAND

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

CLI wrapper around the analytics workbook library.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| CLI wrapper | `main.go` |
| Workbook behavior | `internal/analyticsxlsx` |

## CONVENTIONS

- Write workbook output through provided writer/stdout path in library contract.
- Keep argument parsing minimal; library handles domain validation.

## ANTI-PATTERNS

- No duplicating workbook construction in command code.

## NOTES

- Keep this command boring: args in, workbook bytes out.
- Prefer adding tests in `internal/analyticsxlsx` for export content.
- Only add CLI flags when automation needs them.
- Preserve stdout/stderr split for shell pipelines.
- Exit non-zero on validation or write failures.
- Do not read secrets or app config here unless export source changes.

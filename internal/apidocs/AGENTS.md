# API DOCS PACKAGE

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

OpenAPI/Scalar document generation and docs routes for the Go server.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| OpenAPI document | `document.go`, `schema.go`, `document_test.go` |
| HTTP routes | `routes.go` |

## CONVENTIONS

- Keep schemas aligned with actual `cmd/server` handlers and admin clients.
- Tests should catch route/schema drift.

## ANTI-PATTERNS

- No documenting planned endpoints as live.
- No private/admin-only fields in public schemas unless intentionally exposed.

## NOTES

- Treat this as generated-by-code documentation, not hand-written markdown.
- Route mounting still happens from `cmd/server`.
- Keep examples small and sanitized.
- If auth behavior changes, update security scheme docs here and runtime docs.
- Schema drift is cheaper to catch in unit tests than during frontend integration.

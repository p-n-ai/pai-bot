# RUNTIME SETTINGS

**Generated:** 2026-07-11
**Commit:** bdd0c16

Encrypted persisted runtime settings, effective-value resolution, and live apply coordination.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Effective settings model | `settings.go` |
| Encryption helpers | `crypto.go` |
| Encrypted persistence and in-memory state | `postgres.go` |
| Admin update wiring | `internal/server/handler.go` |
| Integration behavior | `*_integration_test.go` |

## CONVENTIONS

- Environment/config values and persisted values have one explicit precedence path.
- Persisted API keys are encrypted using the auth secret and never serialized back to clients.
- Preserve an undecryptable stored blob unless an authorized update replaces it.
- Updates serialize persistence, current-state replacement, then the non-failing live apply callback.
- Test transaction, degraded-read, corruption, and secret-redaction behavior at real seams.

## ANTI-PATTERNS

- No plaintext secrets in JSON, logs, fixtures, or error context.
- No current-state or apply mutation after a persistence failure.
- No fallback to insecure default auth secrets.
- No config/persistence precedence duplicated in HTTP handlers.

## NOTES

- Runtime settings are platform-global; tenant-scoped fallback is not implicit.

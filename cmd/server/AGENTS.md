# SERVER COMMAND

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Production Go HTTP server: health bootstrap, API routes, admin embedding, chat channels, auth, retrieval, and graceful shutdown.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Startup dependency graph | `main.go` |
| Route registration | `main.go` handler setup sections |
| Admin SPA embedding | `embed_admin.go`, `embed_admin_test.go` |
| Security headers/origins | `security.go`, `security_test.go` |
| Route/wiring regressions | `main_test.go` |
| API shape docs | `internal/apidocs` |

## CONVENTIONS

- Health-only handler comes up before full mux; keep long init after early healthz.
- DB/config failures are fatal; cache failures degrade where existing code does.
- Route handlers parse/auth/encode; service packages own product decisions.
- Admin asset path behavior needs embed tests.

## ANTI-PATTERNS

- No long startup task before healthz availability.
- No direct provider-specific AI setup here; use `internal/platform/airouter`.
- No duplicating admin API logic from `internal/adminapi` in route closures.
- No security header/origin change without route or browser smoke coverage.

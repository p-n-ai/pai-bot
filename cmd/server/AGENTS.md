# SERVER COMMAND

**Generated:** 2026-07-11
**Commit:** bdd0c16

Production composition root: config and dependency wiring for the HTTP server and chat channels.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Startup dependency graph | `main.go` |
| HTTP lifecycle and handler swap | `internal/server/run.go` |
| Routes and admin SPA embedding | `internal/server/handler.go` |
| Security headers/origins | `internal/server/security.go` |
| Route/lifecycle regressions | `internal/server/handler_test.go`, `internal/server/run_test.go` |
| API shape docs | `internal/apidocs` |

## CONVENTIONS

- `internal/server` owns the health-first handler swap, HTTP lifecycle, and mux adapters.
- DB/config failures are fatal; cache failures degrade where existing code does.
- Keep this package focused on dependency construction and channel registration.

## ANTI-PATTERNS

- No duplicate HTTP lifecycle or handler ownership outside `internal/server`.
- No direct provider-specific AI setup here; use `internal/platform/airouter`.
- No reusable server behavior that forces imports from `cmd/`.
- No channel startup detached from the lifecycle owner that observes its failure.

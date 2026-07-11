# HTTP SERVER RUNTIME

**Generated:** 2026-07-11
**Commit:** bdd0c16

HTTP lifecycle, mux composition, security middleware, and admin/chat HTTP adapters.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Health-first startup and shutdown | `run.go` |
| Top-level mounts and API handler | `handler.go` |
| Security headers and origin policy | `security.go` |
| Runtime settings admin surface | `handler.go`, `internal/platform/settings` |
| OpenAPI/docs routes | `handler.go`, `internal/apidocs` |

## CONVENTIONS

- `Run` exposes a health-only handler before dependency initialization, then atomically swaps in the full handler.
- `NewTopMux` owns transport mounts; domain decisions stay in `internal/*` services.
- Preserve explicit tenant and platform-admin authorization at route boundaries.
- Parse, authenticate, and encode here; keep deterministic calculations in owning packages.
- Exercise handler behavior through real HTTP requests/recorders.

## ANTI-PATTERNS

- No slow initialization before health availability.
- No product authorization represented only by client/user-selected filters.
- No duplicate route ownership in `cmd/server`.
- No raw secrets, learner content, prompts, or image data in logs.
- No security/origin change without focused handler tests and browser-facing verification.

## NOTES

- `handler.go` is intentionally the current route composition seam; split only with behavior-preserving tests.

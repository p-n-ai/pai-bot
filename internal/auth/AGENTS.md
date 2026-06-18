# AUTH DOMAIN

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

JWT/session cookies, Google OIDC, guest accounts, password auth, middleware, and Postgres-backed auth service.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| JWT/access tokens | `jwt.go`, `jwt_test.go` |
| Cookie/session behavior | `cookies.go`, `middleware.go` |
| Google login | `google_oidc.go`, `google_oidc_test.go`, `google_integration_test.go` |
| Guest auth | `guest.go`, `guest_test.go` |
| Password auth | `password.go`, `password_test.go` |
| Persistence/service | `service.go`, `postgres.go`, `*_integration_test.go` |

## CONVENTIONS

- Access tokens stay short-lived; refresh/session handling is explicit.
- Middleware attaches identity/tenant context; broad domain fetches stay out.
- Integration tests own DB behavior; unit tests own token/cookie parsing.
- Cookie/token changes need admin browser flow smoke coverage.

## ANTI-PATTERNS

- No broad env/secret dumps in auth tests or debug output.
- No role checks by scattered string comparisons; use central auth/RBAC helpers.
- No tenant fallback where tenant is required.
- No login side effect without matching persistence/session tests.

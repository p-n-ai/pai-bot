# ADMIN SPA LIB

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Typed API client helpers, response guards, RBAC/redirect logic, and view-model calculations for the admin SPA.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Backend calls | `admin-api.ts` |
| Auth/login | `auth-client.ts`, `auth-types.ts`, `auth-errors.ts`, `auth-feedback.ts` |
| RBAC/routes | `rbac.ts`, `rbac-roles.ts`, `rbac-paths.ts`, `router-guards.ts` |
| Dashboard models | `dashboard-*.ts`, `student-detail-*`, `parent-summary-*` |
| AI usage models | `ai-usage-types.ts`, `ai-usage-view.ts` |
| Retrieval lab models | `retrieval-lab*.ts` |
| Onboarding/classes/users | `onboarding-*`, `group-types.ts`, `user-management-types.ts` |
| Search params/redirects | `*-search.ts`, `root-redirect-target.ts`, `redirect-search.ts` |

## CONVENTIONS

- Keep deterministic calculations here; components render returned data.
- Type guards validate server JSON before components trust shape.
- Tests live beside helpers and use representative fixtures.
- Keep backend contract changes synchronized with Go admin API structs/routes.

## ANTI-PATTERNS

- No React component state in lib helpers.
- No silent `as` casts around API responses when a guard is needed.
- No duplicating RBAC route rules between routes and components.

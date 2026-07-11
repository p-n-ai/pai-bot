# ADMIN API SERVICE

**Generated:** 2026-07-11
**Commit:** bdd0c16

Backend service helpers for admin app features: onboarding, classes, groups, users, and school setup.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Service construction | `service.go`, `service_test.go` |
| Onboarding flow | `onboarding.go`, `onboarding_test.go` |
| Classes/groups | `classes.go`, `groups.go` |
| HTTP route wiring | `internal/server/handler.go` |
| SPA shape mirror | `admin-spa/src/lib/admin-api.ts`, `admin-spa/src/lib/*-types.ts` |

## CONVENTIONS

- HTTP parsing/encoding stays in `internal/server`; product decisions live here.
- Return typed structs fit for API JSON responses.
- Tenant-aware queries only; platform-admin/global behavior is explicit.
- Class/group field changes inspect migrations, seed data, and SPA type guards.

## ANTI-PATTERNS

- No admin UI mock-shape drift.
- No onboarding side effects without idempotency tests.
- No missing-role or tenant-mismatch path without coverage.

## NOTES

- Server handlers currently own auth extraction and response encoding.

# ADMIN API SERVICE

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Backend service helpers for admin app features: onboarding, classes, groups, users, and school setup.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Service construction | `service.go`, `service_test.go` |
| Onboarding flow | `onboarding.go`, `onboarding_test.go` |
| Classes/groups | `classes.go`, `groups.go` |
| HTTP route wiring | `cmd/server/main.go` |
| SPA shape mirror | `admin-spa/src/lib/admin-api.ts`, `admin-spa/src/lib/*-types.ts` |

## CONVENTIONS

- HTTP parsing/encoding stays in `cmd/server`; product decisions live here.
- Return typed structs fit for API JSON responses.
- Tenant-aware queries only; platform-admin/global behavior is explicit.
- Class/group field changes inspect migrations, seed data, and SPA type guards.

## ANTI-PATTERNS

- No admin UI mock-shape drift.
- No onboarding side effects without idempotency tests.
- No missing-role or tenant-mismatch path without coverage.

## NOTES

- Server handlers currently own auth extraction and response encoding.

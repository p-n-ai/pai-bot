# TENANT BOOTSTRAP

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Tenant initialization helpers and single/multi-tenant runtime assumptions.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Bootstrap default tenant | `bootstrap.go`, `bootstrap_test.go` |
| Tenant contract notes | `doc.go` |

## CONVENTIONS

- Bootstrap must be idempotent.
- Single-tenant mode still writes tenant IDs; do not special-case storage away.

## ANTI-PATTERNS

- No global user/class rows without tenant ownership.
- No automatic tenant creation from untrusted request input.

## NOTES

- Tenant mode is config-driven; storage shape remains multi-tenant.
- Bootstrap is called from server/dev seed paths, so keep it cheap.
- Test both existing-tenant and missing-tenant paths.
- Prefer slug lookup at boundaries, UUIDs in storage paths.
- Cross-tenant bugs are security bugs.

# ADMIN DOCS

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Admin panel route/auth docs under `docs/admin`, with broader admin docs at docs root.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Route/API map | `routes.md` |
| Admin auth details | `admin-auth.md` |
| Product spec | `../admin-panel.md` |
| UI/UX spec | `../admin-panel-uiux.md` |

## CONVENTIONS

- Keep route docs aligned with `admin/src/app`, `admin-spa/src/routes`, and `cmd/server` handlers.
- Call out whether a route belongs to Next admin or Vite SPA during migration periods.

## ANTI-PATTERNS

- No documenting mock-only UI as production API behavior.
- No duplicate auth flows that diverge from `internal/auth`.

## NOTES

- Admin SPA migration state can be confusing; name the app explicitly.
- Route docs should include auth/role assumptions when non-obvious.
- If server response shape changes, update frontend client references too.
- Keep screenshots/wireframes out unless requested.

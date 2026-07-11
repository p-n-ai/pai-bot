# NEXT ADMIN APP

**Generated:** 2026-07-11
**Commit:** bdd0c16

Standalone Next.js admin surface; distinct from the primary Vite app in `admin-spa/`.

## STRUCTURE

```
admin/
├── src/app/          # App Router pages, layouts, one proxy route
├── src/components/   # feature and UI components
├── src/lib/          # client/server API, auth, RBAC, view logic
├── src/hooks/        # reusable client hooks
├── src/stores/       # Zustand client state
└── e2e/              # Playwright public and backend-auth suites
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Root/landing layout | `src/app/layout.tsx`, `src/app/page.tsx` |
| Authenticated shell | `src/app/(app)/layout.tsx` |
| Cookie-presence redirects | `src/proxy.ts` |
| Browser API client | `src/lib/api.ts` |
| Server API/session/RBAC | `src/lib/server-api.ts`, auth/RBAC helpers |
| Backend proxying | `next.config.ts`, route handlers under `src/app/api` |

## CONVENTIONS

- Use `pnpm`; ignore package-manager suggestions in the scaffold README.
- Proxy cookie checks are navigation hints, not authorization; server/session checks own access.
- Keep browser same-origin API behavior distinct from server-side backend base URLs.
- Dev-only Agentation wiring must not enter production bundles.
- Unit aggregation, Vitest, and Playwright are separate gates; backend-tagged E2E needs explicit env enablement.

## ANTI-PATTERNS

- No shared implementation assumptions with `admin-spa/`; verify which app owns the requested surface.
- No frontend token ownership or client-only authorization.
- No casual removal of the misspelled retrieval route without proving compatibility callers.
- No edits to generated/build output.

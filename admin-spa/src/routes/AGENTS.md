# ADMIN SPA ROUTES

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

TanStack file routes for public auth flows and authenticated admin sections.

## STRUCTURE

```
routes/
├── __root.tsx
├── index.tsx
├── activate.tsx
├── join.$slug.tsx
└── _authenticated/
    ├── dashboard*.tsx
    ├── settings/*.tsx
    ├── setup/onboard.tsx
    ├── students/$id.tsx
    ├── parents/$id.tsx
    └── export.tsx
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Root shell/providers | `__root.tsx`, `_authenticated.tsx` |
| Public redirects | `index.tsx`, `-index.test.ts` |
| Activation/join | `activate.tsx`, `join.$slug.tsx` |
| Dashboard pages | `_authenticated/dashboard*.tsx` |
| Settings pages | `_authenticated/settings/*.tsx` |
| Detail pages | `_authenticated/students/$id.tsx`, `_authenticated/parents/$id.tsx` |
| Generated route tree | `../routeTree.gen.ts` |

## CONVENTIONS

- Route files guard/load/wire; components and `src/lib` own display logic/calculations.
- Authenticated routes use shared guard/RBAC helpers from `src/lib`.
- Search params are parsed by dedicated `src/lib/*search*.ts` helpers.
- Keep `routeTree.gen.ts` aligned after file-route changes.

## ANTI-PATTERNS

- No duplicated redirect/RBAC rules in individual routes.
- No large JSX dashboards in route files; delegate to components.
- No manual route IDs/paths that fight TanStack file-route conventions.

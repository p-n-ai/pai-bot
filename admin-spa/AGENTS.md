# ADMIN SPA

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Vite + React + TanStack Router admin SPA for school/admin onboarding, dashboards, AI usage, retrieval lab, exports, settings, and auth flows.

## STRUCTURE

```
admin-spa/
├── src/routes/       # TanStack file routes (AGENTS.md)
├── src/lib/          # API client, type guards, view models (AGENTS.md)
├── src/components/   # feature/shared UI components (AGENTS.md)
├── src/hooks/        # small reusable hooks
├── src/routeTree.gen.ts
└── vite.config.ts
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Router setup | `src/router.tsx`, `src/routes`, `src/routeTree.gen.ts` |
| Auth state | `src/auth-provider.tsx`, `src/lib/auth-client.ts`, `src/lib/auth-types.ts` |
| Backend API calls | `src/lib/admin-api.ts` |
| RBAC/redirects | `src/lib/rbac*.ts`, `src/lib/router-guards.ts`, `src/lib/*redirect*` |
| Dashboard data/view models | `src/lib/dashboard-*`, `src/components/dashboard` |
| AI usage | `src/lib/ai-usage-*`, `src/components/ai-usage` |
| Onboarding/classes/users | `src/lib/onboarding-*`, `src/components/onboarding`, `src/components/classes`, `src/components/users` |
| Retrieval lab | `src/lib/retrieval-lab*`, `src/components/retrieval` |
| Shared UI primitives | `src/components/ui` |

## CONVENTIONS

- Use `pnpm` in this directory.
- Route files stay thin: load/guard/wire components and lib helpers.
- Backend response validation lives in `src/lib/*-types.ts` guards.
- Keep `src/routeTree.gen.ts` aligned with route changes.
- Intent/shadcn guidance in the block below applies before substantial component/router work.

## ANTI-PATTERNS

- No package-manager swaps.
- No business logic hidden in route components when `src/lib` can own it.
- No admin API shape drift from Go `internal/adminapi`/`cmd/server` responses.
- No hand-authored router tree that disagrees with file routes.

## COMMANDS

```bash
pnpm dev
pnpm test
pnpm typecheck
pnpm build
pnpm check
pnpm run intent:vite
pnpm run intent:router
```

<!-- intent-skills:start -->

## Skill Loading

Before substantial work:

- Skill check: run `pnpm dlx @tanstack/intent@latest list`, or use skills already listed in context.
- Skill guidance: if one local skill clearly matches the task, run `pnpm dlx @tanstack/intent@latest load <package>#<skill>` and follow the returned `SKILL.md`.
- Vite/router config: run `pnpm run intent:vite` and keep `tanstackRouter(...)` before `react()` in `vite.config.ts`.
- Component work: use `building-components`; prefer semantic HTML, accessible defaults, composable props, visible focus states, and lightweight component APIs.
- UI review: use `web-design-guidelines`; fetch the latest Vercel Web Interface Guidelines before review and check changed UI files against accessibility, focus, form, and animation rules.
- shadcn/ui: use `shadcn` before adding or composing UI. Run `pnpm dlx shadcn@latest info --json` for project context, use `pnpm dlx shadcn@latest docs <component>` before component work, prefer installed components, and keep imports aligned with aliases in `components.json`.
- shadcn project context: Vite SPA, Tailwind v4, radix-nova, lucide icons, `@/*` alias, UI components under `src/components/ui`.
- Monorepos: when working across packages, run the skill check from the workspace root and prefer the local skill for the package being changed.
- Multiple matches: prefer the most specific local skill for the package or concern you are changing; load additional skills only when the task spans multiple packages or concerns.
<!-- intent-skills:end -->

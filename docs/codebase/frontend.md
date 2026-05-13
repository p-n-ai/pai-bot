---
title: "Frontend Surfaces"
summary: "Current map of pai-bot frontend folders: legacy Next.js admin, active admin-spa TanStack Router app, Astro site, UI components, routes, clients, and tests."
read_when:
  - You are changing admin UI, login, dashboard, retrieval lab, landing page, or the Astro site.
  - You need to decide whether a frontend change belongs in admin/ or site/.
  - You are updating frontend docs or AGENTS.md folder guidance.
---

# Frontend Surfaces

There are three frontend surfaces:

- `admin/`: operational Next.js app for login, dashboard, admin tools, and product UI.
- `admin-spa/`: active local demo/admin deliverable surface built with Vite, TanStack Router, Tailwind v4, and shadcn/ui. Treat it as the current route-by-route admin implementation workspace unless the task explicitly targets the legacy Next.js app.
- `site/`: Astro public site/docs surface.

Do not mix these responsibilities. Legacy Next.js admin work belongs in `admin/`; current SPA admin work belongs in `admin-spa/`; public/marketing site work belongs in `site/`.

## Admin app

| Path                               | Purpose                                                                                |
| ---------------------------------- | -------------------------------------------------------------------------------------- |
| `admin/src/app/`                   | Next.js App Router routes and layouts.                                                 |
| `admin/src/app/(app)/dashboard/`   | Authenticated dashboard pages: overview, metrics, classes, AI usage, retrieval lab.    |
| `admin/src/app/(onboarding)/`      | Setup/onboarding routes.                                                               |
| `admin/src/app/api/`               | Next.js proxy/API routes used by the admin app.                                        |
| `admin/src/components/`            | App-owned UI components and feature panels.                                            |
| `admin/src/components/ui/`         | shadcn/ui primitives. Keep primitive ownership separate from app shell behavior.       |
| `admin/src/components/login-gate/` | Login-page composition and theme-specific login gate surfaces.                         |
| `admin/src/components/landing/`    | Root landing page sections and copy draft.                                             |
| `admin/src/hooks/`                 | Browser/session/bootstrap hooks.                                                       |
| `admin/src/lib/`                   | Client/server helpers, data transforms, API wrappers, RBAC, metrics, dashboard models. |
| `admin/src/stores/`                | Zustand app/onboarding state.                                                          |
| `admin/e2e/`                       | End-to-end tests.                                                                      |

## Admin SPA migration app

Read [docs/admin-spa-migration.md](../admin-spa-migration.md) before working in `admin-spa/`.

| Path                           | Purpose                                                                                                |
| ------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `admin-spa/src/routes/`        | TanStack Router file-based routes for the SPA migration.                                               |
| `admin-spa/src/components/`    | Feature-grouped route panels and reusable admin components for the SPA.                                |
| `admin-spa/src/components/ai-usage/` | AI usage dashboards, budget editing, provider breakdowns, and related tests.                     |
| `admin-spa/src/components/auth/` | Login, activation, join, and root entry components.                                                  |
| `admin-spa/src/components/classes/` | Class management, invite, assigned-topic, and create-class components.                            |
| `admin-spa/src/components/dashboard/` | Dashboard overview plus parent/student detail panels.                                           |
| `admin-spa/src/components/onboarding/` | Onboarding setup flow components and tests.                                                     |
| `admin-spa/src/components/settings/` | Settings panels such as embed config and WhatsApp setup.                                         |
| `admin-spa/src/components/shared/` | Reusable admin surfaces, state panels, tables, metrics, and page chrome.                          |
| `admin-spa/src/components/ui/` | shadcn/ui primitives for the migration app. Keep these separate from app-specific behavior.             |
| `admin-spa/src/lib/`           | SPA helpers, utilities, typed API clients, route guards, and response guards.                          |
| `admin-spa/src/hooks/`         | SPA hooks for submit, invite, usage, and route state workflows.                                        |
| `admin-spa/src/stores/`        | Zustand app state for SPA shell behavior.                                                              |
| `admin-spa/components.json`    | shadcn project config: Tailwind v4, radix-nova, lucide, `@/*` alias.                                   |

### Admin SPA rules

- Frontend-to-backend connection is the first priority. Prefer typed same-origin API clients in `admin-spa/src/lib/admin-api.ts` and strict response guards before UI polish.
- UI/UX functionality is also priority: route navigation, loading/error/empty states, forms, tables, and disabled/submitting states must work before decorative changes.
- Use `docs/admin-panel.md`, `docs/admin/routes.md`, and backend route contracts to decide which admin features belong live in the SPA.
- Use Refero research as the design and flow driver for route overhaul work: compact operational headers, fixed sidebar, dense tables, badges, filter rows, and clear inline actions.
- Keep shadcn primitives stock. App-specific shell behavior belongs in SPA components such as `admin-spa/src/components/app-sidebar.tsx`.
- Keep UI copy short and functional. Avoid helper prose, framework labels, duplicate eyebrows, or explanatory blocks that do not help the user act.
- Run `pnpm --dir admin-spa exec tsc --noEmit`, `pnpm --dir admin-spa run lint`, focused tests, and Debtmap/Fallow checks after meaningful edit batches.

## Admin route notes

| Route folder                                           | Runtime meaning                                                     |
| ------------------------------------------------------ | ------------------------------------------------------------------- |
| `admin/src/app/page.tsx`                               | Root entry/landing or redirect behavior.                            |
| `admin/src/app/login/page.tsx`                         | Login entry.                                                        |
| `admin/src/app/activate/page.tsx`                      | Account/session activation.                                         |
| `admin/src/app/join/[slug]/page.tsx`                   | Invite join surface.                                                |
| `admin/src/app/settings/whatsapp/page.tsx`             | WhatsApp setup surface.                                             |
| `admin/src/app/(app)/dashboard/retrieval-lab/page.tsx` | Current retrieval lab route.                                        |
| `admin/src/app/(app)/dashboard/retreival-lab/page.tsx` | Legacy misspelled route; keep only if compatibility is intentional. |

## Site app

| Path                   | Purpose                  |
| ---------------------- | ------------------------ |
| `site/src/pages/`      | Astro page routes.       |
| `site/src/components/` | Astro layout components. |
| `site/src/layouts/`    | Astro page layouts.      |
| `site/src/styles/`     | Site-specific styles.    |
| `site/public/`         | Site static assets.      |

## Frontend rules

- Use `pnpm` inside the app being changed.
- Do not move shared shadcn primitives into app-owned behavior.
- Keep product dashboard UI dense and operational; landing/site work can be more editorial.
- Update docs when routes, auth behavior, or dashboard API contracts change.
- For login/auth changes, read `docs/admin/admin-auth.md`.
- For admin shell/layout changes, read `docs/admin-panel-uiux.md`.
- For route/navigation changes, read `docs/admin/routes.md`.

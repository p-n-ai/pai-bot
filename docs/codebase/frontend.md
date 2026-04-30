---
title: "Frontend Surfaces"
summary: "Current map of pai-bot frontend folders: Next.js admin app, Astro site, UI components, app routes, client libraries, and tests."
read_when:
  - You are changing admin UI, login, dashboard, retrieval lab, landing page, or the Astro site.
  - You need to decide whether a frontend change belongs in admin/ or site/.
  - You are updating frontend docs or AGENTS.md folder guidance.
---

# Frontend Surfaces

There are two frontend surfaces:

- `admin/`: operational Next.js app for login, dashboard, admin tools, and product UI.
- `site/`: Astro public site/docs surface.

Do not mix these responsibilities. Admin product UI belongs in `admin/`; public/marketing site work belongs in `site/`.

## Admin app

| Path | Purpose |
|---|---|
| `admin/src/app/` | Next.js App Router routes and layouts. |
| `admin/src/app/(app)/dashboard/` | Authenticated dashboard pages: overview, metrics, classes, AI usage, retrieval lab. |
| `admin/src/app/(onboarding)/` | Setup/onboarding routes. |
| `admin/src/app/api/` | Next.js proxy/API routes used by the admin app. |
| `admin/src/components/` | App-owned UI components and feature panels. |
| `admin/src/components/ui/` | shadcn/ui primitives. Keep primitive ownership separate from app shell behavior. |
| `admin/src/components/login-gate/` | Login-page composition and theme-specific login gate surfaces. |
| `admin/src/components/landing/` | Root landing page sections and copy draft. |
| `admin/src/hooks/` | Browser/session/bootstrap hooks. |
| `admin/src/lib/` | Client/server helpers, data transforms, API wrappers, RBAC, metrics, dashboard models. |
| `admin/src/stores/` | Zustand app/onboarding state. |
| `admin/e2e/` | End-to-end tests. |

## Admin route notes

| Route folder | Runtime meaning |
|---|---|
| `admin/src/app/page.tsx` | Root entry/landing or redirect behavior. |
| `admin/src/app/login/page.tsx` | Login entry. |
| `admin/src/app/activate/page.tsx` | Account/session activation. |
| `admin/src/app/join/[slug]/page.tsx` | Invite join surface. |
| `admin/src/app/(app)/settings/embed/page.tsx` | Embed widget settings for enabled state, trusted origins, theme props, and install snippet. |
| `admin/src/app/settings/whatsapp/page.tsx` | WhatsApp setup surface. |
| `admin/src/app/(app)/dashboard/retrieval-lab/page.tsx` | Current retrieval lab route. |
| `admin/src/app/(app)/dashboard/retreival-lab/page.tsx` | Legacy misspelled route; keep only if compatibility is intentional. |

## Site app

| Path | Purpose |
|---|---|
| `site/src/pages/` | Astro page routes. |
| `site/src/components/` | Astro layout components. |
| `site/src/layouts/` | Astro page layouts. |
| `site/src/styles/` | Site-specific styles. |
| `site/public/` | Site static assets. |

## Frontend rules

- Use `pnpm` inside the app being changed.
- Do not move shared shadcn primitives into app-owned behavior.
- Keep product dashboard UI dense and operational; landing/site work can be more editorial.
- Update docs when routes, auth behavior, or dashboard API contracts change.
- For login/auth changes, read `docs/admin/admin-auth.md`.
- For admin shell/layout changes, read `docs/admin-panel-uiux.md`.
- For route/navigation changes, read `docs/admin/routes.md`.

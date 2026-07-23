---
title: "Admin Routes"
summary: "Current admin frontend route map and backend API route groups for pai-bot."
read_when:
  - You are changing admin pages, navigation, dashboard routes, or admin API endpoints.
  - You need to map a Next.js admin page to its backend route.
  - You are updating admin docs after route or API changes.
---

# Admin Routes

## Frontend routes

| App path | File | Purpose |
|---|---|---|
| `/` | `admin/src/app/page.tsx` | Root landing or app entry behavior. |
| `/login` | `admin/src/app/login/page.tsx` | Login entry. |
| `/activate` | `admin/src/app/activate/page.tsx` | Account activation and invite acceptance support. |
| `/join/[slug]` | `admin/src/app/join/[slug]/page.tsx` | Public invite join surface. |
| `/dashboard` | `admin/src/app/(app)/dashboard/page.tsx` | Main dashboard. |
| `/dashboard/metrics` | `admin/src/app/(app)/dashboard/metrics/page.tsx` | Legacy redirect to AI usage. |
| `/dashboard/classes` | `admin/src/app/(app)/dashboard/classes/page.tsx` | Class management. |
| `/dashboard/ai-usage` | `admin/src/app/(app)/dashboard/ai-usage/page.tsx` | AI usage and budget view. |
| `/dashboard/retrieval-lab` | `admin/src/app/(app)/dashboard/retrieval-lab/page.tsx` | Retrieval lab. |
| `/dashboard/retreival-lab` | `admin/src/app/(app)/dashboard/retreival-lab/page.tsx` | Legacy misspelled redirect to `/dashboard/retrieval-lab`. |
| `/export` | `admin/src/app/(app)/export/page.tsx` | CSV/export tools. |
| `/parents/[id]` | `admin/src/app/(app)/parents/[id]/page.tsx` | Parent summary view. |
| `/students/[id]` | `admin/src/app/(app)/students/[id]/page.tsx` | Student detail view. |
| `/settings/users` | `admin/src/app/(app)/settings/users/page.tsx` | User management settings. |
| `/settings/whatsapp` | `admin/src/app/settings/whatsapp/page.tsx` | WhatsApp setup/status page. |
| `/setup/onboard` | `admin/src/app/(onboarding)/setup/onboard/page.tsx` | Onboarding setup wizard. |
| `/api/retrieval-lab/search` | `admin/src/app/api/retrieval-lab/search/route.ts` | Next.js proxy route for retrieval-lab search. |

## Admin SPA migration routes

| App path | File | Purpose |
|---|---|---|
| `/settings/budget` | `admin-spa/src/routes/_authenticated/settings/budget.tsx` | Admin/platform-admin token budget route using `/api/admin/ai/usage` and `/api/admin/ai/budget-window`. |
| `/settings/embed` | `admin-spa/src/routes/_authenticated/settings/embed.tsx` | Admin/platform-admin embed configuration route using `/api/admin/embed/config` and `/api/admin/embed/origins`. |

## Backend route groups

| Group | Routes |
|---|---|
| health/docs | `GET /healthz`, `GET /readyz`, `GET /openapi.json`, `GET /docs` |
| auth | `POST /api/auth/login`, `GET /api/auth/google/start`, `GET /api/auth/google/callback`, `POST /api/auth/google/link/start`, `GET /api/auth/identities`, `POST /api/auth/invitations/accept`, `GET /api/auth/session`, `POST /api/auth/switch-tenant`, `POST /api/auth/logout` |
| public join | `GET /api/join/{slug}` |
| admin users/onboarding/invites | `GET /api/admin/users` (students plus platform users/invites), `GET|POST /api/admin/onboarding`, `POST /api/admin/invites`, `POST /api/admin/invites/{id}/reissue` |
| dashboard data | `GET /api/admin/classes/{id}/progress`, `GET /api/admin/students/{id}`, `GET /api/admin/students/{id}/conversations`, `POST /api/admin/students/{id}/nudge`, `GET /api/admin/metrics`, `GET /api/admin/ai/usage`, `POST /api/admin/ai/budget-window` |
| exports | `GET /api/admin/export/students`, `GET /api/admin/export/conversations`, `GET /api/admin/export/progress`, `GET /api/admin/analytics/report` |
| parent view | `GET /api/admin/parents/{id}` |
| groups | `GET|POST /api/admin/groups`, `GET|PATCH|DELETE /api/admin/groups/{id}`, `POST /api/admin/groups/{id}/members`, `DELETE /api/admin/groups/{id}/members/{uid}`, `GET /api/admin/groups/{id}/leaderboard` |
| retrieval | `/api/admin/retrieval/sources`, `/collections`, `/documents`, and `POST /api/admin/retrieval/search` |
| embed admin | `GET|PUT /api/admin/embed/config`, `POST|DELETE /api/admin/embed/origins` |
| WhatsApp admin | `GET /api/admin/whatsapp/status`, `POST /api/admin/whatsapp/disconnect` |

## Update rules

- Update this doc when adding, removing, renaming, or redirecting admin routes.
- Update `docs/admin-panel.md` when feature scope changes.
- Update `docs/admin-panel-uiux.md` when navigation or shell behavior changes.
- Update `internal/apidocs` when backend API surface changes.

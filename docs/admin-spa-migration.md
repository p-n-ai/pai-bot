---
title: "Admin SPA Migration"
summary: "Active local admin deliverable and migration plan for the admin-spa TanStack Router surface."
read_when:
  - You are porting admin routes from Next.js to admin-spa.
  - You are changing admin-spa architecture, quality gates, or migration scope.
  - You need the route-by-route plan for replacing or improving the current admin surface.
---

# Admin SPA Migration

> **Status:** Active local admin deliverable. `admin-spa/` is the TanStack Router SPA workspace used for the current demo/prototype admin surface. The legacy `admin/` Next.js app remains useful source material and fallback history, but new admin UI/UX work should default to `admin-spa/` unless the task explicitly targets `admin/`.

## Goal

Port and harden the admin experience from `admin/` Next.js App Router into `admin-spa/` as a Vite + TanStack Router SPA, then keep improving the SPA as the working admin product surface.

The migration is not a cosmetic rewrite. It is a quality reset around explicit client routing, API contracts, typed data boundaries, reusable shadcn components, behavior-first tests, and stricter static analysis.

## Why This Exists

The working concern is that the current React Server Components/App Router shape is a poor fit for this admin surface:

- performance risk: extra framework/runtime complexity for an operational dashboard that should behave like a dense client app;
- security risk: server/client boundary mistakes are easier to introduce when sensitive admin state crosses implicit RSC boundaries;
- testability risk: route behavior, auth redirects, and API reads should be explicit, typed, and exercised through public browser-facing interfaces.

This is a WIP judgment for this migration, not a universal claim about every RSC app.

## Hard Scope

- New admin SPA work lives in `admin-spa/`.
- Do not add SPA migration artifacts to the current `admin/` folder.
- Install JavaScript/TypeScript tooling only inside `admin-spa/`.
- Preserve the current `admin/` app as source/fallback until cutover is explicit.
- Use the existing backend HTTP APIs; do not move backend ownership into the SPA.
- Treat frontend-to-backend connection as absolute priority. A route is not live if it only renders static UI.
- Treat UI/UX functionality as priority alongside connectivity: route flow, sidebar navigation, forms, loading/error/empty states, and table actions must be usable.

## Required Skill Stack

Use these skills for all meaningful migration slices:

- `building-components`: accessible, composable, lightweight component APIs.
- `web-design-guidelines`: fetch and apply the latest interface/accessibility guidelines before UI review.
- `shadcn`: use project context, component docs, installed primitives, and semantic tokens before custom markup.
- `oauth`: use for Google/OIDC login flow checks; SPA must not own tokens, must only pass safe relative return paths to backend-owned OAuth start/callback flows.
- `to-issues`: split work into vertical tracer-bullet issues, not horizontal layer tasks.
- `tdd`: one behavior test, one implementation, then repeat.
- `node`: Node/Vite/TypeScript runtime discipline, scripts, async behavior, and environment handling.
- `nodejs-core`: use when diagnosing Vite/Node process hangs, native build failures, event-loop issues, or runtime performance below normal app code.
- `typescript-magician`: strict types, no `any`, route/data inference, typed guards for API responses.
- `linting-neostandard-eslint9`: ESLint v9 flat config with neostandard baseline.
- `deep-code-review`: self-review substantial slices before handoff or PR.

## Admin SPA Baseline

Current scaffold:

- Vite React SPA.
- TanStack Router file-based routes with `@tanstack/router-plugin`.
- shadcn/ui initialized with Tailwind v4, radix-nova, lucide icons, and `@/*` alias.
- ESLint v9 + neostandard + `typescript-eslint` recommended + shadcn/ui app lint posture where it applies to a Vite SPA + Tailwind class ordering (`tailwindcss/classnames-order`) with Next-only rules excluded.
- Oxlint with React, import, JSX accessibility, performance, TypeScript, unicorn, and oxc checks.
- Prettier formatter.
- Fallow codebase intelligence for dead code, duplication, circular dependencies, complexity, and architecture drift.
- First auth tracer in `admin-spa/`: typed `/api/auth/session` probe with `credentials: include`, strict session contract guard, current-admin RBAC port, role-aware `/` entry route with encoded signed-out `next` links, signed-in safe `next` preservation, source-admin landing subtitle/badge, source-admin landing headline/value proposition, source-admin hero signal trio, source-admin live demo queue/tutor thread/check-in visuals, source-admin daily loop, source-admin workflow summary, source-admin teacher outcomes, source-admin tomorrow plan outcome rail, source-admin teacher-ready evidence spotlight, source-admin command strip, and source-admin footer action, `/login`, protected `/_authenticated` layout, and `/dashboard`.
- Login tracer in `admin-spa/`: password POST to `/api/auth/login`, cookie credentials, typed session contract, tenant-required response branch with selected-school retry coverage, Google start URL with `next`, source-admin login gate hero/auth panel framing, source-admin Google button treatment, source-admin Google/email divider, auth-error copy, and safe post-login routing.
- Dashboard tracer in `admin-spa/`: typed `GET /api/admin/classes/all-students/progress`, preview fallback when live class data fails, source-admin page heading/description, source-admin summary signals (class grade, attention count, strongest/weakest topic), source-admin class-grade and coverage stat notes, source-admin mastery heatmap surface heading/description, source-admin heatmap row/cell framing, source-admin bounded topic labels with keyboard-accessible tooltip triggers, source-admin mastery score chips, heatmap table, student detail drilldown links with chevron affordance, source-admin compact primary nudge button treatment, source-admin loading/error copy, source-admin empty heatmap surface, and `POST /api/admin/students/{id}/nudge` with source-admin Telegram success copy.
- Class management tracer in `admin-spa/`: typed `GET /api/admin/groups?type=class`, `GET /api/admin/groups/{id}`, `POST /api/admin/groups`, and shared invite workflow; source-admin summary stats, first-loaded-class auto-selection, class selector surface heading/description, selected-class subject/syllabus/cadence chips, join-code helper text, roster surface heading/description, roster table, assigned-topic progress panel, create-class form, invite form, empty/loading/error states.
- AI usage tracer in `admin-spa/`: typed `GET /api/admin/ai/usage`, usage summary stats with source-admin metric notes, token-budget priority with USD-budget fallback, source-admin token budget window surface, source-admin daily token trend surface, source-admin provider breakdown surface, provider table, monthly cost/USD budget/top-provider summary panels, empty/loading/error states, and admin-only budget editing with typed `POST /api/admin/ai/budget-window`.
- Legacy metrics redirect in `admin-spa/`: `/dashboard/metrics` redirects to `/dashboard/ai-usage` with route-level coverage.
- Retrieval lab tracer in `admin-spa/`: typed retrieval payload builder, typed `POST /api/admin/retrieval/search`, source-admin `PaiBot Search` title/description, source-admin search-first layout with source-copy advanced settings and raw output triggers collapsed behind accessible controls, accessible labeled controls, result sections hidden until the first run, repeat-run timing, source-admin compact run summary, source-admin unboxed result spacing, source-admin bordered empty-results panel, source-admin result metadata chips, error states, and expired-session redirect back to `/login?next=/dashboard/retrieval-lab`.
- Export tracer in `admin-spa/`: admin/platform-admin-only route with same-origin download links for students, conversations, and progress exports, covered by RBAC plus source title/description rendered-download behavior tests, source-admin export card heading semantics, and source-admin card-grid affordance.
- User management tracer in `admin-spa/`: admin/platform-admin-only route with typed `GET /api/admin/users`, `POST /api/admin/invites`, and `POST /api/admin/invites/{id}/reissue`; summary stats, active-user and pending-invite tenant display/search, pending-invite list with delivery and lifecycle status, accessible invite role selector with source-admin helper copy, issue/reissue flows, source-labeled activation-link copy behavior with source-admin delivery guidance, source-admin generic load-error copy, and loading/error states.
- Public join tracer in `admin-spa/`: typed `GET /api/join/{slug}`, public `/join/$slug` route, source-admin invite summary page/card shell, source-admin invite summary card coverage, loading/error states, and strict public response guard.
- Invite activation tracer in `admin-spa/`: public `/activate?token=...` route, typed `POST /api/auth/invitations/accept`, cookie credentials, strict session guard, source-admin activation two-column shell, source-admin activation framing/trust cues, accessible password guidance, missing-token state, and post-activation route selection.
- Onboarding setup tracer in `admin-spa/`: admin/platform-admin `/setup/onboard` route, typed `GET/POST /api/admin/onboarding`, first-class setup wizard with source-admin progress indicator, slug normalization, source-admin success copy, save result state, join-link copy/open/edit actions, teacher invite success panel, and strict onboarding response guards.
- WhatsApp setup tracer in `admin-spa/`: admin/platform-admin `/settings/whatsapp` route with source-admin RBAC parity, typed `GET /api/admin/whatsapp/status`, typed `POST /api/admin/whatsapp/disconnect`, QR/connected/loading/error states, source-admin QR setup description and scan guidance, source-admin QR waiting surface, source-admin connected active-session surface, retry, and 5-second status polling.
- Token budget tracer in `admin-spa/`: admin/platform-admin `/settings/budget` route using the existing typed AI usage and budget-window contracts, with the source-admin token budget window surface, loading/error states, and admin budget editing.
- Embed setup tracer in `admin-spa/`: admin/platform-admin `/settings/embed` route using typed `GET|PUT /api/admin/embed/config` and `POST|DELETE /api/admin/embed/origins`, with a compact admin setup panel for enablement and allowed-origin management.
- Parent summary tracer in `admin-spa/`: typed `/parents/$id` route, typed `GET /api/admin/parents/{id}`, source-admin `PageHero`, `AdminSurface`, `AdminHighlightPanel`, `AdminInsetPanel`, `Metric`, and `StatCard` primitives ported into the SPA, weekly stats, streak stats, source-shaped mastery rows with admin-style UTC review dates, encouragement state, and load-error behavior that does not fall through to empty mastery.
- Student detail tracer in `admin-spa/`: typed `/students/$id` route, typed `GET /api/admin/students/{id}` and `GET /api/admin/students/{id}/conversations`, source-admin `PageHero`, `AdminSurface`, `AdminHighlightPanel`, `AdminInsetPanel`, `Metric`, Recharts radar, profile card, struggle-area badges plus progress insets, 14-day activity grid, conversation inset cards, admin-style UTC date formatting, and hard-load error behavior.
- Tailwind-first styling pass in `admin-spa/`: retrieval lab, export cards, WhatsApp setup, shared admin page/surface/state/stat components, dashboard heatmap, class management, login form, user invite form, invite activation form, onboarding wizard flow, onboarding result actions, teacher invite outcomes, student detail, and parent summary now prefer component-local Tailwind utility classes over bespoke CSS selectors. Remaining CSS should be treated as migration debt and removed only after its owning component/route has equivalent Tailwind coverage plus behavior tests.

Required local gate:

```bash
cd admin-spa
pnpm run check
```

Current focused gates used during route work:

```bash
pnpm --dir admin-spa exec tsc --noEmit
pnpm --dir admin-spa run lint
pnpm --dir admin-spa exec vitest run <focused-test-files>
debtmap analyze admin-spa/src/components --format markdown --top 12
```

Use Fallow for unused-code or dead-surface checks after larger route reshapes.

## UI/UX Migration Contract

Use [docs/admin-panel-uiux.md](admin-panel-uiux.md) as the source UI reference for the current admin experience. The SPA migration should not treat visual parity as pixel-copying the old Next.js app. It should preserve the same product intent, role context, interaction affordances, and accessibility guarantees while simplifying implementation into explicit client routes.

The admin surface is an operational workspace, not a marketing site. Screens should be dense enough for teachers and admins to scan repeatedly during real work: clear headings, short explanatory copy, stable tables, predictable actions, and quiet feedback states. Avoid decorative redesign while a route is still a tracer. A tracer may look simpler than the source route, but it must not hide important state, remove role context, or make recovery from loading/error/empty states harder.

Route-level UI review should answer four questions before a slice moves past tracer status:

- Does the route still explain the user's current task in product language, not framework language?
- Are primary actions and destructive actions visually distinct, keyboard reachable, and backed by observable success or failure states?
- Do loading, empty, error, expired-session, and permission-denied states preserve the user's next step?
- Does the mobile layout keep navigation, page identity, forms, tables, and long labels usable without overlap or horizontal drift?

Copy should stay concrete and teacher/admin-facing. Prefer "Invite a teacher", "Send parent summary", "Review weak topics", or "Retry WhatsApp status" over generic labels like "Submit", "Action", or "Manage". When a route cannot yet match the current admin visual detail, record that as "visual parity pending" and name the missing behavior or surface in the route inventory rather than leaving parity as a vague bucket.

Interaction feel should remain restrained. Page transitions may use the existing short blur/fade/y movement language, but the work surface should not animate in ways that delay scanning. Theme, school switch, login, invite, export, WhatsApp, and nudge flows need visible feedback because they change session, tenant, or delivery state. Read-only tables and summaries should prioritize stability over flourish.

Accessibility is part of parity. shadcn primitives should carry the hard interaction work where possible; custom components must keep labels, descriptions, focus order, disabled states, and keyboard access explicit. Tooltip-only information is not enough for route-critical state. Tables with compact chips or truncated labels need hover/focus affordances plus readable fallback text.

## Route Port Inventory

Source route truth currently lives in `docs/admin/routes.md`.

| Current Next.js route      | Target admin-spa route     | Migration status | Notes                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| -------------------------- | -------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `/`                        | `/`                        | Tracer only      | Role-aware entry route now sends anonymous users to login with encoded safe `next`, preserves accessible signed-in `next` destinations, sends teachers/admins to dashboard, sends parents to their parent summary, and renders the source-admin landing subtitle/badge, headline/value proposition, hero signal trio, live demo queue/tutor thread/check-in visuals, daily loop, workflow summary, teacher outcomes, tomorrow plan outcome rail, teacher-ready evidence spotlight, command strip, and footer action. Full visual parity still pending.                                                                    |
| `/login`                   | `/login`                   | Tracer only      | Has session-aware redirect, password login, Google start link with `next`, source-admin login gate hero/auth panel framing, source-admin Google button treatment, source-admin Google/email divider, Google redirect pending state, auth-error copy, and tenant-required selected-school retry coverage. Full visual parity and live backend multi-tenant verification still pending.                                                                                                                                                                                                                                     |
| `/activate`                | `/activate`                | Tracer only      | Accepts invite token with typed activation POST, source-admin two-column shell, source-admin framing/trust cues, accessible password guidance, and redirect to the user's default admin route. Full activation QA still pending.                                                                                                                                                                                                                                                                                                                                                                                          |
| `/join/[slug]`             | `/join/$slug`              | Tracer only      | Public invite summary reads `/api/join/{slug}` and renders class/school/curriculum context with source page/card shell and source copy coverage. Invite completion and full visual parity still pending.                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `/dashboard`               | `/dashboard`               | Tracer only      | Protected by current-admin RBAC port. Reads live class progress, falls back to typed preview progress, shows source-admin page heading/description, summary signals, class-grade and coverage stat notes, mastery heatmap surface heading/description, heatmap row/cell framing, bounded topic labels with keyboard-accessible tooltip triggers, mastery score chips, chevron detail links, compact primary nudge buttons, source-admin loading/error copy, and source-admin empty heatmap surface, links students to detail pages, and supports nudge POST with Telegram success copy. Full visual parity still pending. |
| `/dashboard/metrics`       | `/dashboard/metrics`       | Tracer only      | Legacy redirect to AI usage, covered by route redirect tests.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `/dashboard/classes`       | `/dashboard/classes`       | Tracer only      | Lists class groups, auto-selects the first loaded class like source-admin, creates classes, reads selected roster detail, shows source-admin summary stats, singular/plural learner summaries, class selector surface heading/description, join code plus `/join CODE` helper text, selected-class metadata chips, roster surface heading/description, issues invites through the shared invite workflow, and shows assigned-topic progress. Full visual parity still pending.                                                                                                                                            |
| `/dashboard/ai-usage`      | `/dashboard/ai-usage`      | Tracer only      | Reads AI usage summary, shows token/message stats with source-admin metric notes, token-budget priority with USD fallback, source-admin token budget window surface, source-admin daily token trend surface, source-admin provider breakdown surface, provider table, monthly cost/USD budget/top-provider summary panels, and admin-only token budget editing. Full visual parity still pending.                                                                                                                                                                                                                         |
| `/dashboard/retrieval-lab` | `/dashboard/retrieval-lab` | Tracer only      | Typed search form, source-admin title/description, source-admin search-first layout, idle result gating, source-copy collapsed advanced settings/raw output triggers, source-admin compact run summary, source-admin unboxed result spacing, source-admin bordered empty-results panel, source-admin result metadata chips, and direct backend admin retrieval search call are ported. Full visual parity still pending.                                                                                                                                                                                                  |
| `/export`                  | `/export`                  | Tracer only      | Admin/platform-admin-only same-origin download links for students, conversations, and progress exports, with route RBAC, source title/description rendered-download coverage, source-admin export card heading semantics, and source-admin card-grid affordance. Full visual parity still pending.                                                                                                                                                                                                                                                                                                                        |
| `/parents/[id]`            | `/parents/$id`             | Tracer only      | Reads parent summary, renders weekly/streak stats with source-admin day units, mastery rows with admin-style UTC review dates, and encouragement copy. Full visual parity and tone polish still pending.                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `/students/[id]`           | `/students/$id`            | Tracer only      | Reads student profile/progress and recent conversations. Source-admin streak day units, mastery radar snapshot, struggle badges, activity grid, and admin-style UTC date formatting are ported. Full visual parity still pending.                                                                                                                                                                                                                                                                                                                                                                                         |
| `/settings/users`          | `/settings/users`          | Tracer only      | Reads users/invites, renders and searches tenant names for active users and pending invites, shows pending-invite delivery plus lifecycle status, keeps the invite role selector labeled/described like source-admin, issues invites, reissues pending invites, copies source-labeled activation links with source-admin delivery guidance, and keeps the source-admin generic load-error copy. Full visual parity still pending.                                                                                                                                                                                         |
| `/settings/budget`         | `/settings/budget`         | Tracer only      | Admin/platform-admin route for tenant token budget management. Reads the typed AI usage payload, renders the source-admin token budget window surface, and saves budget windows through `POST /api/admin/ai/budget-window`. Full visual parity still pending.                                                                                                                                                                                                                                                                                                                                                             |
| `/settings/embed`          | `/settings/embed`          | Tracer only      | Admin/platform-admin route for embeddable chat setup. Reads and updates the tenant embed config through `GET                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | PUT /api/admin/embed/config`, manages allowed origins through `POST | DELETE /api/admin/embed/origins`, and is available from the grouped sidebar under Tools. Full visual parity still pending. |
| `/settings/whatsapp`       | `/settings/whatsapp`       | Tracer only      | Admin/platform-admin-only route matching source RBAC. Reads WhatsApp status, shows QR/waiting/connected states with source-admin setup description, scan guidance, waiting surface, connected active-session surface, and active-session copy, retries status reads, and disconnects an active session. Full visual parity still pending.                                                                                                                                                                                                                                                                                 |
| `/setup/onboard`           | `/setup/onboard`           | Tracer only      | Reads/saves first classroom setup with typed onboarding contracts, source-admin wizard progress indicator and success copy, join-link copy/open/edit actions, and teacher invite success panel. Multi-step wizard visual parity still pending.                                                                                                                                                                                                                                                                                                                                                                            |

## Vertical Slices

Use these as issue candidates. Each slice should be demoable and independently reviewable.

1. **Admin SPA shell and auth session probe**  
   Type: AFK  
   Blocked by: none  
   Status: tracer implemented in `admin-spa/`; still needs review before calling slice done.  
   Acceptance: shared route shell, session query, unauthenticated redirect, accessible loading/error states, tests.

2. **Login and tenant selection**  
   Type: AFK  
   Blocked by: shell/session probe  
   Status: tracer implemented in `admin-spa/`; still needs visual parity and backend multi-tenant verification before calling slice done.  
   Acceptance: password login, Google start link, tenant-required retry, inline errors, keyboard and screen-reader coverage.

3. **Dashboard overview tracer**  
   Type: AFK  
   Blocked by: shell/session probe  
   Status: tracer implemented in `admin-spa/`; student drilldown links and preview fallback are implemented; still needs current-admin visual parity before calling slice done.  
   Acceptance: authenticated `/dashboard`, typed API client, loading/error/empty states, route guard tests.

4. **User and invite management**  
   Type: AFK  
   Blocked by: dashboard overview tracer  
   Status: tracer implemented in `admin-spa/`; copy-link behavior, shared invite workflow, and component split refinement are implemented; visual parity still pending.  
   Acceptance: list users/invites, create invite, reissue invite, optimistic/failed states, RBAC behavior tests.

5. **Student, parent, and class detail surfaces**  
   Type: AFK  
   Blocked by: dashboard overview tracer  
   Status: class-management, parent-summary, and student-detail tracers implemented in `admin-spa/`; class invites, assigned-topic progress, student date formatting, mastery radar snapshot, struggle badges, and activity grid are implemented; full visual parity still pending.  
   Acceptance: typed route params, typed API responses, detail states, navigation back to dashboard context.

6. **AI usage and budget window**  
   Type: AFK  
   Blocked by: dashboard overview tracer  
   Status: AI usage tracer and dedicated `/settings/budget` tracer implemented in `admin-spa/`; typed budget edit flow and role-gated manage behavior are implemented; visual parity still pending.  
   Acceptance: typed usage read, typed budget update, admin/platform-admin RBAC behavior, provider table states, budget validation tests.

7. **Retrieval lab without Next.js proxy coupling**  
   Type: HITL  
   Blocked by: dashboard overview tracer  
   Status: tracer implemented in `admin-spa/`; now calls backend-owned `POST /api/admin/retrieval/search` directly instead of the old Next.js `/api/retrieval-lab/search` proxy path.  
   Acceptance: explicit backend endpoint contract, no hidden Next.js proxy reliance, review of auth/CORS/security boundary.

8. **Data export tools**  
   Type: AFK  
   Blocked by: dashboard overview tracer  
   Status: tracer implemented in `admin-spa/`; visual parity still pending.  
   Acceptance: admin/platform-admin RBAC, tenant-scoped download links for students/conversations/progress, no external URL escape, route guard tests.

9. **Onboarding and public join flows**  
   Type: AFK  
   Blocked by: login and tenant selection  
   Status: public join, invite activation, and onboarding setup tracers implemented in `admin-spa/`; teacher invite success panel, source-admin invite error title, success-panel `Students` join-link label, and optional school-name review label are ported; wizard parity, invite completion QA, and visual parity still pending.  
   Acceptance: public join route, onboarding wizard, invite acceptance, typed form state, unsaved-change protection where needed.

10. **Embed configuration**  
    Type: AFK  
    Blocked by: dashboard overview tracer  
    Status: tracer implemented in `admin-spa/`; basic enablement and allowed-origin management are connected to existing backend admin embed endpoints.  
    Acceptance: admin/platform-admin RBAC, typed embed config read/update, origin add/remove flows, route guard tests.

11. **Cutover readiness review**  
    Type: HITL  
    Blocked by: all route slices  
    Acceptance: parity checklist, performance check, security review, deep-code-review pass, final route ownership decision.

## Quality Rules

- Write tests by vertical slice using TDD: one behavior test, minimal implementation, repeat.
- Tests should verify public user behavior and route/API contracts, not private component implementation.
- Keep TypeScript strict. Avoid `any`; use `unknown` + type guards for untrusted API payloads.
- Prefer TanStack Router inference over manual casts or duplicated route types.
- For OAuth/OIDC login, keep authorization code, PKCE, state, callback handling, and cookies backend-owned. The SPA may initiate `/api/auth/google/start`, but must not store tokens or pass unsafe external return targets.
- Prefer shadcn components before custom markup; use semantic theme tokens, `gap-*`, and accessible component composition.
- Fetch current web guidelines before UI review.
- Run deep-code-review before PR or cutover decisions.
- Treat `pnpm run check` as the local quality gate.

## Open Questions

- Which backend base URL and auth mode should `admin-spa` use in local dev: same-origin reverse proxy, Vite proxy, or explicit env URL?
- Should `admin-spa` keep a public landing route or redirect `/` straight into login/dashboard after cutover?
- Which route should be the first production-equivalent tracer: `/login` or `/dashboard`?
- What is the accepted fallback for features currently relying on `admin/src/app/api/*` Next.js routes?

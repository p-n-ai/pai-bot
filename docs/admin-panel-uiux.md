---
title: "Admin Panel UI/UX Specification"
summary: "Wireframes, shell behavior, responsive rules, and interaction guidance for the P&AI Bot admin panel, including the root redirect and login entry."
read_when:
  - You are redesigning the admin root redirect behavior or login entry
  - You are changing admin shell layout, responsive behavior, or navigation
  - You need the UI/UX reference before building or refactoring admin-facing pages
---

# Admin Panel — UI/UX Specification & Wireframes

> **Status:** Partially implemented (scaffold complete), ongoing development
>
> **Reference:** [docs/admin-panel.md](admin-panel.md) for feature list, [docs/implementation-guide.md](implementation-guide.md) for code templates

This document provides UI/UX specifications, layout wireframes, and interaction patterns for the P&AI Bot admin panel. Use this as the development reference for building and extending admin views.

---

## Table of Contents

- [Design System](#design-system)
- [Shell & Navigation](#shell--navigation)
- [Login Page](#login-page)
- [Teacher Views](#teacher-views)
  - [Teacher Dashboard (Mastery Heatmap)](#teacher-dashboard)
  - [Student Detail Page](#student-detail-page)
  - [Analytics / Metrics Page](#analytics--metrics-page)
  - [AI Usage Page](#ai-usage-page)
- [Parent Views](#parent-views)
  - [Parent Weekly Summary](#parent-weekly-summary)
- [Admin Views](#admin-views)
  - [Class Management](#class-management)
  - [Token Budget Dashboard](#token-budget-dashboard)
  - [User & Invite Management](#user--invite-management)
  - [Data Export](#data-export)
  - [School Onboarding Wizard](#school-onboarding-wizard)
- [Platform Admin Views](#platform-admin-views)
  - [Tenant Management](#tenant-management)
  - [AI Provider Configuration](#ai-provider-configuration)
  - [Global Analytics](#global-analytics)
- [Shared Components](#shared-components)
- [Responsive Behavior](#responsive-behavior)
- [Interaction Patterns](#interaction-patterns)

---

## Design System

### Colors (OKLch via CSS custom properties)

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--primary` | Slate 950 | White | Headings, primary text |
| `--accent` | Sky 700 | Sky 300 | Links, active states, eyebrow labels |
| `--success` | Emerald | Emerald 400/18 | Mastery ≥ 0.75 |
| `--warning` | Amber | Amber 400/18 | Mastery 0.30–0.74 |
| `--danger` | Rose | Rose 400/18 | Mastery < 0.30, errors |
| `--surface` | White/85 | Slate 950/60 | Card backgrounds |
| `--surface-dark` | Slate 950 | Slate 900/90 | Hero aside, dark cards |

### Typography

- **Eyebrow:** `text-xs font-semibold uppercase tracking-[0.22em]` — sky-700 (light) / sky-300 (dark)
- **Page title:** `text-3xl font-semibold tracking-tight`
- **Card title:** `text-xl tracking-tight`
- **Body:** `text-sm leading-6`
- **Label:** `text-xs uppercase tracking-[0.18em]` — slate-500 / slate-400

### Spacing & Radius

- Page sections: `space-y-6`
- Card border radius: `rounded-[28px]`
- Inner containers: `rounded-[24px]` or `rounded-2xl`
- Card shadow (light): `shadow-[0_18px_60px_rgba(15,23,42,0.05)]`
- Card shadow (dark): `shadow-[0_24px_80px_rgba(2,8,23,0.35)]`

### Component Library (shadcn/ui)

Installed: `Button`, `Card`, `Dialog`, `Input`, `Label`, `Select`, `Table`, `Tabs`, `Badge`, `Textarea`

Custom components: `PageHero`, `StatCard`, `StatePanel`, `Metric`, `AppShell`, `AppSidebar`

Theme note:
- Shared overlay primitives should stay on semantic tokens (`bg-popover`, `text-foreground`, `bg-accent`) and must not force a literal `dark` class on popup surfaces.
- Shell surfaces should read clearly in both modes; the desktop sidebar should stay genuinely light in light mode instead of relying on dark-tinted custom overrides.
- Theme changes should interpolate with short color/border/shadow transitions instead of snapping between light and dark.
- School switching should keep the sidebar mounted, prewarm the destination dashboard data, and avoid flashing through empty shell fallback content during the route handoff.
- Keep the shared shadcn sidebar primitive stock. App brand, school-switch, and role-aware nav belong in the app-owned sidebar layer.

---

## Shell & Navigation

The admin shell provides a persistent sidebar (desktop) or collapsible menu (mobile) with role-aware navigation.

### Desktop Layout (≥ 1024px)

```
┌─────────────────────────────────────────────────────────────────────┐
│                        max-w-[1600px] centered                      │
├────────────────────┬────────────────────────────────────────────────┤
│                    │  ┌──────────────────────────────────────────┐  │
│  SIDEBAR (~268px)  │  │  Section Bar (toggle + context)          │  │
│  ┌──────────────┐  │  │  eyebrow · Dashboard / AI Usage          │  │
│  │ Custom mark  │  │  │  "Budget and provider usage"     🌙 👤   │  │
│  │ Classroom Hub│  │  └──────────────────────────────────────────┘  │
│  └──────────────┘  │                                                │
│                    │  ┌──────────────────────────────────────────┐  │
│  WORKSPACE         │  │                                          │  │
│  ┌──────────────┐  │  │           PAGE CONTENT                   │  │
│  │ 📊 Dashboard │  │  │           (max-w-7xl)                    │  │
│  │ 📚 Classes   │  │  │                                          │  │
│  │ 🪙 AI usage  │  │  │                                          │  │
│  └──────────────┘  │  │                                          │  │
│                    │  │                                          │  │
│  PARENT            │  │                                          │  │
│  ┌──────────────┐  │  │                                          │  │
│  │ 👥 Child     │  │  │                                          │  │
│  └──────────────┘  │  │                                          │  │
│                    │  │                                          │  │
│  FOCUS             │  └──────────────────────────────────────────┘  │
│  ┌──────────────┐  │                                                │
│  │ Context hint │  │                                                │
│  └──────────────┘  │                                                │
│                    │                                                │
│  CURRENT SCOPE     │                                                │
│  ┌──────────────┐  │                                                │
│  │ • Dashboard  │  │                                                │
│  │ • Student    │  │                                                │
│  │ • Parent     │  │                                                │
│  │ • AI usage   │  │                                                │
│  └──────────────┘  │                                                │
└────────────────────┴────────────────────────────────────────────────┘
```

### Mobile Layout (< 1024px)

```
┌──────────────────────────────────┐
│ ☰  Section Title          🌙 👤  │  ← sticky top bar
├──────────────────────────────────┤
│ ┌──────────────────────────────┐ │  ← collapsible nav
│ │  Navigation items (compact)  │ │     (slides down on ☰ tap)
│ └──────────────────────────────┘ │
├──────────────────────────────────┤
│                                  │
│       PAGE CONTENT               │
│       (full width, px-4)         │
│                                  │
└──────────────────────────────────┘
```

### Navigation Items by Role

| Role | Visible Nav Items |
|------|-------------------|
| `teacher` | Dashboard, Classes, AI Usage |
| `parent` | Child Summary |
| `admin` | Dashboard, Classes, AI Usage, Budget, Users, Export |
| `platform_admin` | All admin items + Tenants, Providers, Global Analytics |

---

## Login Page

**Routes:** `/`, `/login`
**Access:** All roles (unauthenticated)
**Status:** Implemented

Current shell note:
- Keep the AI Usage hero as a plain header, not a boxed page card or split summary row.

```
┌───────────────────────────────────────────────────┐
│                                            🌙     │
│                                                   │
│           ┌─────────────────────────┐             │
│           │                         │             │
│           │    ✨ P&AI Bot          │             │
│           │    Admin login          │             │
│           │                         │             │
│           │  ┌───────────────────┐  │             │
│           │  │ Email             │  │             │
│           │  └───────────────────┘  │             │
│           │  ┌───────────────────┐  │             │
│           │  │ Password          │  │             │
│           │  └───────────────────┘  │             │
│           │                         │             │
│           │  ┌───────────────────┐  │  ← shown    │
│           │  │ School ▼          │  │    if user  │
│           │  └───────────────────┘  │    has >1   │
│           │                         │    tenant   │
│           │  [ Sign in            ] │             │
│           │                         │             │
│           │  ⚠ Error message        │  ← on fail  │
│           │                         │             │
│           └─────────────────────────┘             │
│                                                   │
└───────────────────────────────────────────────────┘
```

The root route `/` is now redirect-only. It sends signed-in users to their role-safe default route and sends signed-out users to `/login`. `/login` remains the direct auth URL and renders the login layout.

**Interactions:**
- On success → redirect to the role-appropriate workspace
- Teachers/admins/platform admins land on `/dashboard`
- Parents land on `/parents/{id}` (child summary)
- The hero headline should use a short one-time mount reveal with Emil-style ease-out timing, reduced-motion fallback, and only a very small entry blur that resolves immediately.
- If email maps to multiple schools, the form switches into a guided school-pick state:
  - keep email/password visible as locked summaries
  - show a non-destructive info callout
  - use shadcn `Select` for school choice
  - unlock either credential field by clicking the locked field if the user needs to edit it

---

## Teacher Views

### Teacher Dashboard

**Route:** `/dashboard`
**Access:** Teacher, Admin, Platform Admin
**Status:** Implemented

Interaction notes:
- The shell should sit tight to the sticky top bar; avoid extra dead air above the dashboard header.
- The sticky top-bar trigger should align to the shell edge instead of sitting inside the centered content max-width.
- The sticky top bar should stay minimal; remove duplicated section title/eyebrow copy there and leave page identity to the page header/breadcrumb layer.
- Keep a small visual buffer between the sidebar brand block and the first nav item so the workspace mark does not collide with navigation.
- On deeper routes, the shell header should only carry the breadcrumb and should animate in with the same restrained blur/fade language as the dashboard.
- The desktop sidebar header should lead with the custom Classroom Hub mark and workspace label only; the extra `P&AI Bot` badge is removed.
- Page-to-page navigation should use a short opacity/y/blur transition at the content frame, not heavy full-screen wipes.
- Numeric stat cards can animate upward on first paint, but only for the values that benefit from comparison (`Average mastery`, `Coverage`).
- Heatmap topic headers should truncate in-cell and reveal the full topic label on hover/focus with shadcn `Tooltip`.
- When the signed-in email belongs to multiple tenants, the main sidebar brand block should become the school switcher. Keep the school name primary, use the icon tile as the visual anchor, and do not add redundant product sublabels under the school name.
- Style that main brand switcher as a sharper, flatter block rather than a soft rounded pill.
- Keep the school dropdown in normal shadcn select language: semantic surfaces, grouped items, restrained hover states.
- On successful school change, add a brief blur-overlay handoff that reads `Switched to <school>` before the dashboard reload lands.
- Dashboard stat notes can use semantic emphasis: learner alerts in amber/green, average mastery by mastery band, and weakest/strongest topic labels split into risk/success tones.

```
┌──────────────────────────────────────────────────────────────────┐
│  DASHBOARD                                                        │
│  Track who needs support today across the class.                 │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ Learners    │  │ Class grade │  │ Avg mastery │  │ Coverage │ │
│  │    12       │  │     B       │  │    74%      │  │   92%    │ │
│  │ 2 learners  │  │ 74% average │  │ Weakest:    │  │ 66 of 72 │ │
│  │ need attn   │  │ mastery     │  │ Fractions   │  │ tracked  │ │
│  │             │  │             │  │ Strongest:  │  │          │ │
│  │             │  │             │  │ Geometry    │  │          │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └──────────┘ │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │ MASTERY HEATMAP                                              ││
│  │ Students by topic with direct navigation                     ││
│  │                                                              ││
│  │  Student      │ Algebra │Fractions│Geometry │ Stats │ Nudge  ││
│  │  ─────────────┼─────────┼─────────┼─────────┼───────┼─────── ││
│  │  > Ali        │  82%    │  45%    │  91%    │  33%  │ [🔔]   ││
│  │  > Mei Ling   │  67%    │  78%    │  55%    │  89%  │ [🔔]   ││
│  │  > Raj        │  23%    │  61%    │  44%    │  72%  │ [🔔]   ││
│  │               │         │         │         │       │        ││
│  │  Color key:   ■ ≥80%    ■ ≥60%    ■ ≥40%    ■ <40%           ││
│  │               emerald   lime      amber     rose             ││
│  │                                                              ││
│  │  "Nudge sent to Ali on Telegram."                            ││
│  └──────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────┘
```

**Mastery color coding:**
- `≥ 0.80` → emerald (green) badge
- `≥ 0.60` → lime badge
- `≥ 0.40` → amber (yellow) badge
- `< 0.40` → rose (red) badge

**Interactions:**
- Click student name → navigate to `/students/{id}`
- Click Nudge → sends Telegram notification, shows confirmation message
- Click "Open AI usage" → navigate to `/dashboard/ai-usage`
- If a learner has no tracked score for a topic yet, render `--` in a muted style instead of a red `0%` badge

---

### Student Detail Page

**Route:** `/students/[id]`
**Access:** Teacher, Admin, Platform Admin
**Status:** Implemented

```
┌──────────────────────────────────────────────────────────────────┐
│  STUDENT DETAIL                                   Back to dash → │
│  "Ahmad bin Ibrahim"                                             │
│  Form 1 | telegram | tg_12345                      ┌───────────┐ │
│                                                    │ Streak: 7 │ │
│                                                    │ Longest:12│ │
│                                                    │ XP: 1,240 │ │
│                                                    └───────────┘ │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐ ┌───────────────────┐ ┌──────────────────────┐ │
│  │ PROFILE CARD │ │  MASTERY RADAR    │ │  STRUGGLE AREAS      │ │
│  │              │ │                   │ │                      │ │
│  │ Form         │ │      Algebra      │ │  [Fractions 45%]     │ │
│  │  Form 1      │ │       /\          │ │  [Statistics 33%]    │ │
│  │              │ │      /  \         │ │                      │ │
│  │ Channel      │ │ Geo /    \ Frac   │ │  ┌─────────────────┐ │ │
│  │  telegram    │ │     \    /        │ │  │ Algebra    82%  │ │ │
│  │              │ │      \  /         │ │  │ Last: Mar 25    │ │ │
│  │ External ID  │ │    Stats          │ │  │ Next: Mar 28    │ │ │
│  │  tg_12345    │ │                   │ │  ├─────────────────┤ │ │
│  │              │ │  (Recharts radar  │ │  │ Fractions  45%  │ │ │
│  │ Joined       │ │   with sky-blue   │ │  │ Last: Mar 24    │ │ │
│  │  2026-03-01  │ │   fill, 35%       │ │  │ Next: Mar 26    │ │ │
│  │              │ │   opacity)        │ │  └─────────────────┘ │ │
│  └──────────────┘ └───────────────────┘ └──────────────────────┘ │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ ACTIVITY GRID                                               │ │
│  │ Conversation activity over the last 14 days                 │ │
│  │                                                             │ │
│  │  ┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐┌──┐   │ │
│  │  │  ││▓▓││░░││  ││▓▓││██││░░││  ││░░││▓▓││██││░░││  ││▓▓│   │ │
│  │  └──┘└──┘└──┘└──┘└──┘└──┘└──┘└──┘└──┘└──┘└──┘└──┘└──┘└──┘   │ │
│  │  M13  T14  W15  T16  F17  S18  S19  M20  T21  W22  T23  …   │ │
│  │                                                             │ │
│  │  Less active  ○ ░ ▒ ▓ █  More active                        │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ RECENT CONVERSATIONS                                        │ │
│  │                                                             │ │
│  │  ┌────────────────────────────────────────────────────────┐ │ │
│  │  │ STUDENT                              Mar 25, 10:32 AM  │ │ │
│  │  │ "Can you help me with linear equations?"               │ │ │
│  │  └────────────────────────────────────────────────────────┘ │ │
│  │  ┌────────────────────────────────────────────────────────┐ │ │
│  │  │ AI (sky bg)                          Mar 25, 10:32 AM  │ │ │
│  │  │ "Sure! Let's start by understanding what a linear..."  │ │ │
│  │  └────────────────────────────────────────────────────────┘ │ │
│  │  ┌────────────────────────────────────────────────────────┐ │ │
│  │  │ STUDENT                              Mar 25, 10:34 AM  │ │ │
│  │  │ "Oh I see, so x is the unknown?"                       │ │ │
│  │  └────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

**Layout:** 3-column grid on xl (`grid-cols-[0.75fr_1fr_0.9fr]`), stacks on smaller screens.

**Interactions:**
- "Back to dashboard" link at top
- Struggle area badges highlight topics with mastery < 0.30
- Activity grid cells show tooltip on hover with message count
- Conversations show student (gray bg) vs AI (sky bg) messages

---

### Analytics / Metrics Page

**Route:** `/dashboard/metrics`
**Access:** Teacher, Admin, Platform Admin
**Status:** Legacy redirect to `/dashboard/ai-usage`

```
┌───────────────────────────────────────────────────────────────────┐
│  OPERATIONS                                                       │
│  "Daily learning metrics"                                         │
│  Track active learners, retention, nudge response,     ┌───────┐  │
│  and model activity from the Go admin API.             │Latest │  │
│                                                        │DAU    │  │
│                                                        │  42   │  │
│                                                        └───────┘  │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │ Latest DAU │  │ D7 Retain  │  │ Nudge Resp │  │ AI Msgs    │   │
│  │    42      │  │    68%     │  │    73%     │  │   1.2K     │   │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘   │
│                                                                   │
│  ┌────────────────────────────┐  ┌──────────────────────────────┐ │
│  │ DAILY ACTIVE LEARNERS      │  │ RETENTION COHORTS            │ │
│  │ Last 14 days               │  │ D1, D7, D14 by signup cohort │ │
│  │                            │  │                              │ │
│  │ 2026-03-12  ████████  38   │  │ ┌──────────────────────────┐ │ │
│  │ 2026-03-13  █████████ 42   │  │ │ 2026-03-01               │ │ │
│  │ 2026-03-14  ███████   35   │  │ │ Cohort size 15           │ │ │
│  │ 2026-03-15  ████████  39   │  │ │        D1 87%  D7 68%    │ │ │
│  │ 2026-03-16  ██████    28   │  │ │                D14 52%   │ │ │
│  │ ...                        │  │ └──────────────────────────┘ │ │
│  │                            │  │ ┌──────────────────────────┐ │ │
│  │            Back to dash →  │  │ │ 2026-03-08               │ │ │
│  └────────────────────────────┘  │ │ Cohort size 22           │ │ │
│                                  │ │        D1 91%  D7 72%    │ │ │
│                                  │ └──────────────────────────┘ │ │
│                                  └──────────────────────────────┘ │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  (dark card, 3-col grid)                                     │ │
│  │                                                              │ │
│  │  Nudge follow-through   Token activity    A/B comparison     │ │
│  │        73%                  48.2K             Pending        │ │
│  │  12 of 16 nudges led    Prompt + completion   Experiment     │ │
│  │  to a response within   tokens across the     comparison     │ │
│  │  24 hours.              current snapshot.     stays disabled │ │
│  └──────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

---

### AI Usage Page

**Route:** `/dashboard/ai-usage`
**Access:** Teacher, Admin, Platform Admin
**Status:** Implemented

- Keep the top row as a plain header layout, not a boxed page card, and remove the extra provider snapshot panel entirely.

```
┌──────────────────────────────────────────────────────────────────┐
│  AI OPERATIONS                                                   │
│  "Provider usage at a glance"                                    │
│  Track message volume, token load, and the models      ┌───────┐ │
│  currently carrying the teacher workspace.             │Top    │ │
│                                                        │openai │ │
│                                                        │gpt-4o │ │
│                                                        └───────┘ │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  │
│  │ AI Msgs    │  │ Total Tkns │  │ Input Tkns │  │ Providers  │  │
│  │   1.2K     │  │   48.2K    │  │   32.1K    │  │     4      │  │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ PROVIDER BREAKDOWN                             Back to dash │ │
│  │                                                             │ │
│  │  Provider  │ Model     │ Msgs │ Input │Output│ Total │ Load │ │
│  │  ──────────┼───────────┼──────┼───────┼──────┼───────┼───── │ │
│  │  openai    │ gpt-4o    │  620 │ 18.2K │9.1K  │ 27.3K │████  │ │
│  │  anthropic │ sonnet    │  340 │  9.8K │5.2K  │ 15.0K │██    │ │
│  │  google    │ flash     │  180 │  3.1K │1.5K  │  4.6K │█     │ │
│  │  openrouter│ qwen      │   60 │  1.0K │0.3K  │  1.3K │░     │ │
│  │                                                             │ │
│  │  Load bar = share of total tokens (colored by provider)     │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

## Parent Views

### Parent Weekly Summary

**Route:** `/parents/[id]`
**Access:** Parent
**Status:** Implemented

```
┌────────────────────────────────────────────────────────────────────┐
│  PARENT SUPPORT SUMMARY                                            │
│  "Ahmad this week"                                                 │
│  Form 1 · telegram · tg_12345                          ┌───────┐   │
│                                                        │Streak │   │
│                                                        │  7 d  │   │
│                                                        │Best 12│   │
│                                                        │XP 1240│   │
│                                                        └───────┘   │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐    │
│  │⚡Active     │  │💬 Messages │  │🏆 Quizzes  │  │🤝 Needs     │    │
│  │ days       │  │            │  │            │  │  review    │    │
│  │    5       │  │    28      │  │     3      │  │     2      │    │
│  │ Days with  │  │ Student &  │  │ Quiz       │  │ Topics for │    │
│  │ activity   │  │ AI talks   │  │ completions│  │ parent     │    │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘    │
│                                                                    │
│  ┌──────────────────────────┐  ┌─────────────────────────────────┐ │
│  │ MASTERY PROGRESS         │  │ ENCOURAGEMENT SUGGESTION        │ │
│  │                          │  │                                 │ │
│  │  Algebra                 │  │ ┌─────────────────────────────┐ │ │
│  │  ████████████████░░  82% │  │ │ SUGGESTED MESSAGE           │ │ │
│  │  Next review: Mar 28     │  │ │                             │ │ │
│  │                          │  │ │ "Ahmad is building real     │ │ │
│  │  Fractions               │  │ │  confidence in Algebra!"    │ │ │
│  │  █████████░░░░░░░░░  45% │  │ │                             │ │ │
│  │  Next review: Mar 26     │  │ │  Ahmad completed 3 quizzes  │ │ │
│  │                          │  │ │  this week and improved in  │ │ │
│  │  Geometry                │  │ │  Algebra. Try asking about  │ │ │
│  │  ██████████████████  91% │  │ │  what he learned today.     │ │ │
│  │  Next review: Apr 2      │  │ └─────────────────────────────┘ │ │
│  │                          │  │                                 │ │
│  │  Statistics              │  │ ┌─────────────────────────────┐ │ │
│  │  ██████░░░░░░░░░░░░  33% │  │ │ HOME SUPPORT TIP            │ │ │
│  │  Next review: Mar 27     │  │ │ Keep praise specific,       │ │ │
│  │                          │  │ │ focus on one topic, and ask │ │ │
│  └──────────────────────────┘  │ │ for a short follow-up.      │ │ │
│                                │ └─────────────────────────────┘ │ │
│                                └─────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

**Mastery bar colors:**
- `≥ 0.75` → emerald
- `≥ 0.50` → sky
- `≥ 0.30` → amber
- `< 0.30` → rose

**Key design decisions for parents:**
- Simplified topic labels (no technical IDs)
- Encouragement-first tone — focus on positive progress
- Actionable home support tips
- No access to raw conversation logs or system metrics

---

## Admin Views

### Class Management

**Route:** `/dashboard/classes`
**Access:** Admin, Platform Admin
**Status:** Scaffold with mock data (no backend yet)

- Keep the hero row as a plain header layout, not a boxed page card. The dark mock-data summary can stay as the only emphasized surface in that row.
- Do not stack an extra scaffold-warning card under the hero; let the stat row follow immediately.
- The class picker trigger must render the current class name, never the internal class id.

```
┌───────────────────────────────────────────────────────────────────┐
│  TEACHING OPERATIONS                                              │
│  "Class management"                                               │
│  Frontend scaffold with mock data for class            ┌────────┐ │
│  setup, join codes, member roster, and topic           │Mock    │ │
│  assignment.                                           │data    │ │
│  [+ New class]  [✨ Assign topics]                     │only    │ │
│                                                        └────────┘ │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │ Classes    │  │ Members    │  │ Active     │  │ Avg Mastery│   │
│  │    3       │  │    45      │  │    38      │  │    67%     │   │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘   │
│                                                                   │
│  ┌───────────────────┐  ┌───────────────────────────────────────┐ │
│  │ CLASSES           │  │ SELECTED CLASS                        │ │
│  │                   │  │                                       │ │
│  │ [Select ▼]        │  │ ┌───────────────────────────────────┐ │ │
│  │                   │  │ │ Form 1 Algebra A      Join code   │ │ │
│  │ ┌───────────────┐ │  │ │ KSSM Matematik         ABC-123    │ │ │
│  │ │▸ Form 1 Alg A │ │  │ │ Mon, Wed, Fri                     │ │ │
│  │ │  KSSM Form 1  │ │  │ └───────────────────────────────────┘ │ │
│  │ │  Code: ABC-123│ │  │                                       │ │
│  │ └───────────────┘ │  │ ┌─────────────────┐ ┌───────────────┐ │ │
│  │ ┌───────────────┐ │  │ │ MEMBER ROSTER   │ │ ASSIGNED      │ │ │
│  │ │  Form 1 Alg B │ │  │ │                 │ │ TOPICS        │ │ │
│  │ │  KSSM Form 1  │ │  │ │ Name  │Sta│Ch│Ma│ │               │ │ │
│  │ │  Code: DEF-456│ │  │ │ ──────┼───┼──┼──│ │ Linear Eq.    │ │ │
│  │ └───────────────┘ │  │ │ Ali   │Act│TG│82│ │ ████████ 82%  │ │ │
│  │ ┌───────────────┐ │  │ │ Mei   │Act│TG│67│ │ In progress   │ │ │
│  │ │  Form 2 Geo   │ │  │ │ Raj   │Ina│TG│45│ │               │ │ │
│  │ │  KSSM Form 2  │ │  │ │       │   │  │  │ │ Fractions     │ │ │
│  │ │  Code: GHI-789│ │  │ │       │   │  │  │ │ ████░░░░ 45%  │ │ │
│  │ └───────────────┘ │  │ │       │   │  │  │ │ Upcoming      │ │ │
│  │                   │  │ └─────────────────┘ │               │ │ │
│  └───────────────────┘  │                     │[Assign topics]│ │ │
│                         │                     └───────────────┘ │ │
│                         └───────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

**Create Class Dialog:**
```
┌──────────────────────────────────┐
│  Create class                  ✕ │
│                                  │
│  ┌──────────────────────────┐    │
│  │ Class name               │    │
│  │ Form 1 Algebra A         │    │
│  └──────────────────────────┘    │
│                                  │
│  ┌────────────┐ ┌────────────┐   │
│  │ Syllabus ▼ │ │ Cadence    │   │
│  │ KSSM Form 1│ │ Mon,Wed,Fri│   │
│  └────────────┘ └────────────┘   │
│                                  │
│               [Cancel] [Create]  │
└──────────────────────────────────┘
```

---

### Token Budget Dashboard

**Route:** `/settings/budget`
**Access:** Admin, Platform Admin
**Status:** Planned (Week 4, Day 19)

Current implementation note:
- The shipped admin budget surface is token-allowance based, not real-money based.
- Current scope covers token budget windows, used/remaining tokens, daily token trend, per-student average tokens, and admin create/update flows for tenant token budget windows on the AI usage screen.
- The mockup below is a planned future-state dashboard once USD/provider cost attribution exists.

```
┌──────────────────────────────────────────────────────────────────┐
│  BUDGET & COSTS                                                  │
│  "Token budget tracking"                                         │
│  Monitor AI spending, set limits, and configure        ┌───────┐ │
│  fallback strategies when budgets run low.             │Monthly│ │
│                                                        │$42.50 │ │
│                                                        │of $100│ │
│                                                        └───────┘ │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │ Monthly    │  │ Daily Avg  │  │ Per-Student │  │ Budget     │ │
│  │ Spend      │  │            │  │ Avg         │  │ Remaining  │ │
│  │  $42.50    │  │   $1.42    │  │   $0.89     │  │  $57.50    │ │
│  └────────────┘  └────────────┘  └─────────────┘  └────────────┘ │
│                                                                  │
│  ┌──────────────────────────────┐  ┌───────────────────────────┐ │
│  │ MONTHLY COST TREND           │  │ BY PROVIDER               │ │
│  │                              │  │                           │ │
│  │  $3 ┤                        │  │    ┌──────┐               │ │
│  │     │    ╭─╮                 │  │    │OpenAI│ 56%           │ │
│  │  $2 ┤   │ │ ╭─╮              │  │    │██████│               │ │
│  │     │╭─╮│ │ │ │╭─╮           │  │    └──────┘               │ │
│  │  $1 ┤│ ││ │ │ ││ │╭─╮        │  │    ┌────────────┐         │ │
│  │     ││ ││ │ │ ││ ││ │        │  │    │Anthropic   │ 31%     │ │
│  │  $0 ┼┴─┴┴─┴─┴─┴┴─┴┴─┴──      │  │    └────────────┘         │ │
│  │     M  T  W  T  F  S  S      │  │    ┌──────┐               │ │
│  │                              │  │    │Google│ 10%           │ │
│  └──────────────────────────────┘  │    └──────┘               │ │
│                                    │    ┌────┐                 │ │
│                                    │    │Olla│ 3%              │ │
│                                    │    └────┘                 │ │
│                                    └───────────────────────────┘ │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ BUDGET SETTINGS                                             │ │
│  │                                                             │ │
│  │  Monthly limit        Fallback strategy        Alert at     │ │
│  │  ┌────────────┐       ┌─────────────────┐     ┌──────────┐  │ │
│  │  │ $100.00    │       │ Degrade to free▼│     │ 80%      │  │ │
│  │  └────────────┘       └─────────────────┘     └──────────┘  │ │
│  │                                                             │ │
│  │  Fallback options:                                [Save]    │ │
│  │  • Degrade to free models (Ollama)                          │ │
│  │  • Reduce response length                                   │ │
│  │  • Pause non-critical AI (nudges, summaries)                │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

### User & Invite Management

**Route:** `/settings/users`
**Access:** Admin, Platform Admin
**Status:** Planned (Week 5, Day 24)

```
┌──────────────────────────────────────────────────────────────────┐
│  SCHOOL SETTINGS                                                 │
│  "User & invite management"                                      │
│  Invite teachers and parents, manage access.                     │
│                                          [+ Invite user]         │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  │
│  │ Teachers   │  │ Parents    │  │ Pending    │  │ Total      │  │
│  │    4       │  │    12      │  │ Invites    │  │ Users      │  │
│  │            │  │            │  │    3       │  │    16      │  │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ [Active Users]  [Pending Invites]          🔍 Search...     │ │
│  │                                                             │ │
│  │  Name           │ Email              │ Role    │ Status │ ⋯ │ │
│  │  ───────────────┼────────────────────┼─────────┼────────┼── │ │
│  │  Cikgu Aminah   │ aminah@school.my   │ teacher │ Active │ ⋯ │ │
│  │  Cikgu Rizal    │ rizal@school.my    │ teacher │ Active │ ⋯ │ │
│  │  Puan Siti      │ siti@parent.my     │ parent  │ Active │ ⋯ │ │
│  │  En. Kamal      │ kamal@parent.my    │ parent  │ Pending│ ⋯ │ │
│  │                                                             │ │
│  │  ⋯ menu: [Resend invite] [Revoke access]                    │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

**Invite Dialog:**
```
┌──────────────────────────────────┐
│  Invite user                   ✕ │
│                                  │
│  ┌──────────────────────────┐    │
│  │ Email address            │    │
│  │ cikgu.new@school.my      │    │
│  └──────────────────────────┘    │
│                                  │
│  ┌──────────────────────────┐    │
│  │ Role                    ▼│    │
│  │ teacher                  │    │
│  └──────────────────────────┘    │
│                                  │
│  Invite expires in 7 days.       │
│                                  │
│            [Cancel] [Send invite]│
└──────────────────────────────────┘
```

---

### Data Export

**Route:** `/export`
**Access:** Admin, Platform Admin
**Status:** Planned (Week 5, Day 24)

```
┌──────────────────────────────────────────────────────────────────┐
│  ADMINISTRATION                                                  │
│  "Data export"                                                   │
│  Export student data, conversations, and progress reports.       │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                                                             │ │
│  │  ┌────────────────────┐  ┌─────────────────────┐            │ │
│  │  │ 👥 Students        │  │ Format: CSV         │            │ │
│  │  │                    │  │                     │            │ │
│  │  │ Export all student │  │ Includes: name,     │            │ │
│  │  │ profiles with      │  │ form, channel,      │            │ │
│  │  │ current mastery.   │  │ mastery scores,     │            │ │
│  │  │                    │  │ streak, XP          │            │ │
│  │  │  [Download CSV]    │  │                     │            │ │
│  │  └────────────────────┘  └─────────────────────┘            │ │
│  │                                                             │ │
│  │  ┌────────────────────┐  ┌─────────────────────┐            │ │
│  │  │ 💬 Conversations   │  │ Format: JSON        │            │ │
│  │  │                    │  │                     │            │ │
│  │  │ Export full AI-    │  │ Includes: messages, │            │ │
│  │  │ student chat logs  │  │ timestamps, roles,  │            │ │
│  │  │ with metadata.     │  │ session IDs         │            │ │
│  │  │                    │  │                     │            │ │
│  │  │  [Download JSON]   │  │                     │            │ │
│  │  └────────────────────┘  └─────────────────────┘            │ │
│  │                                                             │ │
│  │  ┌────────────────────┐  ┌─────────────────────┐            │ │
│  │  │ 📊 Progress        │  │ Format: CSV         │            │ │
│  │  │                    │  │                     │            │ │
│  │  │ Export per-student │  │ Includes: topic,    │            │ │
│  │  │ mastery progress   │  │ mastery_score,      │            │ │
│  │  │ across all topics. │  │ last_studied,       │            │ │
│  │  │                    │  │ next_review         │            │ │
│  │  │  [Download CSV]    │  │                     │            │ │
│  │  └────────────────────┘  └─────────────────────┘            │ │
│  │                                                             │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

### School Onboarding Wizard

**Route:** `/setup/onboard`
**Access:** Admin (first-time setup)
**Status:** Planned (Week 6, Day 27)

```
Step 1 of 4                    ● ○ ○ ○

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│                    Welcome to P&AI Bot                           │
│                    Let's set up your school                      │
│                                                                  │
│  Step 1: School Details                                          │
│  ┌──────────────────────────────────┐                            │
│  │ School name                      │                            │
│  │ SMK Taman Megah                  │                            │
│  └──────────────────────────────────┘                            │
│                                                                  │
│  Step 2: Curriculum                     ● ● ○ ○                  │
│  ┌──────────────────────────────────┐                            │
│  │ Which forms do you teach?        │                            │
│  │                                  │                            │
│  │  [✓] KSSM Form 1                 │                            │
│  │  [✓] KSSM Form 2                 │                            │
│  │  [ ] KSSM Form 3                 │                            │
│  └──────────────────────────────────┘                            │
│                                                                  │
│  Step 3: Create First Class             ● ● ● ○                  │
│  ┌──────────────────────────────────┐                            │
│  │ Class name: [Form 1A Matematik ] │                            │
│  │ Syllabus:   [KSSM Form 1     ▼]  │                            │
│  │ Cadence:    [Mon, Wed, Fri     ] │                            │
│  └──────────────────────────────────┘                            │
│                                                                  │
│  Step 4: Invite Teachers                ● ● ● ●                  │
│  ┌──────────────────────────────────┐                            │
│  │ Teacher emails (one per line)    │                            │
│  │ ┌──────────────────────────────┐ │                            │
│  │ │ cikgu.aminah@school.my       │ │                            │
│  │ │ cikgu.rizal@school.my        │ │                            │
│  │ └──────────────────────────────┘ │                            │
│  │                  [Send invites]  │                            │
│  └──────────────────────────────────┘                            │
│                                                                  │
│                        [Back]  [Next →]                          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## Platform Admin Views

### Tenant Management

**Route:** `/tenants`
**Access:** Platform Admin only
**Status:** Planned (Week 5+)

```
┌──────────────────────────────────────────────────────────────────┐
│  PLATFORM                                                        │
│  "Tenant management"                                             │
│  Create and manage schools across the platform.                  │
│                                            [+ Create tenant]     │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  │
│  │ Schools    │  │ Teachers   │  │ Students   │  │ Monthly    │  │
│  │    8       │  │    24      │  │    312     │  │ Spend      │  │
│  │            │  │ across all │  │ across all │  │  $340      │  │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  School            │ Teachers │ Students │ Budget  │ Status │ │
│  │  ──────────────────┼──────────┼──────────┼─────────┼────────│ │
│  │  SMK Taman Megah   │    4     │    48    │ $50/mo  │ Active │ │
│  │  SK Bukit Jalil    │    3     │    35    │ $40/mo  │ Active │ │
│  │  SMK Damansara     │    6     │    72    │ $80/mo  │ Active │ │
│  │  SK Sri Petaling   │    2     │    28    │ $30/mo  │ Trial  │ │
│  │                                                             │ │
│  │  Click row → manage tenant settings, users, budget          │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

### AI Provider Configuration

**Route:** `/settings/providers`
**Access:** Platform Admin only
**Status:** Planned (Week 5+)

```
┌───────────────────────────────────────────────────────────────────┐
│  PLATFORM SETTINGS                                                │
│  "AI provider configuration"                                      │
│  Manage API keys, routing rules, and fallback chains.             │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ PROVIDER STATUS                                              │ │
│  │                                                              │ │
│  │  Provider    │ Status  │ Latency │ Errors (24h) │ Config     │ │
│  │  ────────────┼─────────┼─────────┼──────────────┼──────────  │ │
│  │  OpenAI      │ ● Live  │  320ms  │      2       │ [Edit]     │ │
│  │  Anthropic   │ ● Live  │  280ms  │      0       │ [Edit]     │ │
│  │  Google      │ ● Live  │  450ms  │      1       │ [Edit]     │ │
│  │  OpenRouter  │ ● Live  │  380ms  │      0       │ [Edit]     │ │
│  │  Ollama      │ ○ Off   │   —     │      —       │ [Enable]   │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ ROUTING RULES                                                │ │
│  │                                                              │ │
│  │  Task              │ Primary        │ Fallback1   │ Fallback2│ │
│  │  ──────────────────┼────────────────┼─────────────┼──────────│ │
│  │  Teaching          │ Claude Sonnet  │ GPT-4o      │ Ollama   │ │
│  │  Grading           │ DeepSeek V3    │ GPT-4o-mini │ Gemini   │ │
│  │  Question Gen      │ GPT-4o-mini    │ Gemini Flash│ Ollama   │ │
│  │  Nudges            │ Gemini Flash   │ Ollama      │   —      │ │
│  │                                                              │ │
│  │                                          [Edit routing]      │ │
│  └──────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

---

### Global Analytics

**Route:** `/analytics`
**Access:** Platform Admin only
**Status:** Planned (Week 6, Day 29)

```
┌──────────────────────────────────────────────────────────────────┐
│  PLATFORM ANALYTICS                                              │
│  "Global metrics across all schools"                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  │
│  │ Schools    │  │ Total      │  │ Messages   │  │ Monthly    │  │
│  │    8       │  │ Students   │  │ This Month │  │ AI Spend   │  │
│  │            │  │   312      │  │   28.4K    │  │  $340      │  │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  │
│                                                                  │
│  ┌──────────────────────────────┐  ┌───────────────────────────┐ │
│  │ STUDENT GROWTH               │  │ SPEND BY SCHOOL           │ │
│  │                              │  │                           │ │
│  │  350 ┤              ╭──      │  │ SMK Taman    ████████ $50 │ │
│  │  300 ┤         ╭────╯        │  │ SMK Damansara██████████$80│ │
│  │  250 ┤    ╭────╯             │  │ SK Bukit     ██████   $40 │ │
│  │  200 ┤╭───╯                  │  │ SK Sri       ████     $30 │ │
│  │  150 ┤╯                      │  │                           │ │
│  │      W1  W2  W3  W4  W5      │  │                           │ │
│  └──────────────────────────────┘  └───────────────────────────┘ │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ PLATFORM HEALTH (dark card)                                 │ │
│  │                                                             │ │
│  │  Avg Retention (D7)    Nudge Response     Provider Uptime   │ │
│  │       68%                   73%               99.2%         │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

## Shared Components

### PageHero

Used at the top of every page. Contains eyebrow label, title, description, and an optional dark aside card.

- Default mode: boxed card surface
- Plain mode: same grid/layout, but no outer border/background/shadow; use this when the page should read more like a dashboard header than a standalone card

```
┌──────────────────────────────────────────────────────────────────┐
│  EYEBROW LABEL                                                   │
│  "Page Title"                                                    │
│  Description text that explains what this page     ┌───────────┐ │
│  shows and what actions are available.             │ Dark aside│ │
│                                                    │ with key  │ │
│  [Optional action buttons]                         │ metric    │ │
│                                                    └───────────┘ │
│  [Optional child content like breadcrumb links]                  │
└──────────────────────────────────────────────────────────────────┘
```

### StatCard

A compact metric display card with icon, title, value, and note.

```
┌─────────────────┐
│ 📊 Title        │
│    42           │
│ Explanatory note│
└─────────────────┘
```

### StatePanel

Used for loading, empty, and error states within cards.

```
Loading:                     Empty:                      Error:
┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
│                  │   │                  │   │                  │
│  ⏳ Loading...   │   │  📭 No data yet  │   │  ⚠ Unavailable   │
│  Pulling the     │   │  Data will show  │   │  Please try      │
│  latest data...  │   │  once recorded.  │   │  again later.    │
│                  │   │                  │   │                  │
└──────────────────┘   └──────────────────┘   └──────────────────┘
```

### Mastery Badge

Inline colored badge showing mastery percentage.

```
Score ≥ 80%:  [  82%  ] emerald bg
Score ≥ 60%:  [  67%  ] lime bg
Score ≥ 40%:  [  45%  ] amber bg
Score < 40%:  [  23%  ] rose bg
```

---

## Responsive Behavior

| Breakpoint | Behavior |
|------------|----------|
| `< 768px` (mobile) | Single column, sidebar hidden, hamburger menu, full-width cards |
| `768px–1023px` (tablet) | 2-column grids for stat cards, sidebar hidden |
| `≥ 1024px` (desktop) | Sticky neutral sidebar (~16.75rem), tighter top bar, 3-card summary row, side-by-side layouts where content earns it |
| `≥ 1280px` (xl) | Full 3-column layouts for student detail, wider tables |

### Key responsive patterns:
- **Stat cards:** `grid md:grid-cols-2 xl:grid-cols-4`
- **Two-panel layouts:** `grid xl:grid-cols-[1.1fr_0.9fr]`, stacks on mobile
- **Three-panel layouts:** `grid xl:grid-cols-[0.75fr_1fr_0.9fr]`, stacks on mobile
- **Heatmap table:** Horizontal scroll on mobile (`overflow-x-auto`, `min-w-[760px]`)
- **Activity grid:** `grid-cols-7 md:grid-cols-14`

---

## Interaction Patterns

### Nudge Flow
1. Teacher clicks "Nudge" button on heatmap row
2. Button shows "Sending..." (disabled state)
3. API call: `POST /api/admin/students/{id}/nudge`
4. Success: confirmation message appears below heatmap
5. Failure: error message with retry suggestion

### Navigation
- Student name in heatmap → `/students/{id}` (client-side navigation)
- "Back to dashboard" link on detail pages
- Sidebar items highlight active route
- Dashboard root suppresses the duplicate shell breadcrumb/title block; deeper pages still show breadcrumb + page title.

### Data Loading
- Server components use `force-dynamic` for SSR data fetching
- Client components use `useAsyncResource` hook with loading/error states
- Every data section has empty state, loading state, and error state via `StatePanel`

### Theme
- System preference detected on load
- Manual toggle via `ThemeToggle` component (sun/moon icon)
- Preference stored in `localStorage`
- All components support light and dark via Tailwind `dark:` classes

### Session Management
- Go auth endpoints issue `HttpOnly` session cookies for access, refresh, and SSR profile state
- `/login` now shows a first-class `Continue with Google` action above the email/password form; multi-school chooser mode keeps the user focused by suppressing the Google CTA until tenant resolution is complete
- Frontend auth state hydrates from server cookies into the Zustand app store; auth tokens are not persisted in `localStorage`
- The signed-in shell exposes linked-provider state plus a `Link Google` action in the sidebar footer, so different-email Google linking happens only from an authenticated session
- Logout is a button-triggered `POST /api/auth/logout` action, never a `GET` navigation
- Logout clears the server-owned auth cookies and the local school-switch snapshot

---

## File Reference

| Component / Page | File Path |
|------------------|-----------|
| Admin Shell | `admin/src/components/app-shell.tsx` |
| Root Redirect | `admin/src/app/page.tsx` |
| Login Page | `admin/src/app/login/page.tsx` |
| Login Gate Entry | `admin/src/components/login-gate.tsx` |
| Login Gate Components | `admin/src/components/login-gate/` |
| Teacher Dashboard | `admin/src/app/dashboard/page.tsx` |
| Student Detail | `admin/src/app/students/[id]/page.tsx` |
| Metrics Redirect | `admin/src/app/dashboard/metrics/page.tsx` |
| AI Usage Page | `admin/src/app/dashboard/ai-usage/page.tsx` |
| Class Management | `admin/src/app/dashboard/classes/page.tsx` |
| Parent Summary | `admin/src/app/parents/[id]/page.tsx` |
| API Client | `admin/src/lib/api.ts` |
| Server API | `admin/src/lib/server-api.ts` |
| Navigation Logic | `admin/src/lib/navigation.mjs` |
| RBAC Logic | `admin/src/lib/rbac.mjs` |
| Dashboard View Model | `admin/src/lib/dashboard-view.mjs` |
| Async Resource Hook | `admin/src/hooks/use-async-resource.ts` |
| Student View Model | `admin/src/lib/student-view.mjs` |
| Parent View Model | `admin/src/lib/parent-view.mjs` |
| Shared Components | `admin/src/components/` (page-hero, stat-card, state-panel, metric) |
| shadcn/ui Components | `admin/src/components/ui/` |

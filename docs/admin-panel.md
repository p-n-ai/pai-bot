---
title: "Admin Panel Feature Specification"
summary: "Roles, routes, auth flow, and planned capabilities for the P&AI Bot teacher, parent, admin, and platform-admin web panel."
read_when:
  - You are changing admin panel routes, role access, or auth entry flow
  - You are adding teacher, parent, school-admin, or platform-admin features
  - You need the source-of-truth feature map before editing admin UI or admin APIs
---

# Admin / Management Panel — Feature Specification

> **Status:** Partially implemented. Current shipped scope includes the public gate (`/`), direct login (`/login`), teacher dashboard (`/dashboard`), metrics, AI usage, class dashboard, student detail, and parent summary. Remaining sections below stay planned until implemented.
>
> **Stack:** Next.js 16 (App Router) · TypeScript · Refine v5+ · shadcn/ui · Tailwind CSS 4 · TanStack Query v5 · Recharts/Tremor

This document describes all planned features for the P&AI Bot admin panel, organized by user role. The admin panel serves teachers, parents, school administrators, and platform operators — students interact exclusively via chat (Telegram / WhatsApp / WebSocket).

---

## Table of Contents

- [Roles Overview](#roles-overview)
- [Authentication & Authorization](#authentication--authorization)
- [Teacher Dashboard](#teacher-dashboard)
- [Parent View](#parent-view)
- [School Admin](#school-admin)
- [Platform Admin](#platform-admin)
- [Pages & Routes](#pages--routes)
- [API Endpoints](#api-endpoints)
- [Development Timeline](#development-timeline)

---

## Roles Overview

| Role | Access | Primary Interface |
|------|--------|-------------------|
| `student` | No admin panel access | Telegram / WhatsApp / WebSocket chat |
| `teacher` | Teacher dashboard, student details, analytics | Web (admin panel) |
| `parent` | Child progress view, weekly reports | Web + Telegram (automated reports) |
| `admin` | School-wide management + all teacher features | Web (admin panel) |
| `platform_admin` | Multi-tenant management + all admin features | Web (admin panel) |

---
Unauthenticated entry now starts at a public gate page on `/`, with `/login` kept as a direct login entrypoint rendering the same gate experience.

## Authentication & Authorization

### Flow

1. **Enter** — User lands on `/` or `/login`
2. **Login** — Email + password → JWT access token (15 min) + refresh token (7 days, rotated on use)
3. **Resolve tenant when needed** — If the same email belongs to more than one school, the backend returns `tenant_required` and the UI asks the user to choose the school before retrying sign-in
4. **Route guards** — Next.js enforces role-based page access on the frontend
5. **API middleware** — Go backend enforces RBAC on all `/api/admin/*` endpoints

### Database Tables

| Table | Purpose |
|-------|---------|
| `users` | Profile + role (`student`, `teacher`, `parent`, `admin`, `platform_admin`) |
| `auth_identities` | Login credentials (provider: `password`, `telegram`, `whatsapp`, `google`, `microsoft`) |
| `auth_invites` | Invite tokens with email, role, expiry, acceptance tracking |
| `auth_refresh_tokens` | Rotating refresh tokens (hashed), with user agent and IP |

### Security

- JWT access tokens: 15-minute expiry
- Refresh tokens: 7-day expiry, single-use rotation, stored hashed in PostgreSQL
- All admin endpoints require `Authorization: Bearer <token>` header
- RBAC middleware validates user role against endpoint requirements
- Tenant isolation via `tenant_id` on all queries

---

## Teacher Dashboard

Teachers monitor student progress, send nudges, and track class performance.

### Features

| Feature | Description |
|---------|-------------|
| **Mastery Heatmap** | Grid of students (rows) × topics (columns), color-coded by mastery score (0.0–1.0, green at ≥0.75) with drill-down on click |
| **Student Detail Page** | Profile card, mastery radar chart, activity timeline, struggle areas (mastery < 0.3), conversation summaries, streak/XP tracking |
| **Nudge Button** | Send AI-powered proactive learning prompt to an individual student via their chat channel |
| **Class Leaderboard** | Weekly rankings by mastery gain and XP earned |
| **Analytics Dashboard** | DAU, retention snapshots (D7/D14/D30), AI token usage by provider, nudge response rate |
| **Conversation History** | View full AI–student conversation logs with timestamps |
| **Topic Assignment** | Assign specific curriculum topics to individual students or entire class |

### Mastery Heatmap Detail

```
             Algebra   Fractions   Geometry   Statistics
Student A    ■ 0.82    ■ 0.45      ■ 0.91     ■ 0.33
Student B    ■ 0.67    ■ 0.78      ■ 0.55     ■ 0.89
Student C    ■ 0.23    ■ 0.61      ■ 0.44     ■ 0.72
```

- Green (≥ 0.75): mastered
- Yellow (0.30–0.74): developing
- Red (< 0.30): struggling

### Student Detail Page Components

- **Profile Card** — Name, form/grade, join date, chat channel
- **Mastery Radar Chart** — Visual 360° view across all topics
- **Activity Grid** — Recent conversations, quiz attempts, session timestamps
- **Struggle Areas** — Topics with mastery < 0.3 highlighted
- **Conversation Summaries** — AI-generated summaries of recent sessions
- **Streak Display** — Current streak, longest streak, total XP
- **Actions** — Nudge, assign topic, view full conversation history

---

## Parent View

Parents monitor their child's learning progress. Reports are primarily delivered via Telegram; the web dashboard is supplementary.

### Features

| Feature | Description |
|---------|-------------|
| **Child Progress Dashboard** | Summary card (name, current topics, form), weekly stats (messages sent, quizzes completed, time spent), mastery progress bars per topic |
| **Streak & XP Display** | Current streak, milestone celebrations, total XP, weekly breakdown |
| **Weekly Progress Reports** | Automated AI-generated 3-paragraph summary delivered Sunday at 20:00 via Telegram or email |
| **Encouragement Suggestions** | Personalized tips for parents based on child's progress and struggle areas |

### Weekly Report Structure

Delivered automatically every Sunday at 20:00:

1. **What your child studied this week** — Topics covered, sessions completed
2. **Where they're strong** — Mastered topics and recent achievements
3. **How you can help** — Specific suggestions based on struggle areas

---

## School Admin

School administrators manage classes, teachers, parents, and budgets. They have full access to all teacher features plus school-wide management.

### Features

| Feature | Description |
|---------|-------------|
| **Multi-Class Management** | Create/manage multiple classes, assign Form 1/2/3 KSSM syllabi, view consolidated metrics |
| **Teacher Management** | Invite teachers via email, assign to classes, revoke access |
| **Parent Provisioning** | Invite and manage parent accounts, link to students |
| **Class Configuration** | Create classes, generate join codes, assign curriculum |
| **Token Budget Management** | Set tenant-level budget limits, monitor consumption by class/student, configure AI fallback strategies, view cost projections |
| **School Onboarding Wizard** | Interactive setup: school name → curriculum selection → bot setup → class creation → teacher invitation |
| **Data Export** | Export students (CSV), conversations (JSON), progress data (CSV) |
| **All Teacher Features** | Full access to mastery heatmaps, student details, analytics, nudges |

### Token Budget Dashboard

- Monthly cost visualization (bar chart)
- By-provider breakdown (pie chart)
- Daily usage trend (line graph)
- Per-student average cost
- Budget limit configuration with alerts
- Fallback strategy settings (paid → free model degradation)

---

## Platform Admin

Platform administrators manage the entire multi-tenant deployment across all schools.

### Features

| Feature | Description |
|---------|-------------|
| **Multi-Tenant Management** | Create/manage school tenants, configure per-tenant settings, monitor global health |
| **AI Provider Configuration** | Manage API keys, set provider fallback chains, configure model routing rules, monitor provider health/latency |
| **Global Analytics** | Cross-school metrics: total schools, teachers, students, messages, token usage, revenue tracking |
| **System Administration** | User management across all roles and tenants, audit logs, system health monitoring |
| **All Admin Features** | Full access to everything school admins can do, across all tenants |

---

## Pages & Routes

| Page | Route | Accessible By | Status |
|------|-------|---------------|--------|
| Home Gate | `/` | All web roles | Current |
| Login | `/login` | All web roles | Current |
| Teacher Dashboard | `/dashboard` | Teacher, Admin, Platform Admin | Current |
| Student Detail | `/students/[id]` | Teacher, Admin, Platform Admin | Current |
| Analytics | `/dashboard/metrics` | Teacher, Admin, Platform Admin | Current |
| AI Usage | `/dashboard/ai-usage` | Teacher, Admin, Platform Admin | Current |
| Class Management | `/dashboard/classes` | Teacher, Admin, Platform Admin | Current |
| Parent Child View | `/parents/[id]` | Parent | Current |
| Token Budget | `/settings/budget` | Admin, Platform Admin | Planned |
| Data Export | `/export` | Admin, Platform Admin | Planned |
| School Onboarding | `/setup/onboard` | Admin | Planned |

---

## API Endpoints

All endpoints are under `/api/admin/` and require JWT authentication with RBAC validation.

### Class & Progress

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/admin/classes/{classId}/progress` | Teacher, Admin | Mastery heatmap data (students × topics) |
| `GET` | `/api/admin/classes/{classId}/leaderboard` | Teacher, Admin | Weekly rankings by mastery gain and XP |

### Student

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/admin/students/{studentId}/detail` | Teacher, Admin | Student profile + all progress data |
| `GET` | `/api/admin/students/{studentId}/conversations` | Teacher, Admin | Full conversation history |
| `POST` | `/api/admin/students/{studentId}/nudge` | Teacher, Admin | Send proactive nudge to student |

### Analytics & AI

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/admin/ai/usage` | Teacher, Admin, Platform Admin | Token usage by provider, daily trends |
| `GET` | `/api/admin/analytics/report` | Admin, Platform Admin | Comprehensive analytics report |

### Data Export

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/admin/export/students` | Admin | Student data export (CSV) |
| `GET` | `/api/admin/export/conversations` | Admin | Conversation history export (JSON) |
| `GET` | `/api/admin/export/progress` | Admin | Mastery progress export (CSV) |

---

## Development Timeline

| Week | Day | Milestone |
|------|-----|-----------|
| 3 | 14 | Analytics dashboard (`/dashboard/metrics`) — DAU, retention, token usage |
| 4 | 16 | Admin panel scaffold, teacher dashboard, mastery heatmap, student detail, login + route guards |
| 4 | 17 | Admin API endpoints, parent view, form/syllabus selection |
| 4 | 18 | nginx reverse proxy, Docker Compose integration, class management page |
| 4 | 19 | Weekly parent reports scheduler, token budget tracking UI |
| 5 | 24 | Auth hardening (invite acceptance, email/password), data export endpoints |
| 6 | 27 | School onboarding wizard |
| 6 | 29 | Comprehensive analytics API |

See [development-timeline.md](development-timeline.md) for full task breakdown and dependencies.

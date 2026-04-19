---
title: "Admin Dashboard"
sidebar:
  order: 10
description: "Teacher, parent, and school admin web panel."
---

The admin dashboard is a Next.js web application that gives teachers, parents, and administrators visibility into student learning.

## Roles

| Role | Access |
|------|--------|
| **Teacher** | Class dashboard, student details, nudge students, view conversations |
| **Parent** | Child summary, weekly stats, mastery progress |
| **Admin** | All teacher features + class management, user management, data export |
| **Platform Admin** | All admin features + multi-tenant management, AI usage tracking |

## Key Features

### Teacher Dashboard
- **Mastery heatmap** — Students × topics grid, color-coded by mastery level
- **Nudge button** — Send a study reminder to any student directly
- **Student detail** — Profile card, mastery radar chart, activity grid, recent conversations, struggle areas

### Parent View
- **Child summary card** — Weekly stats and mastery progress bars
- **AI-generated encouragement** — Suggestions for how to support learning at home

### Class Management
- Create classes with curriculum/syllabus selection
- Generate join codes for students
- View member roster
- Track topic progress across the class

### School Onboarding Wizard
A guided flow for new schools: curriculum selection → first class setup → bot configuration → invite teachers via email.

### Data Export
- Student list (CSV)
- Conversations (JSON)
- Progress data (CSV)

### AI Usage & Budget
- Token usage dashboard with daily trends
- Per-student average token consumption
- Tenant token budget configuration

### User Management
- Invite teachers, parents, and admins via email
- Pending invite tracking with reissue capability
- Role-based access control

## Authentication

The admin panel supports:
- Email/password login
- Google OIDC sign-in with account linking
- Invite-based onboarding (no self-registration)
- Multi-school switching with password confirmation
- Session cookies (HttpOnly, no localStorage tokens)

## Weekly Parent Reports

An automated scheduler sends parent reports every **Sunday at 8:00 PM** (Malaysia Time) via Telegram. Reports include an AI-generated 3-paragraph summary of the child's week, with a deterministic fallback when AI is unavailable.

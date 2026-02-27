# pai-bot — Daily Development Timeline

> **Repository:** `p-n-ai/pai-bot`
> **Focus:** KSSM Matematik (Form 1, 2, 3) — Algebra first
> **Duration:** 6 weeks (Day 0 → Day 30)

---

## Scope for pai-bot

pai-bot owns the **core platform**: Go backend, AI gateway, Telegram chat adapter, agent engine, progress tracking, motivation features, and Next.js admin panel. Everything a student interacts with flows through this repo.

**Curriculum scope (first 6 months):** KSSM Matematik only — Form 1, Form 2, Form 3. Algebra topics are the primary validation target because they are sequential (clear prerequisites), assessable (right/wrong answers), and high-demand (students struggle most here).

---

## DAY 0 — SETUP (4 hours)

| Task ID | Task | Owner | Time |
|---------|------|-------|------|
| `P-D0-1` | Initialize Go 1.22 project: `cmd/server/main.go`, `internal/{ai,agent,chat,auth,progress,curriculum,platform/{config,database,cache}}`, `migrations/`, `deploy/docker/` | 🤖 Claude Code | 1hr |
| `P-D0-2` | Create `internal/platform/config/config.go` — all env vars with `LEARN_` prefix | 🤖 Claude Code | 30min |
| `P-D0-3` | Create database + cache clients (`pgxpool`, `go-redis`) | 🤖 Claude Code | 1hr |
| `P-D0-4` | Create `docker-compose.yml` (Postgres 17, Dragonfly, app) + multi-stage Dockerfile | 🤖 Claude Code | 30min |
| `P-D0-5` | Create `migrations/001_initial.up.sql` — users, conversations, messages, learning_progress, events tables | 🤖 Claude Code | 30min |
| `P-D0-6` | Create AI gateway: Provider interface + OpenAI implementation + Ollama implementation + router with fallback chain | 🤖 Claude Code | 1hr |
| `P-D0-7` | GitHub Actions CI: build, test, vet, Docker image build | 🤖 Claude Code | 30min |
| `P-D0-8` | Create Telegram bot via @BotFather, save token | 🧑 Human | 15min |

**Exit:** `docker compose up` runs, health check returns 200, AI gateway compiles.

---

## WEEK 1 — THE TALKING SKELETON

### Day 1 (Mon) — Wire Telegram → AI → Student

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W1D1-1` | `internal/chat/gateway.go` — InboundMessage, OutboundMessage, Channel interface, Gateway router | 🤖 |
| `P-W1D1-2` | `internal/chat/telegram.go` — Telegram Bot API adapter with long polling, /start handler, markdown message splitting | 🤖 |
| `P-W1D1-3` | `internal/agent/engine.go` — ProcessMessage: load state → build prompt → call AI → save state → return response | 🤖 |
| `P-W1D1-4` | `internal/curriculum/loader.go` — Load topic YAML + teaching notes markdown from filesystem | 🤖 |
| `P-W1D1-5` | Wire `cmd/server/main.go`: config → db → cache → AI → curriculum → agent → chat → Telegram → start | 🤖 |

**End of Day 1:** Team members can chat with the bot on Telegram. AI responds using curriculum context.

### Day 2 (Tue) — Logging + Quality

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W1D2-1` | Message persistence: save every exchange to `messages` table with conversation_id, model, tokens | 🤖 |
| `P-W1D2-2` | Event logging: `events` table, log session_started, message_sent, ai_response (non-blocking goroutine) | 🤖 |
| `P-W1D2-3` | Anthropic provider: Claude Messages API implementation, update router for task-based routing | 🤖 |
| `P-W1D2-4` | Topic detection: keyword scan → load matching topic's teaching notes into system prompt | 🤖 |
| `P-W1D2-5` | 🧑 Test 30 conversation scenarios, log every bad response, rewrite system prompt v2 | 🧑 Human |

### Day 3 (Wed) — Deploy + First Students

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W1D3-1` | Deploy script: SSH → pull → build → restart → tail logs | 🤖 |
| `P-W1D3-2` | `/start` onboarding: create user record, welcome message, ask what they want to study | 🤖 |
| `P-W1D3-3` | User lookup by telegram_id in chat flow, auto-trigger /start if new | 🤖 |
| `P-W1D3-4` | Error recovery: retry with backoff, provider fallback chain, friendly error messages | 🤖 |
| `P-W1D3-5` | 🧑 Deploy to AWS (t3.medium, Docker Compose), onboard first 3 pilot students (Form 1-3 KSSM) | 🧑 Human |

### Day 4 (Thu) — Iterate on Real Feedback

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W1D4-1` | `scripts/analytics.sh` — DAU, messages/session, AI latency, tokens by model, returning users | 🤖 |
| `P-W1D4-2` | Session management: new conversation after 30min silence, summarize previous session for context | 🤖 |
| `P-W1D4-3` | In-chat rating: after every 5th response ask 1-5 rating, log as event | 🤖 |
| `P-W1D4-4` | 🧑 Read ALL pilot conversations, categorize issues, rewrite system prompt v3 with KSSM-specific instructions | 🧑 Human |
| `P-W1D4-5` | 🧑 Onboard remaining 7 pilot students (total 10 across Form 1-3) | 🧑 Human |

### Day 5 (Fri) — Week 1 Retro

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W1D5-1` | 🧑 Run analytics, compile Week 1 numbers | 🧑 Human |
| `P-W1D5-2` | 🧑 1hr retro: demo, review conversations, identify top 3 problems for Week 2 | 🧑 Team |
| `P-W1D5-3` | 🧑 Call top 3 and bottom 3 students — 10min each | 🧑 Human |

**Week 1 Targets:** 10 students used bot, ≥7 returned, avg session ≥6 messages, system prompt v3+.

---

## WEEK 2 — PROGRESS + ASSESSMENT + 50 STUDENTS

### Day 6 (Mon) — Mastery Tracking

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W2D6-1` | Progress tracking: lightweight AI call after each exchange to assess mastery_delta, update learning_progress | 🤖 |
| `P-W2D6-2` | SM-2 spaced repetition scheduler: calculate next_review based on performance | 🤖 |
| `P-W2D6-3` | `/progress` command: Unicode progress bars per topic, XP, streak, next review | 🤖 |
| `P-W2D6-4` | Progress context in system prompt: "Student mastered X, working on Y, struggles with Z" | 🤖 |
| `P-W2D6-5` | 🧑 Recruit 40 more students from Pandai (KSSM Matematik Form 1-3 users) | 🧑 Human |

### Day 7 (Tue) — Quiz Engine

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W2D7-1` | `/quiz` command: load questions from assessments.yaml, present sequentially, AI-grade free-text answers, hints on wrong answer, summary at end | 🤖 |
| `P-W2D7-2` | Quiz state management: session_mode field (chat/quiz/challenge), route to appropriate handler | 🤖 |
| `P-W2D7-3` | `CompleteJSON` fast-path in AI gateway: structured JSON responses for grading/assessment (use cheapest model) | 🤖 |
| `P-W2D7-4` | 🧑 Review all KSSM Algebra assessments for accuracy and pedagogical quality | 🧑 Human |

### Day 8 (Wed) — Proactive Nudges + Streaks

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W2D8-1` | Agent scheduler: every 5min check due reviews, respect quiet hours (21:00-07:00 MYT), max 3 nudges/day | 🤖 |
| `P-W2D8-2` | Streak tracking: consecutive days, milestones (3/7/14/30), celebrations, bonus XP | 🤖 |
| `P-W2D8-3` | XP system: session XP, quiz XP (by difficulty), mastery XP, streak XP | 🤖 |
| `P-W2D8-4` | 🧑 Check metrics: how many of 50 students active? Message inactive ones directly | 🧑 Human |

### Day 9 (Thu) — Topic Navigation

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W2D9-1` | Topic unlocking: when mastery ≥0.8, check prerequisite graph, notify student of newly unlocked topics | 🤖 |
| `P-W2D9-2` | `/learn [topic]` command: set current topic, load teaching notes, start teaching session | 🤖 |
| `P-W2D9-3` | Daily summary event: scheduler at 22:00 computes per-student daily stats | 🤖 |
| `P-W2D9-4` | 🧑 Interview 5 students: "Did you get a bot message today? How did that feel? Was the quiz helpful?" | 🧑 Human |

### Day 10 (Fri) — Week 2 Retro

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W2D10-1` | 🧑 Compile Week 2 metrics: DAU, Day-7 retention, quiz completion rate, nudge response rate, mastery gain | 🧑 Human |
| `P-W2D10-2` | 🧑 1hr retro. Decision: ready for motivation features or iterate on core teaching? | 🧑 Team |

**Week 2 Targets:** 50 students onboarded, 30+ active, progress tracking + quizzes live, nudge response ≥25%, Day-7 retention ≥35%.

---

## WEEK 3 — MOTIVATION ENGINE

### Day 11 (Mon) — Goals + Challenges

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W3D11-1` | Goal setting: `goals` table, `/goal` command, AI parses natural language goal, store and track | 🤖 |
| `P-W3D11-2` | Goal progress tracking: auto-update after mastery changes, show in /progress and nudges | 🤖 |
| `P-W3D11-3` | Peer challenges: `challenges` table, `/challenge` command, 6-char challenge code, 5-question simultaneous quiz, results with XP | 🤖 |
| `P-W3D11-4` | 🧑 Design battle question sets for all KSSM Algebra topics, standardized per difficulty | 🧑 Human |

### Day 12 (Tue) — Groups + Leaderboards

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W3D12-1` | Class groups: `groups` + `group_members` tables, `/join [code]`, `/create_group [name]` | 🤖 |
| `P-W3D12-2` | Weekly leaderboard: `/leaderboard` shows top 10 by weekly mastery gain within group | 🤖 |
| `P-W3D12-3` | Monday recap: scheduler sends weekly leaderboard summary to all group members | 🤖 |
| `P-W3D12-4` | 🧑 Set up 2 test groups: pilot school group + Pandai beta group | 🧑 Human |

### Day 13 (Wed) — A/B Test + Social Features

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W3D13-1` | A/B test infra: `user_flags` JSONB, alternating motivation_features on/off, flag logged with every event | 🤖 |
| `P-W3D13-2` | Post-challenge learning: review missed questions after battle, +50 XP for completing review | 🤖 |
| `P-W3D13-3` | Milestone celebrations: topic mastered, XP milestones, subject complete — rich Telegram formatting | 🤖 |
| `P-W3D13-4` | 🧑 Partner with 1 Malaysian school: teacher creates class, enrolls 15-20 KSSM students | 🧑 Human |

### Day 14 (Thu) — Analytics Dashboard

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W3D14-1` | Analytics HTML page at `/admin/metrics`: DAU chart, retention cohort, A/B comparison, token costs, nudge rate | 🤖 |
| `P-W3D14-2` | Smart nudge personalization: include streak, goal, struggle area, XP, leaderboard rank in nudge context | 🤖 |
| `P-W3D14-3` | 🧑 Observe school group: are students challenging each other? Call teacher for feedback | 🧑 Human |

### Day 15 (Fri) — Week 3 Retro

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W3D15-1` | 🧑 Week 3 metrics. A/B test early signal? Battle participation? Leaderboard engagement? | 🧑 Human |
| `P-W3D15-2` | 🧑 Retro + go/no-go for admin panel. Any negative signals from competitive features? | 🧑 Team |

**Week 3 Targets:** Goals, challenges, leaderboards live. ≥1 school group active. Challenge participation ≥20%. 80+ students active.

---

## WEEK 4 — ADMIN PANEL + FORM SELECTION

### Day 16 (Mon) — Scaffold Admin Panel

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W4D16-1` | Scaffold `admin/`: Next.js 14 + TypeScript + Tailwind + shadcn/ui + Refine. JWT auth, sidebar layout. | 🤖 |
| `P-W4D16-2` | Teacher dashboard: mastery heatmap grid (students × topics), color-coded, "Nudge" button per student | 🤖 |
| `P-W4D16-3` | Student detail page: profile card, mastery radar chart, activity grid, recent conversations, struggle areas | 🤖 |
| `P-W4D16-4` | 🧑 Brief frontend engineer on 3 dashboard views: teacher, student detail, parent | 🧑 Human |

### Day 17 (Tue) — API Endpoints + Parent View

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W4D17-1` | Admin API: GET classes/{id}/progress, GET students/{id}/detail, GET students/{id}/conversations, GET ai/usage | 🤖 |
| `P-W4D17-2` | Parent view: child summary card, weekly stats, mastery progress bars, AI-generated encouragement suggestion | 🤖 |
| `P-W4D17-3` | Form/syllabus selection: after /start ask "Tingkatan berapa? 1️⃣ Form 1, 2️⃣ Form 2, 3️⃣ Form 3" — load correct curriculum | 🤖 |
| `P-W4D17-4` | 🧑 Show admin panel to 2 pilot teachers via screen share, collect feedback | 🧑 Human |

### Day 18 (Wed) — Deploy Admin + Class Management

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W4D18-1` | Deploy admin: add to docker-compose, nginx reverse proxy (api/* → Go, /* → Next.js) | 🤖 |
| `P-W4D18-2` | Class management page: create class + syllabus, join code, member list, assign topics to class | 🤖 |
| `P-W4D18-3` | 🧑 Test all 3 Forms (F1, F2, F3) with bot — does content switch correctly? | 🧑 Human |

### Day 19 (Thu) — Reports + Budget Tracking

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W4D19-1` | Weekly parent reports: scheduler sends Sunday 20:00, AI-generated 3-paragraph summary via Telegram | 🤖 |
| `P-W4D19-2` | Token budget tracking page: monthly cost, by-provider pie chart, daily trend, per-student avg, budget limits | 🤖 |
| `P-W4D19-3` | 🧑 Test KSSM Form 2 Algebra with 5 Malaysian students. Does teaching quality hold across all 3 forms? | 🧑 Human |

### Day 20 (Fri) — Week 4 Retro

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W4D20-1` | 🧑 Week 4 metrics: Day-14 retention, A/B test results (10 days), teacher dashboard usage | 🧑 Human |
| `P-W4D20-2` | 🧑 Retro. Big decision: ready for open-source prep? | 🧑 Team |

**Week 4 Targets:** Admin panel live. All 3 Forms working. 2+ teachers using dashboard. 100+ students active. Day-14 retention ≥30%.

---

## WEEK 5 — SELF-HOSTABLE + OPEN SOURCE PREP

### Day 21-22 (Mon-Tue) — Cleanup + Documentation

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W5D21-1` | Codebase cleanup: remove hardcoded values, Go doc comments, copyright headers, golangci-lint fixes, .env.example | 🤖 |
| `P-W5D21-2` | Write docs: setup.md, architecture.md, ai-providers.md, curriculum.md, deployment.md | 🤖 |
| `P-W5D21-3` | Comprehensive README.md: hero, quick start (5 steps), features, architecture diagram, providers table, curricula table | 🤖 |
| `P-W5D21-4` | `scripts/setup.sh`: check prereqs → copy .env → prompt for tokens → docker compose up → migrate → seed demo school | 🤖 |
| `P-W5D21-5` | 🧑 Write launch blog post (1500 words) | 🧑 Human |
| `P-W5D21-6` | 🧑 Record 3-min demo video | 🧑 Human |

### Day 23 (Wed) — Self-Host Testing

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W5D23-1` | Multi-tenancy: LEARN_TENANT_MODE single/multi, auto-create default tenant in single mode | 🤖 |
| `P-W5D23-2` | Helm chart: Deployment, StatefulSet (PG, Dragonfly), ConfigMap, Secret, Service, Ingress | 🤖 |
| `P-W5D23-3` | 🧑 Fresh machine test: new AWS instance, follow README only, deploy from scratch, fix every issue | 🧑 Human |

### Day 24-25 (Thu-Fri) — Security + WhatsApp + Data Export

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W5D24-1` | WhatsApp Cloud API adapter (behind LEARN_WHATSAPP_ENABLED flag) | 🤖 |
| `P-W5D24-2` | Data export: GET /export/students (CSV), /export/conversations (JSON), /export/progress (CSV) | 🤖 |
| `P-W5D24-3` | Security audit: auth on all endpoints, tenant isolation middleware, rate limiting, parameterized queries | 🤖 |
| `P-W5D24-4` | 🧑 Final curriculum QA for all KSSM Algebra topics across F1-F3 | 🧑 Human |
| `P-W5D24-5` | 🧑 Gather testimonials from 5 students + 2 teachers | 🧑 Human |

**Week 5 Targets:** Fresh `docker compose up` works in <10min. README + docs complete. Helm chart exists. Security audit done. 150+ students active.

---

## WEEK 6 — LAUNCH + SCALE

### Day 26 (Mon) — LAUNCH DAY

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W6D26-1` | Landing page at `/`: static HTML (Tailwind CDN), "Try on Telegram" + "Self-host" buttons | 🤖 |
| `P-W6D26-2` | K8s health probes: /healthz, /readyz, graceful shutdown on SIGTERM | 🤖 |
| `P-W6D26-3` | 🧑 Publish blog, HN submission, Twitter/LinkedIn/Reddit, 50 personal emails | 🧑 Human |
| `P-W6D26-4` | 🧑 Monitor server + conversations all day | 🧑 Team |

### Day 27 (Tue) — Respond + Onboard

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W6D27-1` | Fix top 5 bugs from launch day | 🤖 |
| `P-W6D27-2` | School onboarding wizard in admin: name → syllabus → bot setup → create class → invite teachers | 🤖 |
| `P-W6D27-3` | 🧑 Respond to every GitHub issue/star/PR. Onboard schools signing up. | 🧑 Team |

### Day 28 (Wed) — i18n + Scale

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W6D28-1` | i18n support: detect Telegram language_code, add to system prompt "Respond in Bahasa Melayu/Chinese/etc." | 🤖 |
| `P-W6D28-2` | 🧑 3-day post-launch metrics. Identify most-requested features. | 🧑 Human |

### Day 29 (Thu) — Analytics API

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W6D29-1` | Comprehensive analytics API: GET /analytics/report — all 6-week metrics in one endpoint | 🤖 |
| `P-W6D29-2` | 🧑 Review community PRs. Plan next 6 weeks. | 🧑 Team |

### Day 30 (Fri) — 6-Week Report

| Task ID | Task | Owner |
|---------|------|-------|
| `P-W6D30-1` | 🧑 Compile 6-week report: metrics, learnings, unit economics, next steps | 🧑 Human |
| `P-W6D30-2` | 🧑 Final retro. Top 3 priorities for next quarter. | 🧑 Team |

**Week 6 Targets:** Public launch. 500+ GitHub stars. 10+ schools. 500-1,000 students. A/B test conclusive.

---

## Task Count Summary

| Week | 🤖 Claude Code | 🧑 Human | Total |
|------|----------------|----------|-------|
| 0 | 8 | 0 | 8 |
| 1 | 17 | 8 | 25 |
| 2 | 15 | 6 | 21 |
| 3 | 11 | 5 | 16 |
| 4 | 11 | 5 | 16 |
| 5 | 9 | 5 | 14 |
| 6 | 6 | 6 | 12 |
| **Total** | **77** | **35** | **112** |

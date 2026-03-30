# pai-bot — Daily Development Timeline

> **Repository:** `p-n-ai/pai-bot`
> **Focus:** KSSM Matematik (Form 1, 2, 3) — Algebra first
> **Duration:** 6 weeks (Day 0 → Day 30)

---

## Scope for pai-bot

pai-bot owns the **core platform**: Go backend, AI gateway, Telegram chat adapter, agent engine, progress tracking, motivation features, and Next.js admin panel. Everything a student interacts with flows through this repo.

**Curriculum scope (first 6 months):** KSSM Matematik only — Form 1, Form 2, Form 3. Algebra topics are the primary validation target because they are sequential (clear prerequisites), assessable (right/wrong answers), and high-demand (students struggle most here).

**TDD note:** All 🤖 tasks include writing tests as part of the task per the TDD workflow in CLAUDE.md. Test-writing is not counted as a separate task — it is embedded in each feature task.

---

## DAY 0 — SETUP (4.5 hours) ✅ COMPLETE

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-D0-1` | Initialize Go 1.22 project: `cmd/server/main.go`, skeleton packages, `Makefile`, `.env.example` | 🤖 | ✅ | |
| `P-D0-2` | Create `internal/platform/config/config.go` — nested config structs, `LEARN_` prefix, `Validate()` | 🤖 | ✅ | |
| `P-D0-3` | Create database + cache clients (`pgxpool`, `go-redis`) with struct wrappers | 🤖 | ✅ | |
| `P-D0-4` | Create `docker-compose.yml` (Postgres 17, Dragonfly, NATS, app, optional Ollama) + multi-stage Dockerfile | 🤖 | ✅ | |
| `P-D0-5` | Create `migrations/20260318100000_initial.sql` — tenants, users, conversations, messages, learning_progress, events + default tenant seed | 🤖 | ✅ | |
| `P-D0-6` | Create AI gateway: `Provider` interface + OpenAI (+ DeepSeek via base URL) + Anthropic + Google Gemini + Ollama + OpenRouter + `MockProvider` + Router with fallback chain + budget tracker | 🤖 | ✅ | |
| `P-D0-7` | GitHub Actions CI: build, test, vet, Docker image build | 🤖 | ✅ | |
| `P-D0-8` | Create Telegram bot via @BotFather, save token | 🧑 | ✅ | |

**What was built (45+ unit tests, all passing):**
- Config: nested structs (`ServerConfig`, `DatabaseConfig`, `AIConfig`, etc.) with `Load()` and `Validate()`
- Database: `DB` struct wrapping `pgxpool.Pool` with `ParseURL`, `New`, `Close`, `HealthCheck`
- Cache: `Cache` struct wrapping `redis.Client` with `ParseURL`, `New`, `Close`, `HealthCheck`
- AI Gateway: `Provider` interface, `Router` (fallback chain), `MockProvider`, `BudgetChecker` interface
- 6 AI providers: OpenAI, DeepSeek (via OpenAI base URL), Anthropic, Google Gemini, Ollama, OpenRouter
- Docker Compose: Postgres 17, Dragonfly, NATS 2.10 (JetStream), app, Ollama (optional `--profile ollama`)
- Dockerfile: Go 1.22 builder → Alpine 3.20 runtime (~25MB)
- HTTP server: `/healthz` + `/readyz` endpoints, graceful SIGTERM shutdown

---

## Developer Onboarding — Getting Ready for Day 1

All Day 0 code is committed. Before starting Day 1 tasks, every engineer must set up their local environment.

### Prerequisites

```bash
# Go 1.22+ (backend)
go version   # Expected: go1.22.x or higher

# Docker + Docker Compose
docker --version && docker compose version

# golangci-lint (linter)
golangci-lint --version   # Expected: ≥1.55
# Install if missing: brew install golangci-lint

# Optional but recommended: Air (hot reload)
go install github.com/air-verse/air@latest
```

### Setup Steps

```bash
# 1. Clone and enter the repo
git clone https://github.com/p-n-ai/pai-bot.git
cd pai-bot

# 2. First-time setup (copies .env.example → .env, downloads Go modules)
just setup

# 3. Edit .env — add your Telegram bot token and at least one AI provider key
#    LEARN_TELEGRAM_BOT_TOKEN=<your-token>
#    LEARN_AI_OPENAI_API_KEY=<key>   (or any other provider)

# 4. Verify all tests pass
just test

# 5. Start infrastructure (Postgres, Dragonfly, NATS)
docker compose up -d postgres dragonfly nats

# 6. Apply database migrations (goose; version-tracked via goose_db_version)
just migrate

# 7. Verify the server runs and health check works
go run ./cmd/server &
curl http://localhost:8080/healthz   # → {"status":"ok"}
kill %1

# 8. Stop infrastructure when done
docker compose down
```

### Day 1 Task Distribution (4 engineers)

Day 1 has 5 tasks. Tasks 1.1–1.4 can be built in parallel; task 1.5 integrates them all.

| Task ID | Task | Assigned To | Dependencies |
|---------|------|-------------|--------------|
| `P-W1D1-1` | Chat Gateway — `internal/chat/gateway.go` (types + interface + router) | Engineer A | None |
| `P-W1D1-2` | Telegram Adapter — `internal/chat/telegram.go` (long polling, /start, markdown splitting) | Engineer A | Uses types from 1.1 |
| `P-W1D1-3` | Agent Engine — `internal/agent/engine.go` (ProcessMessage pipeline) | Engineer B | Uses `ai.Provider` from Day 0 |
| `P-W1D1-4` | Curriculum Loader — `internal/curriculum/loader.go` (load YAML + teaching notes) | Engineer C | None |
| `P-W1D1-5` | Wire main.go — connect all components, start polling | Engineer D (lead) | After 1.1–1.4 merge |

Engineer mapping:
- Engineer A = @djakajaya89

**Refer to `docs/implementation-guide.md` § Day 1 for exact code templates, test specs, and validation commands for each task.**

**Reminder:** Follow TDD — write `_test.go` first → confirm RED → implement → confirm GREEN → run `just test-all`. Never commit until the full suite passes.

---

## WEEK 1 — THE TALKING SKELETON

### Day 1 — Wire Telegram → AI → Student

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W1D1-1` | `internal/chat/gateway.go` — InboundMessage, OutboundMessage, Channel interface, Gateway router | 🤖 | ✅ | |
| `P-W1D1-2` | `internal/chat/telegram.go` — Telegram Bot API adapter with long polling, /start handler, markdown message splitting | 🤖 | ✅ | |
| `P-W1D1-3` | `internal/agent/engine.go` — ProcessMessage: load state → build prompt → call AI → save state → return response | 🤖 | ✅ | |
| `P-W1D1-4` | `internal/curriculum/loader.go` — Load topic YAML + teaching notes markdown from filesystem | 🤖 | ✅ | |
| `P-W1D1-5` | Wire `cmd/server/main.go`: config → db → cache → AI → curriculum → agent → chat → Telegram → start | 🤖 | ✅ | |

**End of Day 1:** Team members can chat with the bot on Telegram. AI responds using curriculum context.

### Day 2 — Logging + Quality

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W1D2-1` | Message persistence: save every exchange to `messages` table with conversation_id, model, tokens | 🤖 | ✅ | |
| `P-W1D2-2` | Event logging: `events` table, log session_started, message_sent, ai_response (non-blocking goroutine) | 🤖 | ✅ | |
| `P-W1D2-3` | Anthropic provider: Claude Messages API implementation, update router for task-based routing | 🤖 | ✅ | |
| `P-W1D2-4` | Topic detection: keyword scan → load matching topic's teaching notes into system prompt | 🤖 | ✅ | |
| `P-W1D2-5` | Structured problem-solving prompt pattern (dual-loop): system prompt v2 instructs AI to follow Understand → Plan → Solve → Verify → Connect steps for every math question. Include curriculum citation in every explanation (e.g., "KSSM Form 1 > Algebra > Linear Equations"). Inspired by [DeepTutor](https://github.com/HKUDS/DeepTutor)'s dual-loop solver | 🤖 | ✅ | |
| `P-W1D2-6` | 🧑 Test 30 conversation scenarios, log every bad response, validate dual-loop solving pattern quality | 🧑 Human | ✅ | |

### Day 3 — Deploy + First Students

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W1D3-1` | Deploy script: SSH → pull → build → restart → tail logs | 🤖 | ✅ | |
| `P-W1D3-2` | `/start` onboarding: create user record, welcome message, ask what they want to study | 🤖 | ✅ | |
| `P-W1D3-3` | User lookup by telegram_id in chat flow, auto-trigger /start if new | 🤖 | ✅ | |
| `P-W1D3-4` | Error recovery: retry with backoff, provider fallback chain, friendly error messages | 🤖 | ✅ | |
| `P-W1D3-5` | 🧑 Deploy to AWS (t3.medium, Docker Compose), onboard first 3 pilot students (Form 1-3 KSSM) | 🧑 Human | ✅ | |

### Day 4 — Iterate on Real Feedback

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W1D4-1` | `scripts/analytics.sh` — DAU, messages/session, AI latency, tokens by model, returning users, and rating summary (count/avg/source) | 🤖 | ✅ | |
| `P-W1D4-2` | Session management (team decision): use rolling compaction + summary for context continuity instead of fixed 30min session split | 🤖 | ✅ | |
| `P-W1D4-3` | In-chat rating: optional flow with Telegram inline stars, delayed callback support, and dedupe per rated assistant message (`messages.id`). Events: `answer_rating_requested/submitted/skipped` with `rated_message_id` + `rating`; configurable interval via `LEARN_RATING_PROMPT_EVERY_REPLIES` | 🤖 | ✅ | |
| `P-W1D4-6` | Additional feature: onboarding language selection (English/BM/中文), `/language` command + Telegram command autocomplete, persist preference in `users.config.preferred_language`, and feature flag `LEARN_DISABLE_MULTI_LANGUAGE` | 🤖 | ✅ | |
| `P-W1D4-4` | 🧑 Read ALL pilot conversations. Evaluate: (a) Is the dual-loop solving pattern (Understand → Plan → Solve → Verify → Connect) producing clear step-by-step explanations? (b) Are curriculum citations accurate? Rewrite system prompt v3 with KSSM-specific instructions and refined solving pattern | 🧑 Human | ✅ | |
| `P-W1D4-5` | 🧑 Onboard remaining 7 pilot students (total 10 across Form 1-3) | 🧑 Human | ✅ | |

#### Additional Tasks (Out of Initial Plan)

Use this section for any completed or in-progress work that was not listed in the original weekly/day plan.  
When adding a new item here, use an `A-WxDy-...` ID and do not backfill it into the original planned task table.

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `A-W1D4-LANG-1` | Language preference persistence decision: keep `preferred_language` in `users.config` (no new table), and continue using `/language` + onboarding selector as write paths | 🤖 | ✅ | |
| `A-W1D4-LANG-2` | Language chooser UX follow-up: interactive `/language` state handling for `lang:*` callbacks and explicit confirmation message after button selection | 🤖 | ✅ | |
| `A-W1D4-AI-LIVE-1` | OpenAI live conversation regression suite: `//go:build integration` test reads 30 scripted YAML conversations (2-10 turns) and validates real `agent.Engine.ProcessMessage` behavior (continuity, language profile, structured solving, concept connection, rating flows). CI explicitly skips these live tests via environment detection. | 🤖 | ✅ | |

### Day 5 — Week 1 Retro

**Implementation note (Day 4 decision):** The team intentionally chose not to enforce a hard 30-minute session boundary. Context continuity is handled via rolling conversation compaction and summary in the agent engine.

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W1D5-1` | 🧑 Run analytics, compile Week 1 numbers | 🧑 Human | ✅ | |
| `P-W1D5-2` | 🧑 1hr retro: demo, review conversations, identify top 3 problems for Week 2 | 🧑 Team | ✅ | |
| `P-W1D5-3` | 🧑 Call top 3 and bottom 3 students — 10min each | 🧑 Human | ✅ | |

**Week 1 Targets:** 10 students used bot, ≥7 returned, avg session ≥6 messages, system prompt v3+. Dual-loop problem-solving pattern and curriculum citations active in all explanations.

**Week 1 Results:** 10 students used bot, 9 returned (90% retention), avg session >10 messages per student. All targets met or exceeded. Proceeding with Week 2 as planned.

#### Additional Tasks (Out of Initial Plan)

Use this section for any completed or in-progress work that was not listed in the original weekly/day plan.  
When adding a new item here, use an `A-WxDy-...` ID and do not backfill it into the original planned task table.

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `A-W1D5-AI-1` | Pulled forward Week 2 AI gateway groundwork for quiz/assessment flows: added `Router.CompleteJSON` with structured-output validation, cheapest-model defaulting, provider fallback on invalid structured responses, and provider-side structured-output support for OpenAI/OpenRouter. This unblocks planned `P-W2D7-3`, but quiz entry/routing and assessment-driven runtime were still pending at that point. | 🤖 | ✅ | |

---

## WEEK 2 — PROGRESS + ASSESSMENT + 50 STUDENTS

### Day 6 — Mastery Tracking

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W2D6-1` | Progress tracking: lightweight AI call after each exchange to assess mastery_delta, update learning_progress | 🤖 | ✅ | |
| `P-W2D6-2` | SM-2 spaced repetition scheduler: calculate next_review based on performance | 🤖 | ✅ | |
| `P-W2D6-3` | `/progress` command: Unicode progress bars per topic, XP, streak, next review | 🤖 | ✅ | |
| `P-W2D6-4` | Adaptive explanation depth in system prompt based on mastery level: mastery <0.3 → simple language, more examples, smaller steps; mastery 0.3–0.6 → standard explanations, introduce formal notation gradually; mastery >0.6 → concise, focus on edge cases and cross-topic connections. Include progress context: "Student mastered X, working on Y, struggles with Z" | 🤖 | ✅ | |
| `P-W2D6-5` | 🧑 Recruit 40 more students from Pandai (KSSM Matematik Form 1-3 users) | 🧑 Human | ✅ | Recruited 45 new students from Telegram group |

### Day 7 — Quiz Engine

**Implementation note (March 16, 2026):** `P-W2D7-3` groundwork was pulled forward on Day 5 via `A-W1D5-AI-1`, and the shipped quiz runtime now covers natural-language/button entry, persisted quiz-state routing, deterministic OSS-backed grading, hint/repeat/continue/stop controls, and clean pause/resume behavior around side conversations or teaching detours. The remaining unshipped Day 7 slice is dynamic question generation plus explicit exam-mimicry prompting.

**Current code note (March 16, 2026):** quiz start already works from natural language without `/quiz`. Current implementation starts immediately on first use with a default mixed intensity instead of blocking on an intensity-selection step, remembers explicit per-user intensity preferences when they exist, reuses the existing progress/XP systems so correct quiz answers award quiz XP and quiz outcomes update topic mastery, and pauses cleanly for side conversation or teaching detours instead of grading every off-topic message as a wrong answer. Quiz content is still loaded from OSS `assessments.yaml`; fallback AI question generation is not yet wired into the live runtime.

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W2D7-1` | Natural-language / button-driven quiz entry: load questions from `assessments.yaml`, present sequentially, deterministic grading for OSS-backed answers, hints on wrong answer, summary at end. Do not require `/quiz` to start. | 🤖 | ✅ | |
| `P-W2D7-2` | Quiz state management: explicit conversation mode in persisted state, route each turn to chat vs quiz handler before tutor AI | 🤖 | ✅ | |
| `P-W2D7-3` | `CompleteJSON` fast-path in AI gateway: structured JSON responses for grading/assessment and dynamic question generation (use cheapest model) | 🤖 | ✅ | |
| `P-W2D7-4` | Exam-style question mimicry: include 2–3 real UASA/SPM exemplar questions per topic in assessments.yaml. AI prompt for dynamic generation says: "Generate a question in the same style, format, and difficulty as these examples: [exemplars]." Inspired by DeepTutor's Mimic Mode | 🤖 | ⬜ | |
| `P-W2D7-5` | 🧑 Review all KSSM Algebra assessments for accuracy and pedagogical quality. **Source 2–3 real UASA/SPM exam questions per Algebra topic** as exemplars for the mimic-mode question generator | 🧑 Human | ✅ | Sourced current PT3/UASA exemplars for Algebra topics and injected into assessments pool. |

### Day 8 — Proactive Nudges + Streaks

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W2D8-1` | Agent scheduler: every 5min check due reviews, respect quiet hours (21:00-07:00 MYT), max 3 nudges/day | 🤖 | ✅ | |
| `P-W2D8-2` | Streak tracking: consecutive days, milestones (3/7/14/30), celebrations, bonus XP | 🤖 | ✅ | |
| `P-W2D8-3` | XP system: session XP, quiz XP (by difficulty), mastery XP, streak XP | 🤖 | ✅ | |
| `P-W2D8-4` | 🧑 Check metrics: how many of 50 students active? Message inactive ones directly | 🧑 Human | ✅ | 42 student active|

### Day 9 — Topic Navigation

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W2D9-1` | Topic unlocking: when mastery ≥0.8, check prerequisite graph, notify student of newly unlocked topics | 🤖 | ✅ | |
| `P-W2D9-2` | `/learn [topic]` command: set current topic, load teaching notes, start teaching session | 🤖 | ✅ | |
| `P-W2D9-3` | Daily summary event: scheduler at 22:00 computes per-student daily stats | 🤖 | ✅ | |
| `P-W2D9-4` | 🧑 Interview 5 students: "Did you get a bot message today? How did that feel? Was the quiz helpful?" | 🧑 Human | ✅ | Interviewed 6 students and they gave suggestion to improve|

### Day 10 — Week 2 Retro

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W2D10-1` | 🧑 Compile Week 2 metrics: DAU, Day-7 retention, quiz completion rate, nudge response rate, mastery gain | 🧑 Human | ✅ | Added the metrics based on Postgres data|
| `P-W2D10-2` | 🧑 1hr retro. Decision: ready for motivation features or iterate on core teaching? | 🧑 Team | ✅ | |

**Retro Feedback:** Core teaching loop for Algebra is locked. Telegram simulation results successfully validated [here](../../oss/docs/qa/P-W2D10-2-telegram-simulation-results.md). Ready for Week 3 Motivation Engine.


**Week 2 Targets:** 50 students onboarded, 30+ active, progress tracking + quizzes live, nudge response ≥25%, Day-7 retention ≥35%, adaptive explanation depth adjusting based on mastery level.

**Current code reality (March 16, 2026):** OSS-backed quiz runtime is live. Dynamic quiz generation and explicit exam-style mimicry are still planned follow-up work.

---

## WEEK 3 — MOTIVATION ENGINE

### Day 11 — Goals + Challenges

Status (2026-03-12): `/goal` shipped with natural-language parsing, pending confirmation for vague goals, multiple active goals, `/goal clear`, and `/progress` goal sync. `/challenge` deferred to the next slice.

Migration note (2026-03-18): the repo now uses `goose` with single-file timestamped SQL migrations tracked in `goose_db_version`. `just migrate` runs `goose up -allow-missing` so older timestamped migrations can still be applied after newer ones in out-of-order branch merges. Existing databases that were previously managed by `golang-migrate` should either recreate the local Postgres volume or be explicitly baselined before switching tools. Do not run both migration tools against the same database long-term.

Status (2026-03-18): current `/challenge` surface now covers invite-code challenge creation/join, human matchmaking, bounded human acceptance, and AI fallback after unmatched queue timeout. Shipped scope: challenge migration groundwork now tracked in timestamped goose files (`20260318102000_challenges`, `20260318102100_challenge_acceptance`, `20260318102200_challenge_matchmaking_question_count`), in-memory + Postgres challenge stores, `/challenge invite <topic>`, `/challenge <code>`, bare `/challenge` search/resume, `/challenge cancel`, `/challenge accept`, queue pairing into `pending_acceptance`, timeout-to-AI fallback, and hardening for search expiry, stale matched-ticket cleanup, one-live-challenge/search exclusivity across invite + queue flows, and idempotent same-user search reopening. AI fallback now also preserves the original stored search input, including question count. Terminal-chat smoke verification now passes after fixing persistent store channel alignment and Postgres invite-join locking. Attempt runtime, settlement, XP, and review remain pending.

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W3D11-1` | Goal setting: `goals` table, `/goal` command, AI parses natural language goal, store and track | 🤖 | ⬜ | |
| `P-W3D11-2` | Goal progress tracking: auto-update after mastery changes, show in /progress and nudges | 🤖 | ⬜ | |
| `P-W3D11-3` | Peer challenges: `challenges` table, `/challenge` command, 6-char challenge code, 5-question simultaneous quiz, results with XP | 🤖 | ⬜ | |
| `P-W3D11-4` | 🧑 Design battle question sets for all KSSM Algebra topics, standardized per difficulty | 🧑 Human | ✅ | 5-5-5 Rule. Injected new pedagogical schema metadata. |

**Implementation note (Late Mar 2026):** All Form 1, Form 2, and Form 3 Algebra assessment pools have been comprehensively standardized for the Battle Engine. 
*   **The "5-5-5" Rule:** Every topic pool now guarantees a minimum baseline of 5 Easy, 5 Medium, and 5 Hard questions.
*   **What's New:** Injected new pedagogical schema metadata not present in earlier versions, including explicit `tp_level` (1-6) routing, `kbat: true/false` flags for higher-order tracking, and `# EXAM: UASA` provenance tags to map AI models directly to national exam formats (OAP, OPB, Subjektif). Upgraded Form 3 pools with brand new TP6 non-routine application problems.

Current note: `P-W3D11-3` is only partially complete. The shipped Day 11 slice is the challenge-creation and matchmaking baseline listed below; the simultaneous quiz runtime, results settlement, XP award, and post-challenge review parts of that planned task still belong to later slices.

#### Additional Tasks (Out of Initial Plan)

Use this section for any completed or in-progress work that was not listed in the original weekly/day plan.  
When adding a new item here, use an `A-WxDy-...` ID and do not backfill it into the original planned task table.

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `A-W3D11-INFRA-1` | Migration workflow moved from `golang-migrate` to `goose`: single-file timestamped SQL migrations, explicit CLI-driven migration step, `just migrate-status`/`just migration-create`, and removal of startup auto-migration with dirty-state auto-force logic. | 🤖 | ✅ | |
| `A-W3D11-CH-1` | Challenge groundwork slice: add `20260318102000_challenges` migration (`challenges`, `challenge_attempts`, `challenge_matchmaking_tickets`), introduce memory/Postgres challenge stores, and ship invite-code `/challenge invite <topic>` create + `/challenge <code>` join command flow with tests. | 🤖 | ✅ | |
| `A-W3D11-CH-2` | Thin human-matchmaking slice: make bare `/challenge` start or resume human matchmaking for a resolved topic, prompt for topic selection when resolution is ambiguous, support `/challenge cancel` to leave search, and pair compatible searchers. | 🤖 | ✅ | |
| `A-W3D11-CH-3` | Challenge hardening slice: enforce matchmaking expiry, expire stale matched tickets before reopening search, enforce one-live-challenge/search exclusivity across invite + queue flows, and make same-user `/challenge` reopen idempotent under store-level races. | 🤖 | ✅ | |
| `A-W3D11-CH-4` | Challenge AI-fallback slice: when a `searching` ticket times out without a human match, claim that ticket exactly once, create a ready `ai_fallback` challenge with `opponent_kind='ai'`, preserve the original stored search topic/syllabus/question_count, and keep invite + human acceptance flows distinct. | 🤖 | ✅ | |

### Day 12 — Groups + Leaderboards

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W3D12-1` | Class groups: `groups` + `group_members` tables, `/join [code]`, `/create_group [name]` | 🤖 | ⬜ | |
| `P-W3D12-2` | Weekly leaderboard: `/leaderboard` shows top 10 by weekly mastery gain within group | 🤖 | ⬜ | |
| `P-W3D12-3` | Monday recap: scheduler sends weekly leaderboard summary to all group members | 🤖 | ⬜ | |
| `P-W3D12-4` | 🧑 Set up 2 test groups: pilot school group + Pandai beta group | 🧑 Human | ✅ | Setup telegram group from existing users|

### Day 13 — A/B Test + Social Features

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W3D13-1` | A/B test infra: `user_flags` JSONB, alternating motivation_features on/off, flag logged with every event | 🤖 | ⬜ | |
| `P-W3D13-2` | Post-challenge learning: review missed questions after battle, +50 XP for completing review | 🤖 | ⬜ | |
| `P-W3D13-3` | Milestone celebrations: topic mastered, XP milestones, subject complete — rich Telegram formatting | 🤖 | ✅ | |
| `P-W3D13-4` | 🧑 Partner with 1 Malaysian school: teacher creates class, enrolls 15-20 KSSM students | 🧑 Human | ✅ | Work with Sekolah Menengah Sains Batu Pahat with Cikgu Akmallina |

### Day 14 — Analytics Dashboard

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W3D14-1` | Metrics page at `/dashboard/metrics`: DAU, retention snapshots, AI usage/token activity, nudge response rate. A/B comparison remains deferred until experiment flags are persisted. | 🤖 | ✅ | |
| `P-W3D14-2` | Smart nudge personalization: include streak, goal, struggle area, and XP in nudge context. Leaderboard rank remains deferred until a stable rank source exists. | 🤖 | ✅ | |
| `P-W3D14-3` | 🧑 Observe school group: are students challenging each other? Call teacher for feedback | 🧑 Human | ✅ | Called Cikgu Akmallina, receive positive feedback|

### Day 15 — Week 3 Retro

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W3D15-1` | 🧑 Week 3 metrics. A/B test early signal? Battle participation? Leaderboard engagement? | 🧑 Human | ⬜ | |
| `P-W3D15-2` | 🧑 Retro + go/no-go for admin panel. Any negative signals from competitive features? | 🧑 Team | ⬜ | |

**Week 3 Targets:** Goals, challenges, leaderboards live. ≥1 school group active. Challenge participation ≥20%. 80+ students active.

---

## WEEK 4 — ADMIN PANEL + FORM SELECTION

### Day 16 — Scaffold Admin Panel

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W4D16-1` | Scaffold `admin/`: Next.js 16 + TypeScript + Tailwind CSS 4 + shadcn/ui + Refine v5. Protect Go admin API with JWT + RBAC, ship the shared public gate on `/` and direct `/login` entrypoint, plus frontend route guards and sidebar layout. | 🤖 | ✅ | |
| `P-W4D16-2` | Teacher dashboard: mastery heatmap grid (students × topics), color-coded, "Nudge" button per student | 🤖 | ✅ | |
| `P-W4D16-3` | Student detail page: profile card, mastery radar chart, activity grid, recent conversations, struggle areas | 🤖 | ✅ | |
| `P-W4D16-4` | 🧑 Brief frontend engineer on 3 dashboard views: teacher, student detail, parent | 🧑 Human | ✅ | |

### Day 17 — API Endpoints + Parent View

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W4D17-1` | Admin API: GET classes/{id}/progress, GET students/{id}/detail, GET students/{id}/conversations, GET ai/usage | 🤖 | ⬜ | |
| `P-W4D17-2` | Parent view: child summary card, weekly stats, mastery progress bars, AI-generated encouragement suggestion | 🤖 | ✅ | |
| `P-W4D17-3` | Form/syllabus selection: after /start ask "Tingkatan berapa? 1️⃣ Form 1, 2️⃣ Form 2, 3️⃣ Form 3" — load correct curriculum | 🤖 | ⬜ | |
| `P-W4D17-4` | 🧑 Show admin panel to 2 pilot teachers via screen share, collect feedback | 🧑 Human | ⬜ | |

### Day 18 — Deploy Admin + Class Management

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W4D18-1` | Deploy admin: add to docker-compose, nginx reverse proxy (api/* → Go, /* → Next.js) | 🤖 | ⬜ | |
| `P-W4D18-2` | Class management page: create class + syllabus, join code, member list, assign topics to class | 🤖 | ⬜ | |
| `P-W4D18-3` | 🧑 Test all 3 Forms (F1, F2, F3) with bot — does content switch correctly? | 🧑 Human | ⬜ | |

### Day 19 — Reports + Budget Tracking

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W4D19-1` | Weekly parent reports: scheduler sends Sunday 20:00, AI-generated 3-paragraph summary via Telegram | 🤖 | ⬜ | |
| `P-W4D19-2` | Token budget tracking page: monthly cost, by-provider pie chart, daily trend, per-student avg, budget limits | 🤖 | ⬜ | |
| `P-W4D19-3` | 🧑 Test KSSM Form 2 Algebra with 5 Malaysian students. Does teaching quality hold across all 3 forms? | 🧑 Human | ⬜ | |

### Day 20 — Week 4 Retro

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W4D20-1` | 🧑 Week 4 metrics: Day-14 retention, A/B test results (10 days), teacher dashboard usage | 🧑 Human | ⬜ | |
| `P-W4D20-2` | 🧑 Retro. Big decision: ready for open-source prep? | 🧑 Team | ⬜ | |

**Week 4 Targets:** Admin panel live. All 3 Forms working. 2+ teachers using dashboard. 100+ students active. Day-14 retention ≥30%.

---

## WEEK 5 — SELF-HOSTABLE + OPEN SOURCE PREP

### Day 21-22 (Mon-Tue) — Cleanup + Documentation

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W5D21-1` | Codebase cleanup: remove hardcoded values, Go doc comments, copyright headers, golangci-lint fixes, .env.example | 🤖 | ⬜ | |
| `P-W5D21-2` | Write docs: setup.md, architecture.md, ai-providers.md, curriculum.md, deployment.md | 🤖 | ⬜ | |
| `P-W5D21-3` | Comprehensive README.md: hero, quick start (5 steps), features, architecture diagram, providers table, curricula table | 🤖 | ⬜ | |
| `P-W5D21-4` | `scripts/setup.sh`: check prereqs → copy .env → prompt for tokens → docker compose up → migrate → seed demo school | 🤖 | ⬜ | |
| `P-W5D21-5` | 🧑 Write launch blog post (1500 words) | 🧑 Human | ⬜ | |
| `P-W5D21-6` | 🧑 Record 3-min demo video | 🧑 Human | ⬜ | |

### Day 23 — Self-Host Testing

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W5D23-1` | Multi-tenancy: LEARN_TENANT_MODE single/multi, auto-create default tenant in single mode | 🤖 | ⬜ | |
| `P-W5D23-2` | Helm chart: Deployment, StatefulSet (PG, Dragonfly), ConfigMap, Secret, Service, Ingress | 🤖 | ⬜ | |
| `P-W5D23-3` | 🧑 Fresh machine test: new AWS instance, follow README only, deploy from scratch, fix every issue | 🧑 Human | ⬜ | |

### Day 24-25 (Thu-Fri) — Security + WhatsApp + Data Export

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W5D24-1` | WhatsApp Cloud API adapter (behind LEARN_WHATSAPP_ENABLED flag) | 🤖 | ⬜ | |
| `P-W5D24-2` | Data export: GET /export/students (CSV), /export/conversations (JSON), /export/progress (CSV) | 🤖 | ⬜ | |
| `P-W5D24-3` | Security audit: auth on all endpoints, tenant isolation middleware, rate limiting, parameterized queries | 🤖 | ⬜ | |
| `P-W5D24-6` | Admin auth hardening: migrations for `auth_identities`, `auth_invites`, `auth_refresh_tokens`; invite acceptance; email/password login; refresh/logout endpoints; Next.js route guards for teacher/parent/admin views | 🤖 | ⬜ | |
| `P-W5D24-4` | 🧑 Final curriculum QA for all KSSM Algebra topics across F1-F3 | 🧑 Human | ⬜ | |
| `P-W5D24-5` | 🧑 Gather testimonials from 5 students + 2 teachers | 🧑 Human | ⬜ | |

**Week 5 Targets:** Fresh `docker compose up` works in <10min. README + docs complete. Helm chart exists. Security audit done. 150+ students active.

---

## WEEK 6 — LAUNCH + SCALE

### Day 26 — LAUNCH DAY

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W6D26-1` | Landing page at `/`: static HTML (Tailwind CDN), "Try on Telegram" + "Self-host" buttons | 🤖 | ⬜ | |
| `P-W6D26-2` | K8s health probes: /healthz, /readyz, graceful shutdown on SIGTERM | 🤖 | ⬜ | |
| `P-W6D26-3` | 🧑 Publish blog, HN submission, Twitter/LinkedIn/Reddit, 50 personal emails | 🧑 Human | ⬜ | |
| `P-W6D26-4` | 🧑 Monitor server + conversations all day | 🧑 Team | ⬜ | |

### Day 27 — Respond + Onboard

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W6D27-1` | Fix top 5 bugs from launch day | 🤖 | ⬜ | |
| `P-W6D27-2` | School onboarding wizard in admin: name → syllabus → bot setup → create class → invite teachers via email invite flow | 🤖 | ⬜ | |
| `P-W6D27-3` | 🧑 Respond to every GitHub issue/star/PR. Onboard schools signing up. | 🧑 Team | ⬜ | |

### Day 28 — i18n + Scale

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W6D28-1` | i18n support: detect Telegram language_code, add to system prompt "Respond in Bahasa Melayu/Chinese/etc." | 🤖 | ⬜ | |
| `P-W6D28-2` | 🧑 3-day post-launch metrics. Identify most-requested features. | 🧑 Human | ⬜ | |

### Day 29 — Analytics API

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W6D29-1` | Comprehensive analytics API: GET /analytics/report — all 6-week metrics in one endpoint | 🤖 | ⬜ | |
| `P-W6D29-2` | 🧑 Review community PRs. Plan next 6 weeks. | 🧑 Team | ⬜ | |

### Day 30 — 6-Week Report

| Task ID | Task | Owner | Status | Remark |
|---------|------|-------|--------|--------|
| `P-W6D30-1` | 🧑 Compile 6-week report: metrics, learnings, unit economics, next steps | 🧑 Human | ⬜ | |
| `P-W6D30-2` | 🧑 Final retro. Top 3 priorities for next quarter. | 🧑 Team | ⬜ | |

**Week 6 Targets:** Public launch. 500+ GitHub stars. 10+ schools. 500-1,000 students. A/B test conclusive.

---

## Task Count Summary

| Week | 🤖 Claude Code | 🧑 Human | Total |
|------|----------------|----------|-------|
| 0 | 8 | 0 | 8 |
| 1 | 18 | 8 | 26 |
| 2 | 17 | 6 | 23 |
| 3 | 11 | 5 | 16 |
| 4 | 11 | 5 | 16 |
| 5 | 9 | 5 | 14 |
| 6 | 6 | 6 | 12 |
| **Total** | **80** | **35** | **115** |

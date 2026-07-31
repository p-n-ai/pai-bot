<p align="center">
  <h1 align="center">P&AI Bot</h1>
  <p align="center">
    <strong>The AI learning companion that keeps students motivated</strong>
  </p>
  <p align="center">
    Open-source · Self-hostable · Model-agnostic · Chat-first
  </p>
  <p align="center">
    <a href="#quick-start">Quick Start</a> ·
    <a href="#features">Features</a> ·
    <a href="#architecture">Architecture</a> ·
    <a href="#deployment">Deployment</a> ·
    <a href="#contributing">Contributing</a>
  </p>
  <p align="center">
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License"></a>
    <a href="https://goreportcard.com/report/github.com/p-n-ai/pai-bot"><img src="https://goreportcard.com/badge/github.com/p-n-ai/pai-bot" alt="Go Report Card"></a>
    <img src="https://img.shields.io/badge/go-%3E%3D1.25-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-Telegram%20%7C%20WhatsApp%20%7C%20Slack%20%7C%20Discord%20%7C%20Teams%20%7C%20Web-green.svg" alt="Platforms">
  </p>
</p>

---

## What is P&AI?

P&AI (Practice & AI) is a proactive AI learning agent that teaches students through chat. It doesn't wait for students to ask — it initiates study sessions, tracks mastery, schedules reviews, and keeps students motivated with battles, streaks, leaderboards, and purpose-driven progress.

Built on [Pandai](https://pandai.org)'s years of proven engagement mechanics that have made learning fun for millions of students across Southeast Asia.

**Content is commodity. Motivation is the moat.**

### What makes P&AI different?

| Feature | ChatGPT / Claude | Khan Academy | **P&AI** |
|---------|------------------|--------------|----------|
| Answers questions | ✅ | ✅ | ✅ |
| Follows a curriculum | ❌ | ✅ | ✅ |
| Structured step-by-step solving | ❌ | Partial | ✅ |
| Adapts explanation to mastery level | ❌ | ❌ | ✅ |
| Cites curriculum source in responses | ❌ | ❌ | ✅ |
| Tracks mastery per topic | ❌ | ✅ | ✅ |
| Generates exam-style practice questions | ❌ | ❌ | ✅ |
| Proactive — initiates sessions | ❌ | ❌ | ✅ |
| Spaced repetition scheduling | ❌ | ❌ | ✅ |
| Battles, streaks, leaderboards | ❌ | ❌ | ✅ |
| Model-agnostic (swap AI providers) | ❌ | ❌ | ✅ |
| Self-hostable | ❌ | ❌ | ✅ |
| Works on $50 phones via Telegram | ❌ | ❌ | ✅ |
| Open source | ❌ | ❌ | ✅ |

---

## Quick Start

Get P&AI running in under 5 minutes.

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) (v2+)
- Credentials for an external chat adapter if you want to receive messages outside local development. Production requires at least one of Telegram, WhatsApp, Slack, Discord, or Microsoft Teams.
- At least one AI provider: an API key, free self-hosted Ollama, or a local Codex CLI login

### 1. Clone and configure

```bash
git clone --recurse-submodules https://github.com/p-n-ai/pai-bot.git
cd pai-bot
git submodule update --init --recursive
cp .env.example .env
```

Edit `.env` with your credentials:

```env
# Optional external chat adapter
LEARN_TELEGRAM_BOT_TOKEN=your-telegram-bot-token

# Preferred: connect your ChatGPT subscription from Admin → AI settings
LEARN_AI_DEFAULT_PROVIDER=codex
LEARN_AI_CODEX_ENABLED=true
LEARN_AI_CODEX_MODEL=gpt-5.4

# Or use OpenRouter
LEARN_AI_DEFAULT_PROVIDER=openrouter
LEARN_AI_OPENROUTER_API_KEY=sk-or-v1-...
LEARN_AI_OPENROUTER_MODEL=qwen/qwen3-max

# Or use free self-hosted AI (no API key needed)
LEARN_AI_DEFAULT_PROVIDER=ollama
LEARN_AI_OLLAMA_ENABLED=true
LEARN_AI_OLLAMA_MODEL=qwen3
```

Telegram works as soon as `LEARN_TELEGRAM_BOT_TOKEN` is set. For another adapter, configure its complete credential set:

| Adapter | Required configuration | Ingress |
|---------|------------------------|---------|
| Slack | `LEARN_SLACK_ENABLED=true`, `LEARN_SLACK_BOT_TOKEN`, `LEARN_SLACK_SIGNING_SECRET` | Events API: `POST /webhook/slack` |
| Discord | `LEARN_DISCORD_ENABLED=true`, `LEARN_DISCORD_BOT_TOKEN`, `LEARN_DISCORD_PUBLIC_KEY`, `LEARN_DISCORD_APPLICATION_ID` | Gateway messages and interactions at `POST /webhook/discord`; enable the Message Content intent |
| Microsoft Teams | `LEARN_TEAMS_ENABLED=true`, `LEARN_TEAMS_APP_ID`, `LEARN_TEAMS_APP_PASSWORD` | Bot Framework activities: `POST /webhook/teams` |
| WhatsApp | `LEARN_WHATSAPP_ENABLED=true` plus the selected backend's credentials | Cloud API: `POST /webhook/whatsapp`; `meow` uses the local WhatsApp session |

Webhook deployments need a public HTTPS base URL that forwards these paths to the Go server. Configure each Slack, Discord, or Teams credential set together; partial sets fail startup validation even when the adapter's enable flag is false.

### 2. Start everything

```bash
docker compose up -d
```

This starts: PostgreSQL, Dragonfly (cache), the Go server, and the admin panel.

If you want demo rows in PostgreSQL for local testing, run:

```bash
just seed
```

If the app is running in Docker, seed through the app container instead:

```bash
just seed-docker
```

When the backend is running in Docker, make sure `.env` uses Compose service names such as `postgres` and `dragonfly` instead of `localhost`. The `app` service already reads `.env`, so school admins can choose AI provider and default model purely with Docker env vars. For Ollama, Compose overrides `LEARN_AI_OLLAMA_URL` inside the app container to `http://ollama:11434`.

The Codex provider is for a self-hosted server with trusted administrators. Sign in to Admin with the existing email/password flow, choose **Connect Codex** in AI settings, open the displayed OpenAI verification page, and enter the one-time device code. The browser can be on a different machine from the server: Codex completes authorization through the device flow and stores the resulting login in an isolated server-owned Codex home. PaiBot uses the structured `codex app-server` account API and never reads the operator's personal `~/.codex` login.

```env
LEARN_AI_CODEX_ENABLED=true
# Optional; defaults to the OS config directory under pai-bot/codex.
LEARN_AI_CODEX_HOME=/var/lib/pai-bot/codex
```

The production app image includes a pinned Codex CLI. Docker Compose mounts `LEARN_AI_CODEX_HOME` from the persistent `codex-data` volume, so the login survives container replacement. When running the backend outside that image, install the Codex CLI on the backend `PATH` and give the backend user write access to `LEARN_AI_CODEX_HOME`.

### 3. Pull a free AI model (optional)

Only do this if you are using Ollama as your AI provider. This downloads the model weights into the Ollama container so the app has something local to run.

Warning: this can be a large download and may take time depending on the model and network speed.

```bash
docker compose exec ollama ollama pull qwen3
```

After that, set `LEARN_AI_OLLAMA_ENABLED=true` and optionally `LEARN_AI_OLLAMA_MODEL=qwen3` in `.env`.

### 4. Chat with your bot

Message the bot through the adapter you configured. On Telegram, find your bot and send `/start`; webhook adapters begin receiving messages after their provider points at the matching `/webhook/...` endpoint.

### 5. Access the admin panel

Open `http://localhost:3000` to access the admin panel. Current scaffolding keeps the shell publicly reachable in local development, but the planned production model is invite-based account activation followed by email + password login for teacher, parent, and admin roles.

### 6. Browse the API docs

Open `http://localhost:8080/docs` for the Scalar-powered API reference. The raw OpenAPI document is served at `http://localhost:8080/openapi.json` and is generated directly from explicit Go request/response schemas.

---

## Features

### 🎓 For Students

- **AI Tutor in Chat** — Learn through Telegram, WhatsApp, Slack, Discord, Microsoft Teams, or the web. The AI uses Socratic method, scaffolding, and growth mindset pedagogy.
- **Step-by-Step Problem Solving** — Every math question is answered with a structured approach: Understand → Plan → Solve → Verify → Connect. Teaches students *how to think*, not just the answer.
- **Adaptive Explanations** — The AI adjusts explanation complexity based on your mastery level. Beginners get simpler language and more examples; proficient students get concise explanations with harder challenges.
- **Curriculum-Cited Responses** — Every explanation references the exact curriculum source (e.g., "KSSM Form 1 > Algebra > Linear Equations"), so students can find it in their textbook.
- **Proactive Study Sessions** — The agent initiates conversations when it's time to review. Spaced repetition ensures long-term retention.
- **Progress Tracking** — See mastery per topic, XP earned, streak length, and progress toward personal goals.
- **Quizzes & Assessments** — Take quizzes in chat with deterministic grading for OSS-backed free-text answers, hints, and detailed feedback.
- **Exam-Style Practice** — Current quiz content comes from OSS KSSM assessment sets reviewed against Algebra topics. Dynamic AI-generated UASA/SPM-style mimicry is planned, not yet live.
- **Peer Challenges** — Battle classmates on the same set of questions. Learn together, compete for fun.
- **Goals & Streaks** — Set a learning goal ("Master algebra by April") and track daily streaks.

### 👩‍🏫 For Teachers

- **Class Dashboard** — Mastery heatmap showing every student's progress across every topic at a glance.
- **Student Detail View** — Deep dive into any student: mastery radar, activity timeline, struggle areas, conversation summaries.
- **Nudge Students** — One-click to have the AI send a personalized study prompt to a specific student.
- **Assign Topics** — Direct the AI to teach a specific topic to a student or entire class. *(Planned)*
- **Weekly Leaderboards** — Motivate the class with weekly rankings by mastery gain.

### 👪 For Parents

- **Child Progress View** — Simple dashboard showing weekly activity, topics studied, streak, and XP.
- **Weekly Reports** — Automated weekly summary: what your child worked on, what they did well, and how you can help.

### 🏫 For Schools & Governments

- **Self-Hostable** — Run on your own infrastructure. Full data sovereignty. No student data leaves your network.
- **Multi-Tenant** — One deployment serves multiple schools, each with isolated data.
- **Token Budget Management** — Allocate AI credits per school, per class, or per student. Automatic fallback to free self-hosted models when budget runs low.
- **Data Export** — Export all student data as CSV/JSON at any time. Your data, your control.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  Chat Channels                                      │
│  Telegram · WhatsApp · Slack · Discord · Teams · Web │
│                     │                               │
│                     ▼                               │
│              Chat Gateway                           │
│                     │                               │
│                     ▼                               │
│  ┌──────────────────────────────────────────┐       │
│  │           Agent Engine                   │       │
│  │  ┌──────────────┐  ┌──────────────────┐  │       │
│  │  │ Conversation │  │ Proactive        │  │       │
│  │  │ State Machine│  │ Scheduler        │  │       │
│  │  └──────────────┘  └──────────────────┘  │       │
│  │  ┌─────────────┐  ┌──────────────────┐   │       │
│  │  │ Progress    │  │ Pedagogical      │   │       │
│  │  │ Tracker     │  │ Prompts          │   │       │
│  │  └─────────────┘  └──────────────────┘   │       │
│  └──────────────────────┬───────────────────┘       │
│                         │                           │
│            ┌────────────┼──────────────┐            │
│            ▼            ▼              ▼            │
│  ┌───────────────┐ ┌──────────┐ ┌───────────────┐   │
│  │  AI Gateway   │ │Curriculum│ │  PostgreSQL   │   │
│  │  ┌─────────┐  │ │ Service  │ │  + Dragonfly  │   │
│  │  │OpenAI   │  │ │  (OSS)   │ │               │   │
│  │  │Anthropic│  │ └──────────┘ └───────────────┘   │
│  │  │Codex    │  │                                  │
│  │  │Ollama   │  │                                  │
│  │  │Custom   │  │                                  │
│  │  └─────────┘  │                                  │
│  └───────────────┘                                  │
│                                                     │
│  ┌──────────────────────────────────────────┐       │
│  │  Admin Panel (Next.js + TanStack Query)  │       │
│  │  Teacher Dashboard · Parent View · Admin │       │
│  └──────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────┘
```

### Tech Stack

| Component | Technology | Why |
|-----------|-----------|-----|
| **Backend** | Go 1.25+ | Concurrent chat and scheduling services ship as a single binary. |
| **Database** | PostgreSQL 17 | Standard, portable. Every cloud has managed Postgres. |
| **Cache** | Dragonfly | Redis-compatible, multi-threaded, 80% less memory. |
| **AI Providers** | OpenAI, Anthropic, Codex, Ollama, OpenRouter | Provider-agnostic gateway. Swap models without code changes. |
| **Chat** | Telegram, WhatsApp, Slack, Discord, Microsoft Teams, WebSocket | One gateway preserves provider identity, thread routes, delivery IDs, and adapter lifecycle. |
| **Admin Panel** | Next.js 16, TypeScript, TanStack Query, shadcn/ui | Teacher dashboards, parent views, school admin. |
| **Curriculum** | [Open School Syllabus](https://github.com/p-n-ai/oss) | Structured YAML curriculum consumed by the agent. |
| **Deployment** | Docker Compose or Helm | Single server ($20/mo) to national deployment (millions of students). |

### Chat Gateway

Every inbound message keeps its provider identity, delivery ID, and provider-qualified thread route. The gateway serializes work per destination, deduplicates webhook retries, and persists the route so replies, scheduled nudges, and focused-page messages return to the learner's originating channel and thread.

### Current Admin Auth

- Teachers, parents, school admins, and platform admins sign in through `/login`; visiting `/` routes signed-in users to their workspace and everyone else to the login flow.
- Ongoing login uses `email + password`; if the same email belongs to multiple schools, the UI asks the user to pick the correct school before finishing sign-in.
- The Go backend owns admin auth with one server session cookie (`pai_session`); bearer JWT parsing remains only as a compatibility lane.
- Students access P&AI through the configured chat adapters; a student web login is not part of the current baseline.

### Project Structure

```
pai-bot/
├── cmd/
│   ├── server/main.go               # Application entrypoint
│   ├── seed/main.go                 # Demo data seeder
│   ├── terminal-chat/main.go        # Terminal chat for testing
│   └── terminal-nudge/main.go       # Terminal nudge for testing
├── internal/
│   ├── ai/                          # AI Gateway
│   │   ├── gateway.go               # Provider interface + types
│   │   ├── router.go                # Model routing + fallback chain + circuit breaker
│   │   ├── budget.go                # Token budget tracking (in-memory)
│   │   ├── provider_openai.go       # OpenAI + DeepSeek (compatible API)
│   │   ├── provider_anthropic.go    # Anthropic Claude
│   │   ├── provider_google.go       # Google Gemini
│   │   ├── provider_ollama.go       # Self-hosted (Llama, Qwen, etc.)
│   │   ├── provider_openrouter_llm_adapter.go   # 100+ models via OpenRouter
│   │   └── provider_codex.go         # Codex through a server-owned app-server login
│   ├── agent/                       # Agent Engine
│   │   ├── engine.go                # Conversation state machine
│   │   ├── scheduler.go             # Proactive nudge scheduler
│   │   ├── quiz.go                  # Quiz engine + assessment
│   │   ├── challenge.go             # Peer battle system
│   │   ├── challenge_runtime.go     # Challenge gameplay + settlement
│   │   └── goals.go                 # Goal tracking
│   ├── chat/                        # Chat Gateway
│   │   ├── gateway.go               # Unified message routing
│   │   ├── telegram.go              # Telegram Bot API adapter
│   │   ├── whatsapp.go              # WhatsApp Cloud API adapter
│   │   ├── whatsapp_meow.go         # WhatsApp linked-device adapter
│   │   ├── slack.go                 # Slack Events API + Web API adapter
│   │   ├── discord.go               # Discord interactions + REST adapter
│   │   ├── discord_gateway.go       # Discord Gateway lifecycle
│   │   ├── teams.go                 # Microsoft Teams Bot Framework adapter
│   │   └── websocket.go             # WebSocket adapter
│   ├── curriculum/                   # Curriculum Service
│   │   ├── loader.go                # Reads YAML from OSS repository
│   │   ├── types.go                 # Go structs matching OSS schema
│   │   └── prerequisites.go         # Prerequisite graph
│   ├── progress/                    # Progress Tracker
│   │   ├── tracker.go               # Mastery scoring
│   │   ├── spaced_rep.go            # SM-2 algorithm
│   │   ├── streaks.go               # Streak tracking
│   │   └── xp.go                    # XP system
│   ├── auth/                        # Authentication
│   │   ├── jwt.go                   # Token generation + validation
│   │   ├── middleware.go            # Role-based access control
│   │   ├── google_oidc.go           # Google OIDC sign-in
│   │   └── service.go              # Login, invites, sessions
│   ├── adminapi/                    # Admin REST API
│   ├── retrieval/                   # BM25 knowledge retrieval
│   ├── tenant/                      # Multi-tenancy bootstrap
│   ├── i18n/                        # Internationalization (BM/EN/ZH)
│   └── platform/                    # Shared infrastructure
│       ├── config/                  # Environment configuration
│       ├── database/                # PostgreSQL connection (pgx)
│       ├── cache/                   # Dragonfly client (go-redis)
│       ├── mailer/                  # SMTP email delivery
│       └── seed/                    # Demo data seeding
├── admin-spa/                       # Vite/TanStack admin SPA
│   └── src/
│       ├── routes/                  # TanStack Router routes
│       └── components/              # Admin UI components (shadcn/ui)
├── migrations/                      # SQL migration files (goose)
├── deploy/
│   ├── docker/
│   │   ├── Dockerfile               # Multi-stage Go build
│   │   └── Dockerfile.admin         # Multi-stage admin SPA build
│   ├── caddy/                       # Reverse proxy config
│   └── nginx/                       # Alternative reverse proxy
├── scripts/
│   ├── setup.sh                     # First-time setup wizard
│   ├── deploy-remote.sh             # Production deployment
│   └── analytics.sh                 # Quick metrics from CLI
├── docker-compose.yml               # Local development
├── docker-compose.prod.yml          # Production single-server
├── justfile                         # Preferred task runner
├── .env.example                     # All configuration documented
└── .github/workflows/               # CI pipeline
```

---

## AI Providers

P&AI is not locked to any AI model. Configure one or more providers:

| Provider | Models | Cost | Setup |
|----------|--------|------|-------|
| **OpenAI** | GPT-5.4, GPT-5.4 mini | Paid API | Set `LEARN_AI_OPENAI_API_KEY` and optionally `LEARN_AI_OPENAI_MODEL` |
| **Anthropic** | Claude Sonnet 4.6, Claude Haiku 4.5 | Paid API | Set `LEARN_AI_ANTHROPIC_API_KEY` and optionally `LEARN_AI_ANTHROPIC_MODEL` |
| **DeepSeek** | DeepSeek-V3.2 (`deepseek-chat`), DeepSeek-R1 (`deepseek-reasoner`) | Paid API (very cheap) | Set `LEARN_AI_DEEPSEEK_API_KEY` and optionally `LEARN_AI_DEEPSEEK_MODEL` |
| **Google Gemini** | Gemini 3 Flash Preview, Gemini 3 Pro Preview | Paid API | Set `LEARN_AI_GOOGLE_API_KEY` and optionally `LEARN_AI_GOOGLE_MODEL` |
| **Ollama** | Qwen3, Qwen3 14B, Qwen3 30B | Free (self-hosted) | Set `LEARN_AI_OLLAMA_ENABLED=true` and optionally `LEARN_AI_OLLAMA_MODEL` |
| **OpenRouter** | Qwen3 Max, Qwen3 Coder Next, 100+ others | Varies | Set `LEARN_AI_OPENROUTER_API_KEY` and optionally `LEARN_AI_OPENROUTER_MODEL` |
| **Codex** | GPT-5.6 Sol | ChatGPT subscription | Install the Codex CLI, then connect from Admin → AI settings |

DeepSeek uses the OpenAI-compatible API format — no extra code, just a different API key and base URL. Its official `deepseek-chat` alias already tracks the current DeepSeek-V3.2 non-thinking model. Gemini 3 models are the latest family, but note that the current Flash/Pro API IDs are preview models. Preview Gemini IDs can have different or tighter rate limits, so for steadier production behavior it is usually safer to set `LEARN_AI_GOOGLE_MODEL` to a non-preview model name such as `gemini-2.5-flash`. Qwen, Kimi, and other models are accessible via OpenRouter or self-hosted via Ollama.

To prefer one provider first, set `LEARN_AI_DEFAULT_PROVIDER` to one of: `openai`, `anthropic`, `deepseek`, `google`, `ollama`, `openrouter`, `codex`.

The AI Gateway automatically routes by task type:

- **Teaching** (complex explanations) → Best available model (Claude Sonnet, GPT-4o, Gemini Pro)
- **Grading** (quick JSON responses) → Cheapest model (DeepSeek V3, GPT-4o-mini, Gemini Flash)
- **Question generation** (dynamic quiz/exam-style) → Cheapest model (DeepSeek V3, GPT-4o-mini)
- **Nudges** (short messages) → Any available model
- **Fallback** → Self-hosted Ollama (always free)

When paid API budgets run low, the system automatically degrades to cheaper models, then to self-hosted. **No student is ever cut off from learning.**

---

## Supported Curricula

P&AI reads structured curriculum data from the [Open School Syllabus (OSS)](https://github.com/p-n-ai/oss) repository.

Currently supported:

| Curriculum | Subjects | Status |
|-----------|----------|--------|
| Malaysia KSSM Form 1 | Matematik (Algebra) | Live |
| Malaysia KSSM Form 2 | Matematik (Algebra) | Live |
| Malaysia KSSM Form 3 | Matematik (Algebra) | Live |
| Cambridge IGCSE 0580 | Mathematics | Planned |
| *More coming — contributions welcome!* | | |

Adding a new curriculum doesn't require code changes — just add YAML files to the OSS repository and P&AI picks them up automatically. See the [OSS contribution guide](https://github.com/p-n-ai/oss/blob/main/CONTRIBUTING.md).

### Updating OSS Submodule Pointer

To sync to the latest `oss` commit from its default branch:

```bash
git submodule update --remote oss
```

Note: the submodule wiring is currently a bootstrap stub for upcoming curriculum sync work, not a finalized end-user feature.

---

## Deployment

Repository maintainers use the controlled
[nightly candidate and stable release flow](docs/releases.md). A successful
merge creates a candidate after `main` CI passes; it does not deploy
production.

### Option 1: Single Server (Docker Compose)

For a single school or small deployment. Runs on any VPS with 2GB+ RAM.

```bash
git clone https://github.com/p-n-ai/pai-bot.git
cd pai-bot
./scripts/setup.sh     # Interactive setup wizard
# Set private PAI_AUTH_SECRET, PAI_CONFIG_ENCRYPTION_KEY, and
# PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD in .env, then:
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

**Cost:** ~$20/month on any VPS provider. Supports 100-500 students.

### Option 2: Kubernetes (Helm)

For districts, states, or national deployments. A Helm chart is available at `deploy/helm/pai/`.

Create a private, untracked `values.production.yaml`; don't pass secrets through
command-line `--set` arguments.

```yaml
secrets:
  authSecret: "<private auth secret>"
  configEncryptionKey: "<independent 32+ character key>"
  bootstrapAdminPassword: "<private bootstrap password>"
  telegramBotToken: "<telegram token>"
  ai:
    openaiApiKey: "<provider key>"
ingress:
  enabled: true
  host: learn.yourschool.edu.my
```

```bash
chmod 600 values.production.yaml
helm install pai deploy/helm/pai -f values.production.yaml
```

**Scales:** Horizontally to millions of students. Each school gets a namespace with isolated data.

### Option 3: Cloud-Agnostic

P&AI is designed to run on any cloud without lock-in:

| Component | AWS | GCP | Azure | Self-Hosted |
|-----------|-----|-----|-------|-------------|
| Compute | EKS | GKE | AKS | Any K8s |
| Database | RDS PostgreSQL | Cloud SQL | Azure DB | PostgreSQL |
| Cache | (self-hosted Dragonfly) | (self-hosted) | (self-hosted) | Dragonfly/Redis |
| Storage | S3 | GCS | Blob | MinIO |

---

## Configuration Reference

Environment variables define the immutable deployment baseline. Platform
administrators can override the supported AI runtime fields in Admin; resetting
an override returns control to the current environment value without rewriting
the process environment or `.env` files. Core app variables use `LEARN_`; auth
variables use `PAI_AUTH_` only. See [`.env.example`](.env.example) for the
complete baseline list.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LEARN_TELEGRAM_BOT_TOKEN` | No | — | Telegram bot token from @BotFather |
| `LEARN_SLACK_ENABLED` | No | `false` | Enable the signed Slack Events API webhook at `/webhook/slack` |
| `LEARN_SLACK_BOT_TOKEN` | With Slack | — | Slack bot token used for replies |
| `LEARN_SLACK_SIGNING_SECRET` | With Slack | — | Slack request-signing secret |
| `LEARN_DISCORD_ENABLED` | No | `false` | Enable Discord Gateway ingress, signed interactions at `/webhook/discord`, and REST replies |
| `LEARN_DISCORD_BOT_TOKEN` | With Discord | — | Discord bot token; enable the privileged Message Content intent in the Developer Portal |
| `LEARN_DISCORD_PUBLIC_KEY` | With Discord | — | Discord application's Ed25519 interaction public key |
| `LEARN_DISCORD_APPLICATION_ID` | With Discord | — | Discord application ID |
| `LEARN_TEAMS_ENABLED` | No | `false` | Enable authenticated Bot Framework activities at `/webhook/teams` and Connector API replies |
| `LEARN_TEAMS_APP_ID` | With Teams | — | Microsoft Bot Framework application ID |
| `LEARN_TEAMS_APP_PASSWORD` | With Teams | — | Microsoft Bot Framework client secret |
| `LEARN_TEAMS_APP_TENANT_ID` | No | `botframework.com` | Tenant used for Teams client-credential tokens |
| `LEARN_WHATSAPP_ENABLED` | No | `false` | Enable the selected WhatsApp backend |
| `LEARN_WHATSAPP_BACKEND` | No | `meow` | WhatsApp backend: `meow` for a linked device or `cloudapi` for Meta's Cloud API |
| `LEARN_WHATSAPP_ACCESS_TOKEN` | With Cloud API | — | Meta Cloud API access token |
| `LEARN_WHATSAPP_PHONE_ID` | With Cloud API | — | Meta WhatsApp phone number ID |
| `LEARN_WHATSAPP_VERIFY_TOKEN` | With Cloud API | — | Token used to verify the Cloud API webhook |
| `LEARN_DATABASE_URL` | No | `postgres://pai:pai@localhost:5432/pai` | PostgreSQL connection string |
| `LEARN_CACHE_URL` | No | `redis://localhost:6379` | Dragonfly/Redis connection |
| `LEARN_AI_DEFAULT_PROVIDER` | No | — | Preferred provider to try first (`openai`, `anthropic`, `deepseek`, `google`, `ollama`, `openrouter`, `codex`) |
| `LEARN_AI_OPENAI_API_KEY` | No | — | OpenAI API key |
| `LEARN_AI_OPENAI_MODEL` | No | — | Default OpenAI model when request model is not set |
| `LEARN_AI_ANTHROPIC_API_KEY` | No | — | Anthropic API key |
| `LEARN_AI_ANTHROPIC_MODEL` | No | — | Default Anthropic model when request model is not set |
| `LEARN_AI_DEEPSEEK_API_KEY` | No | — | DeepSeek API key (OpenAI-compatible) |
| `LEARN_AI_DEEPSEEK_MODEL` | No | — | Default DeepSeek model when request model is not set |
| `LEARN_AI_GOOGLE_API_KEY` | No | — | Google Gemini API key |
| `LEARN_AI_GOOGLE_MODEL` | No | — | Default Google model when request model is not set |
| `LEARN_AI_OPENROUTER_API_KEY` | No | — | OpenRouter API key (100+ models) |
| `LEARN_AI_OPENROUTER_MODEL` | No | — | Default OpenRouter model when request model is not set |
| `LEARN_AI_CODEX_ENABLED` | No | `false` | Enable authenticated Admin device authorization for the local Codex provider |
| `LEARN_AI_CODEX_HOME` | No | OS config directory under `pai-bot/codex` | Isolated server-owned Codex credential directory |
| `LEARN_AI_CODEX_ACCESS_TOKEN` | No | — | Legacy manual Codex access token; Admin device authorization is preferred |
| `LEARN_AI_CODEX_REFRESH_TOKEN` | No | — | Legacy manual Codex refresh token |
| `LEARN_AI_CODEX_ACCOUNT_ID` | No | — | Legacy manual ChatGPT account ID |
| `LEARN_AI_CODEX_MODEL` | No | `gpt-5.4` | Default Codex model |
| `LEARN_AI_OLLAMA_ENABLED` | No | `false` | Enable self-hosted Ollama |
| `LEARN_AI_OLLAMA_URL` | No | `http://localhost:11434` | Ollama server URL |
| `LEARN_AI_OLLAMA_MODEL` | No | — | Default Ollama model when request model is not set |
| `LEARN_AI_PERSONALIZED_NUDGES_ENABLED` | No | `true` | Let AI personalize proactive nudge messages; falls back to template text on failure |
| `PAI_AUTH_SECRET` | No | `change-me-in-production` | Root auth secret used for JWTs and focused-page capabilities; use a private value in production |
| `PAI_CONFIG_ENCRYPTION_KEY` | No | — | Active independent high-entropy root used for versioned encryption of API keys stored through admin settings |
| `PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS` | No | `[]` | JSON array of up to eight retired encryption roots retained while stored credentials and backups are migrated. `PAI_AUTH_SECRET` remains a read-only candidate only for legacy unversioned ciphertext |
| `LEARN_SERVER_PORT` | No | `8080` | HTTP server port |
| `LEARN_TENANT_MODE` | No | `single` | `single` or `multi` tenant mode |

Outside development mode, configure at least one external chat adapter and one AI provider.
See [Runtime AI settings](docs/operations/runtime-ai-settings.md) for the
env/override reconciliation contract and the safe encryption-key rotation
sequence.

Generate independent auth and configuration-encryption roots in a new private
file with:

```bash
go run ./cmd/init-secrets -out /path/to/pai-bot-secrets.env
```

Before a production rollout, run `go run
./cmd/validate-production-secrets`. The production Compose overlay and Helm
chart also block startup or rendering when required secrets are missing, use
public defaults, reuse a root, or contain an invalid retired-key list.

### First-Boot Tenant Flow

The first setup behavior depends on `LEARN_TENANT_MODE`:

- `single` mode:
  - On server startup, P&AI ensures tenant slug `default` exists.
  - If it is missing (for example, on a fresh DB), startup will auto-create/upsert it.
  - Tenant-bound runtime services use this single tenant context.
- `multi` mode:
  - Startup does not auto-create tenants.
  - Tenant lifecycle is managed explicitly (seed/admin/invite workflows).

Recommended first setup sequence:

1. Run migrations.
2. Set `LEARN_TENANT_MODE` in `.env`.
3. Start the server.

---

## Development

### Prerequisites

- Go 1.25+
- Node.js 20+ (for admin panel)
- Docker and Docker Compose

### Local Development

Note: `just` recipes are supported on macOS/Linux for now. On Windows, prefer Docker/WSL2 instead of `just go` / `just admin-spa`.

The shortest path uses [ONCE](https://github.com/basecamp/once):

```bash
./scripts/once-dev.sh
# or: just once-dev
```

This installs a pinned, checksum-verified ONCE binary under `~/.local/bin` when
needed, builds the all-in-one development image, and runs P&AI at
`http://localhost` with persistent storage, migrations, and demo data. Later
runs rebuild and update the existing application. Use `just once-stop` to stop
it or `just once-remove` to remove it. Set `PAI_ONCE_SEED_DEMO=false` for a clean
database. Non-empty `LEARN_*` and `PAI_AUTH_*` values from `.env` are passed to
the ONCE application, while its internal database, cache, and HTTP settings stay
isolated inside the image.

The lower-level workflow remains available when you need to run services
independently:

`just go` / `just admin-spa` require `LEARN_DATABASE_URL` to be present in `.env`; the local bootstrap path no longer falls back to an implicit default DSN or shell override.

```bash
# Start infrastructure (Postgres, Dragonfly, Ollama)
docker compose up -d postgres dragonfly ollama

# Run database migrations
just migrate

# Check the current migration version
just migrate-version

# Seed demo data (optional)
just seed

# Or, if the app itself is running in Docker
just seed-docker

# Start the Go server (turnkey deps + local Postgres/Dragonfly; auto-seeds only for the default local dev DB target)
just go

# Start the admin SPA and try to boot the Go server if needed
# If backend boot fails, Vite still starts; check /tmp/pai-go.log for backend errors
# Ctrl-C also stops the backend started by this command
just admin-spa

# Start the backend + admin SPA through one wrapper script
# Good target for Codex app "play" / run-button flows
./scripts/run-dev.sh

# Stop docker services plus local backend/frontend/Agentation listeners
just stop

# Stop only the local run-dev.sh processes
./scripts/stop-dev.sh
```

### Running Tests

```bash
go test ./...     # Run all Go tests
go test -tags=integration ./...   # Run integration tests
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"${GOLANGCI_LINT_VERSION:-v2.4.0}" run ./...
cd admin-spa && pnpm test  # Admin SPA unit + component tests
cd admin-spa && pnpm test:e2e  # Public login + protected-route browser tests
cd admin-spa && E2E_ADMIN_EMAIL=admin@example.com E2E_ADMIN_PASSWORD=demo-password pnpm test:e2e:backend  # Real backend auth/session browser test
just admin-spa-check       # Typecheck, lint, format, tests, and build
just test-all              # Go lint/tests + full admin SPA check
```

OpenAI live conversation integration suite:

- Fixture source: `internal/agent/testdata/openai_live_conversations.yaml` (30 scripted conversations, 2-10 turns each)
- Test harness: `internal/agent/engine_openai_integration_test.go` (`//go:build integration`)
- Required env for live run: `LEARN_AI_OPENAI_API_KEY`
- Optional env:
  - `LEARN_AI_LIVE_TIMEOUT_SECONDS` (default `45`)
  - `LEARN_AI_LIVE_MAX_CASES` (default `30`)
- CI behavior: the live OpenAI suite is explicitly skipped in CI (`CI`/`GITHUB_ACTIONS` detection) to avoid external paid API calls in pipeline runs.

Terminal chat workflow:

```bash
just chat-terminal
# Codex-only local chat, with device login when needed:
just chat-codex
# or:
docker compose run --rm --entrypoint /pai-terminal-chat app --user-id demo-user --lang en
# for an ephemeral local-only session:
docker compose run --rm --entrypoint /pai-terminal-chat app --memory
```

The terminal chat uses the same `agent.Engine` and AI router as the app. By default it uses PostgreSQL-backed conversation state for production parity; pass `--memory` for an ephemeral local-only session. `just chat-codex` pins the session to the managed Codex provider with no fallback. It uses PaiBot's isolated Codex home, not personal `~/.codex` credentials, and prints the existing OpenAI device URL and one-time code if that home is not connected yet.

`just chat-codex` also enables the interactive test controls `/status`, `/new`, `/reload`, `/character <id>`, and `/interrupt <message>`. Normal messages entered during a reply are queued FIFO. It watches `.codex/chat-codex-candidate/candidate.yaml`, but applies a valid candidate only after `/reload` starts a fresh in-memory session. `/status` identifies the active candidate only by `sha256:<hash>`.

Terminal nudge workflow:

```bash
just nudge-terminal USER_ID=demo-user
# or:
docker compose run --rm --entrypoint /pai-terminal-nudge app --user-id demo-user
```

The terminal nudge command triggers the real scheduler path for one user and prints any generated nudge message to stdout.

### Useful Commands

```bash
just setup        # First-time setup
just install-deps # Install Go modules + frontend packages
just install-local-runtime  # Install missing Postgres client tools via Homebrew when available
just start        # Start all services via Docker Compose
just stop         # Stop all services
just logs         # Tail application logs
just migrate      # Run database migrations
just migrate-status   # Show applied/pending goose migrations
just migrate-version  # Show current goose migration version
just migrate-down # Roll back the most recent migration
just migration-create add_parent_invites  # Create a new timestamped SQL migration
just seed         # Seed demo tenant/users/messages/progress/events
just seed-docker  # Seed through the running app container
just analytics    # Print quick metrics from the database
just analytics-xlsx   # Export a styled Excel workbook to output/spreadsheet/
just analytics-example  # Generate a sample Excel workbook without a database
just ollama-pull  # Download a free AI model for Ollama
just chat-terminal  # Open a local terminal chat session
just nudge-terminal USER_ID=<user-id>  # Trigger a due-review nudge for one user
```

Excel export notes:

- `scripts/analytics.sh --xlsx output/spreadsheet/pai-analytics.xlsx` keeps the terminal report and also writes a formatted workbook.
- `scripts/analytics.sh --example-xlsx output/spreadsheet/pai-analytics-example.xlsx` creates a sample workbook for layout review without touching the database.
- The analytics script loads `.env` automatically when present. When `PAI_DB_URL` is unset, it falls back to `LEARN_DATABASE_URL` from the app environment before using Docker Compose PostgreSQL.
- The workbook builder now runs through `go run ./cmd/analyticsxlsx`, so there is no separate Python runtime or spreadsheet dependency to install.

Migration notes:

- The repo now uses `goose` with single-file timestamped SQL migrations and `goose_db_version` tracking.
- `just migrate` runs `goose up -allow-missing` so older timestamped migrations can still be applied after newer ones in out-of-order branch merges.
- Existing databases from the pre-goose migration flow should be recreated in local dev or explicitly baselined before switching tools. Do not run both migration systems against the same database long-term.

### Historical Rating Analytics

- The in-chat rating path is retired. Existing `answer_rating_submitted` events remain queryable for historical analytics with:
  - `data.rating` (1-5)
  - `data.rated_message_id` (assistant `messages.id` being rated)
  - `data.source`, `data.channel`, `data.delayed_submit`

---

## Contributing

We welcome contributions! P&AI is built by a community that believes every student deserves a patient, always-available learning companion.

### Ways to contribute

- **Code** — Pick up a [good first issue](https://github.com/p-n-ai/pai-bot/labels/good%20first%20issue) or propose a feature.
- **Curriculum** — Add topics, teaching notes, or assessments to [OSS](https://github.com/p-n-ai/oss).
- **Translation** — Help translate the bot's messages and admin panel.
- **Testing** — Try P&AI with real students and report what works and what doesn't.
- **Documentation** — Improve guides, fix typos, add examples.

### Development workflow

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`go test ./...` and `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"${GOLANGCI_LINT_VERSION:-v2.4.0}" run ./...`)
5. Commit (`git commit -m 'Add amazing feature'`)
6. Push to your branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

---

## Related Repositories

| Repository | Description |
|-----------|-------------|
| [p-n-ai/oss](https://github.com/p-n-ai/oss) | Open School Syllabus — structured curriculum data for any learning platform. |
| [p-n-ai/oss-bot](https://github.com/p-n-ai/oss-bot) | GitHub bot + CLI tools for contributing to Open School Syllabus |

---

## License

P&AI Bot is licensed under the [Apache License 2.0](LICENSE).

You are free to use, modify, and distribute this software. Self-host it for your school, fork it for your country, build a business on it. The only requirement is that you include the license notice.

**Our promise:** The core learning platform will always be free and open source. We will never sell student data or show ads.

---

## Acknowledgments

P&AI is built on the shoulders of [Pandai](https://pandai.org) — years of making learning fun for millions of students through gamification, battles, leaderboards, and purpose-driven progress. The secret sauce has always been motivation, not content.

---

<p align="center">
  <strong>Every student deserves a patient, always-available learning companion.</strong>
  <br>
  A <a href="https://pandai.org">Pandai</a> initiative. Built with ❤️ by educators and AI, for everyone.
</p>

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
    <img src="https://img.shields.io/badge/go-%3E%3D1.22-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-Telegram%20%7C%20WhatsApp%20%7C%20Web-green.svg" alt="Platforms">
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
- A Telegram bot token (get one from [@BotFather](https://t.me/BotFather))
- At least one AI provider API key (OpenAI, Anthropic, or use free self-hosted Ollama)

### 1. Clone and configure

```bash
git clone --recurse-submodules https://github.com/p-n-ai/pai-bot.git
cd pai-bot
git submodule update --init --recursive
cp .env.example .env
```

Edit `.env` with your credentials:

```env
# Required
LEARN_TELEGRAM_BOT_TOKEN=your-telegram-bot-token

# AI Providers (at least one required)
LEARN_AI_OPENAI_API_KEY=sk-...
LEARN_AI_ANTHROPIC_API_KEY=sk-ant-...

# Or use free self-hosted AI (no API key needed)
LEARN_AI_OLLAMA_ENABLED=true
```

### 2. Start everything

```bash
docker compose up -d
```

This starts: PostgreSQL, Dragonfly (cache), NATS (messaging), the Go server, and the admin panel.

If you want demo rows in PostgreSQL for local testing, run:

```bash
make seed
```

If the app is running in Docker, seed through the app container instead:

```bash
make seed-docker
```

When the backend is running in Docker, make sure `.env` uses Compose service names such as `postgres`, `dragonfly`, and `nats` instead of `localhost`.

### 3. Pull a free AI model (optional)

If using Ollama for free self-hosted AI:

```bash
docker compose exec ollama ollama pull llama3:8b
```

### 4. Chat with your bot

Open Telegram, find your bot, and send `/start`. That's it — you're learning.

### 5. Access the admin panel

Open `http://localhost:3000` to manage schools, classes, and view student progress.

---

## Features

### 🎓 For Students

- **AI Tutor on Telegram** — Learn any topic through natural chat conversation. The AI uses Socratic method, scaffolding, and growth mindset pedagogy.
- **Step-by-Step Problem Solving** — Every math question is answered with a structured approach: Understand → Plan → Solve → Verify → Connect. Teaches students *how to think*, not just the answer.
- **Adaptive Explanations** — The AI adjusts explanation complexity based on your mastery level. Beginners get simpler language and more examples; proficient students get concise explanations with harder challenges.
- **Curriculum-Cited Responses** — Every explanation references the exact curriculum source (e.g., "KSSM Form 1 > Algebra > Linear Equations"), so students can find it in their textbook.
- **Proactive Study Sessions** — The agent initiates conversations when it's time to review. Spaced repetition ensures long-term retention.
- **Progress Tracking** — See mastery per topic, XP earned, streak length, and progress toward personal goals.
- **Quizzes & Assessments** — Take quizzes in chat with AI-graded free-text answers, hints, and detailed feedback. When the question bank runs low, the AI generates new questions dynamically from curriculum content.
- **Exam-Style Practice** — AI-generated questions match the format and difficulty of real PT3/SPM exams, so students practice with questions that feel like the real thing.
- **Peer Challenges** — Battle classmates on the same set of questions. Learn together, compete for fun.
- **Goals & Streaks** — Set a learning goal ("Master algebra by April") and track daily streaks.

### 👩‍🏫 For Teachers

- **Class Dashboard** — Mastery heatmap showing every student's progress across every topic at a glance.
- **Student Detail View** — Deep dive into any student: mastery radar, activity timeline, struggle areas, conversation summaries.
- **Nudge Students** — One-click to have the AI send a personalized study prompt to a specific student.
- **Assign Topics** — Direct the AI to teach a specific topic to a student or entire class.
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
│  ┌──────────┐ ┌──────────┐ ┌──────────┐             │
│  │ Telegram │ │ WhatsApp │ │ WebSocket│             │
│  └────┬─────┘ └─────┬────┘ └────┬─────┘             │
│       └─────────────┼───────────┘                   │
│                     ▼                               │
│              Chat Gateway                           │
│                     │                               │
│                     ▼                               │
│  ┌──────────────────────────────────────────┐       │
│  │           Agent Engine                   │       │
│  │  ┌──────────────┐  ┌──────────────────┐  │       │
│  │  │ Conversation │  │ Proactive        │  │       │
│  │  │ State Machine│  │ Scheduler (NATS) │  │       │
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
│  │  │Ollama   │  │                                  │
│  │  │Custom   │  │                                  │
│  │  └─────────┘  │                                  │
│  └───────────────┘                                  │
│                                                     │
│  ┌──────────────────────────────────────────┐       │
│  │  Admin Panel (Next.js + Refine)          │       │
│  │  Teacher Dashboard · Parent View · Admin │       │
│  └──────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────┘
```

### Tech Stack

| Component | Technology | Why |
|-----------|-----------|-----|
| **Backend** | Go 1.22+ (stdlib) | Goroutines handle millions of concurrent connections. Single binary, ~15MB. |
| **Database** | PostgreSQL 17 | Standard, portable. Every cloud has managed Postgres. |
| **Cache** | Dragonfly | Redis-compatible, multi-threaded, 80% less memory. |
| **Messaging** | NATS + JetStream | Proactive nudge scheduling, background jobs, event-driven communication. |
| **AI Providers** | OpenAI, Anthropic, Ollama, OpenRouter | Provider-agnostic gateway. Swap models without code changes. |
| **Chat** | Telegram Bot API, WhatsApp Cloud API, WebSocket | Works on $50 phones, 2G connections, zero data cost in many countries. |
| **Admin Panel** | Next.js 14, TypeScript, Refine, shadcn/ui | Teacher dashboards, parent views, school admin. |
| **Curriculum** | [Open School Syllabus](https://github.com/p-n-ai/oss) | Structured YAML curriculum consumed by the agent. |
| **Deployment** | Docker Compose / Helm + Kubernetes | Single server ($20/mo) to national deployment (millions of students). |

### Project Structure

```
pai-bot/
├── cmd/
│   └── server/
│       └── main.go                  # Application entrypoint
├── internal/
│   ├── ai/                          # AI Gateway
│   │   ├── gateway.go               # Provider-agnostic interface
│   │   ├── router.go                # Model routing + fallback chains
│   │   ├── budget.go                # Token budget tracking + enforcement
│   │   ├── provider_openai.go       # OpenAI + compatible APIs (DeepSeek, etc.)
│   │   ├── provider_anthropic.go
│   │   ├── provider_google.go       # Google Gemini
│   │   ├── provider_ollama.go       # Self-hosted (Llama, DeepSeek, Qwen)
│   │   └── provider_openrouter.go   # 100+ models (Qwen, Kimi, etc.)
│   ├── agent/                       # Agent Engine
│   │   ├── engine.go                # Conversation state machine
│   │   ├── scheduler.go             # Proactive nudges via NATS
│   │   ├── prompts.go               # Pedagogical system prompts
│   │   ├── quiz.go                  # Assessment engine
│   │   └── challenge.go             # Peer battle system
│   ├── chat/                        # Chat Gateway
│   │   ├── gateway.go               # Unified message routing
│   │   ├── telegram.go              # Telegram adapter
│   │   ├── whatsapp.go              # WhatsApp adapter
│   │   └── websocket.go             # Web chat adapter
│   ├── curriculum/                   # Curriculum Service
│   │   ├── loader.go                # Reads YAML from OSS repository
│   │   ├── cache.go                 # In-memory + Dragonfly curriculum cache
│   │   └── types.go                 # Go structs matching OSS schema
│   ├── progress/                    # Progress Tracker
│   │   ├── tracker.go               # Mastery scoring
│   │   ├── spaced_rep.go            # SM-2 algorithm
│   │   └── streaks.go               # Streak + XP system
│   ├── auth/                        # Authentication
│   │   ├── jwt.go                   # Token generation + validation
│   │   └── middleware.go            # Role-based access control
│   ├── tenant/                      # Multi-tenancy
│   │   ├── tenant.go                # Tenant isolation logic
│   │   └── middleware.go            # Tenant resolution from JWT/subdomain
│   └── platform/                    # Shared infrastructure
│       ├── config/                  # Environment configuration
│       ├── database/                # PostgreSQL connection (pgx)
│       ├── cache/                   # Dragonfly client (go-redis)
│       ├── messaging/               # NATS client + JetStream helpers
│       ├── storage/                 # Object storage interface (S3-compatible)
│       ├── telemetry/               # OpenTelemetry setup
│       └── health/                  # Health check endpoints
├── admin/                           # Next.js admin panel
│   ├── src/
│   │   ├── app/                     # App Router pages
│   │   ├── components/              # Shared UI components
│   │   └── providers/               # Auth + data providers
│   ├── package.json
│   └── next.config.js
├── migrations/                      # SQL migration files (golang-migrate)
├── deploy/
│   ├── docker/
│   │   ├── Dockerfile               # Multi-stage Go + Admin build
│   │   └── Dockerfile.dev           # Development with hot reload
│   └── helm/
│       └── pai/                     # Helm chart for Kubernetes
├── terraform/                       # Infrastructure as Code
├── scripts/
│   ├── setup.sh                     # First-time setup wizard
│   ├── deploy.sh                    # Production deployment
│   └── analytics.sh                 # Quick metrics from CLI
├── docker-compose.yml               # One-command local development
├── docker-compose.prod.yml          # Production compose (single-server)
├── Makefile                         # Dev shortcuts
├── .env.example                     # All configuration documented
├── .github/workflows/               # CI/CD (build, test, lint, release)
└── README.md
```

---

## AI Providers

P&AI is not locked to any AI model. Configure one or more providers:

| Provider | Models | Cost | Setup |
|----------|--------|------|-------|
| **OpenAI** | GPT-4o, GPT-4o-mini, GPT-5 Nano | Paid API | Set `LEARN_AI_OPENAI_API_KEY` |
| **Anthropic** | Claude Sonnet, Claude Haiku | Paid API | Set `LEARN_AI_ANTHROPIC_API_KEY` |
| **DeepSeek** | DeepSeek V3, Reasoner | Paid API (very cheap) | Set `LEARN_AI_DEEPSEEK_API_KEY` |
| **Google Gemini** | Gemini 2.5 Flash, Pro | Paid API | Set `LEARN_AI_GOOGLE_API_KEY` |
| **Ollama** | Llama 3, DeepSeek, Qwen, Mistral | Free (self-hosted) | Set `LEARN_AI_OLLAMA_ENABLED=true` |
| **OpenRouter** | 100+ models (Qwen, Kimi, etc.) | Varies | Set `LEARN_AI_OPENROUTER_API_KEY` |

DeepSeek uses the OpenAI-compatible API format — no extra code, just a different API key and base URL. Qwen, Kimi, and other models are accessible via OpenRouter or self-hosted via Ollama.

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
| Malaysia KSSM Form 1 | Matematik (Algebra) | Planned |
| Malaysia KSSM Form 2 | Matematik (Algebra) | Planned |
| Malaysia KSSM Form 3 | Matematik (Algebra) | Planned |
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

### Option 1: Single Server (Docker Compose)

For a single school or small deployment. Runs on any VPS with 2GB+ RAM.

```bash
git clone https://github.com/p-n-ai/pai-bot.git
cd pai-bot
./scripts/setup.sh     # Interactive setup wizard
docker compose up -d   # Start everything
```

**Cost:** ~$20/month on any VPS provider. Supports 100-500 students.

### Option 2: Kubernetes (Helm)

For districts, states, or national deployments.

```bash
helm repo add pai https://p-n-ai.github.io/pai-bot/charts
helm install pai pai/pai-bot \
  --set telegram.botToken=your-token \
  --set ai.openai.apiKey=sk-... \
  --set database.url=postgresql://...
```

**Scales:** Horizontally to millions of students. Each school gets a namespace with isolated data.

### Option 3: Cloud-Agnostic

P&AI is designed to run on any cloud without lock-in:

| Component | AWS | GCP | Azure | Self-Hosted |
|-----------|-----|-----|-------|-------------|
| Compute | EKS | GKE | AKS | Any K8s |
| Database | RDS PostgreSQL | Cloud SQL | Azure DB | PostgreSQL |
| Cache | (self-hosted Dragonfly) | (self-hosted) | (self-hosted) | Dragonfly/Redis |
| Messaging | (self-hosted NATS) | (self-hosted) | (self-hosted) | NATS |
| Storage | S3 | GCS | Blob | MinIO |

---

## Configuration Reference

All configuration is via environment variables with `LEARN_` prefix. See [`.env.example`](.env.example) for the complete list.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LEARN_TELEGRAM_BOT_TOKEN` | Yes | — | Telegram bot token from @BotFather |
| `LEARN_DATABASE_URL` | No | `postgres://pai:pai@localhost:5432/pai` | PostgreSQL connection string |
| `LEARN_CACHE_URL` | No | `redis://localhost:6379` | Dragonfly/Redis connection |
| `LEARN_NATS_URL` | No | `nats://localhost:4222` | NATS messaging server |
| `LEARN_AI_OPENAI_API_KEY` | No* | — | OpenAI API key |
| `LEARN_AI_ANTHROPIC_API_KEY` | No* | — | Anthropic API key |
| `LEARN_AI_DEEPSEEK_API_KEY` | No* | — | DeepSeek API key (OpenAI-compatible) |
| `LEARN_AI_GOOGLE_API_KEY` | No* | — | Google Gemini API key |
| `LEARN_AI_OPENROUTER_API_KEY` | No* | — | OpenRouter API key (100+ models) |
| `LEARN_AI_OLLAMA_ENABLED` | No* | `false` | Enable self-hosted Ollama |
| `LEARN_AI_OLLAMA_BASE_URL` | No | `http://ollama:11434` | Ollama server URL |
| `LEARN_AUTH_JWT_SECRET` | No | Auto-generated | JWT signing secret |
| `LEARN_PORT` | No | `8080` | HTTP server port |
| `LEARN_TENANT_MODE` | No | `single` | `single` or `multi` tenant mode |

*At least one AI provider must be configured.

---

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+ (for admin panel)
- Docker and Docker Compose

### Local Development

```bash
# Start infrastructure (Postgres, Dragonfly, NATS, Ollama)
docker compose up -d postgres dragonfly nats ollama

# Run database migrations
make migrate

# Seed demo data (optional)
make seed

# Or, if the app itself is running in Docker
make seed-docker

# Start the Go server with hot reload
make dev

# In another terminal — start the admin panel
cd admin && npm install && npm run dev
```

### Running Tests

```bash
make test         # Run all Go tests
make test-integration  # Run integration tests (requires -tags=integration tests)
make test-cover   # Run tests with coverage report
make lint         # Run golangci-lint
```

OpenAI live conversation integration suite:

- Fixture source: `internal/agent/testdata/openai_live_conversations.yaml` (30 scripted conversations, 2-10 turns each)
- Test harness: `internal/agent/engine_openai_integration_test.go` (`//go:build integration`)
- Required env for live run: `LEARN_AI_OPENAI_API_KEY`
- Optional env:
  - `LEARN_AI_LIVE_TIMEOUT_SECONDS` (default `45`)
  - `LEARN_AI_LIVE_MAX_CASES` (default `30`)
- CI behavior: the live OpenAI suite is explicitly skipped in CI (`CI`/`GITHUB_ACTIONS` detection) to avoid external paid API calls in pipeline runs.

### Useful Commands

```bash
make setup        # First-time setup
make start        # Start all services via Docker Compose
make stop         # Stop all services
make logs         # Tail application logs
make migrate      # Run database migrations
make seed         # Seed demo tenant/users/messages/progress/events
make seed-docker  # Seed through the running app container
make analytics    # Print quick metrics from the database
make ollama-pull  # Download a free AI model for Ollama
```

### Rating Analytics Contract

- Rating callbacks use internal assistant message IDs in callback data: `rating:{messages.id}:{score}`.
- Submitted ratings are logged in `events` as `answer_rating_submitted` with:
  - `data.rating` (1-5)
  - `data.rated_message_id` (assistant `messages.id` being rated)
  - `data.source`, `data.channel`, `data.delayed_submit`
- Deduplication is enforced per rated assistant message (`rated_message_id`) to prevent duplicate submissions for the same prompt.

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
4. Run tests (`make test && make lint`)
5. Commit (`git commit -m 'Add amazing feature'`)
6. Push to your branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

---

## Related Repositories

| Repository | Description |
|-----------|-------------|
| [p-n-ai/oss](https://github.com/p-n-ai/oss) | Open School Syllabus — structured curriculum data for any learning platform |
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

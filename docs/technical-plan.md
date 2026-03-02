# Technical Plan — P&AI Bot

> **Repository:** `p-n-ai/pai-bot`
> **License:** Apache 2.0
> **Last updated:** February 2026

---

## 1. Architecture Overview

P&AI Bot is a **modular monolith** — a single deployable Go binary with internally clean domain boundaries that can be split into microservices when specific domains need independent scaling. This gives early-stage development speed (fast iteration, one deployment, easy debugging) while maintaining the option to decompose later.

```
┌──────────────────────────────────────────────────────────────┐
│  Chat Channels                                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                      │
│  │ Telegram │ │ WhatsApp │ │ WebSocket│                      │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘                      │
│       └─────────────┼───────────┘                            │
│                     ▼                                        │
│              Chat Gateway (internal/chat)                    │
│                     │                                        │
│                     ▼                                        │
│  ┌──────────────────────────────────────────────────┐        │
│  │           Agent Engine (internal/agent)          │        │
│  │  ┌──────────────┐  ┌──────────────────┐          │        │
│  │  │ Conversation │  │ Proactive        │          │        │
│  │  │ State Machine│  │ Scheduler (NATS) │          │        │
│  │  └──────────────┘  └──────────────────┘          │        │
│  │  ┌──────────────┐  ┌──────────────────┐          │        │
│  │  │ Progress     │  │ Pedagogical      │          │        │
│  │  │ Tracker      │  │ Prompt Builder   │          │        │
│  │  └──────────────┘  └──────────────────┘          │        │
│  └──────────────────────┬───────────────────────────┘        │
│                         │                                    │
│              ┌──────────┼────────────┐                       │
│              ▼          ▼            ▼                       │
│  ┌───────────────┐ ┌──────────┐ ┌───────────────┐            │
│  │  AI Gateway   │ │Curriculum│ │  PostgreSQL   │            │
│  │  ┌─────────┐  │ │ Service  │ │  + Dragonfly  │            │
│  │  │OpenAI   │  │ │  (OSS)   │ │               │            │
│  │  │Anthropic│  │ └──────────┘ └───────────────┘            │
│  │  │Ollama   │  │                                           │
│  │  │Custom   │  │                                           │
│  │  └─────────┘  │                                           │
│  └───────────────┘                                           │
│                                                              │
│  ┌──────────────────────────────────────────────────┐        │
│  │  Admin Panel (Next.js + Refine)                  │        │
│  │  Teacher Dashboard · Parent View · School Admin  │        │
│  └──────────────────────────────────────────────────┘        │
└──────────────────────────────────────────────────────────────┘
```

---

## 2. Tech Stack

### 2.1 Backend

| Component | Technology | Version | Rationale |
|-----------|-----------|---------|-----------|
| **Language** | Go | ≥1.22 | Goroutines handle millions of concurrent connections natively. Single static binary (~15MB). Explicit, minimal syntax is ideal for agentic AI coding (Claude Code, Cursor). |
| **HTTP Router** | Go stdlib `net/http` | 1.22+ | Go 1.22 introduced pattern-based routing in stdlib — no framework needed. Composable, explicit, no magic. |
| **Database Driver** | `pgx` | v5 | Fastest PostgreSQL driver in any language. Native support for Postgres types, batch queries, and built-in connection pooling. |
| **Cache Client** | `go-redis/redis` | v9 | Redis-protocol client. Works with both Redis and Dragonfly without code changes. |
| **Messaging Client** | `nats-io/nats.go` | ≥2.10 | Native Go client for NATS. JetStream support for persistent message streams. |
| **WebSocket** | `nhooyr.io/websocket` | v1 | Lightweight, idiomatic Go WebSocket library. A single Go instance maintains hundreds of thousands of persistent connections. |
| **JWT** | `golang-jwt/jwt` | v5 | Stateless auth — short-lived access tokens (15 min) + longer refresh tokens (7 days). Middleware validates JWT on every request with zero database calls. |
| **Configuration** | Environment variables | — | All config via `LEARN_` prefixed env vars. `envconfig` or `viper` for parsing. |
| **Linting** | `golangci-lint` | latest | Static analysis with a curated set of linters enforced in CI. |
| **Testing** | Go stdlib `testing` | — | Table-driven tests. `testcontainers-go` for integration tests against real Postgres/Dragonfly. |

### 2.2 Data Layer

| Component | Technology | Version | Rationale |
|-----------|-----------|---------|-----------|
| **Primary Database** | PostgreSQL | 17 | Standard, portable — every cloud has managed Postgres. Handles relational data, JSONB for flexible schemas, full-text search, and pub/sub via `LISTEN/NOTIFY`. |
| **Connection Pooling** | PgBouncer (prod) / pgx built-in (dev) | — | Essential at scale. Multiplexes thousands of app connections into fewer PG connections. On AWS, use RDS Proxy during credits year; swap to PgBouncer on migration. |
| **Cache** | Dragonfly | ≥1.0 | Drop-in Redis replacement that is multi-threaded and uses ~80% less memory at scale. Same Redis protocol, same client libraries. Used for: session state, rate limiting, leaderboards, spaced repetition scheduling queues. |
| **Message Queue** | NATS + JetStream | ≥2.10 | Written in Go, single binary. Handles millions of messages/second. Used for: proactive nudge scheduling, background job processing (report generation, analytics events), event-driven communication between domain modules. Far lighter than Kafka, more capable than Redis pub/sub. |
| **Migrations** | `golang-migrate` | v4 | SQL-based migrations. Each migration is a pair of `.up.sql` / `.down.sql` files in `migrations/`. Run via `make migrate`. |

### 2.3 AI Gateway

The AI Gateway is a provider-agnostic abstraction that routes AI inference requests to the best available model based on task type, budget, and availability.

| Provider | Models | Use Case | Cost |
|----------|--------|----------|------|
| **OpenAI** | GPT-4o, GPT-4o-mini, GPT-5 Nano | Teaching (complex), Grading (fast) | Paid API |
| **Anthropic** | Claude Sonnet, Claude Haiku | Teaching (nuanced pedagogy), Analysis | Paid API |
| **DeepSeek** | DeepSeek V3, DeepSeek Reasoner | Grading (cheapest), Question generation | Paid API (OpenAI-compatible) |
| **Google Gemini** | Gemini 2.5 Flash, Gemini 2.5 Pro | Teaching, Grading (competitive pricing) | Paid API |
| **Ollama** | Llama 3, DeepSeek, Qwen, Mistral, Gemma | Fallback (always free), Privacy-sensitive deployments | Free (self-hosted) |
| **OpenRouter** | 100+ models (Qwen, Kimi, etc.) | Access to any model via single API | Varies |

**DeepSeek uses the OpenAI-compatible API format.** The `provider_openai.go` implementation supports a configurable base URL, so DeepSeek (and any other OpenAI-compatible provider like Groq, Together AI) requires no new code — just a different `LEARN_AI_DEEPSEEK_API_KEY` and base URL (`https://api.deepseek.com`). Qwen and Kimi are accessible via OpenRouter or self-hosted via Ollama.

**Routing logic:**

```
Teaching (complex explanations)  → Best available (Claude Sonnet, GPT-4o, Gemini 2.5 Pro)
Grading (quick JSON responses)   → Cheapest (DeepSeek V3, GPT-4o-mini, Gemini Flash)
Question generation (quiz/exam)  → Cheapest (DeepSeek V3, GPT-4o-mini) via CompleteJSON
Nudges (short messages)          → Any available model
Budget exhausted                 → Automatic fallback to Ollama (free)
```

**Go interface:**

```go
type AIProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
    Models() []ModelInfo
    HealthCheck(ctx context.Context) error
}
```

Each provider implements this interface. The `Router` selects a provider based on task type, model preference, token budget, and circuit breaker state. No student is ever cut off from learning — the system degrades gracefully to free self-hosted models.

### 2.4 Chat Gateway

| Channel | Technology | Protocol | Rationale |
|---------|-----------|----------|-----------|
| **Telegram** | Telegram Bot API | HTTPS long-polling / Webhooks | Works on $50 phones, 2G connections. Zero-rated in many countries. Largest reach for target demographic. |
| **WhatsApp** | WhatsApp Cloud API | Webhooks | Second priority channel. Dominant in Southeast Asia, Africa, Latin America. Higher API cost than Telegram. |
| **Web** | WebSocket (`nhooyr.io/websocket`) | WSS | For web-based admin panel chat testing and future web client. |

**Unified interface:**

```go
type ChatChannel interface {
    SendMessage(ctx context.Context, userID string, msg Message) error
    SendQuiz(ctx context.Context, userID string, quiz QuizMessage) error
    ReceiveMessages(ctx context.Context) <-chan IncomingMessage
}
```

Each channel adapter normalizes platform-specific message formats into a common `Message` struct consumed by the Agent Engine.

### 2.5 Admin Panel (Frontend)

| Component | Technology | Version | Rationale |
|-----------|-----------|---------|-----------|
| **Framework** | Next.js (App Router) | 14 | SSR for SEO, API routes for BFF patterns, edge middleware. AI agents write excellent Next.js code — strongest ecosystem support. |
| **Language** | TypeScript | 5.x | Type safety, excellent agentic coding support. |
| **Admin Framework** | Refine | ≥4 | Filament-equivalent for React. Rapid CRUD generation, headless architecture, data provider abstraction. |
| **UI Components** | shadcn/ui | latest | Copy-paste component library built on Radix. Not a dependency — components live in your codebase. Tailwind-based styling. |
| **Styling** | Tailwind CSS | 3.x | Utility-first. Consistent design system without custom CSS. |
| **Charts** | Recharts or Tremor | — | For mastery heatmaps, progress charts, leaderboard visualizations. |
| **State Management** | React Query (TanStack Query) | v5 | Server state management. Automatic caching, refetching, and invalidation. |
| **Auth** | JWT from Go backend | — | Next.js middleware validates JWT on protected routes. Refresh token rotation handled client-side. |

**Admin panel views:**

- **Teacher Dashboard** — Mastery heatmap, student detail views, nudge controls, topic assignment
- **Parent View** — Child progress, weekly reports, streak/XP tracking
- **School Admin** — Multi-class management, token budget allocation, data export
- **Platform Admin** — Multi-tenant management, AI provider configuration, usage analytics

### 2.6 Pedagogical Prompt Strategies

The Agent Engine uses structured prompt patterns to ensure consistent, high-quality teaching. These are implemented in `internal/agent/prompts.go` as system prompt templates — no additional infrastructure required.

Inspired by [DeepTutor](https://github.com/HKUDS/DeepTutor)'s multi-agent reasoning architecture, adapted for chat-based K-12 math tutoring.

#### 2.6.1 Dual-Loop Problem Solving

When a student asks a math question, the system prompt instructs the AI to follow a structured 5-step pattern:

```
1. UNDERSTAND — Restate the problem. Identify what is given and what is asked.
2. PLAN     — Choose a strategy. Explain why this approach works.
3. SOLVE    — Execute step-by-step with intermediate checks.
4. VERIFY   — Check the answer. Does it make sense? Try a different method.
5. CONNECT  — Link to the curriculum topic. Preview what comes next.
```

This mirrors DeepTutor's dual-loop architecture (Analysis Loop → Solve Loop) but is implemented purely as a prompt pattern. The AI is required to show its reasoning at each step, teaching students *how to think* rather than just giving answers.

#### 2.6.2 Curriculum Citation

Every AI explanation must reference the specific curriculum source:

```
"📖 KSSM Form 1 > Algebra > Linear Equations > Section 2.3"
```

The prompt builder injects the curriculum path (`{syllabus} > {subject} > {topic}`) into the system prompt. This helps students locate content in their textbooks and gives teachers/parents confidence that the bot follows the official syllabus.

#### 2.6.3 Adaptive Explanation Depth

The system prompt adjusts explanation complexity based on the student's mastery level for the current topic:

| Mastery Level | Prompt Behavior |
|---------------|----------------|
| **< 0.3** (Beginner) | Use simple everyday language. More concrete examples. Break into smaller steps. Avoid formal notation until the concept clicks. |
| **0.3 – 0.6** (Developing) | Standard explanations. Introduce formal mathematical notation gradually. Mix worked examples with guided practice. |
| **> 0.6** (Proficient) | More concise. Focus on edge cases, common mistakes, and connections between topics. Challenge with harder variants. |

Mastery score is read from the `progress` table and injected into the system prompt alongside the student's progress context ("mastered X, working on Y, struggles with Z").

#### 2.6.4 Dynamic Question Generation

When the curriculum YAML has fewer than 5 assessment questions for a topic, the quiz engine generates additional questions dynamically using the AI gateway's `CompleteJSON` fast-path (cheapest model). The generation prompt includes:

- The topic's teaching notes as source material
- The difficulty level appropriate for the student's mastery
- 2–3 real PT3/SPM exam exemplar questions (stored in `assessments.yaml`) as style references

This "exam mimicry" approach ensures AI-generated questions match the format, difficulty, and style of real Malaysian national exams, rather than producing generic math problems.

### 2.7 Algorithms

| Algorithm | Purpose | Implementation |
|-----------|---------|----------------|
| **SM-2 (SuperMemo 2)** | Spaced repetition scheduling — determines when to review topics | `internal/progress/spaced_rep.go`. Calculates next review interval based on ease factor, repetition count, and response quality. |
| **Mastery Scoring** | Determines per-topic mastery level (0.0–1.0) | Weighted combination of assessment accuracy, consistency, and recency. Threshold at 0.75 for mastery. |
| **Token Budget Tracking** | Allocates and enforces AI credit budgets per school/class/student | `internal/ai/budget.go`. Real-time tracking in Dragonfly with periodic PostgreSQL sync. |
| **Model Routing** | Selects optimal AI provider per request | Cost-aware routing with circuit breaker pattern. Falls back through provider chain on failure. |
| **Dual-Loop Problem Solving** | Structured step-by-step teaching for math questions | `internal/agent/prompts.go`. System prompt pattern: Understand → Plan → Solve → Verify → Connect. |
| **Adaptive Explanation Depth** | Adjusts explanation complexity per student | `internal/agent/prompts.go`. Mastery-based prompt selection: beginner / developing / proficient. |
| **Dynamic Question Generation** | Generates quiz questions when curriculum has insufficient assessments | `internal/agent/quiz.go`. AI generates questions from teaching notes with exam-style mimicry using PT3/SPM exemplars. |

---

## 3. Project Structure

```
pai-bot/
├── cmd/
│   └── server/
│       └── main.go                  # Application entrypoint
├── internal/
│   ├── ai/                          # AI Gateway
│   │   ├── gateway.go               # Provider-agnostic interface + router
│   │   ├── router.go                # Model routing + fallback chains
│   │   ├── budget.go                # Token budget tracking + enforcement
│   │   ├── provider_openai.go       # OpenAI + OpenAI-compatible APIs (DeepSeek, Groq, etc.)
│   │   ├── provider_anthropic.go    # Anthropic implementation
│   │   ├── provider_google.go       # Google Gemini implementation
│   │   ├── provider_ollama.go       # Self-hosted models (Llama, DeepSeek, Qwen, Mistral)
│   │   └── provider_openrouter.go   # OpenRouter (100+ models: Qwen, Kimi, etc.)
│   ├── agent/                       # Agent Engine (core domain)
│   │   ├── engine.go                # Conversation state machine
│   │   ├── scheduler.go             # Proactive nudges via NATS
│   │   ├── prompts.go               # Pedagogical system prompts
│   │   ├── quiz.go                  # Assessment engine
│   │   └── challenge.go             # Peer battle system
│   ├── chat/                        # Chat Gateway
│   │   ├── gateway.go               # Unified message routing
│   │   ├── telegram.go              # Telegram Bot API adapter
│   │   ├── whatsapp.go              # WhatsApp Cloud API adapter
│   │   └── websocket.go             # Web chat adapter
│   ├── curriculum/                   # Curriculum Service
│   │   ├── loader.go                # Reads YAML from OSS repository
│   │   ├── cache.go                 # In-memory + Dragonfly curriculum cache
│   │   └── types.go                 # Go structs matching OSS schema
│   ├── progress/                    # Progress Tracker
│   │   ├── tracker.go               # Mastery scoring engine
│   │   ├── spaced_rep.go            # SM-2 algorithm implementation
│   │   └── streaks.go               # Streak + XP + leaderboard system
│   ├── auth/                        # Authentication
│   │   ├── jwt.go                   # Token generation + validation
│   │   └── middleware.go            # Role-based access control (student/teacher/parent/admin)
│   ├── tenant/                      # Multi-tenancy
│   │   ├── tenant.go                # Tenant isolation logic
│   │   └── middleware.go            # Tenant resolution from JWT/subdomain
│   └── platform/                    # Shared infrastructure
│       ├── config/                  # Environment configuration (envconfig)
│       ├── database/                # PostgreSQL connection pool (pgx)
│       ├── cache/                   # Dragonfly client (go-redis)
│       ├── messaging/               # NATS client + JetStream helpers
│       ├── storage/                 # Object storage interface (S3-compatible)
│       ├── telemetry/               # OpenTelemetry setup
│       └── health/                  # Health check endpoints
├── admin/                           # Next.js admin panel
│   ├── src/
│   │   ├── app/                     # App Router pages
│   │   │   ├── dashboard/           # Teacher dashboard
│   │   │   ├── students/            # Student detail views
│   │   │   ├── classes/             # Class management
│   │   │   ├── parents/             # Parent portal
│   │   │   ├── settings/            # School/platform settings
│   │   │   └── analytics/           # Usage and learning analytics
│   │   ├── components/              # Shared UI components (shadcn/ui based)
│   │   │   ├── mastery-heatmap.tsx
│   │   │   ├── progress-radar.tsx
│   │   │   ├── leaderboard.tsx
│   │   │   └── ...
│   │   └── providers/               # Refine data provider, auth provider
│   ├── package.json
│   ├── next.config.js
│   ├── tailwind.config.ts
│   └── tsconfig.json
├── migrations/                      # SQL migration files (golang-migrate)
│   ├── 001_init_schema.up.sql
│   ├── 001_init_schema.down.sql
│   └── ...
├── deploy/
│   ├── docker/
│   │   ├── Dockerfile               # Multi-stage: Go build + Admin build → scratch
│   │   └── Dockerfile.dev           # Development with hot reload
│   └── helm/
│       └── pai/                     # Helm chart for Kubernetes
│           ├── Chart.yaml
│           ├── values.yaml
│           └── templates/
├── terraform/                       # Infrastructure as Code
│   ├── environments/
│   │   ├── dev/
│   │   ├── staging/
│   │   └── production/
│   ├── modules/
│   │   ├── eks/
│   │   ├── rds/
│   │   ├── s3/
│   │   └── networking/
│   └── main.tf
├── scripts/
│   ├── setup.sh                     # First-time setup wizard
│   ├── deploy.sh                    # Production deployment
│   └── analytics.sh                 # Quick metrics from CLI
├── docker-compose.yml               # One-command local development
├── docker-compose.prod.yml          # Production compose (single-server)
├── Makefile                         # Dev shortcuts (dev, test, lint, migrate, build)
├── .env.example                     # All configuration documented
├── .github/
│   └── workflows/
│       ├── ci.yml                   # Test + lint + build on every PR
│       └── release.yml              # Build Docker image + Helm chart on tag
├── go.mod
├── go.sum
└── README.md
```

---

## 4. Database Schema (Core Tables)

```sql
-- Multi-tenancy
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    config      JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Users (students, teachers, parents, admins)
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),
    role        TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'parent', 'admin')),
    name        TEXT NOT NULL,
    external_id TEXT,                          -- Telegram user ID, WhatsApp number, etc.
    channel     TEXT NOT NULL DEFAULT 'telegram',
    config      JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Student progress per topic
CREATE TABLE progress (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    syllabus_id     TEXT NOT NULL,
    topic_id        TEXT NOT NULL,
    mastery_score   REAL DEFAULT 0.0,          -- 0.0 to 1.0
    ease_factor     REAL DEFAULT 2.5,          -- SM-2 ease factor
    interval_days   INTEGER DEFAULT 1,         -- SM-2 interval
    repetitions     INTEGER DEFAULT 0,         -- SM-2 repetition count
    next_review_at  TIMESTAMPTZ,
    last_studied_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, syllabus_id, topic_id)
);

-- Conversation history (for context and analytics)
CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id),
    topic_id    TEXT,
    state       TEXT NOT NULL DEFAULT 'idle',   -- idle, teaching, quizzing, reviewing
    messages    JSONB DEFAULT '[]',
    started_at  TIMESTAMPTZ DEFAULT NOW(),
    ended_at    TIMESTAMPTZ
);

-- Assessment results
CREATE TABLE assessments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    topic_id        TEXT NOT NULL,
    question_id     TEXT NOT NULL,
    answer          TEXT,
    score           REAL,                       -- 0.0 to 1.0
    feedback        TEXT,
    time_taken_ms   INTEGER,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Engagement: streaks, XP, challenges
CREATE TABLE streaks (
    user_id         UUID PRIMARY KEY REFERENCES users(id),
    current_streak  INTEGER DEFAULT 0,
    longest_streak  INTEGER DEFAULT 0,
    total_xp        INTEGER DEFAULT 0,
    last_active_at  DATE
);

-- AI token budget tracking
CREATE TABLE token_budgets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID REFERENCES tenants(id),
    user_id         UUID REFERENCES users(id),  -- NULL = tenant-level budget
    budget_tokens   BIGINT NOT NULL,
    used_tokens     BIGINT DEFAULT 0,
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL
);
```

---

## 5. Infrastructure & Deployment

### 5.1 Cloud Strategy

**Primary cloud:** AWS (100K credits in Year 1)
**Design principle:** Cloud-agnostic — use AWS as infrastructure, not application logic. No proprietary AWS services in application code.

| Layer | AWS (Current) | Portable To | Lock-in Risk |
|-------|--------------|-------------|--------------|
| Compute | EKS (Kubernetes) | GKE, AKS, any K8s | None — standard K8s manifests |
| Database | RDS PostgreSQL | Cloud SQL, Azure DB, self-hosted | None — standard Postgres |
| Cache | Self-hosted Dragonfly on K8s | Moves with cluster | None |
| Messaging | Self-hosted NATS on K8s | Moves with cluster | None |
| Object Storage | S3 (via Go interface) | GCS, Azure Blob, MinIO | Abstracted in code |
| Secrets | Secrets Manager (via ESO) | GCP SM, Azure KV, Vault | Abstracted via External Secrets Operator |
| Container Registry | ECR | GHCR, GCR, ACR | OCI-standard images |
| CDN / DNS | Cloudflare | — | Cloud-independent |
| Ingress | Traefik (on K8s) | — | Cloud-independent |
| IaC | Terraform | — | Provider-swappable |
| CI/CD | GitHub Actions → ArgoCD | — | Cloud-independent |
| Observability | Grafana stack on K8s | — | Cloud-independent |

**AWS services actively avoided:** DynamoDB, SQS, SNS, Lambda (as core logic), Cognito, Step Functions, AppSync, Amplify, Aurora-specific features, EventBridge, Kinesis, CloudWatch, CloudFormation.

### 5.2 Deployment Options

**Option A: Single Server (Docker Compose)**
For a single school or small deployment. Any VPS with 2GB+ RAM.

```bash
docker compose up -d
# Starts: PostgreSQL, Dragonfly, NATS, Go server, Admin panel, Ollama (optional)
```

Cost: ~$20/month. Supports 100–500 students.

**Option B: Kubernetes (Helm)**
For districts, states, or national deployments.

```bash
helm install pai pai/pai-bot \
  --set telegram.botToken=your-token \
  --set ai.openai.apiKey=sk-... \
  --set database.url=postgresql://...
```

Scales horizontally to millions of students. Each school gets a namespace with isolated data.

### 5.3 Docker Build

Multi-stage build producing a minimal image:

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.22-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /pai-server ./cmd/server

# Stage 2: Build Admin panel
FROM node:20-alpine AS admin-builder
WORKDIR /admin
COPY admin/package*.json ./
RUN npm ci
COPY admin/ .
RUN npm run build

# Stage 3: Final image (~25MB)
FROM scratch
COPY --from=go-builder /pai-server /pai-server
COPY --from=admin-builder /admin/.next /admin/.next
COPY --from=admin-builder /admin/public /admin/public
ENTRYPOINT ["/pai-server"]
```

### 5.4 Observability

| Signal | Tool | Export Target |
|--------|------|--------------|
| **Metrics** | OpenTelemetry SDK → Prometheus | Grafana dashboards |
| **Logs** | Structured JSON (slog) → Loki | Grafana log explorer |
| **Traces** | OpenTelemetry SDK → Tempo | Grafana trace viewer |
| **Analytics** | PostHog (self-hosted or cloud) | Product analytics |

All telemetry is instrumented in Go via OpenTelemetry from Day 1. The Grafana stack runs inside the K8s cluster. On AWS, S3 is used as the storage backend for Loki and Tempo (swappable to GCS/Azure Blob via config).

### 5.5 CI/CD Pipeline

```
Push to main / PR
    │
    ▼
GitHub Actions
    ├── Go: test, lint (golangci-lint), build
    ├── Admin: npm ci, lint, type-check, build
    ├── Docker: build multi-stage image, push to ECR/GHCR
    └── Helm: lint chart, package
    │
    ▼ (on merge to main)
ArgoCD (running in K8s cluster)
    ├── Detects new image tag in Git
    ├── Applies Helm values
    ├── Rolling update (zero downtime)
    └── Health check + automatic rollback
```

---

## 6. Security

| Concern | Approach |
|---------|----------|
| **Authentication** | JWT with short-lived access tokens (15 min) + refresh token rotation (7 days). Built in Go — no external auth provider dependency. |
| **Authorization** | Role-based access control (RBAC). Roles: `student`, `teacher`, `parent`, `admin`, `platform_admin`. Enforced in middleware. |
| **Data Isolation** | Multi-tenant with tenant_id on every table. Row-level security in PostgreSQL. Tenant resolved from JWT claims. |
| **Data Sovereignty** | Self-hostable by design. No student data leaves the deployment unless explicitly configured. |
| **AI API Keys** | Stored in Secrets Manager (via ESO), injected as K8s secrets. Never in code, env files, or logs. |
| **Transport** | TLS everywhere. Cloudflare → Traefik → services. Internal cluster communication via mTLS (optional Istio/Linkerd). |
| **Input Validation** | All user input sanitized. AI prompts use structured templates — no raw user input in system prompts. |
| **Rate Limiting** | Per-user rate limiting in Dragonfly. Per-tenant API rate limiting at Traefik ingress. |
| **COPPA/PDPA Compliance** | Minimal PII collection. Parental consent flow for users under 13. Data export and deletion APIs. |

---

## 7. Key Go Libraries

| Library | Purpose | Import Path |
|---------|---------|-------------|
| pgx | PostgreSQL driver | `github.com/jackc/pgx/v5` |
| go-redis | Dragonfly/Redis client | `github.com/redis/go-redis/v9` |
| nats.go | NATS messaging | `github.com/nats-io/nats.go` |
| websocket | WebSocket handling | `nhooyr.io/websocket` |
| jwt | JWT auth | `github.com/golang-jwt/jwt/v5` |
| otel | OpenTelemetry | `go.opentelemetry.io/otel` |
| slog | Structured logging | `log/slog` (stdlib) |
| migrate | Database migrations | `github.com/golang-migrate/migrate/v4` |
| testcontainers | Integration testing | `github.com/testcontainers/testcontainers-go` |
| golangci-lint | Linting | `github.com/golangci/golangci-lint` |

---

## 8. Admin Panel Dependencies

```json
{
  "dependencies": {
    "next": "^14",
    "react": "^18",
    "@refinedev/core": "^4",
    "@refinedev/nextjs-router": "^6",
    "@refinedev/react-hook-form": "^4",
    "tailwindcss": "^3",
    "@radix-ui/react-*": "latest",
    "recharts": "^2",
    "tanstack/react-query": "^5",
    "lucide-react": "latest",
    "date-fns": "^3",
    "zod": "^3"
  }
}
```

---

## 9. Performance Targets

| Metric | Target | How |
|--------|--------|-----|
| Concurrent student connections | 100K per Go instance | Goroutines (~4KB each) |
| AI response latency (P95) | <3s for teaching, <1s for grading | Model routing + streaming responses |
| Message processing throughput | 10K messages/second per instance | Async processing via NATS |
| Database queries (P95) | <50ms | pgx with prepared statements, Dragonfly caching |
| Admin panel page load | <1s (LCP) | Next.js SSR + edge caching |
| Docker image size | <30MB | Multi-stage build, scratch base |
| Cold start time | <100ms | Go binary boots instantly |
| Deployment downtime | Zero | Rolling updates via Kubernetes |

---

## 10. Development Workflow

```bash
# Prerequisites
# Go 1.22+, Node.js 20+, Docker, Docker Compose

# First-time setup
make setup                           # Copies .env.example, pulls deps

# Start infrastructure
docker compose up -d postgres dragonfly nats ollama

# Run database migrations
make migrate

# Start Go server with hot reload (air)
make dev

# Start admin panel (separate terminal)
cd admin && npm install && npm run dev

# Run tests
make test                            # Unit tests
make test-integration                # Integration tests (testcontainers)
make lint                            # golangci-lint
make test-all                        # Everything

# Build for production
make build                           # Go binary + admin static
make docker                          # Docker image
```

---

## 11. Curriculum Integration

P&AI Bot consumes curriculum data from the [Open School Syllabus (OSS)](https://github.com/p-n-ai/oss) repository. The integration works as follows:

1. **Git submodule** — OSS is included as a Git submodule at `curriculum/`
2. **Loader** — `internal/curriculum/loader.go` reads YAML files at startup and caches parsed curriculum in memory + Dragonfly
3. **Hot reload** — A filesystem watcher detects changes to curriculum files and reloads without restart
4. **Go types** — `internal/curriculum/types.go` defines Go structs that mirror the OSS JSON Schema (Syllabus, Subject, Topic, Assessment, TeachingNotes)
5. **No code changes needed** — Adding a new curriculum to OSS automatically makes it available in P&AI

---

## 12. Related Repositories

| Repository | Relationship |
|-----------|-------------|
| [p-n-ai/oss](https://github.com/p-n-ai/oss) | Curriculum data consumed as Git submodule. P&AI reads and teaches from this content. |
| [p-n-ai/oss-bot](https://github.com/p-n-ai/oss-bot) | Receives improvement suggestions from P&AI's student interaction data via API. |

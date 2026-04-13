# Architecture

P&AI Bot is a **modular monolith** — a single Go binary with clean domain boundaries. Each domain lives under `internal/` and communicates through well-defined Go interfaces, making it possible to split into microservices later if needed.

## High-Level Flow

```
Student (Telegram/WebSocket)
    │
    ▼
Chat Gateway ──► Agent Engine ──► AI Gateway ──► Provider (OpenAI/Anthropic/...)
    │                │                              │
    │                ▼                              ▼
    │          Curriculum Loader            Fallback Chain
    │          Progress Tracker             Budget Tracker
    │          Quiz Engine
    │          Challenge Runtime
    │          Scheduler (nudges)
    │
    ▼
Admin Panel (Next.js) ◄──► Admin API (Go)
```

## Domain Packages

### Core Learning

| Package | Path | Responsibility |
|---------|------|----------------|
| **Agent** | `internal/agent/` | Conversation state machine, message processing pipeline, quiz engine, challenge runtime, goal tracking, proactive nudge scheduler, group commands |
| **AI Gateway** | `internal/ai/` | Provider-agnostic AI interface, model routing with fallback chain, circuit breaker, structured JSON output, token budget tracking |
| **Curriculum** | `internal/curriculum/` | Loads topic YAML, teaching notes, and assessment questions from the OSS curriculum repository |
| **Progress** | `internal/progress/` | Mastery scoring, SM-2 spaced repetition scheduling, streak tracking, XP system |

### Communication

| Package | Path | Responsibility |
|---------|------|----------------|
| **Chat Gateway** | `internal/chat/` | Unified interface for all chat channels. Routes inbound messages to the agent engine and outbound responses to the correct channel |
| **Telegram** | `internal/chat/telegram.go` | Telegram Bot API adapter — long polling, inline keyboards, markdown formatting, `/start` onboarding |
| **WebSocket** | `internal/chat/websocket.go` | WebSocket channel for web clients and terminal-chat testing |
| **i18n** | `internal/i18n/` | Internationalization — message templates for BM/EN/ZH |

### Platform

| Package | Path | Responsibility |
|---------|------|----------------|
| **Config** | `internal/platform/config/` | Environment variable loading with `LEARN_` prefix, validation |
| **Database** | `internal/platform/database/` | PostgreSQL connection pool (`pgxpool`) wrapper |
| **Cache** | `internal/platform/cache/` | Dragonfly/Redis client wrapper |
| **Mailer** | `internal/platform/mailer/` | SMTP email delivery for admin invites |
| **Seed** | `internal/platform/seed/` | Demo data seeding for development |

### Access Control

| Package | Path | Responsibility |
|---------|------|----------------|
| **Auth** | `internal/auth/` | JWT tokens, email/password login, Google OIDC, invite acceptance, session management |
| **Tenant** | `internal/tenant/` | Multi-tenancy — default tenant bootstrap, tenant isolation |
| **Admin API** | `internal/adminapi/` | REST endpoints for dashboard, class management, student data, exports, analytics |

### Utilities

| Package | Path | Responsibility |
|---------|------|----------------|
| **Retrieval** | `internal/retrieval/` | BM25-based knowledge retrieval over collections and documents |
| **API Docs** | `internal/apidocs/` | OpenAPI spec generation and Scalar docs UI at `/docs` |
| **Analytics XLSX** | `internal/analyticsxlsx/` | Excel export for analytics reports |
| **Terminal Chat** | `internal/terminalchat/` | Terminal-based chat runner for local testing and E2E verification |
| **Terminal Nudge** | `internal/terminalnudge/` | Terminal-based nudge trigger for testing scheduler behavior |

## HTTP Routing

Uses Go 1.22+ stdlib `net/http` with pattern-based routing (no framework):

```go
mux.HandleFunc("POST /api/auth/login", ...)
mux.HandleFunc("GET /api/admin/classes/{id}/progress", ...)
```

Middleware chains enforce access control:
- `authenticated` — valid JWT session required
- `teacherOrAbove` — teacher, admin, or platform_admin role
- `parentOrAbove` — parent, admin, or platform_admin role
- `adminOrAbove` — admin or platform_admin role
- `adminOnly` — admin only (single role)

## Database

PostgreSQL 17 with:
- All tables include `tenant_id` for multi-tenant isolation
- UUID primary keys (`gen_random_uuid()`)
- JSONB for flexible fields (user config, conversation state)
- Migrations managed by `goose` (timestamped SQL files in `migrations/`)
- Parameterized queries only — no string interpolation

## Caching

Dragonfly (Redis-compatible) used for:
- Token budget tracking (real-time counters)
- Rate limiting (per-user, per-tenant)
- Session data

## Messaging (Planned)

NATS with JetStream is configured in Docker Compose and the config package but not yet integrated in application code. Planned uses:
- Event streaming (session events, AI usage)
- Cross-service communication if the monolith splits

## Infrastructure

```
┌─────────────────────────────────────────────┐
│  Caddy (reverse proxy, auto-HTTPS)          │
│  /api/* → app:8080  |  /* → admin:3000      │
├─────────────────────────────────────────────┤
│  Go Server (:8080)  │  Next.js Admin (:3000)│
├─────────────────────────────────────────────┤
│  PostgreSQL 17  │  Dragonfly  │  NATS       │
└─────────────────────────────────────────────┘
```

Docker Compose orchestrates all services for both local development and single-server production deployment.

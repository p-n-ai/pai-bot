---
title: "Backend Packages"
summary: "Current Go backend package map for pai-bot, including cmd entrypoints, internal domains, and ownership boundaries."
read_when:
  - You are changing Go backend behavior.
  - You need to locate the package that owns a runtime, API, auth, chat, AI, curriculum, retrieval, progress, or tenancy change.
  - You are adding a new backend package or moving backend responsibilities.
---

# Backend Packages

The backend is a modular Go monolith. `cmd/` wires binaries. `internal/` owns domain behavior. Prefer adding behavior to the existing owning package instead of creating a new package.

## Binaries

| Path | Purpose |
|---|---|
| `cmd/server/` | Main HTTP server. Wires config, DB, cache, mailer, AI router, chat gateways, admin APIs, OpenAPI docs, and embedded admin assets. |
| `cmd/seed/` | Database seed CLI for demo data and token budgets. |
| `cmd/terminal-chat/` | Local terminal chat runner and WebSocket client for testing tutor behavior. |
| `cmd/terminal-nudge/` | Local nudge runner for checking proactive review messages. |
| `cmd/analyticsxlsx/` | Analytics workbook export CLI. |

## Domain packages

| Path | Purpose | Common change type |
|---|---|---|
| `internal/agent/` | Tutor runtime: conversation state machine, prompt assembly, turn harness, scheduling, quiz, goals, challenges, groups, milestones, event logging. | Bot behavior, prompt safety, quiz/challenge rules, turn tracing. |
| `internal/ai/` | Provider-agnostic AI gateway: model routing, token budgets, image inputs, structured output, OpenAI/Anthropic/Google/OpenRouter/Ollama providers. | Provider changes, model routing, budget behavior. |
| `internal/chat/` | Chat channel adapters and message formatting: Telegram, WhatsApp, WebSocket, embedded chat config, keyboards, commands. | Transport parsing, outbound formatting, channel-specific behavior. |
| `internal/adminapi/` | Admin-facing backend service methods for classes, groups, onboarding, and admin dashboards. | Admin API data shape or school/admin workflows. |
| `internal/auth/` | Login/session identity: cookies, JWT, password auth, guest auth, Google OIDC, middleware, Postgres auth persistence. | Auth flow, session cookies, Google login, RBAC. |
| `internal/curriculum/` | YAML curriculum loader, type model, prerequisite graph. | Curriculum schema or topic loading. |
| `internal/progress/` | Mastery scoring, spaced repetition, streaks, XP, display helpers, Postgres tracker. | Progress metrics, review schedule, XP/streak behavior. |
| `internal/retrieval/` | Generic retrieval platform and BM25 service over sources, collections, documents, and curriculum seed data. | Retrieval lab, curriculum search, indexed document behavior. |
| `internal/tenant/` | Tenant bootstrap and isolation helpers. | School/tenant creation or tenant-scoped access. |
| `internal/apidocs/` | stdlib OpenAPI document and route metadata for API docs. | Public API docs and schema changes. |
| `internal/analyticsxlsx/` | XLSX analytics workbook generation. | Spreadsheet export changes. |
| `internal/i18n/` | Localized message strings. | User-facing bot/admin text keys. |
| `internal/terminalchat/` | Shared state/runner code for terminal chat tools. | Local chat debugging behavior. |
| `internal/terminalnudge/` | Nudge capture channel for terminal nudge tests/tools. | Local nudge debugging behavior. |

## Platform packages

| Path | Purpose |
|---|---|
| `internal/platform/airouter/` | AI router setup shared by server and local tools. |
| `internal/platform/cache/` | Dragonfly/Redis cache client. |
| `internal/platform/config/` | Environment config loading and validation. |
| `internal/platform/database/` | PostgreSQL pool setup. |
| `internal/platform/mailer/` | Mailer abstraction. |
| `internal/platform/seed/` | Demo and token-budget seed data. |

## Backend rules

- Keep `cmd/*` thin. If logic is testable product behavior, put it under `internal/`.
- Keep package tests beside package code.
- For bot prompt/context changes, read `docs/runtime/agent-turn-api.md` before editing.
- For auth/session changes, read `docs/admin/admin-auth.md` before editing.
- For admin API changes, update `docs/admin/routes.md`, `internal/apidocs/`, and admin client code when the surface changes.
- For config/env changes, update `docs/ops/config.md`.
- For Telegram, WhatsApp, embed, or OpenAPI docs surfaces, read the matching `docs/runtime/` file first.

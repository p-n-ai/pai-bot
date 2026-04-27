---
title: "Codebase Map"
summary: "Folder-level map for pai-bot: backend, admin app, site, curriculum mirror, deploy assets, scripts, and local agent files."
read_when:
  - You are new to the pai-bot repository and need to know where code lives.
  - You are deciding which folder owns a change.
  - You are updating AGENTS.md or repo documentation about folder responsibilities.
---

# Codebase Map

`pai-bot` is the runtime/admin/deploy repository for the P&AI bot product. It consumes curriculum/content from `oss`, serves the chat runtime, exposes admin APIs, ships the Next.js admin app, and carries deploy assets.

## Top-level folders

| Path | Owns | Notes |
|---|---|---|
| `cmd/` | Go binaries | Thin entrypoints. Keep product logic in `internal/`.
| `internal/` | Go backend packages | Main runtime code: agent engine, chat channels, admin API, auth, AI, curriculum, retrieval, progress, tenancy, platform adapters.
| `admin/` | Next.js admin app | Teacher/admin/dashboard UI, API proxy routes, auth shell, login/onboarding, retrieval lab.
| `site/` | Astro marketing/docs surface | Public site content and styling. Separate from admin app.
| `oss/` | Checked-in curriculum/content source | Canonical curriculum mirror inside this repo. Do not treat it as backend runtime code.
| `migrations/` | PostgreSQL schema migrations | Goose-style timestamped SQL. Keep schema changes here.
| `deploy/` | Runtime deploy configs | Dockerfiles, Caddy, Nginx, Helm.
| `terraform/` | Cloud infrastructure | IaC docs and modules for hosted environments.
| `scripts/` | Shell helpers | Dev, setup, deploy, analytics scripts.
| `tools/` | Local support tools | Provider emulation and local harnesses.
| `docs/` | Repo documentation | Docs-list indexed docs. Add front matter for new docs.
| `.github/` | GitHub Actions | CI and workflow automation.
| `.agents/`, `.codex/`, `.claude/` | Agent-local tooling | Local agent configs/skills. Do not commit unless explicitly repo policy.

## Primary request paths

| Request | Start here | Then check |
|---|---|---|
| Bot reply behavior | `internal/agent/engine.go` | `internal/agent/prompt_builder.go`, `docs/runtime/agent-turn-api.md`, tests in `internal/agent/`.
| Prompt/context safety | `internal/agent/turn.go` | `internal/agent/context_loader.go`, `internal/agent/context_packets.go`, `docs/runtime/agent-turn-api.md`.
| Telegram/WhatsApp/web chat transport | `internal/chat/` | `cmd/server/main.go`, channel-specific tests.
| Admin API behavior | `internal/adminapi/` | `cmd/server/main.go`, `admin/src/lib/server-api.ts`.
| Admin UI | `admin/src/app/` | `admin/src/components/`, `admin/src/lib/`, `docs/admin-panel-uiux.md`.
| Auth/login/session | `internal/auth/` | `docs/admin/admin-auth.md`, `admin/src/components/login-gate/`.
| Admin routes/API | `cmd/server/main.go`, `admin/src/app/` | `docs/admin/routes.md`, `internal/apidocs/`.
| Curriculum loading | `internal/curriculum/` | `oss/`, `docs/architecture/curriculum.md`.
| Retrieval lab/search | `internal/retrieval/` | `admin/src/components/retrieval-lab.tsx`, `docs/runtime/quiz-mode.md` when agent retrieval affects tutoring.
| Embed chat widget | `internal/chat/embed*`, `cmd/server/embed_admin.go` | `docs/runtime/embeddable-chat.md`, `scripts/test-embed.html`.
| WhatsApp runtime | `internal/chat/whatsapp*.go` | `docs/runtime/whatsapp.md`, `admin/src/app/settings/whatsapp/page.tsx`.
| OpenAPI/Scalar docs | `internal/apidocs/` | `docs/runtime/openapi-scalar.md`, `cmd/server/main.go`.
| Config/env behavior | `internal/platform/config/` | `docs/ops/config.md`, `docs/ops/setup.md`.
| Progress, XP, streaks | `internal/progress/` | `internal/agent/*progress*`, `internal/agent/milestones.go`.
| Deployment | `deploy/`, `terraform/` | `docs/ops/deployment.md`, `.github/workflows/`.

## Documentation rules

- Use `docs-list` first to find existing docs before adding a new one.
- Add new cross-cutting docs under `docs/codebase/` unless a narrower docs folder already exists.
- Keep current-state docs separate from future plans.
- If a folder responsibility changes, update both this doc and `AGENTS.md`.

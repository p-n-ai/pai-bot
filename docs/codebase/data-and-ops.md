---
title: "Data And Operations"
summary: "Map of pai-bot data, curriculum, migrations, deploy, infrastructure, scripts, CI, and local tooling folders."
read_when:
  - You are changing database schema, deployment, infrastructure, curriculum files, scripts, or local tooling.
  - You need to know which non-code folder owns a change.
  - You are updating operational docs or AGENTS.md folder guidance.
---

# Data And Operations

This repo carries runtime code plus the operational assets needed to run it.

## Data and content

| Path | Purpose | Rule |
|---|---|---|
| `migrations/` | PostgreSQL schema migrations. | Add timestamped SQL for schema changes; keep migrations forward-readable. |
| `oss/` | Curriculum/content mirror consumed by runtime. | Treat as content source, not backend package code. `oss/AGENTS.md` has local rules. |
| `docs/qa/` | Pilot QA scripts, forms, and evaluation docs. | Keep user-facing pilot docs separate from codebase docs. |

## Deploy and infrastructure

| Path | Purpose |
|---|---|
| `deploy/docker/` | Backend/admin Dockerfiles. |
| `deploy/caddy/` | Caddy reverse-proxy configs. |
| `deploy/nginx/` | Nginx reverse-proxy config. |
| `deploy/helm/` | Helm chart assets. |
| `terraform/` | Cloud infrastructure as code. |
| `.github/workflows/` | GitHub Actions CI and deployment workflows. |

## Scripts and tools

| Path | Purpose |
|---|---|
| `scripts/setup.sh` | Local setup helper. |
| `scripts/run-dev.sh` | Local dev runner. |
| `scripts/stop-dev.sh` | Local dev stop helper. |
| `scripts/deploy-remote.sh` | Remote deploy helper. |
| `scripts/analytics.sh` | Analytics export helper. |
| `scripts/test-embed.html` | Manual embedded-chat test fixture. |
| `tools/emulate/` | Local provider emulation config and docs. |

## Local agent folders

| Path | Purpose | Commit rule |
|---|---|---|
| `.agents/` | Repo-local agent skills/config experiments. | Do not commit unless explicitly making repo policy. |
| `.codex/` | Codex local environment config. | Do not commit client-local config by default. |
| `.claude/` | Claude local skill/config mirror. | Do not commit unless explicitly requested. |
| `AGENTS.override.md` | Local-only org override. | Do not commit. |

## Operational rules

- Update `docs/deployment.md` when deploy behavior changes.
- Update `docs/setup.md` and `.env.example` when config/env changes.
- Update `docs/curriculum.md` when curriculum schema or loader behavior changes.
- Use `trash` for removals.
- Keep local-only agent config out of commits unless the task explicitly says repo-local agent policy.

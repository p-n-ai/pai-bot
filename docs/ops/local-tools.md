---
title: "Local Tools"
summary: "Current local helper binaries, scripts, and emulation tools for developing and testing pai-bot."
read_when:
  - You are using terminal chat, terminal nudge, analytics export, setup scripts, or provider emulation.
  - You are changing cmd/terminal-chat, cmd/terminal-nudge, cmd/analyticsxlsx, scripts, or tools/emulate.
  - You need a local-only way to exercise bot behavior without the full deployed stack.
---

# Local Tools

## Go binaries

| Command path | Purpose |
|---|---|
| `cmd/terminal-chat` | Local chat runner. Can use memory state, Postgres state, multi-user mode, or WebSocket client mode. |
| `cmd/terminal-nudge` | Local proactive nudge runner for a user ID. |
| `cmd/analyticsxlsx` | Analytics workbook export CLI. |
| `cmd/seed` | Demo and token-budget seed CLI. |

## Scripts

| Path | Purpose |
|---|---|
| `scripts/setup.sh` | Local setup helper. |
| `scripts/run-dev.sh` | Starts local dev services/server. |
| `scripts/stop-dev.sh` | Stops local dev services/server. |
| `scripts/deploy-remote.sh` | Remote deploy helper. |
| `scripts/analytics.sh` | Analytics helper wrapper. |
| `scripts/test-embed.html` | Manual embed widget test fixture. |

## Emulation

| Path | Purpose |
|---|---|
| `tools/emulate/` | Local provider emulation config, including auth-provider emulation notes. |

For Google/OIDC local auth emulation, read `docs/ops/local-auth-emulation.md`.

## Update rules

- Keep local tools documented here when adding new `cmd/*` debug binaries or `scripts/*` helpers.
- Keep scripts small and repo-specific.
- Do not commit local `.codex/`, `.agents/`, or `.claude/` changes unless the task explicitly asks for repo-local agent policy.

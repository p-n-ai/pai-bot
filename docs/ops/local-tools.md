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
| `cmd/conversation-harness` | Replays scored YAML tutor conversations against the real agent engine and AI router. Use for prompt/runtime quality loops, answer-dump regressions, and response naturalness checks. |
| `cmd/terminal-nudge` | Local proactive nudge runner for a user ID. |
| `cmd/analyticsxlsx` | Analytics workbook export CLI. |
| `cmd/seed` | Demo and token-budget seed CLI. |

## AI response quality loop

Run the default quality fixture:

```bash
just conversation-harness
```

Run one case and print turns:

```bash
go run ./cmd/conversation-harness --case Q01 --show-responses
```

The default fixture is `internal/agent/testdata/ai_quality_conversations.yaml`. Add pilot failures there before changing the prompt, then rerun the harness to compare behavior.

For transport-only server checks without a real AI provider key, use the dev mock provider:

```bash
LEARN_AI_MOCK_RESPONSE="Mock tutor response from local dev." just go
go run ./cmd/terminal-chat --ws ws://127.0.0.1:8080/ws/chat --user-id dev-check
```

One-shot WebSocket check:

```bash
go run ./cmd/terminal-chat --ws ws://127.0.0.1:8080/ws/chat --user-id dev-check --message "Solve 3x - 5 = 16. First step only."
```

`LEARN_DEV_MODE=true` keeps `/ws/chat` compatible with terminal-chat first-message auth. Production embed chat still uses JWT subprotocol auth and origin checks.

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

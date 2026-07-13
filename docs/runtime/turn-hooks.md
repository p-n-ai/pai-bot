---
title: "Turn Hooks"
summary: "Internal Tutor Turn hook contract, rollout flag, privacy boundary, and add/remove workflow."
read_when:
  - You are adding, removing, or reviewing an internal Tutor Turn hook.
  - You are changing PAI_FEATURES=turn_hooks behavior.
  - You need to verify hook notices, hook-injected context, or hook privacy rules.
---

# Turn Hooks

Turn Hooks are private runtime extension points for the normal tutor model path.
They let the agent runtime observe or shape a `Tutor Turn` after base context is loaded and before prompt compilation.

They are not React hooks, Git hooks, Codex hooks, plugins, tenant settings, YAML config, or user-installed extensions.

## Quick Start

Enable the hook runner with the deploy-time rollout flag:

```env
PAI_FEATURES=turn_hooks
```

To see local hook calls in Terminal Chat, enable dev mode too:

```env
LEARN_DEV_MODE=true
PAI_FEATURES=turn_hooks
```

Terminal Chat prints one short line per configured hook call. The current catalog is empty, so enabling the flag alone produces no notices.

## Runtime Position

Hooks run only inside the generic tutor model flow:

```text
ProcessMessage
  -> runTeachingTurn
  -> agentTurn
  -> loadContextPackets
  -> runTurnHooks
  -> buildPromptMessagesFromTurn
  -> promptCompiler.compile
  -> aiRouter.Complete
  -> agent_turn_completed
```

Early-return flows such as commands, onboarding, quiz routing, and challenge runtime do not run Turn Hooks unless they later enter the normal tutor model path.

## Rollout Flag

`PAI_FEATURES=turn_hooks` controls whether the hook runner executes at all.

| Flag state | Runtime behavior |
|---|---|
| off | The hook runner is skipped. |
| on | The private Turn Hook Catalog runs in order. |

Do not add per-hook feature flags. The rollout flag controls the hook seam. The Turn Hook Catalog controls which hooks run.

## Current Catalog

The catalog is currently empty. Adding a hook requires a concrete teaching-path use case and focused tests.

## Hook Outcomes

Every hook returns exactly one outcome:

| Outcome | Meaning | Required data |
|---|---|---|
| `continue` | Leave the turn unchanged. | No packets required. |
| `inject` | Append context packets before prompt compilation. | At least one valid `contextPacket`. |
| `block` | Stop before the model call with a runtime-owned response. | Optional block message. No production hook uses this yet. |

Injected packets go through the same packet validation as base context. A non-system-owned packet cannot render as system content.

## Privacy Rules

Turn Hooks must keep the existing prompt boundary intact.

- Do not log packet `Data`.
- Do not add hook name, outcome, raw packet content, learner text, prompt messages, image data URLs, or summaries to `agent_turn_completed`.
- Do not persist Hook Call Notices.
- Do not send Hook Call Notices to the model.
- Use `contextPacket` trust, render, and trace fields for any injected context.
- Treat local dump files as private debug artifacts.

## Add A Hook

1. Add a package-private hook type in `internal/agent/turn_hooks.go` or a small sibling file under `internal/agent/`.
2. Implement `Name() string` and `Run(context.Context, *agentTurn) (turnHookResult, error)`.
3. Add the hook to `defaultTurnHookCatalog()` in the order it should run.
4. Return `continue`, `inject`, or `block`.
5. For `inject`, build trace-safe `contextPacket` values and rely on existing packet validation.
6. Add focused tests in `internal/agent/turn_hooks_test.go`.
7. Add an engine-level test when the hook changes prompt shape, model-call behavior, block behavior, or dev-mode notices.
8. Update this doc when the catalog or operating contract changes.

## Remove A Hook

1. Remove it from `defaultTurnHookCatalog()`.
2. Delete hook-specific code and tests.
3. Preserve or restore the non-hook runtime path if learner behavior must remain unchanged.
4. Update this doc and any feature-flag references.

## Test Checklist

Use focused package tests for normal changes:

```bash
go test ./internal/agent ./internal/platform/featureflags ./internal/platform/config
```

Use the full repo gate before publishing:

```bash
go test ./...
```

For request-shape inspection without calling a real model:

```bash
go run ./cmd/conversation-harness --case Q01 --request-only --dump-requests /tmp/pai-bot-llm-requests.jsonl
```

The request dump uses the real turn harness and router trace, but forces a mock provider when no `--mock-response` is supplied.

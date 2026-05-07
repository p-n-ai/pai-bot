---
title: "AI Turn Harness"
summary: "Current architecture note for the tutor AI turn harness, trust-labeled context packets, prompt compilation, and trace-safe metadata."
read_when:
  - You are changing how chat input reaches the tutor model.
  - You are changing prompt assembly, conversation history, compaction, or dynamic learner context.
  - You need to decide whether context belongs in system, user, tool, agent_turn_completed metadata, or future AI-call tables.
---

# AI Turn Harness

The AI turn harness makes one normal tutoring message explicit:

```text
ProcessMessage
  -> agentTurn
  -> loadContextPackets
  -> runTurnHooks (when PAI_FEATURES=turn_hooks)
  -> buildPromptMessagesFromTurn
  -> promptCompiler.compile
  -> aiRouter.Complete
  -> agent_turn_completed
```

This harness covers the generic tutor model path. Commands, onboarding, challenge runtime, quiz routing, and rating-only submissions can return before this flow.

## Current Implementation

The current implementation lives in `internal/agent/`:

| File | Role |
|---|---|
| `turn.go` | Defines package-private `agentTurn`, `contextPacket`, trust/render/trace enums, prompt manifest, and model result metadata. |
| `context_loader.go` | Reads current stores and creates trust-labeled packets for the turn. |
| `context_packets.go` | Builds, defaults, validates, labels, and summarizes packets. |
| `turn_hooks.go` | Defines the private Turn Hook runner, Hook Outcomes, Turn Hook Catalog, and `rate_convo_hook`. |
| `prompt_builder.go` | Compiles packets and chat history into model-facing `[]ai.Message`. |
| `engine.go` | Owns `ProcessMessage`, turn lifecycle, AI call, response persistence, and `agent_turn_completed`. |
| `tutor_personality.go` | Encodes the active SOUL-style tutor personality block used by the prompt harness. |
| `cmd/conversation-harness` | Replays scored YAML conversations through the real engine and AI router for prompt/runtime quality checks. |
| `prompt_builder_test.go` | Regression coverage for prompt ordering, image handling, untrusted data quoting, and invalid packet rejection. |
| `engine_test.go` | Runtime coverage for event metadata and trace privacy. |

The package-level API remains `Engine.ProcessMessage`. The turn harness types are package-private so they stay an internal review boundary, not a public construction API.

## Trust Boundary

Stored in pai-bot does not mean trusted as instruction.

| Trust | Meaning | Render rule |
|---|---|---|
| `system_owned` | App-owned constrained state, such as form, language preference, topic ID, mastery, streak, XP, image/rating control instruction. | May render as system data or system instruction. |
| `learner_provided` | Learner-authored text or profile text, such as first name, goal summary, replied-to text. | Quote as user data. Never system instruction. |
| `model_generated` | Model-created data from prior turns, such as compacted summary. | Quote as data-only continuity context. |
| `external` | Image attachment or future external/OCR content. | Attach or quote as untrusted external data. |

Validation rejects any non-system-owned packet that tries to render as system content.

## Packet Sources

`loadContextPackets` currently creates packets from:

- profile metadata and learner-provided first name
- conversation state and model-generated summary
- matched curriculum topic and teaching notes
- progress snapshot and due reviews
- active goals, split into system-owned metadata and learner-provided summary
- streak and XP
- replied-to text
- image instruction and image attachment

When `PAI_FEATURES=turn_hooks` is off, existing rating prompt behavior is appended after base packet loading. When `PAI_FEATURES=turn_hooks` is on, `rate_convo_hook` injects the same `rating.prompt` packet only when the turn is already due for rating.

Mixed-trust records must be split. Example: goal target mastery is system-owned metadata; goal summary is learner-provided text.

## Turn Hooks

**Turn Hooks** are internal runtime extension points that observe or shape a **Tutor Turn** without becoming plugins, YAML config, tenant settings, or user-installed extension points.

For the operating contract, add/remove workflow, privacy rules, and test checklist, read [Turn Hooks](turn-hooks.md).

The **Turn Hook Rollout Flag** is:

```env
PAI_FEATURES=turn_hooks
```

When the flag is disabled, the hook runner does not run. When the flag is enabled, the private **Turn Hook Catalog** runs in order. The first catalog contains only **Rate Conversation Turn Hook** (`rate_convo_hook`).

Each **Turn Hook** returns one **Hook Outcome**:

| Outcome | Meaning |
|---|---|
| `continue` | Leave the **Tutor Turn** unchanged. |
| `inject` | Add trace-safe context packets, then validate them with the existing packet validation rules. |
| `block` | Stop the model call with a runtime-owned block response. No production hook uses this yet. |

`rate_convo_hook` is behavior-preserving. It does not decide when rating is due; it only turns the existing rating decision into hook-shaped context.

Add or remove hooks through the **Turn Hook Catalog** only. Do not add per-hook feature flags, dynamic hook config, or public plugin behavior for this slice.

## Prompt Shape

`promptCompiler.compile` renders messages in this order:

1. base tutor system prompt
2. context trust rules, if untrusted packets exist
3. system-owned learner context
4. model-generated conversation summary as quoted user data
5. recent user/assistant chat history
6. learner-provided context as quoted user data
7. system-owned image instruction
8. current user message, with image URLs if present
9. optional system-owned rating prompt instruction

The current user message must appear once. Reply context is separate quoted data, not mixed into current input.

The base tutor prompt includes a SOUL-style `ROBOT PERSONALITY ACTIVE: P&AI Study Buddy` block. The runtime uses a distilled in-code block so production does not depend on local tuning notes.

## Trace Contract

`agent_turn_completed` records metadata only:

- turn ID
- channel
- route and task
- topic ID
- prompt message count
- summary used flag
- context source names and count
- model name, token counts, latency
- status and error text

Never add raw packet data, names, goal summaries, reply text, summaries, image data URLs, full prompts, or chat text to this event.

For local prompt debugging, `cmd/terminal-chat --dump-json <path>` can write an explicit file containing UI-visible turns plus model-facing `messages`. Add `--turn-limit 10` when the UI only needs the latest 10 visible turns/model calls. Keep that path opt-in and local-only; it is not part of `agent_turn_completed` or durable event telemetry.

When `LEARN_DEV_MODE=true` and `PAI_FEATURES=turn_hooks`, Terminal Chat may print one **Hook Call Notice** per hook call:

```text
turn hook called: rate_convo_hook outcome=continue
```

The notice contains only hook name and **Hook Outcome**. It is not sent to the model, saved to chat history, included in `--dump-json`, emitted by `cmd/conversation-harness`, or persisted as an event.

## Future Work

Keep future changes incremental:

| Future slice | Notes |
|---|---|
| richer prompt manifest | Add packet counts or rendered sections without raw content. |
| `agent_turns` persistence | Store metadata/manifest only after runtime fields stabilize. |
| `ai_call_events` persistence | Track model purpose, provider/model, tokens, latency, status, and error code. |
| debug UI | Show metadata by default. Raw prompt inspection requires separate permission, retention, and audit events. |

Do not add prompt snapshots, admin debug surfaces, or schema migrations as part of a prompt-shape cleanup unless the task explicitly asks for persistence.

## Quality Harness

Use `cmd/conversation-harness` to make prompt changes measurable:

```bash
go run ./cmd/conversation-harness --fixture internal/agent/testdata/ai_quality_conversations.yaml
```

The default fixture scores pilot-derived cases for:

- answer dumping under first-step, setup-only, check-only, and direct-answer pressure
- prompt extraction attempts that ask for hidden or system instructions
- naturalness, including short replies and avoiding worksheet-style section labels when the learner asked for a light interaction
- scope redirects for loaded curriculum and form-level boundaries

Add a failing conversation before changing prompt/runtime behavior. Keep checks broad enough to catch regressions without depending on one exact wording.

The harness suppresses warning logs by default so the result transcript stays focused on pass/fail output. Add `--verbose` when diagnosing curriculum loading or async background checks.

For local request inspection without calling a real model, dump the mock provider requests:

```bash
go run ./cmd/conversation-harness --case Q01 --request-only --dump-requests /tmp/pai-bot-llm-requests.jsonl
```

`--dump-requests` writes JSONL records from the real turn harness and router trace, but forces a mock provider when no `--mock-response` is supplied. The file contains model-facing `messages`, model/task/max-token request fields, and provider timing metadata. Treat the dump as local-only because it can include raw learner text and prompt context.

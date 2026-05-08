---
title: "Agent Turn API"
summary: "Runtime API contract for turning one tutor message into trust-labeled context packets, prompt messages, and safe turn metadata."
read_when:
  - You are changing how chat input reaches the tutor model
  - You are changing agentTurn, contextPacket, prompt assembly, or agent_turn_completed metadata
  - You need to add a new learner context source without leaking raw private data into traces
---

# Agent Turn API

This doc describes the current internal API boundary for one generic tutor model turn.

The goal is to make prompt construction reviewable:

- runtime state is collected once
- context is tagged by trust level before rendering
- untrusted text is quoted as data, not promoted into system instructions
- traces record metadata only

This only covers the normal tutor AI path. Early-return flows such as commands, onboarding, challenge runtime, quiz routing, and rating submissions stay outside this harness unless they reach the normal tutor model path.

## Public Surface

The package-level API remains `Engine.ProcessMessage`.

The turn harness is package-private on purpose:

- `agentTurn`
- `contextPacket`
- `loadContextPackets`
- `runTurnHooks`
- `buildPromptMessagesFromTurn`
- `promptCompiler.compile`

Keep these names unexported unless another Go package needs to construct or compile tutor turns directly. The contract is still documented here because these types are the review boundary inside `internal/agent`.

Non-goals for the current surface:

- exporting turn construction for other packages
- persisting full prompts or packet data
- wrapping command, onboarding, goal, challenge, quiz, or rating-only flows in `agentTurn`
- adding a second `TurnContext` object between the loader and compiler

## Runtime flow

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

## agentTurn

`agentTurn` is the runtime record for one inbound message that reaches the tutor model.

It carries:

- request identity: turn ID, user ID, conversation ID, channel, language
- current input: raw text, model-facing user content, reply/image flags
- resolved context: conversation, matched curriculum topic, teaching notes
- packet input: `[]contextPacket`
- output metadata: prompt manifest and model result

It should not become a general session object. State that survives across turns belongs in the existing stores.

## contextPacket

`contextPacket` is the context unit passed from the loader to the prompt compiler.

Fields:

- `ID`
  Stable packet name, such as `profile.name` or `conversation.summary`.
- `Kind`
  Broad domain category: profile, conversation, curriculum, progress, goal, streak, XP, current input, image, or control instruction.
- `Trust`
  Who owns the content.
- `Source`
  Trace-safe source name, such as `profile`, `goals`, or `reply_to`.
- `Data`
  Typed payload consumed by the prompt compiler.
- `RenderAs`
  How the compiler may render the packet.
- `TraceMode`
  Whether the packet source may appear in turn metadata.

Packet data may contain private or learner-provided content. Do not log `Data`.

## Trust levels

`system_owned`

App-owned state that can be rendered as system context or control instruction. Examples: form level, preferred language, curriculum topic metadata, progress snapshot, image handling instruction.

`learner_provided`

Learner text or learner-authored data. Examples: learner name, goal summary, replied-to message. Render as quoted data.

`model_generated`

Model-created data from prior turns. Example: conversation summary. Render as quoted data.

`external`

External attachments or content. Example: image data URL. Attach to the current user message or omit from trace metadata.

Validation rejects any non-system-owned packet that asks to render as system content.

## Render modes

`system_instruction`

System-owned control text. Use for instructions that change model behavior, such as image analysis or rating prompt handling.

`system_data`

System-owned learner/runtime facts. Use for app-owned context like progress, topic metadata, form level, XP, and streak.

`quoted_data`

Untrusted or semi-trusted text. Use for learner-provided and model-generated content. The compiler quotes this content and adds trust rules.

`attachment`

Binary or URL-like payload attached through the model request, not rendered as prompt text.

## Loader contract

`loadContextPackets` gathers context directly from existing stores and returns packets.

Current sources:

- profile
- conversation state and summary
- curriculum topic and teaching notes
- progress and due reviews
- goals
- streak
- XP
- replied-to text
- image instruction and attachment

When `PAI_FEATURES=turn_hooks` is disabled, existing rating prompt behavior is appended after base context loading. When `PAI_FEATURES=turn_hooks` is enabled, the **Rate Conversation Turn Hook** (`rate_convo_hook`) injects the same `rating.prompt` packet only when `agentTurn.RatingPromptRequested` is already true.

Keep the loader direct. Do not add a second `TurnContext` representation unless multiple callers need the same intermediate shape.

## Turn Hook contract

**Turn Hooks** are package-private runtime extension points for the normal **Tutor Turn** path. They are not React hooks, Git hooks, Codex hooks, plugins, YAML config, tenant settings, or user-installed extensions.

Read [Turn Hooks](turn-hooks.md) for the full operating contract, add/remove workflow, privacy rules, and test checklist.

The **Turn Hook Rollout Flag** is `PAI_FEATURES=turn_hooks`. When it is off, the hook runner does not run. When it is on, the private **Turn Hook Catalog** runs in order. The first catalog contains only `rate_convo_hook`.

A **Turn Hook** returns exactly one **Hook Outcome**:

- `continue`: leave the **Tutor Turn** unchanged.
- `inject`: append trace-safe `contextPacket` values and validate them through the existing packet validation rules.
- `block`: stop the model call with a runtime-owned block response. No production hook uses this yet.

Add or remove hooks by editing the **Turn Hook Catalog**. Do not add per-hook feature flags or dynamic hook configuration.

## Prompt compiler contract

`buildPromptMessagesFromTurn` delegates to `promptCompiler.compile`.

The compiler renders messages in this order:

1. base tutor system prompt
2. context trust rules, when untrusted packets exist
3. system-owned learner context
4. model-generated conversation summary as quoted user data
5. recent user/assistant chat history
6. learner-provided context as quoted user data
7. system-owned image instruction
8. current user message, with image URLs if present
9. system-owned rating prompt instruction

The current user message should appear once. Reply context should be separate quoted data, not mixed into current input.

## Trace contract

The `agent_turn_completed` event records turn metadata only.

Allowed fields include:

- turn ID
- channel
- route and task
- topic ID
- prompt message count
- summary used flag
- context source names
- model name, token counts, latency
- status and error text

Do not add raw packet data, learner goal text, reply text, profile name, summaries, image data URLs, or full prompt bodies to this event.

## Adding a context source

1. Add the packet in `loadContextPackets`.
2. Pick the narrowest `Kind`, `Trust`, `RenderAs`, and `TraceMode`.
3. Render it in `promptCompiler` only if it affects model input.
4. Add a regression test if the source can include learner text or model-generated text.
5. Confirm `agent_turn_completed` contains only trace-safe metadata.

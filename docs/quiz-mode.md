---
title: "Quiz Mode"
summary: "How quiz mode works, why it is routed before tutor AI, and how it borrows from OpenClaw's session-first agent runtime."
read_when:
  - You are changing quiz flow, quiz UX, or assessment routing
  - You are deciding whether quiz logic belongs in AI prompts or deterministic runtime code
  - You need to understand why quiz mode uses explicit state, persisted metadata, and existing progress/XP trackers
---

# Quiz Mode

Quiz mode is the assessment runtime for P&AI Bot.

It is designed to feel like part of the same chat, not a separate command-driven subsystem.
Students should be able to say "quiz me on linear equations" and move directly into an assessment flow without learning a special command surface.

This doc explains:

- how quiz mode works today
- why quiz turns are routed before tutor AI
- why quiz state is persisted explicitly
- how quiz reuses the existing progress and XP systems
- how this design is inspired by OpenClaw's agent loop, session, and memory model

## Design goals

Quiz mode is optimized for five product goals:

1. Seamless entry
   Students should not need `/quiz` to start.

2. Deterministic grading
   OSS-backed assessment questions should be graded without an LLM in the critical path.

3. Explicit runtime state
   Active quiz flow should be owned by the runtime, not inferred from prompt history.

4. Durable learner continuity
   Intensity preference, learner profile, and active quiz progress should survive across turns.

5. Shared learning signals
   Quiz results should feed the same progress and XP systems used elsewhere in the product.

## Runtime model

Quiz mode is a conversation mode, not a prompt trick.

The engine resolves the active conversation, checks its persisted mode, and decides whether the incoming turn belongs to:

- onboarding
- language selection
- rating flow
- quiz intensity selection
- active quiz
- normal teaching

That means quiz routing happens before the normal tutor AI call.

This is the key behavior that makes quiz mode feel stable:

- answer grading does not depend on the model being clever
- wrong answers keep the same question active
- default first-run intensity does not block quiz start
- explicit intensity selection is remembered
- side questions can pause the quiz without consuming the question
- normal conversation can resume the quiz later from stored state
- `/clear` resets runtime quiz state cleanly

## Persisted state

Quiz mode does not encode the whole session into brittle string blobs.

Instead, the conversation keeps a small explicit mode plus persisted quiz metadata:

- `quiz_intensity`
  Waiting for the learner to choose `easy`, `medium`, `hard`, or `mixed`

- `quiz_active`
  A live quiz is in progress

Additional quiz details are stored as structured conversation metadata when needed:

- active topic id
- selected intensity
- current question index
- correct answer count
- quiz run state (`active` or `paused`)
- pause reason (`manual_pause`, `side_question`, `teach_first`)

Per-user durable preference is stored separately:

- preferred quiz intensity

If the learner has never chosen an intensity before, quiz mode starts immediately with a default mixed intensity.
That keeps the first quiz turn seamless while still allowing later explicit intensity preference to take over.

Learner profile is also reused when available:

- name
- form level

This keeps the control plane explicit while letting the chat still feel natural.

## Conversation during quiz

Quiz mode now borrows a smaller OpenClaw-style runtime idea:

- the session owns a quiz state object
- the runtime classifies the turn before grading
- only answer turns mutate progress

That means an active quiz turn is no longer treated as "all text must be an answer".

Before grading, the runtime now checks for a few quiz actions:

- answer
- hint
- repeat the current question
- pause/exit
- restart quiz
- teach first
- side conversation

Examples:

- `hint`
  returns the stored hint and keeps the same question active
- `stop`, `done`, `taknak quiz`
  exits quiz mode naturally without requiring a slash command
- `I don't get it`, `tak faham`
  pauses quiz mode and hands the turn back to teaching flow
- `sambung`, `continue quiz`
  resumes the paused quiz from the same checkpoint
- `give me another quiz on linear equations`
  restarts quiz mode instead of being graded as a wrong answer
- `how is the weather today?`
  pauses the quiz, routes the turn back to normal teaching/chat handling, and preserves the quiz checkpoint

This keeps the chat flexible without letting unrelated turns corrupt quiz progress.

## Why route before tutor AI

Normal teaching and quiz assessment are different workloads.

Teaching is open-ended.
Quiz is constrained.

If quiz answers were handled inside the normal tutor prompt, the system would become less reliable:

- grading would be harder to keep consistent
- retries could drift
- state would depend on prompt reconstruction
- wrong answers might accidentally advance the flow
- progress signals would be noisier

Routing before AI gives quiz mode a smaller and more trustworthy loop:

1. detect or resume quiz mode
2. load persisted quiz state
3. grade deterministically
4. update quiz progress
5. render the next quiz response

Only when the turn is not part of quiz mode does the engine fall back to the normal tutor AI path.

## Why this is OpenClaw-inspired

This design is heavily inspired by OpenClaw's runtime philosophy.

Three OpenClaw ideas matter most here:

### 1. Agent loop before model

OpenClaw treats the agent loop as the authoritative runtime:

- intake
- session resolution
- context assembly
- model/tool execution
- persistence

The important lesson for P&AI Bot is that the runtime should decide the path of a turn before the model is called.

Quiz mode follows that.
The engine first checks session state and routes the turn accordingly.

### 2. Session is the source of truth

OpenClaw treats session state as owned by the gateway/runtime, not by the UI or prompt transcript.

P&AI Bot follows the same idea for quiz mode:

- active quiz is persisted in conversation state
- explicit intensity selection is persisted as user preference
- quiz progress is resumed from storage, not reconstructed from text

This is why quiz mode survives normal chat continuity better than a parser-first design.

### 3. Durable memory lives outside the model

OpenClaw's memory model is simple: if something should be remembered, write it to durable state.

P&AI Bot applies the same rule to quiz:

- learner profile is persisted
- quiz intensity is persisted
- active quiz progress is persisted
- quiz outcomes update durable mastery/XP systems

The model is not asked to "remember" quiz progress.
The runtime does.

## Parser boundary

The current design aims to keep parsing at the edge.

Acceptable parsing:

- natural-language intent detection for quiz start
- callback/button payload decoding
- intensity normalization from user input
- optional intensity inference from quiz-start phrasing

Not acceptable:

- encoding the full quiz session into free-form state strings
- making core quiz transitions depend on prompt text
- inferring durable quiz progress from conversation history alone

In other words:

- edge adapters may parse
- runtime state should stay typed and persisted

That boundary is intentional and should be preserved.

## Grading model

Quiz mode currently uses deterministic grading for OSS-backed assessments.

That means:

- correct answers advance immediately
- wrong answers keep the learner on the same question
- hints come from assessment data
- completion summary is generated without an LLM

This is cheaper, faster, and more stable than using the tutor model for every quiz turn.

LLM-based structured generation still has a role for future dynamic quiz generation, but the live OSS-backed answer loop should stay deterministic unless there is a very strong reason to change it.

## Progress and XP integration

Quiz mode reuses the existing learning systems.

It does not maintain a separate quiz-only ledger for learning state.

Current integration:

- correct answers award quiz XP
- quiz outcomes update topic mastery
- crossing mastery threshold can award mastery XP
- paused or side-conversation turns do not award or deduct quiz progress signals
- `/progress` reflects the same shared state

This matters because quiz should deepen the same learner model already used by:

- progress reporting
- review scheduling
- nudges
- streak and XP display

If quiz wrote to a disconnected subsystem, the product would feel fragmented.

## Why not a slash-command-first design

`/quiz` can still exist as a convenience surface later, but it should not be required for the core experience.

Natural entry is better because:

- students already think in chat language
- it lowers command burden
- it feels closer to a real tutor
- it matches how the rest of the product works

The runtime still stays explicit.
Only the entry surface is conversational.

## When to change this design

You should revisit this design only if one of these becomes true:

- deterministic grading is no longer good enough for the question set
- dynamic question generation becomes the dominant source of quiz content
- multiple simultaneous assessment modes need deeper state separation
- channel-specific quiz UI requires a more explicit action contract

Even then, keep the same principles:

- route before AI
- store durable state explicitly
- keep parsing at the edge
- reuse the shared progress model

## Implementation guidance

If you are working on quiz mode, prefer these rules:

1. Treat quiz as runtime mode, not prompt flavor.
2. Persist control state explicitly.
3. Keep deterministic grading for OSS-backed questions.
4. Use the shared progress and XP trackers.
5. Keep parser logic at the boundary only.
6. If the feature touches session/state/runtime design, re-check the relevant OpenClaw concepts before coding.

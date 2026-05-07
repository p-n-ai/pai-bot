# P&AI Bot

This context names the tutoring runtime concepts used by P&AI Bot. It keeps agent, learner, and operator language precise when shaping runtime behavior.

## Language

**Tutor Turn**:
One learner-facing message cycle that reaches the tutor model path.
_Avoid_: request, chat call

**Turn Hook**:
An internal runtime extension point that observes or shapes a **Tutor Turn** without becoming a user-installed extension system.
_Avoid_: React hook, Git hook, Codex hook, plugin

**Hook Outcome**:
The limited result a **Turn Hook** may return: continue the turn, block the turn, or inject trace-safe context.
_Avoid_: arbitrary prompt rewrite, dynamic extension result

**Behavior-Preserving Turn Hook**:
A **Turn Hook** that moves existing tutor-runtime behavior behind the hook seam without changing learner-facing behavior.
_Avoid_: new policy hook, feature hook

**Turn Hook Rollout Flag**:
A `PAI_FEATURES=turn_hooks` flag that controls whether **Turn Hooks** run at all while the hook seam is being delivered.
_Avoid_: per-hook flag, public hook setting

**Rate Conversation Turn Hook**:
A **Behavior-Preserving Turn Hook** that adds the existing conversation-rating prompt when the **Tutor Turn** is already due for rating.
Use code slug `rate_convo_hook`.
_Avoid_: rating_prompt_context, rate_convo_hooks

**Turn Hook Catalog**:
The internal list of **Turn Hooks** that run when the **Turn Hook Rollout Flag** is enabled.
_Avoid_: plugin registry, dynamic hook config

**Hook Call Notice**:
A development-only Terminal Chat console line that prints a **Turn Hook** name and **Hook Outcome**.
_Avoid_: hook artifact, hook dump, model-visible hook trace

## Relationships

- A **Tutor Turn** may pass through zero or more **Turn Hooks**.
- A **Turn Hook** belongs to the internal tutoring runtime, not to external users or tenant configuration.
- A **Turn Hook** returns exactly one **Hook Outcome**.
- A **Behavior-Preserving Turn Hook** should keep the same learner-visible response path unless the existing behavior already blocked the turn.
- The **Turn Hook Rollout Flag** controls the hook seam, not individual hooks.
- The **Rate Conversation Turn Hook** does not decide when rating is due; it only turns that existing decision into hook-shaped context.
- The **Turn Hook Catalog** controls which hooks run and in what order.
- The first **Turn Hook Catalog** should contain only the **Rate Conversation Turn Hook**.
- A **Hook Call Notice** may be printed by Terminal Chat only when developer mode is enabled and the **Turn Hook Rollout Flag** is enabled, but it must not be sent to the model, saved to chat history, or persisted.
- When **Hook Call Notices** are enabled, every **Turn Hook** call prints one short notice, including `continue` outcomes.

## Example dialogue

> **Dev:** "Should this be a plugin?"
> **Domain expert:** "No. For now it is a **Turn Hook** inside the **Tutor Turn** runtime, so we can keep tutor safety and trace rules local."

> **Dev:** "How do I add a hook?"
> **Domain expert:** "Add it to the **Turn Hook Catalog**. Do not add a per-hook flag."

## Flagged ambiguities

- "hooks" was used ambiguously across React hooks, Codex hooks, Git hooks, and agent harness hooks — resolved: in this work, "hooks" means **Turn Hooks**.

## Local PRD: Feature-Flagged Turn Hook Seam

### Problem Statement

P&AI Bot needs a small internal **Turn Hook** seam for the **Tutor Turn** runtime, but without turning hooks into plugins, YAML config, user settings, or model-visible metadata. Current rating prompt behavior is mixed into context loading, making it hard to add/remove runtime shaping behavior cleanly.

### Solution

Ship **Turn Hooks** behind `PAI_FEATURES=turn_hooks`.

First hook only: **Rate Conversation Turn Hook** with slug `rate_convo_hook`.

It moves existing rating prompt injection behind the hook seam while preserving learner-visible behavior. When the flag is off, existing behavior remains. When the flag is on, the **Turn Hook Catalog** runs and `rate_convo_hook` injects the same trace-safe rating prompt context only when the existing runtime already says rating is due.

### User Stories

1. As a developer, I want **Turn Hooks** disabled by default, so that hook code can land safely.
2. As a developer, I want one rollout flag, so that enabling/disabling the seam is simple.
3. As a developer, I want hooks added through a **Turn Hook Catalog**, so that add/remove is one obvious place.
4. As a developer, I want no per-hook flags, so that runtime behavior does not become config sprawl.
5. As a tutor-runtime maintainer, I want `rate_convo_hook` to preserve existing behavior, so that learners see no regression.
6. As a tutor-runtime maintainer, I want rating-due logic outside the hook, so that the hook only shapes context.
7. As a prompt reviewer, I want hook-injected context to use existing context packet validation, so that trust rules still hold.
8. As an operator, I want no hook data persisted, so that hook visibility does not create a privacy surface.
9. As a CLI developer, I want short **Hook Call Notices** in Terminal Chat, so that I can see hooks ran during dev.
10. As a CLI developer, I want every hook call noticed, including `continue`, so that silent no-op hooks are still observable.
11. As a privacy reviewer, I want hook notices to include only name and outcome, so that learner text never leaks.
12. As a future developer, I want `block`, `inject`, and `continue` outcomes available, so that later internal hooks have a stable shape.

### Implementation Decisions

- Add `turn_hooks` as a known feature flag, default off.
- Treat this as an internal rollout exception to the current learner-facing feature flag wording.
- Add a private **Tutor Turn** hook runner module.
- Define **Hook Outcome** as `continue`, `block`, or `inject trace-safe context`.
- Add a private **Turn Hook Catalog** with one initial entry: `rate_convo_hook`.
- Move rating prompt packet creation out of base context loading into `rate_convo_hook`.
- Keep rating-due calculation in the existing **Tutor Turn** flow.
- Run hooks after base context is available and before prompt compilation.
- Keep hook-injected packets under the same validation and trace contract as normal context packets.
- Add Terminal Chat-only **Hook Call Notice** when dev mode and `turn_hooks` are enabled.
- Do not send hook name/outcome to the model.
- Do not persist hook notices to chat history, event telemetry, or request dumps.

### Testing Decisions

- Test feature flag parsing: known flag, default false, explicit true/false, unknown failure.
- Test flag off path preserves existing rating prompt behavior.
- Test flag on path runs `rate_convo_hook`.
- Test `rate_convo_hook` injects rating prompt only when rating is already due.
- Test non-rating turns continue without injection.
- Test fake hooks for runner outcomes: `continue`, `inject`, `block`.
- Test hook ordering through the catalog.
- Test invalid injected context is rejected by existing packet validation.
- Test Terminal Chat notices only appear with dev mode plus `turn_hooks`.
- Test notices include every hook call and only hook name plus outcome.

### Out of Scope

- NSFW/content blocker hooks.
- Public plugin system.
- YAML/dynamic hook config.
- Per-hook feature flags.
- Admin UI/debug UI.
- Persistent hook artifacts.
- Model-visible hook metadata.
- Provider request rewriting.
- Multiple production hooks in first slice.

### Further Notes

This PRD's main architecture choice: make **Turn Hook** a deep internal module with a small interface. The leverage is clean add/remove behavior through the **Turn Hook Catalog**, while locality stays inside **Tutor Turn** runtime. The first shipment should prove the seam with boring existing behavior, not invent new policy.

## Local Issue Breakdown: Turn Hook Tracer Bullets

### 1. Add Turn Hook Rollout Flag

**Type**: AFK

**Blocked by**: None

**User stories covered**: 1, 2

**What to build**: `PAI_FEATURES=turn_hooks` is known, default off, documented as an internal rollout exception. No **Tutor Turn** behavior changes yet.

**Acceptance criteria**:

- [ ] `turn_hooks` is a known feature flag with default disabled behavior.
- [ ] `PAI_FEATURES=turn_hooks` and explicit boolean overrides parse correctly.
- [ ] Unknown feature names still fail config load.
- [ ] Runtime docs explain why this internal rollout flag is allowed.

### 2. Add Private Turn Hook Runner

**Type**: AFK

**Blocked by**: 1. Add Turn Hook Rollout Flag

**User stories covered**: 3, 4, 12

**What to build**: **Tutor Turn** can run an internal **Turn Hook Catalog** when the **Turn Hook Rollout Flag** is enabled. Runner supports `continue`, `inject`, and `block` with test-only hooks. No production hook yet.

**Acceptance criteria**:

- [ ] The **Turn Hook Catalog** is private and ordered.
- [ ] The hook runner is skipped when `turn_hooks` is disabled.
- [ ] Test hooks cover `continue`, `inject`, and `block` outcomes.
- [ ] Hook-injected context still goes through existing packet validation.
- [ ] No per-hook flag or dynamic config is introduced.

### 3. Move Rating Prompt Into `rate_convo_hook`

**Type**: AFK

**Blocked by**: 2. Add Private Turn Hook Runner

**User stories covered**: 5, 6, 7

**What to build**: Existing rating prompt behavior becomes the first catalog hook. Flag off preserves old path. Flag on uses hook path. Prompt output stays equivalent.

**Acceptance criteria**:

- [ ] The first production **Turn Hook Catalog** contains only `rate_convo_hook`.
- [ ] `rate_convo_hook` does not decide when rating is due.
- [ ] When rating is due and `turn_hooks` is enabled, `rate_convo_hook` injects the existing rating prompt context.
- [ ] When rating is not due, `rate_convo_hook` returns `continue`.
- [ ] Behavior with the flag off remains equivalent to current learner-visible behavior.
- [ ] Prompt-shape tests prove the rating instruction still appears in the expected place.

### 4. Add Terminal Chat Hook Call Notices

**Type**: AFK

**Blocked by**: 3. Move Rating Prompt Into `rate_convo_hook`

**User stories covered**: 8, 9, 10, 11

**What to build**: Terminal Chat prints one short line per hook call only when `LEARN_DEV_MODE=true` and `PAI_FEATURES=turn_hooks`. The notice includes hook name and outcome only. It is not persisted.

**Acceptance criteria**:

- [ ] Terminal Chat prints a **Hook Call Notice** for every **Turn Hook** call when dev mode and `turn_hooks` are enabled.
- [ ] Notices include only hook name and **Hook Outcome**.
- [ ] `continue` outcomes are printed too.
- [ ] Notices do not appear when dev mode is off.
- [ ] Notices do not appear when `turn_hooks` is off.
- [ ] Notices are not sent to the model, saved to chat history, or persisted in request dumps/events.

### 5. Document Turn Hook Operating Contract

**Type**: AFK

**Blocked by**: 3. Move Rating Prompt Into `rate_convo_hook`

**User stories covered**: 3, 4, 7, 8, 12

**What to build**: Runtime docs explain **Turn Hook**, **Hook Outcome**, **Turn Hook Catalog**, **Turn Hook Rollout Flag**, privacy boundary, and how to add/remove hooks.

**Acceptance criteria**:

- [ ] Runtime docs use the exact domain terms from this context.
- [ ] Docs explain that **Turn Hooks** are internal, not plugins.
- [ ] Docs explain add/remove through **Turn Hook Catalog** only.
- [ ] Docs explain why no per-hook flags exist.
- [ ] Docs explain the **Hook Call Notice** privacy boundary.
- [ ] Docs note the first catalog contains only `rate_convo_hook`.

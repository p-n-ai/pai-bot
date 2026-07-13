---
title: "Feature Flags"
summary: "Feature flag model for learner-facing product experiments and internal rollout seams, including PAI_FEATURES syntax, goals, and non-goals."
read_when:
  - You are adding or consuming a learner-facing product feature flag.
  - You are adding or consuming an internal rollout feature flag.
  - You are changing PAI_FEATURES parsing or internal/platform/featureflags.
  - You are deciding whether a deploy-time switch is a feature flag or runtime toggle.
---

# Feature Flags

## Context

Feature flags in pai-bot are deploy-time controls for learner-facing product experiments and narrowly scoped internal rollout seams. They are separate from runtime toggles such as dev mode, provider enablement, channel enablement, and local development shortcuts.

`PAI_FEATURES` is the single deploy-time override surface for known feature flags. It accepts comma-separated overrides:

```env
PAI_FEATURES=turn_hooks
PAI_FEATURES=turn_hooks=true
PAI_FEATURES=turn_hooks=false
```

`turn_hooks` is shorthand for `turn_hooks=true`. Unknown feature names, invalid boolean values, and duplicate overrides for the same feature fail config load.

The current registry contains two internal rollout flags:

| Flag | Default | Status | Owner | Behavior |
|---|---:|---|---|---|
| `turn_hooks` | off | `under_development` | Tutor Turn runtime | Enables the internal **Turn Hook** runner and currently empty **Turn Hook Catalog**. |
| `agent_core` | off | `under_development` | Tutor Turn runtime | Enables the native model → sequential tool → model continuation loop for teaching turns. |

Read [Turn Hooks](turn-hooks.md) before adding, removing, or reviewing hook behavior.

## Dev Mode

`LEARN_DEV_MODE` and `PAI_FEATURES` control different things.

`LEARN_DEV_MODE=true` is an operational runtime mode. It lets the process run with development shortcuts, such as relaxed startup requirements or development-only auth behavior.

`PAI_FEATURES=turn_hooks=true` is a deploy-time behavior decision. It enables the internal **Turn Hook** seam for the tutor model path.

Keep them separate so developers can run any combination:

```env
LEARN_DEV_MODE=true
PAI_FEATURES=
```

Run locally with dev shortcuts, but no product experiment active.

```env
LEARN_DEV_MODE=false
PAI_FEATURES=turn_hooks=true
```

Run with `LEARN_DEV_MODE=false` and the **Turn Hook** seam active.

When developing a feature flag locally, use both only when both meanings are needed:

```env
LEARN_DEV_MODE=true
PAI_FEATURES=turn_hooks=true
```

With `LEARN_DEV_MODE=true` and `PAI_FEATURES=turn_hooks`, Terminal Chat may print **Hook Call Notices**. Those notices are local console output only; they are not model context, chat history, request dumps, or durable events.

## Goals

- Keep product experiment and internal rollout flags separate from runtime toggles.
- Provide one deploy-time feature override surface through `PAI_FEATURES`.
- Hard-fail unknown feature names and invalid override values.
- Keep source state minimal: registry defaults plus deploy-time overrides.
- Derive the effective product feature set from defaults and overrides.
- Allow future feature status such as `under_development` and `stable` without mixing status with enabled state.

## Non-Goals

- Do not use feature flags for dev mode, provider enablement, channel enablement, or operational runtime switches.
- Do not add feature flags without a behavior that consumes them.
- Do not add rollout percentages, variants, tenant overrides, or per-feature config structs in this slice.
- Do not add hook-specific flags. The **Turn Hook Rollout Flag** controls the seam; the **Turn Hook Catalog** controls which hooks run.

## Adding a Feature Flag

Add a feature flag only with the behavior that consumes it. Do not add names to the registry as placeholders.

1. Name learner-facing flags after the learner-facing outcome; name internal rollout flags after the internal seam they control.
2. Add the flag to `internal/platform/featureflags` with its status and default.
3. Add or update config tests for `PAI_FEATURES` parsing when the new flag changes expected behavior.
4. Wire the effective feature set into the smallest runtime module that owns the learner-facing decision.
5. Check the flag at the decision point, not in `cmd/server` unless startup itself is the behavior.
6. Add behavior tests with the flag off and on.
7. Update this doc and `docs/ops/config.md` when syntax, defaults, or validation rules change.

Examples of good placement:

- A quiz-start experiment belongs near quiz-start routing.
- A tutor-response experiment belongs near the agent turn or prompt assembly decision.
- A channel-specific learner experience belongs in that channel runtime.
- The `turn_hooks` internal rollout flag belongs in the **Tutor Turn** runtime because it controls whether **Turn Hooks** run.

Examples of poor placement:

- Do not branch in `cmd/server` for product behavior that belongs in the agent.
- Do not put feature-flag checks inside `internal/platform/featureflags`; that package only parses and answers enabled state.
- Do not use feature flags as a substitute for runtime config validation.

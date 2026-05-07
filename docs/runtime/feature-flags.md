---
title: "Feature Flags"
summary: "Feature flag model for learner-facing product experiments, including PAI_FEATURES syntax, goals, and non-goals."
read_when:
  - You are adding or consuming a learner-facing product feature flag.
  - You are changing PAI_FEATURES parsing or internal/platform/featureflags.
  - You are deciding whether a deploy-time switch is a feature flag or runtime toggle.
---

# Feature Flags

## Context

Feature flags in pai-bot are deploy-time controls for learner-facing product experiments. They are separate from runtime toggles such as dev mode, provider enablement, channel enablement, and local development shortcuts.

`PAI_FEATURES` is the single deploy-time override surface for product feature flags. It accepts comma-separated overrides:

```env
PAI_FEATURES=some_feature
PAI_FEATURES=some_feature=true
PAI_FEATURES=some_feature=false
```

These example names are placeholders until the registry contains real product flags. `some_feature` is shorthand for `some_feature=true`. Unknown feature names, invalid boolean values, and duplicate overrides for the same feature fail config load.

The current infrastructure slice intentionally has an empty registry. That means `PAI_FEATURES=` passes, while any non-empty feature name fails until a real product behavior adds a known flag. No agent engine behavior consumes feature flags yet.

## Dev Mode

`LEARN_DEV_MODE` and `PAI_FEATURES` control different things.

`LEARN_DEV_MODE=true` is an operational runtime mode. It lets the process run with development shortcuts, such as relaxed startup requirements or development-only auth behavior.

`PAI_FEATURES=some_feature=true` is a product behavior decision. It enables a learner-facing experiment.

Keep them separate so developers can run any combination:

```env
LEARN_DEV_MODE=true
PAI_FEATURES=
```

Run locally with dev shortcuts, but no product experiment active.

```env
LEARN_DEV_MODE=false
PAI_FEATURES=some_feature=true
```

Run with `LEARN_DEV_MODE=false` and a specific experiment active.

When developing a feature flag locally, use both only when both meanings are needed:

```env
LEARN_DEV_MODE=true
PAI_FEATURES=some_feature=true
```

## Goals

- Keep product experiment flags separate from runtime toggles.
- Provide one deploy-time feature override surface through `PAI_FEATURES`.
- Hard-fail unknown feature names and invalid override values.
- Keep source state minimal: registry defaults plus deploy-time overrides.
- Derive the effective product feature set from defaults and overrides.
- Allow future feature status such as `in_development` and `stable` without mixing status with enabled state.

## Non-Goals

- Do not use feature flags for dev mode, provider enablement, channel enablement, or operational runtime switches.
- Do not wire feature flags into the agent engine before a learner-facing behavior needs them.
- Do not add rollout percentages, variants, tenant overrides, or per-feature config structs in this slice.
- Do not add hook-specific flags before the hook behavior slice exists.

## Adding a Feature Flag

Add a feature flag only with the product behavior that consumes it. Do not add names to the registry as placeholders.

1. Name the flag after the learner-facing outcome, not the internal mechanism.
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

Examples of poor placement:

- Do not branch in `cmd/server` for product behavior that belongs in the agent.
- Do not put feature-flag checks inside `internal/platform/featureflags`; that package only parses and answers enabled state.
- Do not use feature flags as a substitute for runtime config validation.

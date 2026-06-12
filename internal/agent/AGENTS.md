# TUTOR AGENT ENGINE

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Conversation state machine, pedagogy, quiz runtime, nudges, goals, group flows, and learner motivation.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Main turn handling | `engine.go`, `turn.go`, `turn_hooks.go` |
| Prompt behavior | `prompt_builder.go`, `tutor_behavior.go`, `tutor_personality.go` |
| Curriculum context | `context_loader.go`, `context_packets.go`, `context_resolver.go`, `curriculum_retriever.go` |
| Quiz flow | `quiz.go`, `quiz_runtime.go`, `quiz_router.go`, `quiz_generate.go`, `quiz_progress.go` |
| Spaced nudges | `scheduler.go`, `nudge_tracker_postgres.go`, `daily_summary.go` |
| Challenges/groups | `challenge*.go`, `group_*.go`, `weekly_leaderboard_test.go` |
| Learner goals/progression | `goals.go`, `milestones.go`, `topic_unlock.go`, `topics.go` |
| Persistence | `store.go`, `store_postgres.go`, `group_store*.go` |
| Dev commands | `dev_commands.go`, `challenge_command.go`, `group_commands.go` |

## CONVENTIONS

- Tests are behavior-specific; add regression tests beside the exact flow changed.
- AI-dependent tests use integration naming/build tags; unit tests stay deterministic.
- Malay/KSSM tutoring behavior lives in prompts/tests, not scattered literals.
- Generated quiz JSON goes through `ai.CompleteJSON`.
- Side effects use `TurnHooks` where the turn pipeline expects them.

## ANTI-PATTERNS

- No direct provider calls or provider type switches.
- No bypassing `TurnHooks` for progress, events, or notification-like side effects.
- No mastery/XP/streak mutation from terminal defaults unless explicitly enabled.
- No new prompt sections without updating behavior contract tests.
- No mixing group challenge persistence with single-user store paths.

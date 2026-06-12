# CONVERSATION HARNESS COMMAND

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

YAML-driven AI quality harness for scripted tutoring conversations and behavior checks.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Fixture schema/checks | `main.go` structs near top |
| Default fixture | `internal/agent/testdata/ai_quality_conversations.yaml` |
| Command tests | `main_test.go` |

## CONVENTIONS

- Checks should express user-observable tutoring behavior, not provider internals.
- Default fixture path stays stable for local and CI scripts.
- JSON output is for automation; keep fields stable.

## ANTI-PATTERNS

- No golden tests tied to exact model prose unless intentionally narrow.
- No real-provider call in regular unit tests.

## NOTES

- This command is a QA harness, not production runtime.
- Use broad behavior checks for model variance.
- Keep failure output actionable: case ID, turn, violated check.
- When prompts change, update fixtures/checks together.
- Do not hide fallback responses; harness should catch them.

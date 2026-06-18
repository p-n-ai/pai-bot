# RUNTIME BEHAVIOR DOCS

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Live bot/channel behavior contracts: Telegram, WhatsApp, embed, turn hooks, quizzes, challenges, harnesses, and tutor behavior.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Telegram behavior | `telegram.md` |
| WhatsApp behavior | `whatsapp.md` |
| Embed widget/runtime | `embeddable-chat.md` |
| OpenAPI/Scalar runtime | `openapi-scalar.md` |
| Turn API/hooks | `agent-turn-api.md`, `turn-hooks.md` |
| Quiz/challenge behavior | `quiz-mode.md`, `challenge-invite-slice.md` |
| Tutor behavior contract | `tutor-behavior-contract.md`, `ai-turn-harness.md` |

## CONVENTIONS

- Runtime docs describe observable behavior and operational flags.
- Keep command names aligned with `internal/chat/commands.go`.
- Harness docs must match fixture schema and CLI flags.

## ANTI-PATTERNS

- No channel-specific workaround documented as product contract unless tested.
- No prompt copy drift from `internal/agent` tests.

## NOTES

- Runtime docs should link back to tests or commands when behavior is verifiable.

# AI GATEWAY

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Provider-neutral completion gateway, model routing, token budget enforcement, image input, and structured output.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Gateway contracts | `gateway.go`, `mock.go` |
| Model routing/fallback | `router.go`, `router_test.go` |
| Token budgets | `budget.go`, `budget_test.go` |
| Structured JSON | `complete_json.go`, `structured_output*.go`, related tests |
| OpenAI/DeepSeek-compatible | `provider_openai.go` |
| Anthropic/Gemini/Ollama/OpenRouter | `provider_anthropic.go`, `provider_google.go`, `provider_ollama.go`, `provider_openrouter_llm_adapter.go` |
| Image inputs | `image_input.go` |

## CONVENTIONS

- Provider tests use local fake HTTP servers; real providers only in explicit integration tests.
- DeepSeek stays OpenAI-compatible config, not a separate provider type.
- Budget checks happen before provider calls; usage accounting happens after response when known.
- Structured-output helpers return typed errors callers can degrade from.
- Preserve multimodal message support when changing request shapes.

## ANTI-PATTERNS

- No provider imports in `internal/agent` or `internal/chat`.
- No silent fallback loops that hide all provider errors.
- No text-only assumptions in gateway contracts.
- No broad API key/config dumps in test failures.

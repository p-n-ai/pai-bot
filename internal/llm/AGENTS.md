# LLM PROVIDER PROTOCOL

**Generated:** 2026-07-11
**Commit:** bdd0c16

Provider-neutral request/response types, concurrent registry, streaming protocol, and provider adapters.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Public protocol and content types | `types.go` |
| Provider registry and lookup | `registry.go` |
| OpenAI/OpenRouter adapters | `openai.go`, `openrouter.go` |
| Streaming/SSE behavior | `stream.go`, provider tests |
| Deterministic provider behavior | `faux.go` |

## CONVENTIONS

- Preserve provider-neutral typed content, reasoning, tool, and multimodal shapes.
- Registry mutation and lookup remain concurrency-safe.
- Expected provider/stream failures are typed or wrapped with safe operational context.
- Adapter tests use fake HTTP servers; live-provider tests stay explicit integration tests.
- Do not add comments here; attribution belongs in `NOTICE`.

## ANTI-PATTERNS

- No provider-specific decisions in `internal/agent` or `internal/chat`.
- No text-only assumptions that discard images, tools, reasoning, or stream terminal state.
- No silent fallback loop that hides every provider error.
- No broad config, API-key, request, or response dumps.

## NOTES

- Product-level routing, budgets, and fallback policy remain in `internal/ai`.

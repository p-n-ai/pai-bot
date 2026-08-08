# OpenRouter

**Status:** configured and implemented.

- **Identity and protocol:** OpenRouter; OpenRouter-flavored OpenAI Chat Completions SSE.
- **Authentication:** `LEARN_AI_OPENROUTER_API_KEY` (Bearer token).
- **Base URL:** `https://openrouter.ai/api/v1`.
- **Model scope:** Default `qwen/qwen3-max`; arbitrary routed model IDs are accepted; no local full catalog.
- **Capabilities:** Native streaming transport, text/images, tools in native contract, usage/cache accounting, attribution headers.
- **pai-bot seam:** `internal/ai/provider_openrouter_llm_adapter.go` over `internal/llm`.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/openrouter.ts`, `openrouter.models.ts` (when present), shared API transports, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** The teaching `Complete` projection returns text and hides reasoning/tool calls; provider model availability is external.
- **Acceptance tests:** Accepted when the documented command succeeds: `go test ./internal/ai -run OpenRouter` and `go test ./internal/llm -run OpenRouter`.

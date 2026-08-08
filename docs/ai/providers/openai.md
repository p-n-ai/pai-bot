# OpenAI

**Status:** configured and implemented.

- **Identity and protocol:** OpenAI; OpenAI Chat Completions; native `llm` path also supports OpenAI Responses.
- **Authentication:** `LEARN_AI_OPENAI_API_KEY` (Bearer token).
- **Base URL:** `https://api.openai.com/v1`.
- **Model scope:** Provider default `gpt-4o`; `Models()` is the small in-code list, not discovery.
- **Capabilities:** Text chat, usage, images, JSON Schema structured output; native seam adds tools/streaming.
- **pai-bot seam:** `internal/ai/provider_openai.go`; configured in `internal/platform/config` and server wiring.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/openai.ts`, `openai.models.ts` (when present), shared API transports, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** The catalog is curated and can age; the simple `Complete` response only returns text.
- **Acceptance tests:** Accepted when the documented command succeeds: `go test ./internal/ai -run OpenAI`; live tests require the key.

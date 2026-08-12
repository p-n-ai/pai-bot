# DeepSeek

**Status:** configured and implemented.

- **Identity and protocol:** DeepSeek; OpenAI-compatible Chat Completions.
- **Authentication:** `LEARN_AI_DEEPSEEK_API_KEY` (Bearer token).
- **Base URL:** `https://api.deepseek.com`.
- **Model scope:** Default `deepseek-v4-flash`; curated catalog: `deepseek-v4-flash`, `deepseek-v4-pro`. There is no runtime discovery.
- **Capabilities:** Text chat and usage; JSON-object structured-output fallback.
- **pai-bot seam:** `NewDeepSeekProvider` in `internal/ai/provider_openai.go`; config/server registration.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/deepseek.ts`, `deepseek.models.ts` (when present), shared API transports, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** No vision claim; schema strictness is not transmitted, and stream is not incremental.
- **Acceptance tests:** Accepted when the documented command succeeds: `go test ./internal/ai -run DeepSeek`.

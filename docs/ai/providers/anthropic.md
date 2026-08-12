# Anthropic

**Status:** configured and implemented.

- **Identity and protocol:** Anthropic; Anthropic Messages API.
- **Authentication:** `LEARN_AI_ANTHROPIC_API_KEY` (`x-api-key`).
- **Base URL:** `https://api.anthropic.com/v1`.
- **Model scope:** Default `claude-sonnet-4-6`; in-code `Models()` scope only.
- **Capabilities:** Text, system instruction, URL/data images, usage, stable JSON-schema output config.
- **pai-bot seam:** `internal/ai/provider_anthropic.go`; config/server provider registration.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/anthropic.ts`, `anthropic.models.ts` (when present), shared API transports, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** pai-bot adapter does not expose Anthropic tools or true incremental streaming.
- **Acceptance tests:** Accepted when the documented command succeeds: `go test ./internal/ai -run Anthropic`.

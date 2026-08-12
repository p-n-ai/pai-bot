# OpenAI Codex

**Status:** implemented; opt-in.

- **Identity and protocol:** OpenAI Codex; ChatGPT Codex Responses SSE, or managed `codex app-server`.
- **Authentication:** `LEARN_AI_CODEX_ENABLED`, `LEARN_AI_CODEX_HOME`; optional access/refresh tokens and account ID variables.
- **Base URL:** `https://chatgpt.com/backend-api` (auth: `https://auth.openai.com`).
- **Model scope:** Configured default `LEARN_AI_CODEX_MODEL`, currently `gpt-5.4`; no discovery catalog.
- **Capabilities:** Streaming Responses parsing, text/images, tools, reasoning metadata, structured output, token refresh.
- **pai-bot seam:** `internal/ai/provider_codex.go`, `provider_codex_appserver.go`, and `internal/platform/codexauth`.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/openai-codex.ts`, `openai-codex.models.ts` (when present), shared API transports, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** Requires a dedicated Codex home or valid OAuth material; not an OpenAI API-key alias.
- **Acceptance tests:** Accepted when the documented command succeeds: `go test ./internal/ai -run Codex` and `go test ./internal/platform/codexauth`.

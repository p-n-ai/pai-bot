# Ollama

**Status:** implemented; opt-in local provider.

- **Identity and protocol:** Ollama; Ollama OpenAI-compatible `/v1/chat/completions`; native `/api/tags` health check.
- **Authentication:** No credential; `LEARN_AI_OLLAMA_ENABLED`, `LEARN_AI_OLLAMA_URL`, `LEARN_AI_OLLAMA_MODEL`.
- **Base URL:** Default `http://localhost:11434`.
- **Model scope:** Configured model overrides; adapter default/catalog entry is `qwen3`. Installed models are not imported into the catalog.
- **Capabilities:** Text chat, usage when returned, health check.
- **pai-bot seam:** `internal/ai/provider_ollama.go`; config/server registration.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/api/openai-completions.ts` and the provider/model registry patterns in `/tmp/pi-provider-review`; Pi has no first-class Ollama provider file. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** No tools, images, structured output, or incremental streaming in this adapter.
- **Acceptance tests:** Accepted when the documented command succeeds: `go test ./internal/ai -run Ollama`; acceptance also requires the selected model installed locally.

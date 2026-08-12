# Groq

**Status:** configured and implemented through the built-in catalog, with first-class runtime admin support.

- **Identity and protocol:** Groq; OpenAI-compatible Chat Completions.
- **Authentication:** `LEARN_AI_GROQ_API_KEY` (Bearer token); optional `LEARN_AI_GROQ_MODEL`.
- **Base URL:** `https://api.groq.com/openai/v1`.
- **Model scope:** Default `llama-3.3-70b-versatile`; curated catalog: `llama-3.1-8b-instant`, `llama-3.3-70b-versatile`, `openai/gpt-oss-120b`, `openai/gpt-oss-20b`, `openai/gpt-oss-safeguard-20b`, `qwen/qwen3.6-27b`. `LEARN_AI_GROQ_MODEL` overrides the default; there is no runtime discovery.
- **Capabilities:** Every curated model supports text chat. Llama models use JSON-object mode; GPT-OSS models use JSON-schema mode. No curated model advertises vision.
- **pai-bot seam:** `internal/ai/provider_catalog.go`; environment/runtime settings in `internal/platform/config` and `internal/platform/settings`; registration in `internal/platform/airouter/setup.go`; admin contract/UI in `internal/apidocs`, `internal/server`, and `admin-spa/src/components/settings/ai-settings-panel.tsx`.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/groq.ts`, `groq.models.ts`, generated `data/groq.json`, shared OpenAI Completions transport, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** Catalog providers do not implement pai-bot's native tool-continuation contract. The shared `Complete` stream remains a final buffered chunk.
- **Acceptance tests:** Accepted when backend catalog/config/router/settings and admin SPA settings tests pass; environment or encrypted runtime credentials register the named provider; a blank model selects the catalog default; and OpenAPI/UI selectors expose the provider without returning secret material.

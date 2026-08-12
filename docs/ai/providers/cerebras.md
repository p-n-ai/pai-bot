# Cerebras

**Status:** configured and implemented through the built-in catalog, with first-class runtime admin support.

- **Identity and protocol:** Cerebras; OpenAI-compatible Chat Completions.
- **Authentication:** `LEARN_AI_CEREBRAS_API_KEY` (Bearer token); optional `LEARN_AI_CEREBRAS_MODEL`.
- **Base URL:** `https://api.cerebras.ai/v1`.
- **Model scope:** Default `gpt-oss-120b`; curated catalog: `gemma-4-31b`, `gpt-oss-120b`, `zai-glm-4.7`. `LEARN_AI_CEREBRAS_MODEL` overrides the default; there is no runtime discovery.
- **Capabilities:** Catalog advertises chat, streaming, tools, structured output; vision is false.
- **pai-bot seam:** `internal/ai/provider_catalog.go`; environment/runtime settings in `internal/platform/config` and `internal/platform/settings`; registration in `internal/platform/airouter/setup.go`; admin contract/UI in `internal/apidocs`, `internal/server`, and `admin-spa/src/components/settings/ai-settings-panel.tsx`.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/cerebras.ts`, `cerebras.models.ts`, generated `data/cerebras.json`, shared OpenAI Completions transport, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** Catalog capability flags describe intended protocol compatibility; the shared `Complete` stream remains a final buffered chunk.
- **Acceptance tests:** Accepted when backend catalog/config/router/settings and admin SPA settings tests pass; environment or encrypted runtime credentials register the named provider; a blank model selects the catalog default; and OpenAPI/UI selectors expose the provider without returning secret material.

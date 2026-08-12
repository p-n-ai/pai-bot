# Mistral

**Status:** configured and implemented through the built-in catalog, with first-class runtime admin support.

- **Identity and protocol:** Mistral; OpenAI-compatible Chat Completions.
- **Authentication:** `LEARN_AI_MISTRAL_API_KEY` (Bearer token); optional `LEARN_AI_MISTRAL_MODEL`.
- **Base URL:** `https://api.mistral.ai/v1`.
- **Model scope:** Default `mistral-large-latest`; curated catalog: `codestral-latest`, `devstral-latest`, `magistral-medium-latest`, `magistral-small`, `mistral-large-latest`, `mistral-medium-latest`, `mistral-small-latest`, `pixtral-large-latest`. `LEARN_AI_MISTRAL_MODEL` overrides the default; there is no runtime discovery.
- **Capabilities:** Catalog advertises chat, streaming, tools, structured output, and vision.
- **pai-bot seam:** `internal/ai/provider_catalog.go`; environment/runtime settings in `internal/platform/config` and `internal/platform/settings`; registration in `internal/platform/airouter/setup.go`; admin contract/UI in `internal/apidocs`, `internal/server`, and `admin-spa/src/components/settings/ai-settings-panel.tsx`.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/mistral.ts`, `mistral.models.ts`, generated `data/mistral.json`, the native Mistral Conversations transport, and `env-api-keys.ts` in `/tmp/pi-provider-review`. pai-bot instead uses Mistral’s OpenAI-compatible endpoint. This is design provenance, not a dependency.
- **Compatibility gaps:** Catalog capability flags describe intended protocol compatibility; the shared `Complete` stream remains a final buffered chunk.
- **Acceptance tests:** Accepted when backend catalog/config/router/settings and admin SPA settings tests pass; environment or encrypted runtime credentials register the named provider; a blank model selects the catalog default; and OpenAPI/UI selectors expose the provider without returning secret material.

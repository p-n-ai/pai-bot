# xAI

**Status:** configured and implemented through the built-in catalog, with first-class runtime admin support.

- **Identity and protocol:** xAI; OpenAI-compatible Chat Completions.
- **Authentication:** `LEARN_AI_XAI_API_KEY` (Bearer token); optional `LEARN_AI_XAI_MODEL`.
- **Base URL:** `https://api.x.ai/v1`.
- **Model scope:** Default `grok-4.3`; curated catalog: `grok-4.3`, `grok-build-0.1`, `grok-4.5`. `LEARN_AI_XAI_MODEL` overrides the default; there is no runtime discovery.
- **Capabilities:** Catalog advertises chat, streaming, tools, structured output, and vision.
- **pai-bot seam:** `internal/ai/provider_catalog.go`; environment/runtime settings in `internal/platform/config` and `internal/platform/settings`; registration in `internal/platform/airouter/setup.go`; admin contract/UI in `internal/apidocs`, `internal/server`, and `admin-spa/src/components/settings/ai-settings-panel.tsx`.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/xai.ts`, `xai.models.ts`, generated `data/xai.json`, shared OpenAI transports, and `env-api-keys.ts` in `/tmp/pi-provider-review`. Pi supports both Completions and Responses plus OAuth; pai-bot uses API-key Completions only. This is design provenance, not a dependency.
- **Compatibility gaps:** Catalog capability flags describe intended protocol compatibility; the shared `Complete` stream remains a final buffered chunk.
- **Acceptance tests:** Accepted when backend catalog/config/router/settings and admin SPA settings tests pass; environment or encrypted runtime credentials register the named provider; a blank model selects the catalog default; and OpenAPI/UI selectors expose the provider without returning secret material.

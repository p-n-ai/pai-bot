# Google Gemini

**Status:** configured and implemented.

- **Identity and protocol:** Google Gemini; Gemini `generateContent` REST API.
- **Authentication:** `LEARN_AI_GOOGLE_API_KEY` (`key` query parameter).
- **Base URL:** `https://generativelanguage.googleapis.com/v1beta`.
- **Model scope:** Default `gemini-3-flash-preview`; small in-code model list, no runtime discovery.
- **Capabilities:** Text/system roles, usage, JSON schema, data images and fetched remote images.
- **pai-bot seam:** `internal/ai/provider_google.go`; config/server registration.
- **Pi mapping and provenance:** Pi reference: `packages/ai/src/providers/google.ts`, `google.models.ts` (when present), shared API transports, and `env-api-keys.ts` in `/tmp/pi-provider-review`. This is design provenance only; pai-bot does not import Pi.
- **Compatibility gaps:** No tools or true streaming; remote image fetching adds a separate network failure surface.
- **Acceptance tests:** Accepted when the documented command succeeds: `go test ./internal/ai -run Google`.

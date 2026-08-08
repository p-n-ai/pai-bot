# AI provider specifications

Use this index when adding, wiring, upgrading, or testing an AI provider. Read only the provider file for the branch you are changing; compare multiple files only for shared protocol work.

## Provider index

| Provider | pai-bot status | Specification |
| --- | --- | --- |
| OpenAI | Configured and implemented | [openai.md](openai.md) |
| Codex | Implemented; opt-in | [codex.md](codex.md) |
| Anthropic | Configured and implemented | [anthropic.md](anthropic.md) |
| DeepSeek | Configured and implemented | [deepseek.md](deepseek.md) |
| Google Gemini | Configured and implemented | [google.md](google.md) |
| Ollama | Implemented; opt-in local | [ollama.md](ollama.md) |
| OpenRouter | Configured and implemented | [openrouter.md](openrouter.md) |
| Groq | Configured, runtime-admin supported | [groq.md](groq.md) |
| xAI | Configured, runtime-admin supported | [xai.md](xai.md) |
| Mistral | Configured, runtime-admin supported | [mistral.md](mistral.md) |
| Cerebras | Configured, runtime-admin supported | [cerebras.md](cerebras.md) |

“Implemented” means a pai-bot adapter exists. “Configured” additionally means normal server configuration can register it. “Configured, runtime-admin supported” means environment or encrypted runtime settings can register the shared compatible adapter and the admin API/UI expose its controls without returning stored secrets.

## Specification template

Keep each provider fact and its caveat together. Use these fields in this order:

1. **Status** — distinguish adapter, catalog, configuration, and production readiness.
2. **Identity and protocol** — provider identity and actual wire family.
3. **Authentication** — pai-bot environment keys and wire authentication.
4. **Base URL** — the code default; label overrides.
5. **Model scope** — distinguish a request default, curated catalog, and discovery.
6. **Capabilities** — only behavior represented by the current adapter path.
7. **pai-bot seam** — implementation and registration files.
8. **Pi mapping and provenance** — exact reference files; Pi is comparative provenance, not a dependency.
9. **Compatibility gaps** — co-locate intended flags with end-to-end limitations.
10. **Acceptance tests** — commands and observable completion criteria.

A provider change is complete when its spec matches the branch, every claimed capability has a request/response contract test, configuration claims have registration tests, and secrets remain absent from fixtures and prose. For catalog-backed providers, a blank model setting must resolve to `ProviderDefinition.DefaultModel`. Model names are snapshots: update them from code or generated provenance rather than treating this directory as live provider discovery.

# AI Providers

P&AI Bot uses a provider-agnostic AI gateway. All AI calls go through a unified `Provider` interface with automatic fallback, circuit breaking, and budget enforcement.

## Supported Providers

| Provider | Env Variable | Chat Default | Structured Default | Notes |
|----------|-------------|-------------|-------------------|-------|
| **OpenAI** | `LEARN_AI_OPENAI_API_KEY` | `gpt-4o-mini` | `gpt-4o-mini` | Primary provider |
| **Anthropic** | `LEARN_AI_ANTHROPIC_API_KEY` | `claude-sonnet-4-6` | `claude-haiku-4-5-20251001` | Sonnet for teaching, Haiku for grading |
| **DeepSeek** | `LEARN_AI_DEEPSEEK_API_KEY` | (via OpenAI compat) | `deepseek-chat` | OpenAI-compatible API with custom base URL |
| **Google Gemini** | `LEARN_AI_GOOGLE_API_KEY` | `gemini-2.5-flash` | `gemini-2.5-flash` | Fast and cheap |
| **OpenRouter** | `LEARN_AI_OPENROUTER_API_KEY` | `qwen/qwen-2.5-72b-instruct` | `qwen/qwen-2.5-72b-instruct` | Access to 100+ models |
| **Ollama** | `LEARN_AI_OLLAMA_ENABLED=true` | `llama3` | — (not supported) | Self-hosted, free, last-resort fallback |

Chat defaults are set in each provider's `Complete()` method. Structured defaults are set centrally in `router.go` (`defaultStructuredModelForProvider`). DeepSeek reuses the OpenAI provider with a different base URL; its chat requests typically specify a model explicitly.

*At least one provider must be configured.*

## Fallback Chain

Providers are tried in registration order. If one fails, the next is attempted:

```
OpenAI → Anthropic → DeepSeek → Google → OpenRouter → Ollama
```

### Circuit Breaker

Each provider has an independent circuit breaker:
- **Threshold:** 3 consecutive failures triggers cooldown
- **Cooldown:** 30 seconds before retrying the provider
- **Retry backoff:** 1s → 2s → 4s between attempts

When a provider's circuit is open, it is skipped in the fallback chain.

## Task-Based Routing

Different tasks use different model tiers for cost optimization:

| Task | Preferred Models | Why |
|------|-----------------|-----|
| Teaching / Explanation | Claude Sonnet, GPT-4o, Gemini Pro | Best reasoning quality |
| Grading / Assessment | DeepSeek V3, GPT-4o-mini, Gemini Flash | Cheap, fast, structured output |
| Question Generation | Any with structured output | Uses `CompleteJSON` for schema-validated responses |
| Nudges | Any available | Simple text generation |

## Structured Output (`CompleteJSON`)

For tasks that need validated JSON (grading, quiz generation), the gateway provides `CompleteJSON`:

1. Takes a JSON schema and a Go struct to unmarshal into
2. Iterates the fallback chain, checking each provider's structured output capability
3. Uses provider-native structured output (OpenAI `response_format`, Anthropic `output_config`, etc.)
4. Validates the response against the JSON schema
5. Falls through to the next provider if validation fails

Ollama does not support structured output and is always skipped for `CompleteJSON` calls.

## Budget Enforcement

Token usage is tracked per tenant via an in-memory budget tracker (`InMemoryBudget` in `internal/ai/budget.go`). Dragonfly-based real-time tracking with periodic PostgreSQL sync is planned but not yet implemented.

- Admins can set token budget windows via the admin panel (`POST /api/admin/ai/budget-window`)
- When budget is exhausted, the gateway degrades to free models (Ollama) instead of cutting off the student
- Budget tracking is token-based, not USD-based

## Configuration

### Minimal Setup (One Provider)

```env
LEARN_AI_OPENAI_API_KEY=sk-...
```

### Full Setup (All Providers)

```env
LEARN_AI_OPENAI_API_KEY=sk-...
LEARN_AI_ANTHROPIC_API_KEY=sk-ant-...
LEARN_AI_DEEPSEEK_API_KEY=sk-...
LEARN_AI_GOOGLE_API_KEY=AIza...
LEARN_AI_OPENROUTER_API_KEY=sk-or-...
LEARN_AI_OLLAMA_ENABLED=true
LEARN_AI_OLLAMA_URL=http://localhost:11434
```

### Free Setup (Ollama Only)

```env
LEARN_AI_OLLAMA_ENABLED=true
LEARN_AI_OLLAMA_URL=http://localhost:11434
```

Run `just ollama-pull` to download the default model (`llama3`).

## Adding a New Provider

1. Implement the `Provider` interface in `internal/ai/provider_<name>.go`:

```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
    Models() []ModelInfo
    HealthCheck(ctx context.Context) error
}
```

2. For structured output (`CompleteJSON`) support, add the provider name to `structuredProviderCapabilities()` in `internal/ai/router.go` and add a default model entry in `defaultStructuredModelForProvider()`. Structured output capability is determined by provider name lookup, not by interface implementation.

3. Register the provider in `cmd/server/main.go`:

```go
router.Register("my-provider", ai.NewMyProvider(cfg.AI.MyProvider.APIKey))
```

4. Add config fields in `internal/platform/config/config.go` and `.env.example`.

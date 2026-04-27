---
title: "Configuration Reference"
summary: "Current environment variable reference for pai-bot server, database, cache, AI providers, email, Telegram, WhatsApp, auth, tenancy, logging, features, and curriculum path."
read_when:
  - You are changing internal/platform/config or environment variable behavior.
  - You are updating setup, deployment, AI provider, auth, WhatsApp, or local dev docs.
  - You need to know defaults and validation rules for pai-bot configuration.
---

# Configuration Reference

`internal/platform/config` loads runtime configuration from environment variables. Core app variables use `LEARN_`; auth variables use `PAI_AUTH_`.

## Server and infrastructure

| Env var | Default | Purpose |
|---|---:|---|
| `LEARN_SERVER_PORT` | `8080` | HTTP server port. |
| `LEARN_SERVER_HOST` | `0.0.0.0` | HTTP server host. |
| `LEARN_DATABASE_URL` | local Postgres URL | PostgreSQL connection string. |
| `LEARN_DATABASE_MAX_CONNS` | `25` | Max Postgres pool connections. |
| `LEARN_DATABASE_MIN_CONNS` | `5` | Min Postgres pool connections. |
| `LEARN_CACHE_URL` | `redis://localhost:6379` | Dragonfly/Redis URL. |
| `LEARN_NATS_URL` | `nats://localhost:4222` | NATS URL. |

## AI providers

| Env var | Default | Purpose |
|---|---:|---|
| `LEARN_AI_DEFAULT_PROVIDER` | empty | Optional default provider. Must be one of `openai`, `anthropic`, `deepseek`, `google`, `ollama`, `openrouter` when set. |
| `LEARN_AI_OPENAI_API_KEY` | empty | OpenAI API key. |
| `LEARN_AI_OPENAI_MODEL` | empty | OpenAI model override. |
| `LEARN_AI_ANTHROPIC_API_KEY` | empty | Anthropic API key. |
| `LEARN_AI_ANTHROPIC_MODEL` | empty | Anthropic model override. |
| `LEARN_AI_DEEPSEEK_API_KEY` | empty | DeepSeek API key. |
| `LEARN_AI_DEEPSEEK_MODEL` | empty | DeepSeek model override. |
| `LEARN_AI_GOOGLE_API_KEY` | empty | Google Gemini API key. |
| `LEARN_AI_GOOGLE_MODEL` | empty | Google model override. |
| `LEARN_AI_OLLAMA_ENABLED` | `false` | Enables Ollama as a provider. |
| `LEARN_AI_OLLAMA_URL` | `http://localhost:11434` | Ollama base URL. |
| `LEARN_AI_OLLAMA_MODEL` | empty | Ollama model override. |
| `LEARN_AI_OPENROUTER_API_KEY` | empty | OpenRouter API key. |
| `LEARN_AI_OPENROUTER_MODEL` | empty | OpenRouter model override. |

## Channels and auth

| Env var | Default | Purpose |
|---|---:|---|
| `LEARN_TELEGRAM_BOT_TOKEN` | empty | Telegram bot token. Required unless `LEARN_DEV_MODE=true`. |
| `LEARN_WHATSAPP_ENABLED` | `false` | Enables WhatsApp channel. |
| `LEARN_WHATSAPP_BACKEND` | `meow` | WhatsApp backend: `meow` or `cloudapi`. |
| `LEARN_WHATSAPP_ACCESS_TOKEN` | empty | Meta Cloud API access token. |
| `LEARN_WHATSAPP_PHONE_ID` | empty | Meta Cloud API phone number ID. |
| `LEARN_WHATSAPP_VERIFY_TOKEN` | empty | Meta webhook verification token. |
| `LEARN_WHATSAPP_MEOW_DB` | SQLite DSN | whatsmeow session DB path. |
| `LEARN_WHATSAPP_QR_TOKEN` | empty | Token for QR/admin WhatsApp flows. |
| `PAI_AUTH_SECRET` | `change-me-in-production` | JWT signing secret. |
| `PAI_AUTH_GOOGLE_CLIENT_ID` | empty | Google OAuth client ID. |
| `PAI_AUTH_GOOGLE_CLIENT_SECRET` | empty | Google OAuth client secret. |
| `PAI_AUTH_GOOGLE_ALLOWED_DOMAIN` | empty | Optional Google hosted-domain allowlist. |
| `PAI_AUTH_GOOGLE_DISCOVERY_URL` | Google discovery URL | OIDC discovery document. |
| `PAI_AUTH_GOOGLE_EMULATOR_SIGNING_SECRET` | empty | Local Google auth emulator signing secret. |
| `PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL` | `platform-admin@example.com` | Bootstrap platform admin email. |
| `PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD` | `demo-password` | Bootstrap platform admin password. |

## Email, tenancy, logging, features

| Env var | Default | Purpose |
|---|---:|---|
| `LEARN_EMAIL_SMTP_ADDR` | empty | SMTP server address. Required when email delivery is configured. |
| `LEARN_EMAIL_SMTP_USERNAME` | empty | SMTP username. |
| `LEARN_EMAIL_SMTP_PASSWORD` | empty | SMTP password. |
| `LEARN_EMAIL_FROM_ADDRESS` | empty | From address. Required when email delivery is configured. |
| `LEARN_EMAIL_FROM_NAME` | `P&AI Bot` | From name. |
| `LEARN_EMAIL_BASE_URL` | empty | Base URL for invite emails. |
| `LEARN_TENANT_MODE` | `single` | Must be `single` or `multi`. |
| `LEARN_LOG_LEVEL` | `info` | Log level. |
| `LEARN_LOG_FORMAT` | `json` | Log format. |
| `LEARN_DEV_MODE` | `false` | Relaxes provider/token requirements for local dev. |
| `LEARN_DISABLE_MULTI_LANGUAGE` | `false` | Disables multilingual behavior. |
| `LEARN_RATING_PROMPT_EVERY_REPLIES` | `5` | Rating prompt cadence. |
| `LEARN_AI_PERSONALIZED_NUDGES_ENABLED` | `true` | Enables AI-personalized nudges. |
| `LEARN_AI_NUDGES_ENABLED` | fallback only | Legacy fallback for personalized nudges. |
| `LEARN_CURRICULUM_PATH` | `./oss` | Curriculum source path. |

## Validation rules

- `LEARN_TELEGRAM_BOT_TOKEN` is required unless `LEARN_DEV_MODE=true`.
- At least one AI provider is required unless `LEARN_DEV_MODE=true`.
- `LEARN_AI_DEFAULT_PROVIDER`, when set, must be a known provider.
- `LEARN_TENANT_MODE` must be `single` or `multi`.
- Partial email configuration requires `LEARN_EMAIL_SMTP_ADDR` and `LEARN_EMAIL_FROM_ADDRESS`.

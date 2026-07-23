---
title: "Configuration"
sidebar:
  order: 3
description: "Environment variables, AI provider setup, and feature flags."
---

P&AI Bot is configured entirely through environment variables, all prefixed with `LEARN_` (or `PAI_AUTH_` for authentication).

## Required Variables

| Variable | Description |
|----------|-------------|
| `LEARN_TELEGRAM_BOT_TOKEN` | Telegram Bot API token from @BotFather |

At least one AI provider must be configured (see below).

## AI Provider Configuration

Each provider needs an API key and optionally a model override:

| Provider | API Key Variable | Model Variable | Default Model |
|----------|-----------------|----------------|---------------|
| OpenAI | `LEARN_AI_OPENAI_API_KEY` | `LEARN_AI_OPENAI_MODEL` | gpt-4o |
| Anthropic | `LEARN_AI_ANTHROPIC_API_KEY` | `LEARN_AI_ANTHROPIC_MODEL` | claude-sonnet |
| Google Gemini | `LEARN_AI_GOOGLE_API_KEY` | `LEARN_AI_GOOGLE_MODEL` | gemini-pro |
| DeepSeek | `LEARN_AI_DEEPSEEK_API_KEY` | `LEARN_AI_DEEPSEEK_MODEL` | deepseek-chat |
| OpenRouter | `LEARN_AI_OPENROUTER_API_KEY` | `LEARN_AI_OPENROUTER_MODEL` | — |
| Ollama | `LEARN_AI_OLLAMA_ENABLED=true` | `LEARN_AI_OLLAMA_MODEL` | llama3 |

Set `LEARN_AI_DEFAULT_PROVIDER` to choose which provider handles requests by default. The router automatically falls back to other configured providers if the primary fails.

## Infrastructure

| Variable | Default | Description |
|----------|---------|-------------|
| `LEARN_SERVER_PORT` | `8080` | HTTP server port |
| `LEARN_SERVER_HOST` | `0.0.0.0` | HTTP server bind address |
| `LEARN_DATABASE_URL` | `postgres://localhost:5432/pai` | PostgreSQL connection string |
| `LEARN_CACHE_URL` | `redis://localhost:6379` | Dragonfly/Redis connection |

## WhatsApp (Optional)

| Variable | Description |
|----------|-------------|
| `LEARN_WHATSAPP_ENABLED` | Set to `true` to enable |
| `LEARN_WHATSAPP_ACCESS_TOKEN` | Cloud API access token |
| `LEARN_WHATSAPP_PHONE_ID` | WhatsApp Business phone number ID |
| `LEARN_WHATSAPP_VERIFY_TOKEN` | Webhook verification token |

## Authentication

| Variable | Description |
|----------|-------------|
| `PAI_AUTH_SECRET` | JWT signing secret |
| `PAI_AUTH_GOOGLE_CLIENT_ID` | Google OAuth client ID (admin panel) |
| `PAI_AUTH_GOOGLE_CLIENT_SECRET` | Google OAuth client secret |

The admin SPA shows Google sign-in automatically when both Google credentials
are configured. Register `/api/auth/google/callback` on the public admin origin
as an authorized redirect URI in Google Cloud.

## Feature Flags

| Variable | Default | Description |
|----------|---------|-------------|
| `LEARN_DISABLE_MULTI_LANGUAGE` | `false` | Disable language selection |
| `LEARN_AI_PERSONALIZED_NUDGES_ENABLED` | `true` | Use AI for nudge personalization |
| `LEARN_DEV_MODE` | `false` | Enable dev commands |
| `LEARN_TENANT_MODE` | `single` | `single` or `multi` tenant mode |

## Email (Optional)

| Variable | Description |
|----------|-------------|
| `LEARN_EMAIL_SMTP_ADDR` | SMTP server address (e.g. `smtp.gmail.com:587`) |
| `LEARN_EMAIL_FROM_ADDRESS` | Sender email address |
| `LEARN_EMAIL_FROM_NAME` | Sender display name |

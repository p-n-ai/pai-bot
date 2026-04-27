---
title: "Telegram Runtime"
summary: "Current Telegram Bot API runtime behavior, command registration, media handling, and backend ownership for pai-bot."
read_when:
  - You are changing Telegram runtime behavior, Bot API polling, command menus, inline keyboards, media handling, or proactive nudges.
  - You are changing LEARN_TELEGRAM_BOT_TOKEN or Telegram validation behavior.
  - You need to distinguish Telegram transport behavior from WhatsApp or embedded chat behavior.
---

# Telegram Runtime

Telegram is the primary chat channel for students. The server registers it when `LEARN_TELEGRAM_BOT_TOKEN` is set. In dev mode, the config validator allows the token to be omitted so local HTTP/admin work can run without a real bot.

## Runtime Flow

| Step | Code | Notes |
|---|---|---|
| Configure | `internal/platform/config/config.go` | Loads `LEARN_TELEGRAM_BOT_TOKEN`; required unless `LEARN_DEV_MODE=true`. |
| Register channel | `cmd/server/main.go` | Creates `chat.NewTelegramChannel`, sets dev command mode, and registers channel name `telegram`. |
| Sync commands | `internal/chat/telegram.go` | Calls Telegram `setMyCommands` during startup. |
| Receive messages | `internal/chat/telegram.go` | Uses Bot API long polling through `getUpdates`; no webhook route is registered. |
| Normalize inbound | `internal/chat/telegram.go` | Maps messages, captions, callbacks, reply context, profile names, language, and image metadata into `chat.InboundMessage`. |
| Send replies | `internal/chat/telegram.go` | Sends Bot API `sendMessage`, splits overlong text at 4096 chars, and retries plain text if Markdown parsing fails. |
| Format output | `cmd/server/main.go` | Applies Telegram Markdown normalization and Telegram-specific reply/inline keyboards. |

## Environment

| Env var | Purpose |
|---|---|
| `LEARN_TELEGRAM_BOT_TOKEN` | Bot token from BotFather. Required unless `LEARN_DEV_MODE=true`. |
| `LEARN_DEV_MODE` | Allows local startup without Telegram token and exposes dev-only bot commands when the channel is enabled. |

## Commands

Bot commands are centralized in `internal/chat/commands.go`.

| Command set | Code | Notes |
|---|---|---|
| Student commands | `RegisteredCommands` | `help`, `start`, `clear`, `language`, `progress`, `goal`, `learn`, `create_group`, `join`, `leaderboard`, `challenge`. |
| Dev commands | `DevCommands` | `dev_reset`, `dev_boost`, `dev_close_group`; included only when dev mode is enabled. |
| Command merge | `AllCommands(devMode)` | Source used by Telegram startup sync. |

When adding a new command, add it to `internal/chat/commands.go` so Telegram autocomplete stays aligned with runtime behavior.

## Message Capabilities

| Capability | Current behavior |
|---|---|
| Text and captions | Empty text falls back to photo/document caption when present. |
| Images | Photos and image documents are fetched through Telegram `getFile`, downloaded, and attached as an image data URL for AI input. |
| Reply context | Replies carry the replied text/caption; replies to prior media can reuse the image context. |
| Inline callbacks | Callback data becomes inbound text, callback queries are acknowledged, and rating prompts are deduped per chat/message. |
| Reply keyboard | Runtime can attach persistent resized Telegram reply keyboards. |
| Inline keyboard | Runtime can attach inline buttons, especially for rating/progress interactions. |
| Long replies | Replies are split into Bot API sized chunks before sending. |

## Admin And Proactive Nudge Notes

Manual student nudges are Telegram-only today. `cmd/server/main.go` rejects the admin nudge path unless the student channel is `telegram` and the stored external ID is a real Telegram chat ID.

## Ownership

- Server wiring and output formatting: `cmd/server/main.go`
- Telegram adapter: `internal/chat/telegram.go`
- Command catalog: `internal/chat/commands.go`
- Telegram markdown/keyboards: `internal/chat/formatting.go`, `internal/chat/keyboards.go`
- Config loading/validation: `internal/platform/config/config.go`

## Update Rules

- Update `docs/ops/config.md` when adding or changing Telegram environment variables.
- Update this doc and `internal/chat/commands.go` together when command behavior changes.
- Keep channel behavior behind the `chat.Channel` interface.
- If Telegram-specific admin APIs are added, update `docs/admin/routes.md`.

# CHAT GATEWAYS

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Channel adapters for Telegram, WhatsApp, WebSocket, and embeddable chat plus command metadata.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Slash command list/autocomplete | `commands.go` |
| Telegram inbound/outbound | `telegram.go`, `telegram_*_test.go` |
| WhatsApp runtime | `whatsapp.go`, `whatsapp_meow.go`, `whatsapp_test.go` |
| WebSocket chat | `websocket.go`, `websocket_test.go` |
| Embeddable widget API | `embed_handler.go`, `embed_config.go`, `embed_ratelimit.go` |
| Message formatting/keyboards | `formatting.go`, `inline_keyboard.go`, `reply_keyboard.go` |
| Agent handoff | `gateway.go` |

## CONVENTIONS

- New bot commands go in `RegisteredCommands`; dev-only commands go in `DevCommands`.
- Channel structs implement shared gateway/channel contracts; agent logic remains channel-neutral.
- Telegram command sync happens on startup; command list changes need tests.
- Embed rate limits use cache-compatible behavior and degrade safely.

## ANTI-PATTERNS

- No tutoring decisions in channel adapters.
- No channel-specific command names unless product explicitly needs them.
- No direct Telegram/WhatsApp API calls from `internal/agent`.
- No rate-limit failure that blocks normal embed operation when cache is unavailable.

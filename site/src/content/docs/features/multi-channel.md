---
title: "Multi-Channel"
sidebar:
  order: 7
description: "Telegram, WhatsApp, and embeddable web chat."
---

P&AI Bot works across multiple messaging platforms. Students get the same tutoring experience regardless of which channel they use.

## Telegram

The primary channel. Features include:
- Long polling for reliable message delivery
- Inline keyboard buttons for quizzes, language selection, and ratings
- Command autocomplete menu (synced on bot startup)
- Markdown message formatting
- Automatic message splitting for responses exceeding 4,096 characters
- Typing indicators while the AI generates a response

## WhatsApp

WhatsApp Cloud API integration (behind the `LEARN_WHATSAPP_ENABLED` flag):
- Webhook-based message receiving (GET verify + POST inbound)
- Text message sending via Cloud API v21.0
- Same tutoring pipeline as Telegram

Enable by setting `LEARN_WHATSAPP_ENABLED=true` along with `LEARN_WHATSAPP_ACCESS_TOKEN`, `LEARN_WHATSAPP_PHONE_ID`, and `LEARN_WHATSAPP_VERIFY_TOKEN`.

## Embeddable Web Chat

A drop-in JavaScript widget that lets any website host the tutor:

```html
<script
  src="https://your-server.com/embed/pai-chat.js"
  data-tenant="your-tenant-id"
  data-color="#9a4a1a"
  data-position="right"
  data-language="en">
</script>
```

Features:
- Sandboxed iframe for security
- WebSocket transport with automatic reconnection (exponential backoff 1s–30s)
- Offline message queue
- Message history (last 50 messages in localStorage)
- Typing indicator
- Customizable theme color, position, and language
- Per-tenant allowed-origin configuration
- Guest JWT authentication (1-hour tokens, tenant-scoped)

### Security

The web embed includes several security measures:
- Origin allowlist validation before WebSocket upgrade
- Per-IP handshake rate limiting (10/min)
- Per-user message rate limiting (30/min)
- Content filtering for prompt injection attempts
- CSP `frame-ancestors` headers
- 8KB message size limit

## Channel Architecture

All channels implement the same `Channel` interface and feed into the same agent engine. This means:
- Same AI tutoring quality across all platforms
- Same progress tracking and mastery scoring
- Same quiz engine and challenge system
- Per-channel event logging for analytics

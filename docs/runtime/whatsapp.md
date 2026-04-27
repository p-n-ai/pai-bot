---
title: "WhatsApp Runtime"
summary: "Current WhatsApp runtime modes, admin setup routes, environment variables, and backend ownership for pai-bot."
read_when:
  - You are changing WhatsApp runtime behavior, setup UI, webhook handling, QR flow, or config.
  - You are changing LEARN_WHATSAPP_* environment variables.
  - You need to distinguish Meta Cloud API mode from whatsmeow mode.
---

# WhatsApp Runtime

WhatsApp is an optional chat channel. Enable it with `LEARN_WHATSAPP_ENABLED=true`.

## Runtime Flow

| Step | Cloud API | whatsmeow |
|---|---|---|
| Configure | `LEARN_WHATSAPP_BACKEND=cloudapi` plus access token, phone ID, verify token | `LEARN_WHATSAPP_BACKEND=meow` plus session DB path |
| Register channel | `cmd/server/main.go` registers channel name `whatsapp` | `cmd/server/main.go` registers channel name `whatsapp` |
| Receive messages | `/webhook/whatsapp` | Long-lived whatsmeow client session |
| Setup/admin | Meta app configuration outside this repo | Admin QR/status/disconnect routes and `/settings/whatsapp` |
| Session storage | Meta Cloud API account state | SQLite DSN from `LEARN_WHATSAPP_MEOW_DB` |

## Backends

| Backend | Env value | Code | Notes |
|---|---|---|---|
| Meta Cloud API | `cloudapi` | `internal/chat/whatsapp.go` | Uses access token, phone ID, verify token, and `/webhook/whatsapp`. |
| whatsmeow | `meow` | `internal/chat/whatsapp_meow.go` | Default. Stores session in SQLite DSN from `LEARN_WHATSAPP_MEOW_DB`. |

## Environment

| Env var | Purpose |
|---|---|
| `LEARN_WHATSAPP_ENABLED` | Enables WhatsApp channel registration. |
| `LEARN_WHATSAPP_BACKEND` | Selects `meow` or `cloudapi`. |
| `LEARN_WHATSAPP_ACCESS_TOKEN` | Cloud API access token. |
| `LEARN_WHATSAPP_PHONE_ID` | Cloud API phone number ID. |
| `LEARN_WHATSAPP_VERIFY_TOKEN` | Cloud API webhook verification token. |
| `LEARN_WHATSAPP_MEOW_DB` | whatsmeow session DB path. |
| `LEARN_WHATSAPP_QR_TOKEN` | Token for QR/admin setup flow. |

## Message Capabilities

| Capability | Current behavior |
|---|---|
| Text | Both backends map incoming WhatsApp text into `chat.InboundMessage`. |
| Channel name | Both backends use `whatsapp` so downstream agent behavior stays channel-agnostic. |
| Setup state | whatsmeow exposes admin-visible connection and QR state. |
| Cloud webhook verification | Cloud API backend verifies Meta webhook requests through the configured verify token. |

## Routes

| Route | Backend | Purpose |
|---|---|---|
| `/webhook/whatsapp` | Cloud API | Webhook verification and inbound messages. |
| `GET /api/admin/whatsapp/status` | whatsmeow | Admin status and QR state. |
| `POST /api/admin/whatsapp/disconnect` | whatsmeow | Logs out/disconnects current session. |
| `/settings/whatsapp` | admin app | WhatsApp setup/status page. |

## Ownership

- Server wiring: `cmd/server/main.go`
- Cloud API adapter: `internal/chat/whatsapp.go`
- whatsmeow adapter: `internal/chat/whatsapp_meow.go`
- Admin client helpers: `admin/src/lib/api.ts`
- Admin UI: `admin/src/app/settings/whatsapp/page.tsx`, `admin/src/components/whatsapp-setup-panel.tsx`

## Update rules

- Update `docs/ops/config.md` when adding or changing `LEARN_WHATSAPP_*`.
- Update `docs/admin/routes.md` when admin setup routes change.
- Keep this doc parallel with `docs/runtime/telegram.md` when shared chat channel behavior changes.
- Keep channel behavior behind the `chat.Channel` interface.

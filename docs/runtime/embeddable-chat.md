---
title: "Embeddable Chat Surface"
summary: "Current runtime contract for the pai-bot embeddable web chat widget, guest auth, WebSocket transport, admin embed config, and security boundaries."
read_when:
  - You are changing the embeddable chat widget, embed auth, WebSocket chat, or tenant origin checks.
  - You are changing /embed, /api/embed, /ws/chat, or /api/admin/embed routes.
  - You need to separate shipped embed behavior from future tawk.to-style improvements.
---

# Embeddable Chat Surface

`pai-bot` now has a shipped embeddable chat runtime surface. It is not just a future plan.

## Runtime Pieces

| Piece | Code |
|---|---|
| loader script | `internal/chat/embed/widget.js`, served at `GET /embed/pai-chat.js` |
| iframe chat shell | `internal/chat/embed/chat.html`, served at `GET /embed/chat` |
| static handlers | `internal/chat/embed_handler.go` |
| embed config store | `internal/chat/embed_config.go` |
| embed rate limit | `internal/chat/embed_ratelimit.go` |
| WebSocket channel | `internal/chat/websocket.go`, mounted at `GET /ws/chat` |
| guest auth and admin routes | `cmd/server/embed_admin.go` |
| server wiring | `cmd/server/main.go` |
| manual fixture | `scripts/test-embed.html` |

## Public Routes

| Route | Purpose |
|---|---|
| `GET /embed/pai-chat.js` | Host-page loader script. Creates launcher and iframe. |
| `GET /embed/chat?tenant=<slug>&color=<hex>&lang=<code>` | Iframe chat UI. |
| `POST /api/embed/auth/guest` | Issues a tenant-scoped guest JWT after origin and tenant validation. |
| `POST /api/embed/auth/upgrade` | Upgrades a guest user to a student account. |
| `GET /api/embed/messages?before=<cursor>&limit=20` | Returns authenticated message history. |
| `GET /ws/chat` | WebSocket transport. Embed clients authenticate with JWT subprotocol. |

## Admin Routes

| Route | Purpose |
|---|---|
| `GET /api/admin/embed/config` | Read tenant embed config. |
| `PUT /api/admin/embed/config` | Update enabled/theme settings. |
| `POST /api/admin/embed/origins` | Add an allowed origin. |
| `DELETE /api/admin/embed/origins` | Remove an allowed origin. |

Admin embed routes require admin or platform-admin role.

## Security Boundary

- Guest auth validates tenant slug plus request origin through `EmbedConfigStore`.
- Embed WebSocket connections require JWT auth.
- Embed WebSocket origin checks use tenant embed config.
- Embed guest auth and guest upgrade are CORS-enabled and IP-rate-limited.
- Message size is capped for embed connections.
- The WebSocket read loop applies simple prompt-injection content filtering for embed connections.

## Operational Notes

- The widget is tenant-routed by slug.
- Embed origins must be explicitly configured per tenant.
- The chat shell is intentionally minimal and iframe-based.
- Future polish can add widget events, unread state, resize events, richer theming, and admin UI for embed config.

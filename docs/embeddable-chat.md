---
title: "Embeddable Chat Surface"
summary: "Future-plan note for a tawk.to-style embeddable chat, grounded in what pai-bot already has and what is still missing."
read_when:
  - You are discussing an embeddable chat widget for external users, like tawk.to or Intercom.
  - You need to separate existing chat transport primitives from a real hosted chat widget product.
  - You are planning iframe chat, website chat embed, or third-party site integration.
---

# Embeddable Chat Surface

Status:

- future plan
- not a shipped product surface today

This doc answers a narrower question than "embeddable software":

> does `pai-bot` already have an embeddable chat surface that external websites can use, like `tawk.to`?

Short answer today:

- not yet as a shipped product surface
- but the backend primitives are partly there

## What exists today

### 1. WebSocket chat transport

There is a working WebSocket channel in the backend:

- `internal/chat/websocket.go`
- wired in `cmd/server/main.go`
- exercised by `cmd/terminal-chat/main.go`

What this gives us:

- realtime message send / receive
- typing events
- a browser-friendly transport path

What it does **not** give us by itself:

- an embeddable widget
- a hosted chat launcher bubble
- an external-site snippet
- tenant-safe visitor session bootstrap

### 2. Multi-channel runtime

The tutoring runtime already supports multiple channels:

- Telegram
- WhatsApp
- WebSocket

That matters because an embedded website chat can be introduced as another surface on the same runtime instead of a separate tutoring backend.

### 3. Public unauthenticated entry

There is already a public route:

- `admin/src/app/join/[slug]/page.tsx`
- backed by `GET /api/join/{slug}`

That proves public entry is acceptable in the product shape.

But this route is not a chat widget.
It only exposes class metadata and the join surface scaffold.

## What is missing for a real tawk.to-style embed

Right now the repo does **not** have:

- a JS embed snippet
- a hosted widget shell
- an iframe-ready chat app
- a launcher bubble UI
- anonymous or visitor identity issuance for external users
- per-site tenant routing for embeds
- allowed-origin / host validation for external websites
- widget events like open, close, unread, ready, or resize
- conversation persistence rules for external visitors
- theme / size / placement configuration for host websites

So the truthful statement is:

> `pai-bot` has a realtime transport that could power an embeddable chat, but it does not yet ship a tawk.to-style embedded chat product.

## Future-plan next slice

If the goal is "external users can drop in a small chat widget on their website", the smallest sensible slice is:

1. Hosted widget app
   - one minimal web chat UI
   - separate from admin panel
2. Embed loader
   - one `<script>` snippet that mounts an iframe
3. Visitor session bootstrap
   - issue a tenant-scoped visitor/session token
4. WebSocket reuse
   - use existing `websocket` channel instead of inventing another realtime stack
5. Host controls
   - allowed origins
   - widget title
   - theme
   - position

## Best current product wording

Use this wording unless the embed is actually built:

> `pai-bot` does not yet have a shipped embeddable chat widget for external websites. It does already have the WebSocket transport and multi-channel runtime needed to build one.

## Proposed first implementation shape

If we build it, the clean first version is:

- `widget.pandai.org/embed.js`
- host page loads one script
- script mounts an iframe
- iframe app talks to backend over existing WebSocket channel
- backend maps widget session to tenant / school context

That is much closer to `tawk.to` than trying to expose admin pages or the current join route directly.

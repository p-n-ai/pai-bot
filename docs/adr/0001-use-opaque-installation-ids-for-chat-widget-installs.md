---
title: "Use Opaque Installation IDs For Chat Widget Installs"
summary: "Decision record for identifying public Chat Widget installs by opaque Installation ID instead of tenant slug."
read_when:
  - You are changing the embeddable Chat Widget install snippet, public identifier, or tenant routing.
  - You are changing Allowed Website checks or Admin Preview security.
---

# Use opaque installation IDs for chat widget installs

Pai-bot's public chat widget should identify an **Embed Installation** with an opaque **Installation ID**, not a school tenant slug. This keeps the public script from exposing tenant identity, lets each school keep one install for now, and preserves exact **Allowed Website** checks; admin-side preview uses an authenticated preview token rather than weakening public origin rules.

# P&AI Bot

P&AI Bot is a chat-first learning agent for schools. This context captures product language that should stay stable across runtime, admin, and documentation.

## Language

**Embed Installation**:
An administrator-managed website chat installation that lets an external site host the tutor for one school context.
_Avoid_: Tenant snippet, widget tenant, embed tenant

**Chat Widget**:
The user-facing admin label for an **Embed Installation**.
_Avoid_: Website Chat, Widget, Embed Settings

**Installation ID**:
An opaque public identifier used by a website script to load one **Embed Installation** without exposing the school tenant.
_Avoid_: Tenant slug, widget ID, school ID

**Allowed Origin**:
A web origin explicitly trusted to host an **Embed Installation**.
_Avoid_: Domain whitelist, URL whitelist

**Allowed Website**:
The user-facing admin label for an **Allowed Origin**.
_Avoid_: Allowed Origin, whitelist

**Parent Origin**:
The actual web origin of the page that opened the **Chat Widget** iframe.
_Avoid_: iframe origin, admin origin, backend origin

**Guest Learner**:
An anonymous learner session created through an **Embed Installation** before account upgrade.
_Avoid_: Anonymous user, shadow student, ghost user, patched Telegram user

**Admin Preview**:
An authenticated admin-side preview of an **Embed Installation** before or without external origin access.
_Avoid_: Public test mode, localhost allowlist

**School Administrator**:
A school-scoped admin user who can manage school integration settings.
_Avoid_: Teacher, platform operator

**Teacher**:
A school-scoped educator who can use teaching/admin surfaces without managing security-sensitive integration settings.
_Avoid_: School Administrator

## Relationships

- An **Embed Installation** belongs to exactly one school context.
- A **Chat Widget** is the UI name for that school's **Embed Installation**.
- A school context has at most one **Embed Installation** for now.
- An **Embed Installation** is loaded publicly by one **Installation ID**.
- An **Embed Installation** has zero or more exact-match **Allowed Origins**.
- An **Allowed Website** input may be a page URL, but it resolves to one exact-match **Allowed Origin**.
- A **Parent Origin** must match one exact **Allowed Origin** before guest or WebSocket access is issued.
- A **Guest Learner** starts from exactly one **Embed Installation** and lasts for one browser session.
- A **Guest Learner** is an embed-channel learner identity, not a patched Telegram identity.
- A **Guest Learner** can have server-side chat history retained for 30 days by default.
- A **Guest Learner** history stays separate from any later registered learner account.
- Teachers, school administrators, and platform administrators can review retained **Guest Learner** history.
- A **Teacher** can view and copy an **Embed Installation** snippet.
- A **School Administrator** or platform administrator can manage **Allowed Origins**, enabled state, and security-sensitive **Embed Installation** settings.
- An **Embed Installation** can be enabled before any **Allowed Origin** is added, for **Admin Preview**.
- **Admin Preview** uses the real chat widget UI with an admin-authorized preview token.

## Example dialogue

> **Dev:** "Should the school paste a tenant slug into the site script?"
> **Domain expert:** "No. The school manages an **Embed Installation**; any public identifier should represent the installation, not reveal the school tenant directly."

## Flagged ambiguities

- "embed tenant" was used to mean the public website install. Resolved: the domain term is **Embed Installation**; tenant routing is an implementation detail, not the product language.
- Multiple website installs per school are not in scope yet. Resolved: one school context has at most one **Embed Installation**.
- Public script identity should not be a tenant slug. Resolved: use an opaque **Installation ID**.
- Allowed origin matching is exact. Resolved: `https://school.edu`, `https://www.school.edu`, and `https://learn.school.edu` are separate **Allowed Origins**.
- Admin UI should say **Allowed Website** and accept full URLs, then store/show the exact origin.
- Teachers can view and copy the **Embed Installation** snippet, but only school administrators and platform administrators manage origins, enabled state, and security-sensitive settings.
- Current UI should allow teachers to open Embed settings in read-only mode for snippet, preview, and retained guest history; security controls remain admin-only.
- Embed Installation uses one admin page with role-based permissions, not separate teacher/admin pages.
- Admin UI should say **Chat Widget**; domain docs may use **Embed Installation** for precision.
- **Chat Widget** page sections are Preview, Install, Allowed Websites, Guest Chats, Status, and Appearance.
- Teachers can use Preview, Install, and Guest Chats. School administrators and platform administrators can additionally manage Allowed Websites, Status, and Appearance.
- The **Embed Installation** snippet remains copyable while disabled. Disabled means guest access is blocked, not that installation instructions disappear.
- Enabled with no **Allowed Origins** is valid for **Admin Preview**, but external sites still cannot create guest learner sessions until an exact origin is added.
- **Admin Preview** must not weaken public origin rules; it uses admin authorization rather than adding localhost or admin origin to **Allowed Websites**.
- Guest access must not create ghost learners or reuse Telegram-shaped identities. Resolved: **Guest Learner** is distinct embed identity.
- Guest access creates a new **Guest Learner** per browser session.
- Guest learner chat history is retained server-side for 30 days by default, with school-configurable retention deferred.
- Guest history is not attached to a registered learner account during upgrade.
- Retained guest history is visible to teachers, school administrators, and platform administrators.
- Iframe/backend/admin origins are transport details and must not substitute for **Parent Origin** validation.

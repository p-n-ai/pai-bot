# P&AI Focused Conversation Page Design Doc

## Problem Context

P&AI conversations currently end in text. Occasionally the learner needs one message to feel more deliberate and easier to revisit than another chat bubble.

The previous proposal gave the model a page-document schema with several block types. That makes the model choose content, structure, and presentation at once. It increases generation failure modes before we have proved the basic product value.

The first slice needs one private temporary page containing one personalized message. Everything else should be deterministic application policy.

## Proposed Solution

Add focused conversation pages:

- The model supplies one `message` string through the normal `create_focused_page` agent-core tool.
- The server injects the trusted learner name and conversation identity.
- One fixed page template renders recipient, message, expiry, and a fixed “Continue with P&AI” action.
- The server creates a private capability link and sends it with the same chat turn.
- The page expires or can be revoked; it never becomes a permanent dashboard.

Compared with the current text response, the user receives a focused page they can open and revisit briefly. Compared with the prior block-document proposal, the model has one decision: what message to say.

## Goals and Non-Goals

### Goals

- Same-turn delivery: reply text and one private focused-page link arrive together.
- Low model burden: model outputs only a message.
- Personalization: server injects the resolved learner identity.
- Temporary privacy: every page has enforced expiry and revocation.
- Retry safety: a retried turn reuses the same page.

### Non-Goals

- Model-selected layouts, blocks, titles, CTA labels, themes, lifetimes, or URLs.
- Arbitrary HTML, Markdown, scripts, media, or embedded external content.
- Permanent page history or artifact library.
- Page-side mutation of goals, progress, or account state.
- Cross-channel account linking.

## Design

The focused page is an immutable message snapshot owned by one learner and conversation. The application owns every presentation and lifecycle decision.

```mermaid
sequenceDiagram
    participant U as User
    participant C as Chat adapter
    participant E as Agent engine
    participant P as Focused page service
    participant W as Page endpoint

    U->>C: Ask for a focused page
    C->>E: Process turn with resolved actor
    E->>P: Tool creates page for trusted actor and turn
    P->>P: Parse message and apply fixed template policy
    P-->>E: Tool success only; link retained outside model context
    E-->>C: TurnResult{reply text, private temporary link}
    C-->>U: Reply text + URL button
    U->>W: Redeem private capability
    W-->>U: Recipient + message + fixed CTA, or expired state
```

### Key Components

#### Focused Page Draft

```go
type TurnResult struct {
    Text string
    Page *FocusedPageDraft
}

type FocusedPageArtifact struct {
    URL       string
    ExpiresAt time.Time
}
```

The strict tool schema contains exactly one required string and rejects additional fields. The boundary parser trims it, rejects empty or messages above 4,000 characters, and returns a refined message value.

For the first slice, the tool description limits use to goal and report flows. Normal tutoring turns may finish without a tool call and remain text-only.

#### Fixed Page Template

The application supplies:

- P&AI branding.
- “A message for {learner name}” heading.
- Parsed model message rendered as plain text.
- Expiry status.
- Fixed “Continue with P&AI” action.
- Expired-page copy that reveals no previous message.

The model cannot alter these fields. The renderer never interprets HTML or Markdown.

#### Page Creation

`FocusedPageService.Create` receives the resolved actor, conversation ID, turn ID, parsed message, and server-selected lifetime.

Creation sequence:

1. Parse the message.
2. Resolve the learner display name from trusted application data.
3. Generate a public ID and derive a high-entropy capability from the server secret and trusted idempotency key.
4. Store only the capability hash.
5. Persist the immutable message and expiry.
6. Return the private link.

Use `(tenant_id, turn_id, page_index)` as the idempotency key. The first version allows at most one page per turn, so `page_index` is always zero. Deterministic HMAC derivation lets a retry recreate the same link while the database stores only its SHA-256 hash.

#### Page Instance Data

| Field | Purpose |
|---|---|
| `id`, `public_id` | Internal identity and non-secret route identity |
| `tenant_id`, `owner_user_id` | Enforced ownership scope |
| `conversation_id`, `turn_id` | Provenance and idempotency |
| `recipient_name`, `message` | Immutable page snapshot |
| `token_hash` | Verify capability without storing raw secret |
| `status` | `active`, `revoked`, or `expired` |
| `expires_at`, lifecycle timestamps | Access enforcement and diagnosis |

Tenant and owner must be enforced together in the database and every repository query.

#### Private Link and Expiry

Send `/a/{public_id}#secret`:

- Initial request contains no secret.
- Page JavaScript posts the fragment secret to the same-origin redeem endpoint.
- Server hashes the secret and compares it to `token_hash`.
- Browser removes the fragment after redemption.
- Responses use `Cache-Control: private, no-store`, restrictive CSP, and `Referrer-Policy: no-referrer`.
- Logs never contain capability, message, or full URL.

The endpoint rejects expired or revoked pages before returning recipient or message data. Cleanup later deletes expired rows; cleanup delay never extends access.

#### Conversation Delivery and Failure

The agent tool creates the page before channel send. `internal/agent` assembles the final text and artifact, then `internal/chat` appends the private URL button after existing Telegram keyboard rows.

If message generation or page creation fails, send the useful text response without a link. If channel delivery fails, retain the idempotent page for retry until normal expiry.

The page CTA is application-owned and only returns to a trusted P&AI conversation. It does not mutate learner state.

The slice is enabled only when `LEARN_FOCUSED_PAGE_BASE_URL` and `LEARN_FOCUSED_PAGE_TELEGRAM_CTA_URL` are both set. The server refuses focused-page startup with the development-default `PAI_AUTH_SECRET`, because that secret derives capabilities with a focused-page-specific HMAC domain.

## Alternatives Considered

| Alternative | Pros | Cons | Why Not Chosen |
|-------------|------|------|----------------|
| Fixed focused-message page | Small model contract, predictable UI, easy validation | One presentation shape | Chosen for first proof |
| Bounded block document | Supports plans, lists, metrics, and reports | Model must choose structure; larger renderer and schema | Defer until one-message page proves insufficient |
| Preset-specific templates | Strong control per use case | New code for each conversational result | More surface than the first slice needs |
| Model-generated HTML | Maximum flexibility | Unsafe and inconsistent | Invalid trust boundary |
| Text-only chat | No new infrastructure | Message remains buried in conversation | Does not provide the focused surface requested |

## Fixed First-Slice Decisions

- Product use cases: goal and report conversations.
- Lifetime: exactly one hour, enforced by server and database policy.
- Redemption: repeatable until expiry unless revoked.
- First channel: Telegram; terminal chat is planned later.
- Message limit: 4,000 Unicode characters.
- Delivery: final tutor text plus at most one focused-page artifact; the tool never sends chat messages.

## Implementation Status

Implemented in the focused Telegram slice:

- Normal agent-core tool execution with the strict `{message}` contract.
- Trusted tenant, owner, conversation, and turn derivation in `internal/agent`.
- PostgreSQL and in-memory stores with one-hour expiry, revocation, idempotency, and hash-only capability storage.
- Fixed read-only page shell and fragment redemption with no-store, restrictive CSP, no-referrer, and frame protections.
- Final tutor text plus one Telegram URL button, with retry of the same assembled artifact after delivery failure.
- Unit and migration-backed integration coverage for text-only turns, tool continuation, duplicate calls, one-artifact enforcement, wrong token, expiry, revocation, isolation, and Telegram order.

Still planned:

- Terminal-chat delivery.
- Native provider support beyond OpenRouter.
- Expired-row cleanup and durable queued channel retries.
- Browser-level layout, keyboard, fragment-removal, and reduced-motion smoke coverage.

## Appendix

Implemented seams:

- `internal/agentcore/core.go`: native sequential continuation loop.
- `internal/agent/focused_page_tool.go`: trusted tool policy and `TurnResult` assembly.
- `internal/ai/router.go`: native provider routing and fallback.
- `internal/focusedpage`: persistence, capability lifecycle, and HTTP renderer.
- `internal/chat/turn_render.go`: Telegram formatting and URL-button order.

Interactive companion: `docs/planning/user-artifacts-design.html`.

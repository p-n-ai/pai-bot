---
title: "Admin Auth Runtime"
summary: "Current admin authentication model: cookies, login flows, Google linking, refresh, tenant switching, and POST-only logout."
read_when:
  - You are changing admin login, logout, refresh, or session-cookie behavior
  - You are changing Google sign-in or identity-linking behavior
  - You need a current runtime explanation of how admin auth actually works today
---

# Admin Auth Runtime

Current.

This doc describes the auth model that is live today for the admin app.

## Source Of Truth

- Go backend owns auth state and cookie issuance
- Next.js admin reads session state from server-owned cookies
- browser JavaScript does not own access tokens

Main files:

- [cmd/server/main.go](/Users/thor/.codex/worktrees/5665/pai-bot/cmd/server/main.go)
- [internal/auth/postgres.go](/Users/thor/.codex/worktrees/5665/pai-bot/internal/auth/postgres.go)
- [internal/auth/google_oidc.go](/Users/thor/.codex/worktrees/5665/pai-bot/internal/auth/google_oidc.go)
- [internal/auth/cookies.go](/Users/thor/.codex/worktrees/5665/pai-bot/internal/auth/cookies.go)
- [admin/src/lib/api.ts](/Users/thor/.codex/worktrees/5665/pai-bot/admin/src/lib/api.ts)
- [admin/src/stores/app-store.ts](/Users/thor/.codex/worktrees/5665/pai-bot/admin/src/stores/app-store.ts)

## Cookie Model

Go sets and clears these cookies:

- `pai_admin_access`
  short-lived JWT access token, `HttpOnly`
- `pai_admin_refresh`
  opaque refresh token, `HttpOnly`
- `pai_admin_user`
  URL-escaped SSR/profile cookie for frontend hydration

Frontend rule:

- no auth tokens in `localStorage`
- browser requests use `credentials: include`
- protected auth responses send `Cache-Control: private, no-store`

## Login Flows

### Email + password

1. Frontend sends `POST /api/auth/login`
2. Go validates password identity from `auth_identities`
3. If one account matches, Go issues cookies and returns session payload
4. If multiple tenant matches exist, Go returns `tenant_required`
5. Frontend asks user to choose school, then retries login with `tenant_id`

### Google sign-in

Endpoints:

- `GET /api/auth/google/start`
- `GET /api/auth/google/callback`

Flow:

1. Frontend redirects to `GET /api/auth/google/start?next=...`
2. Go creates `auth_oidc_flows` row with hashed state, nonce, PKCE verifier, flow type, next path
3. Browser goes to Google or local emulate provider
4. Provider redirects back to `GET /api/auth/google/callback?state=...&code=...`
5. Go exchanges code, verifies ID token, fetches userinfo, then resolves the local account

Account resolution rules:

- existing Google link by `provider_account_id = sub`
  sign in directly
- verified authoritative Google email and exactly one matching password identity
  auto-link, then sign in
- multiple matching schools
  reject with `tenant_required`
- no authoritative single match
  reject with `link_required`

Authoritative email rule:

- `@gmail.com` verified addresses qualify
- hosted-domain Google identities can also qualify
- arbitrary verified non-Google emails do not auto-link

That means:

- `teacher@gmail.com` can auto-link when there is exactly one local password account with the same email
- `teacher@yahoo.com` cannot be auto-linked from Google sign-in
- a signed-in user can still explicitly link a different Google account later

## Explicit Identity Linking

Endpoints:

- `POST /api/auth/google/link/start`
- `GET /api/auth/identities`

Rules:

- link start requires an authenticated session
- link start requires an allowed browser `Origin`
- different-email Google linking only happens from an authenticated workspace
- one Google link per local user; re-link replaces the previous Google identity for that user
- Google `sub` is the stable identity key; provider email is metadata only

## Refresh

Endpoint:

- `POST /api/auth/refresh`

Flow:

1. Frontend receives `401`
2. frontend calls refresh once
3. Go validates refresh token from cookie or body
4. Go rotates refresh token, reissues access token, resets cookies

## Tenant Switch

Endpoint:

- `POST /api/auth/switch-tenant`

Rules:

- requires password confirmation
- reissues session in place for the selected tenant
- does not force a logout/login round trip

## Logout

Endpoint:

- `POST /api/auth/logout`

Important:

- logout is POST-only
- logout must not use GET
- frontend logout is triggered by a button action, not a link

Flow:

1. Frontend calls `POST /api/auth/logout`
2. Go reads refresh token from cookie or optional body
3. Go revokes the refresh token server-side
4. Go clears auth cookies
5. Frontend clears local UI/session metadata and redirects to `/login`

Guardrail:

- `GET /api/auth/logout` is rejected and covered by server tests

## Local Emulation

For local Google auth testing:

- use [docs/local-auth-emulation.md](/Users/thor/.codex/worktrees/5665/pai-bot/docs/local-auth-emulation.md)
- local emulate seed currently includes `teacher@gmail.com`

One emulator nuance:

- browser auth still uses the provider issuer/authorization URL
- server-side token, userinfo, and JWKS calls may need transport rewriting when discovery is fetched from `host.docker.internal` inside Docker

## Mental Model

Short version:

- Go owns identity
- cookies own session
- Next hydrates state from cookies
- Google `sub` owns provider identity
- logout is POST, not GET

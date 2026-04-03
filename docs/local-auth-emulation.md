---
title: "Local Auth Emulation"
summary: "Repo-local dev/test support for Google and Vercel provider emulation via emulate."
read_when:
  - You are adding Google login, OIDC callback handling, or provider token exchange to the Go auth layer
  - You need to test external OAuth flows locally or in CI without calling the real Google or Vercel APIs
  - You are wiring dev scripts, local seed data, or auth-related integration tests
---

# Local Auth Emulation

Status:

- Current: repo-local dev tooling only
- Current: Go admin auth can now point its Google OIDC transport at these emulator endpoints
- Not current: production auth still uses the Go backend and real providers

This repo now carries a shared [`emulate`](https://github.com/vercel-labs/emulate) seed config under [`tools/emulate/emulate.config.yaml`](/Users/thor/.codex/worktrees/5665/pai-bot/tools/emulate/emulate.config.yaml).

Why:

- local OAuth/OIDC testing without real Google calls
- deterministic provider state in local dev and CI
- cleaner future auth work: callback handling, token exchange, userinfo fetch, provider-link tests

## Commands

```bash
just emulate-auth
just emulate-google
just emulate-vercel
```

Pinned package:

- `emulate@0.4.1`

Default local URLs from the current recipes:

- Vercel emulator: `http://127.0.0.1:4000`
- Google emulator: `http://127.0.0.1:4002`

## Seeded identities

Google:

- `teacher@gmail.com`
- `platform-admin@example.com`

Google OAuth client:

- client id: `pai-google-local.apps.googleusercontent.com`
- client secret: `emulate-google-secret`

Seeded redirect URIs:

- `http://127.0.0.1:8080/api/auth/google/callback`
- `http://127.0.0.1:8082/api/auth/google/callback`
- `http://localhost:8080/api/auth/google/callback`
- `http://127.0.0.1:3000/api/auth/callback/google`
- `http://localhost:3000/api/auth/callback/google`

## Recommended boundary

- Keep `emulate` separate from the Go auth runtime
- Treat it as dev/test support only
- Keep production auth as real Go session, RBAC, API-key, and provider logic

## Current env contract

The Go admin auth slice now switches Google OIDC transport through config instead of hardcoding provider URLs:

- `LEARN_AUTH_GOOGLE_CLIENT_ID`
- `LEARN_AUTH_GOOGLE_CLIENT_SECRET`
- `LEARN_AUTH_GOOGLE_REDIRECT_URL`
- `LEARN_AUTH_GOOGLE_DISCOVERY_URL`
- `LEARN_AUTH_GOOGLE_EMULATOR_SIGNING_SECRET`
- `LEARN_AUTH_ADMIN_BASE_URL`

For local `emulate`:

- discovery endpoint: `http://127.0.0.1:4002/.well-known/openid-configuration`
- emulator signing secret: `emulate-google-jwt-secret`

The callback, token exchange, and userinfo fetch all derive from discovery. The emulator signing secret is only for local HS256 ID-token verification; real Google still uses discovery issuer + JWKS.

Do not bake emulator-specific assumptions into the core auth model. Keep the provider transport switchable.

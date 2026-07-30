# camelAI configuration and secrets research

Research snapshot: [`qaml-ai/camelAI@b75546e`](https://github.com/qaml-ai/camelAI/tree/b75546e0b94917e16d0711c86f93a19fd53996e7), compared with P&AI pull request [#224](https://github.com/p-n-ai/pai-bot/pull/224) at `7b5b4b0`.

## Conclusion

P&AI should keep PR #224's separate authentication and runtime-settings
encryption keys. camelAI supports a broader provider configuration shape and
has a useful explicit precedence rule for deployment-managed configuration,
but its single `INTEGRATION_SECRET_KEY` encrypts provider, integration, and SSO
credentials. That is a wider blast radius than P&AI's new purpose-specific
key, and camelAI's inspected code has no key version, key ring, or read-old /
write-new rotation path.

## What camelAI does

camelAI has two configuration modes:

- Hosted mode stores one active provider record per organization in the
  organization's Durable Object. The record separates an encrypted credential
  blob from non-secret provider configuration and audit fields
  ([schema](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/workers/main/src/identity/org-do.ts#L1552-L1564),
  [upsert](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/workers/main/src/identity/org-do.ts#L8866-L8917)).
  Organization admins can switch among Anthropic, Bedrock, custom, OpenAI, and
  OpenRouter at runtime; custom providers include base URL, model ID, auth
  header, and protocol selection
  ([validation and persistence](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/src/routes/api/orgs.%24id.llm-provider.ts#L205-L383)).
- Self-host mode reads `SELFHOST_AI_*` variables. A configured deployment
  provider overrides the organization record, and the API rejects UI changes
  while that override exists
  ([precedence](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/src/lib/selfhost-ai-provider.ts#L89-L102),
  [write guard](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/src/routes/api/orgs.%24id.llm-provider.ts#L73-L100)).
  Partial configurations fail validation, including a missing key for every
  provider except Bedrock/IAM and missing endpoint/model/protocol fields for a
  custom provider
  ([validation](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/src/lib/selfhost-ai-provider.ts#L111-L178)).

Provider switching is data-driven rather than a process restart: the current
record is loaded for a turn, decrypted, refined into provider-specific
credentials, then used to select compatible models
([turn-time resolution](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/workers/main/src/chat-thread/pi-model-config.ts#L731-L802),
[compatibility rules](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/src/lib/llm-provider-config.ts#L409-L439)).

Secrets are purpose-separated from token signing at the environment boundary:
`TOKEN_SIGNING_SECRET` and `INTEGRATION_SECRET_KEY` are distinct required
Compose inputs. Self-host initialization generates each independently with 32
random bytes, writes them to a mode-`0600` env file, and refuses to overwrite
it unless forced
([Compose](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/docker-compose.selfhost.yml#L11-L32),
[initializer](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/scripts/selfhost-init.mjs#L14-L19),
[generation and write](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/scripts/selfhost-init.mjs#L58-L79)).
Stored credentials use AES-256-GCM with a random 12-byte IV and a key derived
from `INTEGRATION_SECRET_KEY` by PBKDF2-SHA-256
([implementation](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/src/lib/integration-crypto.ts#L1-L65)).

Deployment is materially heavier than P&AI's modular monolith. The self-host
Compose stack runs the app/workerd bundle, a separate project-runtime service
with Docker socket access, and a local artifact service
([stack](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/docker-compose.selfhost.yml#L1-L139)).
Its infrastructure templates target a single-node AWS, Azure, or GCP VM, bind
services to loopback, and place Caddy in front
([infrastructure guide](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/infra/selfhost/README.md#L5-L57)).

## Comparison with P&AI PR #224

PR #224 is stronger on encryption-key lifecycle. It introduces
`PAI_CONFIG_ENCRYPTION_KEY`, rejects short values and reuse of
`PAI_AUTH_SECRET`, uses the dedicated key for new writes, retains the auth
secret only as a legacy read key, and re-encrypts legacy ciphertext on the next
settings update
([configuration validation](https://github.com/p-n-ai/pai-bot/blob/7b5b4b0d878da95b3c4715f45622ddc48e0eb93e/internal/platform/config/config.go#L397-L418),
[read-old/write-new path](https://github.com/p-n-ai/pai-bot/blob/7b5b4b0d878da95b3c4715f45622ddc48e0eb93e/internal/platform/settings/postgres.go#L149-L175),
[secret write guard](https://github.com/p-n-ai/pai-bot/blob/7b5b4b0d878da95b3c4715f45622ddc48e0eb93e/internal/platform/settings/postgres.go#L234-L270)).
Production Compose also requires independent auth, encryption, and bootstrap
admin secrets before interpolation
([Compose](https://github.com/p-n-ai/pai-bot/blob/7b5b4b0d878da95b3c4715f45622ddc48e0eb93e/docker-compose.prod.yml#L23-L35)).

camelAI is stronger in provider configuration breadth. It models provider,
endpoint, protocol, auth method, model, and region as validated data and
applies provider changes to active work without restarting. P&AI's runtime
settings currently store one OpenRouter key and overlay a process-wide
environment baseline, so copying camelAI's full UI or per-organization storage
would be a product and tenancy expansion, not a secrets hardening step.

## What P&AI should borrow

1. **Make configuration precedence explicit.** If P&AI later supports both
   operator-managed and admin-managed provider credentials, define one rule
   such as `deployment override > tenant runtime setting > environment
   baseline`, expose who manages the effective value, and reject writes that
   cannot take effect.
2. **Model provider configuration as validated data.** A provider record should
   contain a discriminated provider kind plus only that provider's legal
   endpoint, protocol, model, region, and auth fields. Keep credentials in the
   encrypted secret payload and return only a redacted hint to clients.
3. **Add deployment ergonomics around PR #224.** A small initializer that
   generates independent secrets, creates a private env file with mode `0600`,
   and refuses accidental overwrite would reduce setup errors without weakening
   the production checks.
4. **Expose a redacted configuration-health result.** camelAI's self-host
   health route reports missing bindings and invalid provider combinations
   without returning secret values
   ([health checks](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/src/routes/api/selfhost.health.ts#L14-L64)).

## What P&AI should not borrow

- Do not replace the dedicated encryption root with a shared
  integration-wide key. Purpose separation and the migration path in PR #224
  are safer.
- Do not adopt Cloudflare Durable Objects, workerd, or camelAI's project-runtime
  deployment topology for provider settings. P&AI already has Postgres,
  tenant-aware backend boundaries, Compose, and Helm; the operational cost
  would not buy better secret handling.
- Do not add per-tenant runtime provider switching until P&AI has an explicit
  tenant authorization contract, cross-instance invalidation strategy, and
  provider/model compatibility rules. A user-selected tenant filter is not an
  authorization boundary.
- Do not copy camelAI's deployment documentation wholesale. At this snapshot
  its README links to `docs/self-hosting.md`, but that file is absent from the
  repository
  ([README link](https://github.com/qaml-ai/camelAI/blob/b75546e0b94917e16d0711c86f93a19fd53996e7/README.md#L59-L70)).

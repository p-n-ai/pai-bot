# Runtime AI settings

Use this runbook to configure the platform-wide AI router at runtime, diagnose
desired-versus-applied drift, and rotate stored provider credentials without
exposing them.

AI configuration has two layers:

1. Environment variables are the immutable boot baseline.
2. The single global database row contains optional runtime overrides.

The admin API and UI always return the effective value, its source, the boot
baseline, the stored override, and revision state. Resetting a field deletes its
database override, so the effective value immediately returns to the environment
baseline. PaiBot never edits `.env` files or process environment variables.

Security, database, listener, and bootstrap settings remain environment-only.
Runtime AI settings are global platform configuration and require platform-admin
authorization; they are not tenant-scoped.

## Runtime provider model

The runtime API accepts a closed provider variant, so each provider can expose
only fields that its production adapter supports:

- API-key providers (`openai`, `anthropic`, `deepseek`, `google`, and
  `openrouter`) allow model and write-only credential overrides.
- Ollama allows enabled and model overrides. Its URL remains environment-only
  because runtime custom endpoints require a separately reviewed SSRF policy.
- Managed Codex allows a model override. Enablement and device authentication
  remain owned by the environment and the managed login flow.

An omitted field is unchanged. A JSON `null` deletes that field's database
override and restores the environment baseline. Empty or whitespace-only
models and credentials are rejected rather than treated as reset aliases.

For example, replace an OpenRouter model and credential:

```json
{
  "expectedRevision": 7,
  "provider": {
    "type": "api_key",
    "name": "openrouter",
    "model": "anthropic/claude-sonnet-4.5",
    "apiKey": "write-only-value"
  }
}
```

Reset only the stored credential override:

```json
{
  "expectedRevision": 8,
  "provider": {
    "type": "api_key",
    "name": "openrouter",
    "apiKey": null
  }
}
```

Select a default using the same closed discriminator:

```json
{
  "expectedRevision": 9,
  "defaultProvider": {
    "type": "ollama"
  }
}
```

The settings response returns baseline, override, effective value, and source
per non-secret field. Credentials return only set state, source, an optional
safe last-four hint, and envelope health; plaintext and ciphertext are never
returned.

## Generate independent roots

Create a new env fragment at an explicit, previously unused path:

```bash
go run ./cmd/init-secrets -out /path/to/pai-bot-secrets.env
```

The command generates independent 256-bit random roots for `PAI_AUTH_SECRET`
and `PAI_CONFIG_ENCRYPTION_KEY`, creates the file with mode `0600`, refuses
symlinks and existing files, and never prints either value. Load the fragment
through the deployment's secret mechanism; do not commit it or concatenate it
into a tracked `.env` file.

## Validate production secrets

Run the canonical preflight before changing a production deployment:

```bash
go run ./cmd/validate-production-secrets
```

It reads only `PAI_AUTH_SECRET`, `PAI_CONFIG_ENCRYPTION_KEY`,
`PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS`, and
`PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD`. It rejects public defaults, missing active
roots, malformed or oversized previous-key lists, short or low-diversity roots,
duplicates, and reuse across auth, active, and retired roots without printing
any secret.

The production Compose overlay runs the same check before the app starts, and
the remote deployment runs it before migrations. The GitHub deployment
validates before copying files and replaces the mode-`0600` server `.env`
atomically. The Helm chart rejects the same unsafe value classes during
rendering, before a workload is installed or upgraded.

## Encryption key rotation

`PAI_CONFIG_ENCRYPTION_KEY` encrypts new or replaced provider credentials.
`PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS` is a JSON array of retired roots that may
decrypt existing credentials during migration. Use independent, random secrets
of at least 32 non-whitespace characters; never reuse `PAI_AUTH_SECRET`.

New credentials use the authenticated envelope
`pai:v1:a256gcm:<key-id>:<payload>`. The payload contains a random nonce and
AES-256-GCM ciphertext/tag; authenticated context binds the envelope header to
runtime-settings row `1`, the provider, and the credential slot. The key ID
selects the active or approved previous encryption root directly. Only
unprefixed legacy ciphertext may try the bounded PR #224 migration keys;
unknown versions, algorithms, or key IDs never fall back to legacy decryption.

Rotate without downtime in two deployments:

1. Deploy the new active root as `PAI_CONFIG_ENCRYPTION_KEY` and move the old
   active root into `PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS`.
2. Save the affected runtime settings once. PaiBot rewrites credentials that
   were encrypted with a retired or legacy root using the active root.
3. Restart and verify the provider using only the active root in a staging or
   controlled production rollout.
4. Remove the retired root only after every live database and restorable backup
   that must remain usable has been migrated or expired.

To roll back application code while new envelopes exist, keep the new root as
active and the former root in the previous-key list. Do not roll back to a
binary that cannot parse the current envelope. If configuration must revert,
restore the former root as active while retaining the newer root as a previous
key, restart, rewrite the credential, and verify restart/use before retiring
either root.

Keep no more than eight previous roots. The server rejects duplicate roots,
roots shared with the active encryption key or auth secret, short roots, and
oversized key lists.

Losing every root that can decrypt a stored credential is recoverable only by
resetting or replacing that credential. Unrelated settings can still be changed,
and an undecryptable credential can be explicitly cleared without exposing its
ciphertext or plaintext.

## Revision and drift

Each semantic database change increments `revision`. A successful synchronous
live apply advances `appliedRevision`; `drift` is true when they differ. Admin
writes include `expectedRevision`, so a stale browser receives HTTP 409 instead
of silently overwriting a newer operator change.

After a restart, the server loads the committed row before serving admin
traffic, builds the router from the same effective configuration, and marks that
revision applied. If drift is reported, stop making further edits and inspect
the server startup or provider-construction error before retrying.

`/health/ai` exercises the production completion router and returns only
`ok` or `unavailable`; it never returns provider errors or configuration
material. The authenticated Admin AI settings response is the diagnostic
surface for source and revision state.

Runtime edits are intentionally process-local. One process serializes commit
and live apply order, but there is no cross-instance invalidation. In a
multi-replica deployment, restart or roll every replica after an edit before
treating the fleet as converged.

## Common failures

- **HTTP 409 when saving:** another administrator saved a newer revision.
  Reload the settings page, review the effective values, then apply the intended
  change again.
- **Provider update rejected:** the selected default cannot be constructed from
  the effective configuration and current managed capabilities. Restore its
  credential or select a registrable provider.
- **`drift` is true:** PostgreSQL contains a desired revision that this process
  has not applied. Check startup/provider-construction logs, which contain only
  safe provider and error classifications, then restart or correct the
  configuration.
- **Stored credential cannot decrypt:** keep every candidate root in place,
  inspect the configured active/previous-key set, and replace or explicitly
  reset the credential. Unrelated updates preserve the unreadable ciphertext
  byte-for-byte.

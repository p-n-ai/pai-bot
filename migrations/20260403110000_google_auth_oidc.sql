-- +goose Up
ALTER TABLE auth_identities
    ADD COLUMN provider_account_id TEXT,
    ADD COLUMN provider_email TEXT,
    ADD COLUMN provider_email_normalized TEXT,
    ADD COLUMN provider_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN linked_at TIMESTAMPTZ,
    ADD COLUMN last_used_at TIMESTAMPTZ;

CREATE UNIQUE INDEX idx_auth_identities_provider_account_id
    ON auth_identities(provider, provider_account_id)
    WHERE provider_account_id IS NOT NULL;

CREATE INDEX idx_auth_identities_provider_email
    ON auth_identities(provider, provider_email_normalized)
    WHERE provider_email_normalized IS NOT NULL;

CREATE TABLE auth_oidc_flows (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider      TEXT NOT NULL CHECK (provider IN ('google')),
    flow_type     TEXT NOT NULL CHECK (flow_type IN ('login', 'link')),
    state_hash    TEXT NOT NULL UNIQUE,
    nonce         TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    next_path     TEXT,
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_auth_oidc_flows_expires_at ON auth_oidc_flows(expires_at);

-- +goose Down
DROP TABLE IF EXISTS auth_oidc_flows;
DROP INDEX IF EXISTS idx_auth_identities_provider_email;
DROP INDEX IF EXISTS idx_auth_identities_provider_account_id;

ALTER TABLE auth_identities
    DROP COLUMN IF EXISTS last_used_at,
    DROP COLUMN IF EXISTS linked_at,
    DROP COLUMN IF EXISTS provider_profile,
    DROP COLUMN IF EXISTS provider_email_normalized,
    DROP COLUMN IF EXISTS provider_email,
    DROP COLUMN IF EXISTS provider_account_id;

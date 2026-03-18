-- +goose Up
-- P&AI Bot - Auth tables for invite onboarding and web login

CREATE TABLE auth_identities (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id             UUID REFERENCES tenants(id) ON DELETE CASCADE,
    provider              TEXT NOT NULL CHECK (provider IN ('password', 'telegram', 'whatsapp', 'google', 'microsoft')),
    identifier            TEXT NOT NULL,
    identifier_normalized TEXT NOT NULL,
    password_hash         TEXT,
    email_verified_at     TIMESTAMPTZ,
    last_login_at         TIMESTAMPTZ,
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, provider, identifier_normalized)
);

CREATE INDEX idx_auth_identities_user_id ON auth_identities(user_id);
CREATE INDEX idx_auth_identities_tenant_provider ON auth_identities(tenant_id, provider);

CREATE TABLE auth_invites (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email            TEXT NOT NULL,
    email_normalized TEXT NOT NULL,
    role             TEXT NOT NULL CHECK (role IN ('teacher', 'parent', 'admin', 'platform_admin')),
    token_hash       TEXT NOT NULL UNIQUE,
    invited_by       UUID REFERENCES users(id),
    expires_at       TIMESTAMPTZ NOT NULL,
    accepted_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_auth_invites_tenant_email ON auth_invites(tenant_id, email_normalized);
CREATE INDEX idx_auth_invites_expires_at ON auth_invites(expires_at);

CREATE TABLE auth_refresh_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id    UUID REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    user_agent   TEXT,
    ip_address   TEXT,
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    replaced_by  UUID REFERENCES auth_refresh_tokens(id),
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_auth_refresh_tokens_user_id ON auth_refresh_tokens(user_id);
CREATE INDEX idx_auth_refresh_tokens_tenant_id ON auth_refresh_tokens(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS auth_refresh_tokens;
DROP TABLE IF EXISTS auth_invites;
DROP TABLE IF EXISTS auth_identities;

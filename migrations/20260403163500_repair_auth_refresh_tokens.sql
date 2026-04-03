-- +goose Up
-- Repair drifted local/dev databases where goose recorded auth migrations but
-- auth_refresh_tokens was later dropped or never materialized.

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
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

CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_user_id ON auth_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_tenant_id ON auth_refresh_tokens(tenant_id);

-- +goose Down
-- Intentionally no-op. This migration repairs drift and should not drop a live
-- refresh token table during rollback.
SELECT 1;

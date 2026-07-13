-- +goose Up
CREATE TABLE focused_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id TEXT NOT NULL UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    owner_user_id UUID NOT NULL REFERENCES users(id),
    conversation_id UUID NOT NULL REFERENCES conversations(id),
    turn_id TEXT NOT NULL,
    page_index SMALLINT NOT NULL DEFAULT 0 CHECK (page_index = 0),
    recipient_name TEXT NOT NULL,
    message TEXT NOT NULL CHECK (char_length(message) BETWEEN 1 AND 4000),
    token_hash BYTEA NOT NULL CHECK (octet_length(token_hash) = 32),
    status TEXT NOT NULL CHECK (status IN ('active', 'revoked', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    UNIQUE (tenant_id, turn_id, page_index),
    CHECK (expires_at = created_at + INTERVAL '1 hour'),
    CHECK ((status = 'revoked') = (revoked_at IS NOT NULL)),
    CHECK ((status = 'expired') = (expired_at IS NOT NULL))
);

CREATE INDEX focused_pages_expiry_idx ON focused_pages (expires_at) WHERE status = 'active';
CREATE INDEX focused_pages_owner_idx ON focused_pages (tenant_id, owner_user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS focused_pages;

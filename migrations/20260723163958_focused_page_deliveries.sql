-- +goose Up
ALTER TABLE focused_pages
    ADD CONSTRAINT focused_pages_tenant_public_turn_unique
    UNIQUE (tenant_id, public_id, turn_id);

CREATE TABLE focused_page_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    turn_id TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (char_length(channel) > 0),
    recipient_id TEXT NOT NULL CHECK (char_length(recipient_id) > 0),
    final_text TEXT NOT NULL CHECK (char_length(final_text) > 0),
    focused_page_public_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'leased', 'delivered')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_token TEXT,
    lease_expires_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (tenant_id, focused_page_public_id, turn_id)
        REFERENCES focused_pages(tenant_id, public_id, turn_id) ON DELETE CASCADE,
    UNIQUE (tenant_id, turn_id, channel),
    CHECK (
        (status = 'pending' AND lease_token IS NULL AND lease_expires_at IS NULL AND delivered_at IS NULL)
        OR (status = 'leased' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL AND delivered_at IS NULL)
        OR (status = 'delivered' AND lease_token IS NULL AND lease_expires_at IS NULL AND delivered_at IS NOT NULL)
    )
);

CREATE INDEX focused_page_deliveries_pending_idx
    ON focused_page_deliveries (next_attempt_at, created_at)
    WHERE status = 'pending';

CREATE INDEX focused_page_deliveries_lease_idx
    ON focused_page_deliveries (lease_expires_at, created_at)
    WHERE status = 'leased';

-- +goose Down
DROP TABLE IF EXISTS focused_page_deliveries;
ALTER TABLE focused_pages
    DROP CONSTRAINT IF EXISTS focused_pages_tenant_public_turn_unique;

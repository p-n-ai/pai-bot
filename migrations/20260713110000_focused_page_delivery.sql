-- +goose Up
ALTER TABLE focused_pages
    ADD CONSTRAINT focused_pages_tenant_public_id_unique UNIQUE (tenant_id, public_id);

CREATE TABLE focused_page_deliveries (
    tenant_id UUID NOT NULL,
    page_public_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (channel = 'telegram'),
    recipient_external_id TEXT NOT NULL,
    tutor_text TEXT NOT NULL CHECK (char_length(tutor_text) > 0),
    status TEXT NOT NULL CHECK (status IN ('pending', 'sent', 'expired', 'cancelled')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL,
    last_attempt_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, page_public_id),
    FOREIGN KEY (tenant_id, page_public_id) REFERENCES focused_pages (tenant_id, public_id) ON DELETE CASCADE,
    CHECK ((status = 'sent') = (sent_at IS NOT NULL))
);

CREATE INDEX focused_page_deliveries_due_idx
    ON focused_page_deliveries (next_attempt_at)
    WHERE status = 'pending';

-- +goose Down
DROP TABLE IF EXISTS focused_page_deliveries;
ALTER TABLE focused_pages DROP CONSTRAINT IF EXISTS focused_pages_tenant_public_id_unique;

-- +goose Up
CREATE TABLE chat_inbound_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel TEXT NOT NULL CHECK (char_length(channel) > 0),
    delivery_id TEXT NOT NULL CHECK (char_length(delivery_id) > 0),
    learner_key TEXT NOT NULL CHECK (char_length(learner_key) > 0),
    destination_key TEXT NOT NULL CHECK (char_length(destination_key) > 0),
    accepted_sequence BIGSERIAL NOT NULL,
    inbound_payload JSONB NOT NULL,
    inbound_payload_hash BYTEA NOT NULL CHECK (octet_length(inbound_payload_hash) = 32),
    result_payload JSONB,
    status TEXT NOT NULL DEFAULT 'received'
        CHECK (status IN ('received', 'processing', 'delivery_pending', 'delivering', 'delivered', 'failed')),
    processing_attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (processing_attempt_count >= 0),
    delivery_attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (delivery_attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_token TEXT,
    lease_expires_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, channel, delivery_id),
    CHECK (
        (status = 'received' AND result_payload IS NULL
            AND lease_token IS NULL AND lease_expires_at IS NULL
            AND delivered_at IS NULL AND failed_at IS NULL)
        OR (status = 'processing' AND result_payload IS NULL
            AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL
            AND delivered_at IS NULL AND failed_at IS NULL)
        OR (status = 'delivery_pending' AND result_payload IS NOT NULL
            AND lease_token IS NULL AND lease_expires_at IS NULL
            AND delivered_at IS NULL AND failed_at IS NULL)
        OR (status = 'delivering' AND result_payload IS NOT NULL
            AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL
            AND delivered_at IS NULL AND failed_at IS NULL)
        OR (status = 'delivered' AND result_payload IS NOT NULL
            AND lease_token IS NULL AND lease_expires_at IS NULL
            AND delivered_at IS NOT NULL AND failed_at IS NULL)
        OR (status = 'failed' AND result_payload IS NOT NULL
            AND lease_token IS NULL AND lease_expires_at IS NULL
            AND delivered_at IS NULL AND failed_at IS NOT NULL)
    )
);

CREATE INDEX chat_inbound_deliveries_received_idx
    ON chat_inbound_deliveries (next_attempt_at, accepted_sequence)
    WHERE status = 'received';

CREATE INDEX chat_inbound_deliveries_processing_lease_idx
    ON chat_inbound_deliveries (lease_expires_at, accepted_sequence)
    WHERE status = 'processing';

CREATE INDEX chat_inbound_deliveries_delivery_idx
    ON chat_inbound_deliveries (next_attempt_at, accepted_sequence)
    WHERE status = 'delivery_pending';

CREATE INDEX chat_inbound_deliveries_delivery_lease_idx
    ON chat_inbound_deliveries (lease_expires_at, accepted_sequence)
    WHERE status = 'delivering';

CREATE INDEX chat_inbound_deliveries_learner_order_idx
    ON chat_inbound_deliveries (tenant_id, learner_key, accepted_sequence)
    WHERE status NOT IN ('delivered', 'failed');

CREATE INDEX chat_inbound_deliveries_destination_order_idx
    ON chat_inbound_deliveries (tenant_id, destination_key, accepted_sequence)
    WHERE status NOT IN ('delivered', 'failed');

CREATE INDEX chat_inbound_deliveries_delivered_idx
    ON chat_inbound_deliveries (delivered_at, accepted_sequence)
    WHERE status = 'delivered';

CREATE INDEX chat_inbound_deliveries_failed_idx
    ON chat_inbound_deliveries (failed_at, accepted_sequence)
    WHERE status = 'failed';

-- +goose Down
DROP TABLE IF EXISTS chat_inbound_deliveries;

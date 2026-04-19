-- +goose Up
-- Embed configuration per tenant for the embeddable web chat widget.

CREATE TABLE embed_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    enabled         BOOLEAN NOT NULL DEFAULT false,
    allowed_origins TEXT[] NOT NULL DEFAULT '{}',
    theme_config    JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id)
);

CREATE INDEX idx_embed_configs_tenant_id ON embed_configs(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS embed_configs;

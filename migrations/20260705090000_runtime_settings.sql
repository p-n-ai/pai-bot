-- +goose Up
-- Platform-global runtime settings for the single process-wide AI router;
-- deliberately no tenant_id.

CREATE TABLE runtime_settings (
    id         SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    ai         JSONB NOT NULL DEFAULT '{}',
    flags      JSONB NOT NULL DEFAULT '{}',
    secrets    JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS runtime_settings;

-- +goose Up
ALTER TABLE runtime_settings
    ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision >= 0);

-- +goose Down
ALTER TABLE runtime_settings
    DROP COLUMN revision;

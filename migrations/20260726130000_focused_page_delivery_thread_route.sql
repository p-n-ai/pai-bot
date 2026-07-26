-- +goose Up
ALTER TABLE focused_page_deliveries
    ADD COLUMN thread_id TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE focused_page_deliveries
    DROP COLUMN thread_id;

-- +goose Up
ALTER TABLE groups ADD COLUMN closed BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE groups DROP COLUMN IF EXISTS closed;

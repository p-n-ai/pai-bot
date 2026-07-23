-- +goose Up
DROP INDEX focused_pages_expiry_idx;
CREATE INDEX focused_pages_expiry_idx ON focused_pages (expires_at, id);

-- +goose Down
DROP INDEX focused_pages_expiry_idx;
CREATE INDEX focused_pages_expiry_idx ON focused_pages (expires_at) WHERE status = 'active';

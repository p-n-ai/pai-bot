-- +goose NO TRANSACTION
-- +goose Up
DROP INDEX CONCURRENTLY IF EXISTS focused_pages_cleanup_idx;
CREATE INDEX CONCURRENTLY focused_pages_cleanup_idx ON focused_pages (expires_at, id);
DROP INDEX CONCURRENTLY IF EXISTS focused_pages_expiry_idx;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS focused_pages_expiry_idx;
CREATE INDEX CONCURRENTLY focused_pages_expiry_idx ON focused_pages (expires_at) WHERE status = 'active';
DROP INDEX CONCURRENTLY IF EXISTS focused_pages_cleanup_idx;

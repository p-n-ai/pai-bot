-- +goose Up
-- Rename auth_refresh_tokens to auth_sessions to match the current session-first contract.

ALTER TABLE auth_refresh_tokens RENAME TO auth_sessions;
ALTER TABLE auth_sessions RENAME CONSTRAINT auth_refresh_tokens_replaced_by_fkey TO auth_sessions_replaced_by_fkey;
ALTER INDEX idx_auth_refresh_tokens_user_id RENAME TO idx_auth_sessions_user_id;
ALTER INDEX idx_auth_refresh_tokens_tenant_id RENAME TO idx_auth_sessions_tenant_id;

-- +goose Down
ALTER INDEX idx_auth_sessions_tenant_id RENAME TO idx_auth_refresh_tokens_tenant_id;
ALTER INDEX idx_auth_sessions_user_id RENAME TO idx_auth_refresh_tokens_user_id;
ALTER TABLE auth_sessions RENAME CONSTRAINT auth_sessions_replaced_by_fkey TO auth_refresh_tokens_replaced_by_fkey;
ALTER TABLE auth_sessions RENAME TO auth_refresh_tokens;

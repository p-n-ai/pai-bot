-- +goose Up
-- Allow 'guest' role in users table for anonymous embed visitors.

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('student', 'teacher', 'parent', 'admin', 'platform_admin', 'guest'));

-- Update the tenant scope check: guests require a tenant_id (like students).
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_tenant_scope_check;
ALTER TABLE users ADD CONSTRAINT users_tenant_scope_check
    CHECK (
        (role = 'platform_admin' AND tenant_id IS NULL) OR
        (role <> 'platform_admin' AND tenant_id IS NOT NULL)
    );

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('student', 'teacher', 'parent', 'admin', 'platform_admin'));

-- Tenant scope check is unchanged in the down migration (same logic).

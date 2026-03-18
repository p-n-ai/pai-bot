-- +goose Up
ALTER TABLE users
    ALTER COLUMN tenant_id DROP NOT NULL;

UPDATE users
SET tenant_id = NULL
WHERE role = 'platform_admin';

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_tenant_scope_check;

ALTER TABLE users
    ADD CONSTRAINT users_tenant_scope_check
    CHECK (
        (role = 'platform_admin' AND tenant_id IS NULL) OR
        (role <> 'platform_admin' AND tenant_id IS NOT NULL)
    );

ALTER TABLE auth_identities
    ALTER COLUMN tenant_id DROP NOT NULL;

UPDATE auth_identities ai
SET tenant_id = NULL
FROM users u
WHERE ai.user_id = u.id
  AND u.role = 'platform_admin';

ALTER TABLE auth_refresh_tokens
    ALTER COLUMN tenant_id DROP NOT NULL;

UPDATE auth_refresh_tokens rt
SET tenant_id = NULL
FROM users u
WHERE rt.user_id = u.id
  AND u.role = 'platform_admin';

-- +goose Down
UPDATE users
SET tenant_id = COALESCE(
    tenant_id,
    (SELECT id FROM tenants ORDER BY created_at ASC LIMIT 1)
)
WHERE role = 'platform_admin';

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_tenant_scope_check;

ALTER TABLE users
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE users
    ADD CONSTRAINT users_tenant_scope_check
    CHECK (tenant_id IS NOT NULL);

UPDATE auth_identities ai
SET tenant_id = u.tenant_id
FROM users u
WHERE ai.user_id = u.id
  AND ai.tenant_id IS NULL;

ALTER TABLE auth_identities
    ALTER COLUMN tenant_id SET NOT NULL;

UPDATE auth_refresh_tokens rt
SET tenant_id = u.tenant_id
FROM users u
WHERE rt.user_id = u.id
  AND rt.tenant_id IS NULL;

ALTER TABLE auth_refresh_tokens
    ALTER COLUMN tenant_id SET NOT NULL;

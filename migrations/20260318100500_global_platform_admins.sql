-- +goose Up
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_tenant_scope_check;

ALTER TABLE auth_identities
    DROP CONSTRAINT IF EXISTS auth_identities_user_id_tenant_id_fkey;

ALTER TABLE auth_identities
    ADD CONSTRAINT auth_identities_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE;

ALTER TABLE auth_identities
    ALTER COLUMN tenant_id DROP NOT NULL;

ALTER TABLE auth_refresh_tokens
    ALTER COLUMN tenant_id DROP NOT NULL;

UPDATE auth_identities
SET tenant_id = NULL
WHERE user_id IN (
    SELECT id
    FROM users
    WHERE role = 'platform_admin'
);

UPDATE auth_refresh_tokens
SET tenant_id = NULL
WHERE user_id IN (
    SELECT id
    FROM users
    WHERE role = 'platform_admin'
);

ALTER TABLE users
    ALTER COLUMN tenant_id DROP NOT NULL;

UPDATE users
SET tenant_id = NULL
WHERE role = 'platform_admin';

ALTER TABLE users
    ADD CONSTRAINT users_tenant_scope_check
        CHECK (
            (role = 'platform_admin' AND tenant_id IS NULL)
            OR (role <> 'platform_admin' AND tenant_id IS NOT NULL)
        );

CREATE UNIQUE INDEX idx_auth_identities_global_provider_identifier
    ON auth_identities(provider, identifier_normalized)
    WHERE tenant_id IS NULL;

-- +goose Down
UPDATE users
SET tenant_id = (
    SELECT id
    FROM tenants
    ORDER BY created_at ASC, id ASC
    LIMIT 1
)
WHERE role = 'platform_admin'
  AND tenant_id IS NULL;

UPDATE auth_identities ai
SET tenant_id = u.tenant_id
FROM users u
WHERE ai.user_id = u.id
  AND ai.tenant_id IS NULL
  AND u.tenant_id IS NOT NULL;

UPDATE auth_refresh_tokens rt
SET tenant_id = u.tenant_id
FROM users u
WHERE rt.user_id = u.id
  AND rt.tenant_id IS NULL
  AND u.tenant_id IS NOT NULL;

DROP INDEX IF EXISTS idx_auth_identities_global_provider_identifier;

ALTER TABLE auth_identities
    DROP CONSTRAINT IF EXISTS auth_identities_user_id_fkey;

ALTER TABLE auth_identities
    ADD CONSTRAINT auth_identities_user_id_tenant_id_fkey
        FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id)
        ON DELETE CASCADE;

ALTER TABLE auth_refresh_tokens
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE auth_identities
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_tenant_scope_check;

ALTER TABLE users
    ALTER COLUMN tenant_id SET NOT NULL;

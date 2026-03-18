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

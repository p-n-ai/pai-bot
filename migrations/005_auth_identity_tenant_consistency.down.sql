ALTER TABLE auth_identities
    DROP CONSTRAINT IF EXISTS auth_identities_user_id_tenant_id_fkey;

ALTER TABLE auth_identities
    ADD CONSTRAINT auth_identities_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_id_tenant_id_key;

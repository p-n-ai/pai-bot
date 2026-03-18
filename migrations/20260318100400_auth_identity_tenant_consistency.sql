-- +goose Up
ALTER TABLE users
    ADD CONSTRAINT users_id_tenant_id_key UNIQUE (id, tenant_id);

ALTER TABLE auth_identities
    DROP CONSTRAINT IF EXISTS auth_identities_user_id_fkey;

ALTER TABLE auth_identities
    ADD CONSTRAINT auth_identities_user_id_tenant_id_fkey
        FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id)
        ON DELETE CASCADE;

-- +goose Down
ALTER TABLE auth_identities
    DROP CONSTRAINT IF EXISTS auth_identities_user_id_tenant_id_fkey;

ALTER TABLE auth_identities
    ADD CONSTRAINT auth_identities_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_id_tenant_id_key;

\set ON_ERROR_STOP on
\pset pager off

BEGIN;

DO $$
DECLARE
    target_tenant CONSTANT UUID := '8962d547-307c-48e9-970b-2d6064a64f0b';
    canonical_user CONSTANT UUID := 'b363e99e-11ad-460c-a7a1-a2e4218b3284';
    duplicate_users CONSTANT UUID[] := ARRAY[
        '6d505f34-0171-44d7-ada8-722974b68c0f'::UUID,
        '8760f550-1aa9-4007-9830-f319d3542030'::UUID,
        '0aa356bf-abea-475a-82b8-aa7777fbae5b'::UUID
    ];
    expected_users CONSTANT UUID[] := ARRAY[
        canonical_user,
        duplicate_users[1],
        duplicate_users[2],
        duplicate_users[3]
    ];
    actual_users UUID[];
    changed_rows INTEGER;
BEGIN
    PERFORM 1
    FROM users
    WHERE tenant_id = target_tenant
      AND channel = 'telegram'
      AND external_id = '6888713047'
    FOR UPDATE;

    SELECT ARRAY_AGG(id ORDER BY created_at, id)
    INTO actual_users
    FROM users
    WHERE tenant_id = target_tenant
      AND channel = 'telegram'
      AND external_id = '6888713047';

    IF actual_users = ARRAY[canonical_user]::UUID[] THEN
        RAISE NOTICE 'Telegram identity is already reconciled';
        RETURN;
    END IF;

    IF actual_users IS DISTINCT FROM expected_users THEN
        RAISE EXCEPTION
            'Refusing Telegram identity repair: current users do not match the reviewed conflict';
    END IF;

    UPDATE users
    SET external_id = NULL
    WHERE id = ANY(duplicate_users)
      AND tenant_id = target_tenant
      AND channel = 'telegram'
      AND external_id = '6888713047';

    GET DIAGNOSTICS changed_rows = ROW_COUNT;
    IF changed_rows <> 3 THEN
        RAISE EXCEPTION
            'Telegram identity repair changed % rows, expected 3',
            changed_rows;
    END IF;
END
$$;

COMMIT;

SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE external_id IS NOT NULL
    GROUP BY tenant_id, channel, external_id
    HAVING COUNT(*) > 1
) AS duplicate_identities_exist
\gset

\if :duplicate_identities_exist
\echo 'ERROR: duplicate provider-qualified user identities block conversation thread migration.'
SELECT tenant_id,
       channel,
       external_id,
       ARRAY_AGG(id ORDER BY created_at, id) AS user_ids
FROM users
WHERE external_id IS NOT NULL
GROUP BY tenant_id, channel, external_id
HAVING COUNT(*) > 1
ORDER BY tenant_id, channel, external_id;
\echo 'Reconcile the listed users using docs/operations/conversation-identity-migration.md, then rerun this preflight.'
DO $$
BEGIN
    RAISE EXCEPTION 'conversation identity preflight failed';
END
$$;
\endif

\echo 'Conversation identity preflight passed.'

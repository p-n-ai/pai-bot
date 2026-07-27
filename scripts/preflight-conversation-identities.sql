\set ON_ERROR_STOP on
\pset pager off

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

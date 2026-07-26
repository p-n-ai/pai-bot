-- +goose Up
ALTER TABLE conversations
    ADD COLUMN thread_id TEXT NOT NULL DEFAULT '';

-- The runtime resolves users by this exact provider-qualified identity. Refuse
-- to guess when historical duplicates exist because merging user rows would
-- also require reconciling learning progress, goals, auth, and challenge data.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        WHERE external_id IS NOT NULL
        GROUP BY tenant_id, channel, external_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION
            'conversation thread migration requires unique users by tenant, channel, and external_id; reconcile duplicate users before retrying';
    END IF;
END
$$;
-- +goose StatementEnd

CREATE UNIQUE INDEX uniq_users_tenant_channel_external_id
    ON users (tenant_id, channel, external_id)
    WHERE external_id IS NOT NULL;

-- Existing schemas did not prevent more than one active conversation for a
-- learner. Keep the newest row active before adding the scoped uniqueness
-- invariant so this migration is safe on databases that already contain
-- duplicates.
WITH ranked_active AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY tenant_id, user_id, thread_id
               ORDER BY started_at DESC, id DESC
           ) AS active_rank
    FROM conversations
    WHERE ended_at IS NULL
)
UPDATE conversations AS conversation
SET ended_at = GREATEST(NOW(), conversation.started_at)
FROM ranked_active
WHERE conversation.id = ranked_active.id
  AND ranked_active.active_rank > 1;

CREATE UNIQUE INDEX uniq_conversations_active_thread
    ON conversations (tenant_id, user_id, thread_id)
    WHERE ended_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS uniq_conversations_active_thread;
DROP INDEX IF EXISTS uniq_users_tenant_channel_external_id;
ALTER TABLE conversations DROP COLUMN IF EXISTS thread_id;

-- Ending duplicate legacy conversations in the Up migration is deliberately
-- irreversible: restoring them as active would violate the scoped invariant.

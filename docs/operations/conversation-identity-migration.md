# Conversation identity migration

Migration `20260726120000_conversation_threads.sql` makes
`(tenant_id, channel, external_id)` unique for non-null external IDs. Run the
preflight before migrating:

```bash
just conversation-identity-preflight
```

The preflight reports every conflicting identity and exits without changing
data. For each conflict, choose the user that should continue receiving
messages from that provider. Preserve the other user and its learning data by
clearing only its obsolete provider identity:

```sql
BEGIN;

SELECT id, tenant_id, channel, external_id, name, created_at
FROM users
WHERE tenant_id = '<tenant-id>'
  AND channel = '<channel>'
  AND external_id = '<external-id>'
FOR UPDATE;

UPDATE users
SET external_id = NULL
WHERE id = '<duplicate-user-id>'
  AND tenant_id = '<tenant-id>'
  AND channel = '<channel>'
  AND external_id = '<external-id>';

COMMIT;
```

Repeat the update for every non-canonical duplicate, then rerun the preflight.
Do not delete or merge user rows as part of this migration: progress, goals,
authentication, and challenge records need domain-specific reconciliation.
The deployment script runs the same preflight and stops before applying any
migration when conflicts remain.

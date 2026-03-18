ALTER TABLE challenges
    DROP COLUMN IF EXISTS opponent_accepted_at,
    DROP COLUMN IF EXISTS creator_accepted_at;

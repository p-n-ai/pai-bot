-- +goose Up
ALTER TABLE challenges
    ADD COLUMN creator_accepted_at TIMESTAMPTZ,
    ADD COLUMN opponent_accepted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE challenges
    DROP COLUMN IF EXISTS opponent_accepted_at,
    DROP COLUMN IF EXISTS creator_accepted_at;

ALTER TABLE challenges
    ADD COLUMN creator_accepted_at TIMESTAMPTZ,
    ADD COLUMN opponent_accepted_at TIMESTAMPTZ;

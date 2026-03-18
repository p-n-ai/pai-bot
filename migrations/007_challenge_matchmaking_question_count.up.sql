ALTER TABLE challenge_matchmaking_tickets
    ADD COLUMN question_count INTEGER NOT NULL DEFAULT 5 CHECK (question_count > 0);

UPDATE challenge_matchmaking_tickets AS t
SET question_count = c.question_count
FROM challenges AS c
WHERE t.matched_challenge_id = c.id;

-- P&AI Bot — Challenges with matchmaking, private invites, and AI fallback

CREATE TABLE challenges (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                      TEXT NOT NULL UNIQUE,
    source                    TEXT NOT NULL CHECK (source IN ('public_queue', 'private_code', 'ai_fallback')),
    opponent_type             TEXT NOT NULL CHECK (opponent_type IN ('human', 'ai')),
    creator_user_id           UUID NOT NULL REFERENCES users(id),
    opponent_user_id          UUID REFERENCES users(id),
    tenant_id                 UUID NOT NULL REFERENCES tenants(id),
    topic_id                  TEXT NOT NULL,
    topic_name                TEXT NOT NULL,
    subject_id                TEXT NOT NULL DEFAULT '',
    syllabus_id               TEXT NOT NULL,
    questions                 JSONB NOT NULL DEFAULT '[]'::jsonb,
    question_count            INTEGER NOT NULL DEFAULT 0 CHECK (question_count >= 0),
    state                     TEXT NOT NULL DEFAULT 'waiting' CHECK (state IN ('waiting', 'ready', 'active', 'completed')),
    creator_ready_at          TIMESTAMPTZ,
    opponent_ready_at         TIMESTAMPTZ,
    creator_correct_count     INTEGER NOT NULL DEFAULT 0,
    opponent_correct_count    INTEGER NOT NULL DEFAULT 0,
    creator_completed_at      TIMESTAMPTZ,
    opponent_completed_at     TIMESTAMPTZ,
    creator_finish_xp_granted BOOLEAN NOT NULL DEFAULT FALSE,
    opponent_finish_xp_granted BOOLEAN NOT NULL DEFAULT FALSE,
    winner_user_id            UUID REFERENCES users(id),
    winner_xp_granted         BOOLEAN NOT NULL DEFAULT FALSE,
    metadata                  JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_challenges_tenant_state ON challenges(tenant_id, state);
CREATE INDEX idx_challenges_tenant_source_state ON challenges(tenant_id, source, state);
CREATE INDEX idx_challenges_creator ON challenges(creator_user_id);
CREATE INDEX idx_challenges_opponent ON challenges(opponent_user_id);

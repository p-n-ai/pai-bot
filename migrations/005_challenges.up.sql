-- P&AI Bot — challenge groundwork

CREATE TABLE challenges (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    creator_user_id     UUID NOT NULL REFERENCES users(id),
    opponent_user_id    UUID REFERENCES users(id),
    topic_id            TEXT NOT NULL,
    topic_name          TEXT NOT NULL,
    syllabus_id         TEXT NOT NULL,
    match_source        TEXT NOT NULL CHECK (match_source IN ('invite_code', 'queue', 'ai_fallback')),
    opponent_kind       TEXT NOT NULL DEFAULT 'human' CHECK (opponent_kind IN ('human', 'ai')),
    invite_code         TEXT,
    question_count      INTEGER NOT NULL CHECK (question_count > 0),
    state               TEXT NOT NULL CHECK (state IN ('waiting', 'pending_acceptance', 'ready', 'active', 'completed', 'expired', 'cancelled')),
    question_snapshot   JSONB,
    settlement_metadata JSONB,
    settlement_version  INTEGER NOT NULL DEFAULT 0,
    join_deadline_at    TIMESTAMPTZ,
    ready_at            TIMESTAMPTZ,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    settled_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT challenges_distinct_participants CHECK (creator_user_id IS DISTINCT FROM opponent_user_id)
);

CREATE UNIQUE INDEX uniq_challenges_active_invite_code
    ON challenges (tenant_id, invite_code)
    WHERE invite_code IS NOT NULL AND state IN ('waiting', 'pending_acceptance', 'ready', 'active');

CREATE INDEX idx_challenges_tenant_state_ready
    ON challenges (tenant_id, state, join_deadline_at, ready_at);

CREATE TABLE challenge_attempts (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id              UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    tenant_id                 UUID NOT NULL REFERENCES tenants(id),
    user_id                   UUID REFERENCES users(id),
    participant_kind          TEXT NOT NULL DEFAULT 'human' CHECK (participant_kind IN ('human', 'ai')),
    participant_profile       JSONB,
    state                     TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'active', 'completed', 'timed_out', 'cancelled')),
    answer_snapshot           JSONB,
    correct_count             INTEGER NOT NULL DEFAULT 0 CHECK (correct_count >= 0),
    total_time_ms             INTEGER NOT NULL DEFAULT 0 CHECK (total_time_ms >= 0),
    xp_awarded_at             TIMESTAMPTZ,
    review_reward_awarded_at  TIMESTAMPTZ,
    review_completed_at       TIMESTAMPTZ,
    started_at                TIMESTAMPTZ,
    completed_at              TIMESTAMPTZ,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uniq_challenge_attempts_user
    ON challenge_attempts (challenge_id, user_id)
    WHERE user_id IS NOT NULL;

CREATE TABLE challenge_matchmaking_tickets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    user_id             UUID NOT NULL REFERENCES users(id),
    topic_id            TEXT NOT NULL,
    topic_name          TEXT NOT NULL,
    syllabus_id         TEXT NOT NULL,
    status              TEXT NOT NULL CHECK (status IN ('searching', 'matched', 'cancelled', 'expired')),
    matched_challenge_id UUID REFERENCES challenges(id),
    expires_at          TIMESTAMPTZ NOT NULL,
    cancelled_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uniq_matchmaking_active_ticket
    ON challenge_matchmaking_tickets (tenant_id, user_id)
    WHERE status = 'searching';

CREATE INDEX idx_matchmaking_search
    ON challenge_matchmaking_tickets (tenant_id, topic_id, status, created_at);

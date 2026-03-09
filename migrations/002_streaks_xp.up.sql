-- P&AI Bot — Streaks & XP Schema (Day 8)

-- Streak tracking per user
CREATE TABLE streaks (
    user_id         UUID PRIMARY KEY REFERENCES users(id),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    current_streak  INTEGER NOT NULL DEFAULT 0,
    longest_streak  INTEGER NOT NULL DEFAULT 0,
    last_active_date DATE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_streaks_tenant_id ON streaks(tenant_id);

-- XP ledger: immutable log of all XP awards
CREATE TABLE xp_ledger (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    source      TEXT NOT NULL CHECK (source IN ('session', 'quiz', 'mastery', 'streak', 'challenge', 'review')),
    amount      INTEGER NOT NULL,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_xp_ledger_user_id ON xp_ledger(user_id);
CREATE INDEX idx_xp_ledger_tenant_id ON xp_ledger(tenant_id);

-- Nudge tracking for scheduler (max 3/day enforcement)
CREATE TABLE nudge_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    nudge_type  TEXT NOT NULL,
    topic_id    TEXT,
    sent_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_nudge_log_user_id_sent ON nudge_log(user_id, sent_at);

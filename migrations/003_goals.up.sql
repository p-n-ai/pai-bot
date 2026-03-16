-- P&AI Bot — Goal tracking (Day 11)

CREATE TABLE goals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    summary         TEXT NOT NULL,
    topic_id        TEXT NOT NULL,
    topic_name      TEXT NOT NULL,
    syllabus_id     TEXT NOT NULL,
    target_mastery  REAL NOT NULL CHECK (target_mastery >= 0.0 AND target_mastery <= 1.0),
    current_mastery REAL NOT NULL DEFAULT 0.0 CHECK (current_mastery >= 0.0 AND current_mastery <= 1.0),
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'archived')),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_goals_user_status ON goals(user_id, status);
CREATE INDEX idx_goals_tenant_status ON goals(tenant_id, status);

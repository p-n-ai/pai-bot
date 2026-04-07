-- +goose Up
-- Groups: generic grouping entity. A "class" is a group with type='class'.
-- Bot-created groups default to 'study_group'; 'class' is admin-only.

CREATE TABLE groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    type        TEXT NOT NULL DEFAULT 'study_group' CHECK (type IN ('class', 'study_group')),
    description TEXT NOT NULL DEFAULT '',
    syllabus    TEXT NOT NULL DEFAULT '',
    subject     TEXT NOT NULL DEFAULT '',
    cadence     TEXT NOT NULL DEFAULT '',
    join_code   TEXT NOT NULL,
    created_by  UUID REFERENCES users(id),
    config      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(join_code)
);

CREATE INDEX idx_groups_tenant_id ON groups(tenant_id);
CREATE INDEX idx_groups_type ON groups(type);
CREATE INDEX idx_groups_join_code ON groups(join_code);

CREATE TABLE group_members (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id  UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    role      TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'leader', 'teacher')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, user_id)
);

CREATE INDEX idx_group_members_group_id ON group_members(group_id);
CREATE INDEX idx_group_members_user_id ON group_members(user_id);
CREATE INDEX idx_group_members_tenant_id ON group_members(tenant_id);

-- Enforce tenant consistency: member must belong to same tenant as group.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_group_member_tenant() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM groups g
        JOIN users u ON u.tenant_id = g.tenant_id
        WHERE g.id = NEW.group_id
          AND u.id = NEW.user_id
          AND g.tenant_id = NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'group_members: tenant mismatch between group, user, and membership';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_group_member_tenant_check
    BEFORE INSERT OR UPDATE ON group_members
    FOR EACH ROW EXECUTE FUNCTION check_group_member_tenant();

-- Mastery snapshots: daily snapshot of mastery_score for leaderboard history.
-- The scheduler writes one row per user/topic/day so we can compute weekly deltas.
CREATE TABLE mastery_snapshots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    topic_id    TEXT NOT NULL,
    mastery_score REAL NOT NULL,
    snapshot_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, topic_id, snapshot_date)
);

CREATE INDEX idx_mastery_snapshots_user_date ON mastery_snapshots(user_id, snapshot_date);
CREATE INDEX idx_mastery_snapshots_tenant_date ON mastery_snapshots(tenant_id, snapshot_date);

-- +goose Down
DROP TRIGGER IF EXISTS trg_group_member_tenant_check ON group_members;
DROP FUNCTION IF EXISTS check_group_member_tenant();
DROP TABLE IF EXISTS mastery_snapshots;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;

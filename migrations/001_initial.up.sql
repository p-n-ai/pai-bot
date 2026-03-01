-- P&AI Bot — Initial Schema
-- All tables include tenant_id for multi-tenancy.

-- Multi-tenancy
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    config      JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Users (students, teachers, parents, admins)
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    role        TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'parent', 'admin', 'platform_admin')),
    name        TEXT NOT NULL,
    external_id TEXT,
    channel     TEXT NOT NULL DEFAULT 'telegram',
    form        TEXT,
    config      JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_external_id ON users(external_id);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- Conversations
CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    topic_id    TEXT,
    state       TEXT NOT NULL DEFAULT 'idle',
    metadata    JSONB DEFAULT '{}',
    started_at  TIMESTAMPTZ DEFAULT NOW(),
    ended_at    TIMESTAMPTZ
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);

-- Messages
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    role            TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content         TEXT NOT NULL,
    model           TEXT,
    input_tokens    INTEGER,
    output_tokens   INTEGER,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);

-- Learning progress per topic (SM-2 data)
CREATE TABLE learning_progress (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    syllabus_id     TEXT NOT NULL,
    topic_id        TEXT NOT NULL,
    mastery_score   REAL DEFAULT 0.0,
    ease_factor     REAL DEFAULT 2.5,
    interval_days   INTEGER DEFAULT 1,
    repetitions     INTEGER DEFAULT 0,
    next_review_at  TIMESTAMPTZ,
    last_studied_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, syllabus_id, topic_id)
);

CREATE INDEX idx_learning_progress_user_id ON learning_progress(user_id);
CREATE INDEX idx_learning_progress_next_review ON learning_progress(next_review_at);

-- Events (analytics / audit log)
CREATE TABLE events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    user_id     UUID REFERENCES users(id),
    event_type  TEXT NOT NULL,
    data        JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_events_user_id ON events(user_id);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_created_at ON events(created_at);

-- Insert default tenant for single-tenant mode
INSERT INTO tenants (name, slug) VALUES ('Default', 'default');

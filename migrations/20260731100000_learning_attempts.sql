-- +goose Up
CREATE TABLE learning_attempts (
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    user_id         UUID NOT NULL,
    attempt_id      TEXT NOT NULL CHECK (length(attempt_id) BETWEEN 1 AND 256),
    syllabus_id     TEXT NOT NULL,
    topic_id        TEXT NOT NULL,
    source_kind     TEXT NOT NULL,
    source_id       TEXT NOT NULL,
    source_revision TEXT NOT NULL CHECK (length(source_revision) BETWEEN 1 AND 256),
    score           DOUBLE PRECISION NOT NULL CHECK (score BETWEEN 0 AND 1),
    payload_hash    BYTEA NOT NULL CHECK (octet_length(payload_hash) = 32),
    mastery_before  DOUBLE PRECISION NOT NULL CHECK (mastery_before BETWEEN 0 AND 1),
    mastery_after   DOUBLE PRECISION NOT NULL CHECK (mastery_after BETWEEN 0 AND 1),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, user_id, attempt_id),
    CONSTRAINT learning_attempts_user_tenant_fkey
        FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id)
        ON DELETE CASCADE
);

CREATE INDEX idx_learning_attempts_topic
    ON learning_attempts(tenant_id, user_id, syllabus_id, topic_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS learning_attempts;

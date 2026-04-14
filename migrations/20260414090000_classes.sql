-- +goose Up
-- Minimal persisted class entity for onboarding and public join resolution.

CREATE TABLE classes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL,
    syllabus_id   TEXT NOT NULL,
    config        JSONB DEFAULT '{}',
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (slug)
);

CREATE INDEX idx_classes_tenant_id ON classes(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS classes;

-- +goose Up
-- P&AI Bot - Token budget tracking

CREATE TABLE token_budgets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    user_id         UUID REFERENCES users(id),
    budget_tokens   BIGINT NOT NULL CHECK (budget_tokens > 0),
    used_tokens     BIGINT NOT NULL DEFAULT 0 CHECK (used_tokens >= 0),
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT token_budgets_period_check CHECK (period_end > period_start),
    CONSTRAINT token_budgets_scope_unique UNIQUE (tenant_id, user_id, period_start, period_end)
);

CREATE INDEX idx_token_budgets_tenant_period ON token_budgets(tenant_id, period_start, period_end);
CREATE INDEX idx_token_budgets_user_period ON token_budgets(user_id, period_start, period_end);

-- +goose Down
DROP TABLE IF EXISTS token_budgets;

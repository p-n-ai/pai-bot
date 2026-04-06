-- +goose Up
ALTER TABLE auth_invites
ADD COLUMN IF NOT EXISTS delivery_status TEXT NOT NULL DEFAULT 'pending' CHECK (delivery_status IN ('pending', 'sent', 'failed')),
ADD COLUMN IF NOT EXISTS delivery_attempted_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS delivery_sent_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS delivery_error TEXT;

UPDATE auth_invites
SET delivery_status = 'pending'
WHERE delivery_status IS NULL;

-- +goose Down
ALTER TABLE auth_invites
DROP COLUMN IF EXISTS delivery_error,
DROP COLUMN IF EXISTS delivery_sent_at,
DROP COLUMN IF EXISTS delivery_attempted_at,
DROP COLUMN IF EXISTS delivery_status;

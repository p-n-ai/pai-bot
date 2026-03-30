-- +goose Up
UPDATE users SET config = config || '{"ab_group":"A"}'
WHERE config->>'ab_group' IS NULL;

-- +goose Down
UPDATE users SET config = config - 'ab_group';

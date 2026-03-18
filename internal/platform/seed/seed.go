package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p-n-ai/pai-bot/internal/auth"
)

const tenantUpsertSQL = `
INSERT INTO tenants (name, slug, config)
VALUES ('Demo School', 'default', '{"seeded": true, "source": "demo"}'::jsonb)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    config = tenants.config || EXCLUDED.config
RETURNING id::text
`

type beginner interface {
	Begin(ctx context.Context) (txLike, error)
}

type txLike interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type poolBeginner struct {
	pool *pgxpool.Pool
}

func (p poolBeginner) Begin(ctx context.Context) (txLike, error) {
	return p.pool.Begin(ctx)
}

// SeedDemo inserts a small idempotent demo dataset into the current database.
func SeedDemo(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("pool is nil")
	}
	return seedDemo(ctx, poolBeginner{pool: pool})
}

func seedDemo(ctx context.Context, db beginner) (err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if err == nil {
			return
		}
		_ = tx.Rollback(ctx)
	}()

	var tenantID string
	if err = tx.QueryRow(ctx, tenantUpsertSQL).Scan(&tenantID); err != nil {
		return fmt.Errorf("upsert default tenant: %w", err)
	}

	for i, stmt := range demoStatements(tenantID) {
		if _, err = tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("seed statement %d: %w", i+1, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}

func demoStatements(tenantID string) []string {
	passwordHash, _ := auth.HashPassword("demo-password")
	studentEmail := "student@example.com"
	teacherEmail := "teacher@example.com"
	parentEmail := "parent@example.com"
	adminEmail := "admin@example.com"
	platformAdminEmail := "platform-admin@example.com"

	return []string{
		fmt.Sprintf(`
INSERT INTO users (id, tenant_id, role, name, external_id, channel, form, config)
VALUES
('10000000-0000-0000-0000-000000000001', '%[1]s', 'teacher', 'Aisyah Teacher', 'teacher_1', 'telegram', 'Form 1', '{"subject":"Matematik"}'::jsonb),
('10000000-0000-0000-0000-000000000002', '%[1]s', 'student', 'Alya Sofea', 'stu_1', 'telegram', 'Form 1', '{"preferred_language":"bm"}'::jsonb),
('10000000-0000-0000-0000-000000000003', '%[1]s', 'student', 'Hakim Firdaus', 'stu_2', 'telegram', 'Form 1', '{"preferred_language":"en"}'::jsonb),
('10000000-0000-0000-0000-000000000004', '%[1]s', 'student', 'Mei Lin', 'stu_3', 'telegram', 'Form 2', '{"preferred_language":"bm"}'::jsonb),
('10000000-0000-0000-0000-000000000005', '%[1]s', 'parent', 'Farah Parent', 'parent_1', 'telegram', NULL, '{"children":["stu_1"]}'::jsonb),
('10000000-0000-0000-0000-000000000006', '%[1]s', 'admin', 'Nadia Admin', 'admin_1', 'web', NULL, '{"scope":"school"}'::jsonb),
('10000000-0000-0000-0000-000000000007', NULL, 'platform_admin', 'P&AI Platform Admin', 'platform_admin_1', 'web', NULL, '{"scope":"platform"}'::jsonb)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    tenant_id = EXCLUDED.tenant_id,
    external_id = EXCLUDED.external_id,
    channel = EXCLUDED.channel,
    form = EXCLUDED.form,
    config = EXCLUDED.config,
    updated_at = NOW()
`, tenantID),
		fmt.Sprintf(`
INSERT INTO auth_identities (
    user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at, last_login_at, created_at, updated_at
)
VALUES
('10000000-0000-0000-0000-000000000001', '%[1]s', 'password', '%[2]s', '%[3]s', '%[10]s', NOW(), NOW(), NOW(), NOW()),
('10000000-0000-0000-0000-000000000002', '%[1]s', 'password', '%[4]s', '%[5]s', '%[10]s', NOW(), NOW(), NOW(), NOW()),
('10000000-0000-0000-0000-000000000005', '%[1]s', 'password', '%[6]s', '%[7]s', '%[10]s', NOW(), NOW(), NOW(), NOW()),
('10000000-0000-0000-0000-000000000006', '%[1]s', 'password', '%[8]s', '%[9]s', '%[10]s', NOW(), NOW(), NOW(), NOW())
ON CONFLICT (tenant_id, provider, identifier_normalized) DO UPDATE
SET password_hash = EXCLUDED.password_hash,
    user_id = EXCLUDED.user_id,
    identifier = EXCLUDED.identifier,
    identifier_normalized = EXCLUDED.identifier_normalized,
    email_verified_at = EXCLUDED.email_verified_at,
    last_login_at = EXCLUDED.last_login_at,
    updated_at = NOW()
`, tenantID,
			teacherEmail, auth.NormalizeIdentifier(teacherEmail),
			studentEmail, auth.NormalizeIdentifier(studentEmail),
			parentEmail, auth.NormalizeIdentifier(parentEmail),
			adminEmail, auth.NormalizeIdentifier(adminEmail),
			passwordHash,
			platformAdminEmail, auth.NormalizeIdentifier(platformAdminEmail)),
		fmt.Sprintf(`
WITH existing_identity AS (
    SELECT id
    FROM auth_identities
    WHERE user_id = '10000000-0000-0000-0000-000000000007'
       OR identifier_normalized = '%[2]s'
    ORDER BY created_at ASC
    LIMIT 1
),
updated_identity AS (
    UPDATE auth_identities
    SET user_id = '10000000-0000-0000-0000-000000000007',
        tenant_id = NULL,
        provider = 'password',
        identifier = '%[1]s',
        identifier_normalized = '%[2]s',
        password_hash = '%[3]s',
        email_verified_at = NOW(),
        last_login_at = NOW(),
        updated_at = NOW()
    WHERE id IN (SELECT id FROM existing_identity)
    RETURNING id
)
INSERT INTO auth_identities (
    id, user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at, last_login_at, created_at, updated_at
)
SELECT
    '60000000-0000-0000-0000-000000000005',
    '10000000-0000-0000-0000-000000000007',
    NULL,
    'password',
    '%[1]s',
    '%[2]s',
    '%[3]s',
    NOW(),
    NOW(),
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM updated_identity)
  AND NOT EXISTS (
      SELECT 1
      FROM auth_identities
      WHERE user_id = '10000000-0000-0000-0000-000000000007'
         OR identifier_normalized = '%[2]s'
  )
`, platformAdminEmail, auth.NormalizeIdentifier(platformAdminEmail), passwordHash),
		fmt.Sprintf(`
INSERT INTO conversations (id, user_id, tenant_id, topic_id, state, metadata, started_at)
VALUES
('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', '%[1]s', 'kssm-f1-algebra-linear-equations', 'active', '{"source":"seed"}'::jsonb, NOW() - INTERVAL '2 day'),
('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000003', '%[1]s', 'kssm-f2-algebra-patterns', 'idle', '{"source":"seed"}'::jsonb, NOW() - INTERVAL '1 day'),
('20000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000004', '%[1]s', 'kssm-f3-algebra-simultaneous-equations', 'active', '{"source":"seed"}'::jsonb, NOW() - INTERVAL '6 hour')
ON CONFLICT (id) DO UPDATE
SET topic_id = EXCLUDED.topic_id,
    state = EXCLUDED.state,
    metadata = EXCLUDED.metadata
`, tenantID),
		fmt.Sprintf(`
INSERT INTO messages (id, conversation_id, tenant_id, role, content, model, input_tokens, output_tokens, created_at)
VALUES
('30000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '%[1]s', 'user', 'Cikgu, macam mana nak selesaikan 2x + 3 = 11?', 'openai:gpt-4o-mini', 48, 0, NOW() - INTERVAL '2 day'),
('30000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '%[1]s', 'assistant', 'Tolak 3 pada kedua-dua belah dahulu, jadi 2x = 8. Kemudian bahagi 2, maka x = 4.', 'openai:gpt-4o-mini', 48, 72, NOW() - INTERVAL '2 day' + INTERVAL '5 second'),
('30000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000002', '%[1]s', 'user', 'How do I continue a number pattern that increases by 5?', 'openai:gpt-4o-mini', 36, 0, NOW() - INTERVAL '1 day'),
('30000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000002', '%[1]s', 'assistant', 'Look at the gap between each term. If the difference is always 5, add 5 to get the next value.', 'openai:gpt-4o-mini', 36, 54, NOW() - INTERVAL '1 day' + INTERVAL '4 second'),
('30000000-0000-0000-0000-000000000005', '20000000-0000-0000-0000-000000000003', '%[1]s', 'user', 'Saya keliru dengan persamaan serentak.', 'openai:gpt-4o-mini', 29, 0, NOW() - INTERVAL '6 hour'),
('30000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000003', '%[1]s', 'assistant', 'Kita selesaikan satu pemboleh ubah dahulu, kemudian gantikan ke dalam persamaan yang satu lagi.', 'openai:gpt-4o-mini', 29, 61, NOW() - INTERVAL '6 hour' + INTERVAL '3 second')
ON CONFLICT (id) DO UPDATE
SET content = EXCLUDED.content,
    model = EXCLUDED.model,
    input_tokens = EXCLUDED.input_tokens,
    output_tokens = EXCLUDED.output_tokens
`, tenantID),
		fmt.Sprintf(`
INSERT INTO learning_progress (id, user_id, tenant_id, syllabus_id, topic_id, mastery_score, ease_factor, interval_days, repetitions, next_review_at, last_studied_at)
VALUES
('40000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', '%[1]s', 'kssm-form-1', 'linear-equations', 0.86, 2.5, 6, 4, NOW() + INTERVAL '1 day', NOW() - INTERVAL '1 day'),
('40000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002', '%[1]s', 'kssm-form-1', 'algebraic-expressions', 0.62, 2.2, 4, 3, NOW() + INTERVAL '12 hour', NOW() - INTERVAL '1 day' + INTERVAL '20 minute'),
('40000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000002', '%[1]s', 'kssm-form-1', 'inequalities', 0.44, 1.9, 2, 2, NOW() + INTERVAL '8 hour', NOW() - INTERVAL '2 day'),
('40000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000002', '%[1]s', 'kssm-form-1', 'functions', 0.30, 1.8, 1, 1, NOW() + INTERVAL '6 hour', NOW() - INTERVAL '3 day'),
('40000000-0000-0000-0000-000000000005', '10000000-0000-0000-0000-000000000003', '%[1]s', 'kssm-form-1', 'linear-equations', 0.38, 1.9, 2, 2, NOW() + INTERVAL '10 hour', NOW() - INTERVAL '1 day'),
('40000000-0000-0000-0000-000000000006', '10000000-0000-0000-0000-000000000003', '%[1]s', 'kssm-form-1', 'algebraic-expressions', 0.57, 2.1, 3, 2, NOW() + INTERVAL '14 hour', NOW() - INTERVAL '1 day' + INTERVAL '20 minute'),
('40000000-0000-0000-0000-000000000007', '10000000-0000-0000-0000-000000000003', '%[1]s', 'kssm-form-1', 'inequalities', 0.21, 1.7, 1, 1, NOW() + INTERVAL '5 hour', NOW() - INTERVAL '2 day'),
('40000000-0000-0000-0000-000000000008', '10000000-0000-0000-0000-000000000003', '%[1]s', 'kssm-form-1', 'functions', 0.18, 1.6, 1, 1, NOW() + INTERVAL '3 hour', NOW() - INTERVAL '2 day' + INTERVAL '30 minute'),
('40000000-0000-0000-0000-000000000009', '10000000-0000-0000-0000-000000000004', '%[1]s', 'kssm-form-2', 'linear-equations', 0.92, 2.6, 7, 5, NOW() + INTERVAL '2 day', NOW() - INTERVAL '1 day'),
('40000000-0000-0000-0000-000000000010', '10000000-0000-0000-0000-000000000004', '%[1]s', 'kssm-form-2', 'algebraic-expressions', 0.84, 2.5, 6, 5, NOW() + INTERVAL '36 hour', NOW() - INTERVAL '1 day' + INTERVAL '25 minute'),
('40000000-0000-0000-0000-000000000011', '10000000-0000-0000-0000-000000000004', '%[1]s', 'kssm-form-2', 'inequalities', 0.74, 2.3, 4, 4, NOW() + INTERVAL '18 hour', NOW() - INTERVAL '2 day'),
('40000000-0000-0000-0000-000000000012', '10000000-0000-0000-0000-000000000004', '%[1]s', 'kssm-form-2', 'functions', 0.59, 2.1, 3, 3, NOW() + INTERVAL '12 hour', NOW() - INTERVAL '2 day' + INTERVAL '20 minute')
ON CONFLICT (user_id, syllabus_id, topic_id) DO UPDATE
SET mastery_score = EXCLUDED.mastery_score,
    ease_factor = EXCLUDED.ease_factor,
    interval_days = EXCLUDED.interval_days,
    repetitions = EXCLUDED.repetitions,
    next_review_at = EXCLUDED.next_review_at,
    last_studied_at = EXCLUDED.last_studied_at,
    updated_at = NOW()
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000001', '%[1]s', '10000000-0000-0000-0000-000000000002', 'session_started', '{"topic_id":"kssm-f1-algebra-linear-equations"}'::jsonb, NOW() - INTERVAL '2 day'),
('50000000-0000-0000-0000-000000000002', '%[1]s', '10000000-0000-0000-0000-000000000002', 'answer_rating_submitted', '{"rating":5,"source":"seed"}'::jsonb, NOW() - INTERVAL '2 day' + INTERVAL '1 minute'),
('50000000-0000-0000-0000-000000000003', '%[1]s', '10000000-0000-0000-0000-000000000003', 'quiz_completed', '{"score":4,"out_of":5,"source":"seed"}'::jsonb, NOW() - INTERVAL '1 day'),
('50000000-0000-0000-0000-000000000004', '%[1]s', '10000000-0000-0000-0000-000000000004', 'nudge_sent', '{"reason":"review_due","source":"seed"}'::jsonb, NOW() - INTERVAL '5 hour')
ON CONFLICT (id) DO UPDATE
SET data = EXCLUDED.data,
    created_at = EXCLUDED.created_at
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000005', '%[1]s', '10000000-0000-0000-0000-000000000002', 'progress_viewed', '{"surface":"seed-demo"}'::jsonb, NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000006', '%[1]s', '10000000-0000-0000-0000-000000000003', 'topic_selected', '{"topic_id":"kssm-f2-algebra-patterns","source":"seed"}'::jsonb, NOW() - INTERVAL '22 hour')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000007', '%[1]s', '10000000-0000-0000-0000-000000000004', 'help_requested', '{"channel":"telegram","source":"seed"}'::jsonb, NOW() - INTERVAL '5 hour')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000008', '%[1]s', '10000000-0000-0000-0000-000000000001', 'teacher_dashboard_opened', '{"surface":"demo"}'::jsonb, NOW() - INTERVAL '3 hour')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000009', '%[1]s', '10000000-0000-0000-0000-000000000005', 'parent_report_viewed', '{"surface":"demo"}'::jsonb, NOW() - INTERVAL '2 hour')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000010', '%[1]s', '10000000-0000-0000-0000-000000000002', 'study_streak_extended', '{"days":4,"source":"seed"}'::jsonb, NOW() - INTERVAL '90 minute')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000011', '%[1]s', '10000000-0000-0000-0000-000000000003', 'ai_response', '{"model":"openai:gpt-4o-mini","source":"seed"}'::jsonb, NOW() - INTERVAL '1 day' + INTERVAL '4 second')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000012', '%[1]s', '10000000-0000-0000-0000-000000000004', 'review_due', '{"topic_id":"kssm-f3-algebra-simultaneous-equations","source":"seed"}'::jsonb, NOW() + INTERVAL '4 hour')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000013', '%[1]s', '10000000-0000-0000-0000-000000000001', 'class_summary_generated', '{"students":3,"source":"seed"}'::jsonb, NOW() - INTERVAL '30 minute')
ON CONFLICT (id) DO NOTHING
`, tenantID),
		fmt.Sprintf(`
INSERT INTO events (id, tenant_id, user_id, event_type, data, created_at)
VALUES
('50000000-0000-0000-0000-000000000014', '%[1]s', '10000000-0000-0000-0000-000000000002', 'goal_set', '{"goal":"Master linear equations","source":"seed"}'::jsonb, NOW() - INTERVAL '15 minute')
ON CONFLICT (id) DO NOTHING
`, tenantID),
	}
}

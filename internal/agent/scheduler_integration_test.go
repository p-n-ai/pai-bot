//go:build integration
// +build integration

package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/progress"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestSchedulerIntegration_PostgresBackedNudges(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)

	t.Run("sends the most overdue due review and records it", func(t *testing.T) {
		gateway := chat.NewGateway()
		channel := &chat.MockChannel{}
		gateway.Register("telegram", channel)

		tracker := progress.NewPostgresTracker(pool, tenantID)
		nudges := NewPostgresNudgeTracker(pool, tenantID)
		scheduler := NewScheduler(
			DefaultSchedulerConfig(),
			tracker,
			progress.NewMemoryStreakTracker(),
			progress.NewMemoryXPTracker(),
			nudges,
			gateway,
			nil,
			nil,
		)

		userID := "scheduler-student-overdue"
		seedSchedulerUser(t, ctx, pool, tenantID, userID)
		now := time.Now()
		seedLearningProgress(t, ctx, pool, tenantID, userID, "topic-less-overdue", 0.44, now.Add(-2*time.Hour))
		seedLearningProgress(t, ctx, pool, tenantID, userID, "topic-most-overdue", 0.81, now.Add(-72*time.Hour))

		checkTime := activeMYTTime(now)
		if err := scheduler.checkUser(ctx, userID, checkTime); err != nil {
			t.Fatalf("checkUser() error = %v", err)
		}

		if len(channel.SentMessages) != 1 {
			t.Fatalf("sent messages = %d, want 1", len(channel.SentMessages))
		}
		t.Logf("outbound nudge:\n%s", channel.SentMessages[0].Text)
		if !strings.Contains(channel.SentMessages[0].Text, "topic-most-overdue") {
			t.Fatalf("nudge text = %q, want topic-most-overdue", channel.SentMessages[0].Text)
		}

		count, err := nudges.NudgeCountToday(userID)
		if err != nil {
			t.Fatalf("NudgeCountToday() error = %v", err)
		}
		if count != 1 {
			t.Fatalf("nudge count = %d, want 1", count)
		}

		recordedTopic := fetchLatestNudgeTopic(t, ctx, pool, tenantID, userID)
		if recordedTopic != "topic-most-overdue" {
			t.Fatalf("latest recorded topic = %q, want topic-most-overdue", recordedTopic)
		}
	})

	t.Run("skips sending when the daily nudge cap is already reached", func(t *testing.T) {
		gateway := chat.NewGateway()
		channel := &chat.MockChannel{}
		gateway.Register("telegram", channel)

		tracker := progress.NewPostgresTracker(pool, tenantID)
		nudges := NewPostgresNudgeTracker(pool, tenantID)
		scheduler := NewScheduler(
			DefaultSchedulerConfig(),
			tracker,
			progress.NewMemoryStreakTracker(),
			progress.NewMemoryXPTracker(),
			nudges,
			gateway,
			nil,
			nil,
		)

		userID := "scheduler-student-at-limit"
		now := time.Now()
		seedSchedulerUser(t, ctx, pool, tenantID, userID)
		seedLearningProgress(t, ctx, pool, tenantID, userID, "topic-cap-test", 0.51, now.Add(-6*time.Hour))

		for i := 0; i < MaxNudgesPerDay; i++ {
			seedNudgeLog(t, ctx, pool, tenantID, userID, "review_due", "topic-cap-test", now.Add(time.Duration(-i-1)*time.Minute))
		}

		checkTime := activeMYTTime(now)
		if err := scheduler.checkUser(ctx, userID, checkTime); err != nil {
			t.Fatalf("checkUser() error = %v", err)
		}

		if len(channel.SentMessages) != 0 {
			t.Fatalf("sent messages = %d, want 0", len(channel.SentMessages))
		}
		t.Log("outbound nudge: none (daily cap reached)")

		count, err := nudges.NudgeCountToday(userID)
		if err != nil {
			t.Fatalf("NudgeCountToday() error = %v", err)
		}
		if count != MaxNudgesPerDay {
			t.Fatalf("nudge count = %d, want %d", count, MaxNudgesPerDay)
		}
	})
}

func startSchedulerPostgres(t *testing.T, ctx context.Context) (*pgxpool.Pool, string) {
	t.Helper()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("pai"),
		tcpostgres.WithUsername("pai"),
		tcpostgres.WithPassword("pai"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("container connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	waitForPostgresReady(t, ctx, pool)

	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "001_initial.up.sql"))
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "002_streaks_xp.up.sql"))

	var tenantID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM tenants WHERE slug = 'default'`).Scan(&tenantID); err != nil {
		t.Fatalf("load default tenant: %v", err)
	}

	return pool, tenantID
}

func waitForPostgresReady(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for {
		if err := pool.Ping(ctx); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("postgres did not become ready before timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func applyMigrationFile(t *testing.T, ctx context.Context, pool *pgxpool.Pool, path string) {
	t.Helper()

	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
		t.Fatalf("apply migration %s: %v", path, err)
	}
}

func seedSchedulerUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, externalID string) {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (tenant_id, role, name, external_id, channel)
		 VALUES ($1::uuid, 'student', $2, $3, 'telegram')`,
		tenantID,
		"Student "+externalID,
		externalID,
	)
	if err != nil {
		t.Fatalf("seed user %s: %v", externalID, err)
	}
}

func seedLearningProgress(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, externalID, topicID string, mastery float64, nextReviewAt time.Time) {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO learning_progress (user_id, tenant_id, syllabus_id, topic_id, mastery_score, ease_factor, interval_days, repetitions, next_review_at, last_studied_at)
		 VALUES (
			(SELECT id FROM users WHERE tenant_id = $1::uuid AND external_id = $2 AND channel = 'telegram' LIMIT 1),
			$1::uuid, 'kssm-form1', $3, $4, 2.5, 1, 1, $5, $6
		 )`,
		tenantID,
		externalID,
		topicID,
		mastery,
		nextReviewAt,
		nextReviewAt.Add(-24*time.Hour),
	)
	if err != nil {
		t.Fatalf("seed learning progress %s: %v", topicID, err)
	}
}

func seedNudgeLog(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, externalID, nudgeType, topicID string, sentAt time.Time) {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO nudge_log (user_id, tenant_id, nudge_type, topic_id, sent_at)
		 VALUES (
			(SELECT id FROM users WHERE tenant_id = $1::uuid AND external_id = $2 AND channel = 'telegram' LIMIT 1),
			$1::uuid, $3, $4, $5
		 )`,
		tenantID,
		externalID,
		nudgeType,
		topicID,
		sentAt,
	)
	if err != nil {
		t.Fatalf("seed nudge log for %s: %v", externalID, err)
	}
}

func fetchLatestNudgeTopic(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, externalID string) string {
	t.Helper()

	var topicID string
	err := pool.QueryRow(ctx,
		`SELECT nl.topic_id
		 FROM nudge_log nl
		 JOIN users u ON u.id = nl.user_id
		 WHERE nl.tenant_id = $1::uuid
		   AND u.external_id = $2
		 ORDER BY nl.sent_at DESC
		 LIMIT 1`,
		tenantID,
		externalID,
	).Scan(&topicID)
	if err != nil {
		t.Fatalf("fetch latest nudge topic for %s: %v", externalID, err)
	}
	return topicID
}

func activeMYTTime(now time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		loc = time.FixedZone("MYT", 8*60*60)
	}
	myt := now.In(loc)
	return time.Date(myt.Year(), myt.Month(), myt.Day(), 10, 0, 0, 0, loc)
}

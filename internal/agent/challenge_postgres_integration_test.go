//go:build integration
// +build integration

package agent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresChallengeStore_GetChallengeIgnoresCompletedInviteCodes(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "005_challenges.up.sql"))

	creatorUUID := seedChallengeUser(t, ctx, pool, tenantID, "challenge-completed-creator", "telegram")

	_, err := pool.Exec(ctx,
		`INSERT INTO challenges (
			tenant_id,
			creator_user_id,
			topic_id,
			topic_name,
			syllabus_id,
			match_source,
			invite_code,
			question_count,
			state,
			completed_at
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, NOW())`,
		tenantID,
		creatorUUID,
		"F1-02",
		"Linear Equations",
		"kssm-f1",
		ChallengeMatchSourceInviteCode,
		"ABC123",
		5,
		"completed",
	)
	if err != nil {
		t.Fatalf("insert completed challenge: %v", err)
	}

	store := NewPostgresChallengeStore(pool, tenantID)
	_, err = store.GetChallenge("ABC123")
	if err != ErrChallengeNotFound {
		t.Fatalf("GetChallenge() error = %v, want ErrChallengeNotFound", err)
	}
}

func TestPostgresChallengeStore_CreateInviteChallengeSupportsExplicitChannel(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "005_challenges.up.sql"))

	seedChallengeUser(t, ctx, pool, tenantID, "terminal-challenge-user", "terminal")

	store := NewPostgresChallengeStoreForChannel(pool, tenantID, "terminal")
	challenge, err := store.CreateInviteChallenge("terminal-challenge-user", ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}
	if challenge.CreatorID != "terminal-challenge-user" {
		t.Fatalf("CreatorID = %q, want terminal-challenge-user", challenge.CreatorID)
	}
	if challenge.State != ChallengeStateWaiting {
		t.Fatalf("State = %q, want %q", challenge.State, ChallengeStateWaiting)
	}
}

func seedChallengeUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, externalID, channel string) string {
	t.Helper()

	var userID string
	err := pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, external_id, channel)
		 VALUES ($1::uuid, 'student', $2, $3, $4)
		 RETURNING id::text`,
		tenantID,
		"Student "+externalID,
		externalID,
		channel,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user %s/%s: %v", externalID, channel, err)
	}
	return userID
}

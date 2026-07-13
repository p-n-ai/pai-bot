//go:build integration

package focusedpage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresStoreIdempotencyIsolationAndLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := startFocusedPagePostgres(t, ctx)
	var tenantID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM tenants WHERE slug = 'default'`).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	var ownerID, otherOwnerID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (tenant_id, role, name, external_id, channel) VALUES ($1, 'student', 'Aina', 'aina', 'telegram') RETURNING id::text`, tenantID).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users (tenant_id, role, name, external_id, channel) VALUES ($1, 'student', 'Other', 'other', 'telegram') RETURNING id::text`, tenantID).Scan(&otherOwnerID); err != nil {
		t.Fatal(err)
	}
	var otherTenantID string
	if err := pool.QueryRow(ctx, `INSERT INTO tenants (name, slug) VALUES ('Other tenant', 'other-tenant') RETURNING id::text`).Scan(&otherTenantID); err != nil {
		t.Fatal(err)
	}
	var otherTenantOwnerID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (tenant_id, role, name, external_id, channel) VALUES ($1, 'student', 'Other tenant learner', 'other-tenant-learner', 'telegram') RETURNING id::text`, otherTenantID).Scan(&otherTenantOwnerID); err != nil {
		t.Fatal(err)
	}
	var conversationID string
	if err := pool.QueryRow(ctx, `INSERT INTO conversations (tenant_id, user_id, state) VALUES ($1, $2, 'teaching') RETURNING id::text`, tenantID, ownerID).Scan(&conversationID); err != nil {
		t.Fatal(err)
	}
	var otherConversationID string
	if err := pool.QueryRow(ctx, `INSERT INTO conversations (tenant_id, user_id, state) VALUES ($1, $2, 'teaching') RETURNING id::text`, otherTenantID, otherTenantOwnerID).Scan(&otherConversationID); err != nil {
		t.Fatal(err)
	}
	assertFocusedPageInsertRejected(t, ctx, pool, "cross-tenant-owner", tenantID, otherTenantOwnerID, conversationID)
	assertFocusedPageInsertRejected(t, ctx, pool, "cross-tenant-conversation", tenantID, ownerID, otherConversationID)

	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	service, err := NewService(NewPostgresStore(pool), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	input := CreateInput{TenantID: tenantID, OwnerUserID: ownerID, ConversationID: conversationID, TurnID: "turn-1", RecipientName: "Aina", Message: "Goal report"}
	first, err := service.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.URL != second.URL {
		t.Fatalf("idempotent URLs differ: %q %q", first.URL, second.URL)
	}

	wrongOwner := input
	wrongOwner.OwnerUserID = otherOwnerID
	wrongOwner.TurnID = "turn-2"
	if _, err := service.Create(ctx, wrongOwner); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong owner create error = %v", err)
	}
	if _, err := service.Redeem(ctx, first.PublicID, "wrong"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong token error = %v", err)
	}
	token := strings.Split(first.URL, "#")[1]
	if _, err := service.Redeem(ctx, first.PublicID, token); err != nil {
		t.Fatal(err)
	}
	now = first.ExpiresAt
	if _, err := service.Redeem(ctx, first.PublicID, token); !errors.Is(err, ErrExpired) {
		t.Fatalf("expiry error = %v", err)
	}

	now = time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	revocable := input
	revocable.TurnID = "turn-3"
	revokedArtifact, err := service.Create(ctx, revocable)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Revoke(ctx, revokedArtifact.PublicID, tenantID, otherOwnerID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong owner revoke error = %v", err)
	}
	if err := service.Revoke(ctx, revokedArtifact.PublicID, tenantID, ownerID); err != nil {
		t.Fatal(err)
	}
	revokedToken := strings.Split(revokedArtifact.URL, "#")[1]
	if _, err := service.Redeem(ctx, revokedArtifact.PublicID, revokedToken); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked redeem error = %v", err)
	}
}

func startFocusedPagePostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("pai"),
		tcpostgres.WithUsername("pai"),
		tcpostgres.WithPassword("pai"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	deadline := time.Now().Add(15 * time.Second)
	for pool.Ping(ctx) != nil {
		if time.Now().After(deadline) {
			t.Fatal("postgres did not become ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	applyFocusedPageMigration(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318100000_initial.sql"))
	applyFocusedPageMigration(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318100300_auth_tables.sql"))
	applyFocusedPageMigration(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318100400_auth_identity_tenant_consistency.sql"))
	applyFocusedPageMigration(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260713100000_focused_pages.sql"))
	return pool
}

func assertFocusedPageInsertRejected(t *testing.T, ctx context.Context, pool *pgxpool.Pool, publicID, tenantID, ownerID, conversationID string) {
	t.Helper()
	createdAt := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, `INSERT INTO focused_pages (
		public_id, tenant_id, owner_user_id, conversation_id, turn_id, recipient_name,
		message, token_hash, status, created_at, expires_at
	) VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $1, 'Learner', 'Report', $5, 'active', $6, $7)`,
		publicID, tenantID, ownerID, conversationID, make([]byte, 32), createdAt, createdAt.Add(Lifetime))
	if err == nil {
		t.Fatalf("database accepted mismatched focused page %q", publicID)
	}
}

func applyFocusedPageMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	up := strings.Index(text, "-- +goose Up")
	down := strings.Index(text, "-- +goose Down")
	if up < 0 || down < 0 || down <= up {
		t.Fatalf("invalid goose migration %s", path)
	}
	if _, err := pool.Exec(ctx, text[up+len("-- +goose Up"):down]); err != nil {
		t.Fatalf("apply %s: %v", path, err)
	}
}

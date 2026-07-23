//go:build integration

package focusedpage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreCleanupExpiredBatchPolicy(t *testing.T) {
	ctx := context.Background()
	pool := startFocusedPagePostgres(t, ctx)
	owners := seedCleanupOwners(t, ctx, pool)
	store := NewPostgresStore(pool)
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)

	t.Run("zero rows", func(t *testing.T) {
		truncateFocusedPages(t, ctx, pool)
		deleted, err := store.CleanupExpired(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 0 {
			t.Fatalf("deleted rows = %d, want 0", deleted)
		}
	})

	t.Run("exactly 100 rows", func(t *testing.T) {
		truncateFocusedPages(t, ctx, pool)
		for index := 1; index <= CleanupBatchSize; index++ {
			seedCleanupPage(t, ctx, pool, owners[index%len(owners)], index, StatusActive, now)
		}
		deleted, err := store.CleanupExpired(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != CleanupBatchSize {
			t.Fatalf("deleted rows = %d, want %d", deleted, CleanupBatchSize)
		}
		assertFocusedPageCount(t, ctx, pool, 0)
	})

	t.Run("more than 100 equal-expiry rows drain deterministically across tenants", func(t *testing.T) {
		truncateFocusedPages(t, ctx, pool)
		const total = 205
		for index := 1; index <= total; index++ {
			seedCleanupPage(t, ctx, pool, owners[index%len(owners)], index, StatusActive, now)
		}

		assertCleanupProgress(t, ctx, store, pool, now, CleanupBatchSize, CleanupBatchSize+1, total-CleanupBatchSize)
		assertCleanupProgress(t, ctx, store, pool, now, CleanupBatchSize, 2*CleanupBatchSize+1, total-2*CleanupBatchSize)
		assertCleanupProgress(t, ctx, store, pool, now, total-2*CleanupBatchSize, 0, 0)

		deleted, err := store.CleanupExpired(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 0 {
			t.Fatalf("final deleted rows = %d, want 0", deleted)
		}
	})

	t.Run("expires_at alone governs every lifecycle state", func(t *testing.T) {
		truncateFocusedPages(t, ctx, pool)
		statuses := []Status{StatusActive, StatusRevoked, StatusExpired}
		for index, status := range statuses {
			seedCleanupPage(t, ctx, pool, owners[index%len(owners)], index+1, status, now)
			seedCleanupPage(t, ctx, pool, owners[(index+1)%len(owners)], index+4, status, now.Add(time.Second))
		}

		deleted, err := store.CleanupExpired(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != int64(len(statuses)) {
			t.Fatalf("deleted rows = %d, want %d", deleted, len(statuses))
		}
		assertFocusedPageCount(t, ctx, pool, int64(len(statuses)))
		for _, status := range statuses {
			var count int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM focused_pages WHERE status = $1`, status).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("%s rows after cleanup = %d, want one unexpired row", status, count)
			}
		}
		for index := 4; index <= 6; index++ {
			var tenantID string
			if err := pool.QueryRow(ctx, `SELECT tenant_id::text FROM focused_pages WHERE id = $1::uuid`, cleanupPageID(index)).Scan(&tenantID); err != nil {
				t.Fatalf("unexpired row %d was not preserved: %v", index, err)
			}
			wantOwner := owners[(index-3)%len(owners)]
			if tenantID != wantOwner.tenantID {
				t.Fatalf("unexpired row %d tenant = %s, want %s", index, tenantID, wantOwner.tenantID)
			}
		}

		deleted, err = store.CleanupExpired(ctx, now.Add(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if deleted != int64(len(statuses)) {
			t.Fatalf("deleted rows at later expiry = %d, want %d", deleted, len(statuses))
		}
		assertFocusedPageCount(t, ctx, pool, 0)
	})

	t.Run("cancellation preserves eligible rows", func(t *testing.T) {
		truncateFocusedPages(t, ctx, pool)
		seedCleanupPage(t, ctx, pool, owners[0], 1, StatusActive, now)
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := store.CleanupExpired(cancelled, now); !errors.Is(err, context.Canceled) {
			t.Fatalf("cleanup error = %v, want context cancellation", err)
		}
		assertFocusedPageCount(t, ctx, pool, 1)
	})
}

func TestPostgresStoreCleanupExpiredReportsDatabaseFailure(t *testing.T) {
	ctx := context.Background()
	pool := startFocusedPagePostgres(t, ctx)
	store := NewPostgresStore(pool)
	pool.Close()
	if _, err := store.CleanupExpired(ctx, time.Now()); err == nil {
		t.Fatal("cleanup succeeded with a closed database pool")
	}
}

type cleanupOwner struct {
	tenantID       string
	ownerID        string
	conversationID string
}

func seedCleanupOwners(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []cleanupOwner {
	t.Helper()
	owners := make([]cleanupOwner, 0, 2)
	for index := 1; index <= 2; index++ {
		var owner cleanupOwner
		if err := pool.QueryRow(ctx,
			`INSERT INTO tenants (name, slug) VALUES ($1, $2) RETURNING id::text`,
			fmt.Sprintf("Cleanup tenant %d", index), fmt.Sprintf("cleanup-tenant-%d", index),
		).Scan(&owner.tenantID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (tenant_id, role, name, external_id, channel)
			VALUES ($1, 'student', $2, $3, 'telegram')
			RETURNING id::text`,
			owner.tenantID, fmt.Sprintf("Cleanup learner %d", index), fmt.Sprintf("cleanup-learner-%d", index),
		).Scan(&owner.ownerID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO conversations (tenant_id, user_id, state)
			VALUES ($1, $2, 'teaching')
			RETURNING id::text`,
			owner.tenantID, owner.ownerID,
		).Scan(&owner.conversationID); err != nil {
			t.Fatal(err)
		}
		owners = append(owners, owner)
	}
	return owners
}

func seedCleanupPage(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	owner cleanupOwner,
	index int,
	status Status,
	expiresAt time.Time,
) {
	t.Helper()
	var lifecycleAt any
	if status != StatusActive {
		lifecycleAt = expiresAt.Add(-time.Minute)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO focused_pages (
			id, public_id, tenant_id, owner_user_id, conversation_id, turn_id, recipient_name,
			message, token_hash, status, created_at, expires_at, revoked_at, expired_at
		)
		VALUES (
			$1::uuid, $2, $3::uuid, $4::uuid, $5::uuid, $6, 'Learner',
			'Private report', $7, $8, $9, $10,
			CASE WHEN $8 = 'revoked' THEN $11::timestamptz END,
			CASE WHEN $8 = 'expired' THEN $11::timestamptz END
		)`,
		cleanupPageID(index), fmt.Sprintf("cleanup-public-%03d", index),
		owner.tenantID, owner.ownerID, owner.conversationID, fmt.Sprintf("cleanup-turn-%03d", index),
		make([]byte, 32), status, expiresAt.Add(-Lifetime), expiresAt, lifecycleAt,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func assertCleanupProgress(
	t *testing.T,
	ctx context.Context,
	store *PostgresStore,
	pool *pgxpool.Pool,
	now time.Time,
	wantDeleted int64,
	wantFirstIndex int,
	wantRemaining int64,
) {
	t.Helper()
	deleted, err := store.CleanupExpired(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != wantDeleted {
		t.Fatalf("deleted rows = %d, want %d", deleted, wantDeleted)
	}
	assertFocusedPageCount(t, ctx, pool, wantRemaining)
	if wantFirstIndex == 0 {
		return
	}
	var firstID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM focused_pages ORDER BY expires_at, id LIMIT 1`).Scan(&firstID); err != nil {
		t.Fatal(err)
	}
	if want := cleanupPageID(wantFirstIndex); firstID != want {
		t.Fatalf("first remaining id = %s, want %s", firstID, want)
	}
}

func assertFocusedPageCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want int64) {
	t.Helper()
	var count int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM focused_pages`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("focused page rows = %d, want %d", count, want)
	}
}

func truncateFocusedPages(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE focused_pages`); err != nil {
		t.Fatal(err)
	}
}

func cleanupPageID(index int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", index)
}

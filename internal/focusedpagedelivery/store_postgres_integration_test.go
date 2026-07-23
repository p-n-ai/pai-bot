//go:build integration

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpagedelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresStoreDurableLeasingAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	pool := startDeliveryPostgres(t, ctx)
	tenant1, page1 := seedDeliveryIdentity(t, ctx, pool, "tenant-one", "turn-1", "page-one")
	tenant2, page2 := seedDeliveryIdentity(t, ctx, pool, "tenant-two", "turn-1", "page-two")
	tenant3, page3 := seedDeliveryIdentity(t, ctx, pool, "tenant-three", "turn-1", "page-three")
	store := NewPostgresStore(pool)
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)

	firstInput := EnqueueInput{
		TenantID: tenant1, TurnID: "turn-1", Channel: "telegram", RecipientID: "learner-one",
		FinalText: "Original text", FocusedPagePublicID: page1,
	}
	first, err := store.Enqueue(ctx, firstInput, now)
	if err != nil {
		t.Fatal(err)
	}
	changed := firstInput
	changed.RecipientID = "changed"
	changed.FinalText = "Changed text"
	duplicate, err := store.Enqueue(ctx, changed, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != first.ID || duplicate.RecipientID != firstInput.RecipientID ||
		duplicate.FinalText != firstInput.FinalText || duplicate.FocusedPagePublicID != page1 {
		t.Fatalf("idempotent enqueue changed row: %#v", duplicate)
	}
	otherTenant, err := store.Enqueue(ctx, EnqueueInput{
		TenantID: tenant2, TurnID: "turn-1", Channel: "telegram", RecipientID: "learner-two",
		FinalText: "Other tenant text", FocusedPagePublicID: page2,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if otherTenant.ID == first.ID {
		t.Fatal("delivery identity collided across tenants")
	}
	leased, err := store.Enqueue(ctx, EnqueueInput{
		TenantID: tenant3, TurnID: "turn-1", Channel: "telegram", RecipientID: "learner-three",
		FinalText: "Leased tenant text", FocusedPagePublicID: page3,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	leased, ok, err := store.Claim(ctx, leased.ID, "cleanup-lease", now, now.Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("cleanup lease claim ok = %t, err = %v", ok, err)
	}
	if _, err := store.Enqueue(ctx, EnqueueInput{
		TenantID: tenant1, TurnID: "turn-1", Channel: "whatsapp", RecipientID: "learner-one",
		FinalText: "Wrong page tenant", FocusedPagePublicID: page2,
	}, now); err == nil {
		t.Fatal("cross-tenant focused-page reference was accepted")
	}

	var claims atomic.Int32
	var claimed Delivery
	var claimMu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			delivery, ok, claimErr := store.Claim(ctx, first.ID, randomToken(), now, now.Add(time.Minute))
			if claimErr != nil {
				t.Errorf("claim: %v", claimErr)
				return
			}
			if ok {
				claims.Add(1)
				claimMu.Lock()
				claimed = delivery
				claimMu.Unlock()
			}
		}()
	}
	wg.Wait()
	if claims.Load() != 1 {
		t.Fatalf("concurrent successful claims = %d, want 1", claims.Load())
	}

	restarted := NewPostgresStore(pool)
	if _, ok, err := restarted.Claim(ctx, first.ID, "early", now.Add(30*time.Second), now.Add(90*time.Second)); err != nil || ok {
		t.Fatalf("claim before lease expiry ok = %t, err = %v", ok, err)
	}
	reclaimed, ok, err := restarted.Claim(ctx, first.ID, "restart", now.Add(time.Minute), now.Add(2*time.Minute))
	if err != nil || !ok {
		t.Fatalf("restart reclaim ok = %t, err = %v", ok, err)
	}
	if reclaimed.FinalText != claimed.FinalText || reclaimed.FocusedPagePublicID != claimed.FocusedPagePublicID {
		t.Fatal("restart reclaim changed payload")
	}
	if err := restarted.ScheduleRetry(ctx, reclaimed.ID, reclaimed.LeaseToken, now.Add(2*time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := restarted.Claim(ctx, first.ID, "before-backoff", now.Add(90*time.Second), now.Add(3*time.Minute)); err != nil || ok {
		t.Fatalf("claim before retry backoff ok = %t, err = %v", ok, err)
	}
	retry, ok, err := restarted.Claim(ctx, first.ID, "after-backoff", now.Add(2*time.Minute), now.Add(3*time.Minute))
	if err != nil || !ok {
		t.Fatalf("claim after retry backoff ok = %t, err = %v", ok, err)
	}
	if retry.AttemptCount != 1 {
		t.Fatalf("attempt count = %d, want 1", retry.AttemptCount)
	}
	if err := restarted.MarkDelivered(ctx, retry.ID, retry.LeaseToken, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := restarted.Claim(ctx, first.ID, "after-delivery", now.Add(4*time.Minute), now.Add(5*time.Minute)); err != nil || ok {
		t.Fatalf("delivered row claim ok = %t, err = %v", ok, err)
	}

	t.Run("parent deletion cascades pending leased and delivered rows", func(t *testing.T) {
		assertDeliveryStatus(t, ctx, pool, otherTenant.ID, StatusPending)
		assertDeliveryStatus(t, ctx, pool, leased.ID, StatusLeased)
		assertDeliveryStatus(t, ctx, pool, first.ID, StatusDelivered)
		tag, err := pool.Exec(ctx, `
			DELETE FROM focused_pages
			WHERE public_id = ANY($1::text[])`,
			[]string{page1, page2, page3})
		if err != nil {
			t.Fatalf("delete expired parent pages: %v", err)
		}
		if tag.RowsAffected() != 3 {
			t.Fatalf("deleted parent pages = %d, want 3", tag.RowsAffected())
		}
		var remaining int
		if err := pool.QueryRow(ctx, `
			SELECT count(*)
			FROM focused_page_deliveries
			WHERE id = ANY($1::uuid[])`,
			[]string{first.ID, otherTenant.ID, leased.ID}).Scan(&remaining); err != nil {
			t.Fatal(err)
		}
		if remaining != 0 {
			t.Fatalf("orphaned delivery rows = %d, want 0", remaining)
		}
	})
}

func assertDeliveryStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, deliveryID string, want Status) {
	t.Helper()
	var got Status
	if err := pool.QueryRow(ctx, `
		SELECT status
		FROM focused_page_deliveries
		WHERE id = $1::uuid`,
		deliveryID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("delivery %s status = %q, want %q", deliveryID, got, want)
	}
}

func startDeliveryPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
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
	for _, name := range []string{
		"20260318100000_initial.sql",
		"20260318100300_auth_tables.sql",
		"20260318100400_auth_identity_tenant_consistency.sql",
		"20260713100000_focused_pages.sql",
		"20260723163958_focused_page_deliveries.sql",
		"20260723170429_focused_page_delivery_cleanup_cascade.sql",
	} {
		applyDeliveryMigration(t, ctx, pool, filepath.Join("..", "..", "migrations", name))
	}
	return pool
}

func seedDeliveryIdentity(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slug, turnID, publicID string) (string, string) {
	t.Helper()
	var tenantID string
	if err := pool.QueryRow(ctx, `INSERT INTO tenants (name, slug) VALUES ($1, $1) RETURNING id::text`, slug).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, role, name, external_id, channel)
		VALUES ($1, 'student', $2, $2, 'telegram') RETURNING id::text`,
		tenantID, slug+"-learner").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	var conversationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO conversations (tenant_id, user_id, state)
		VALUES ($1, $2, 'teaching') RETURNING id::text`,
		tenantID, userID).Scan(&conversationID); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO focused_pages (
			public_id, tenant_id, owner_user_id, conversation_id, turn_id, recipient_name,
			message, token_hash, status, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, 'Learner', 'Private content', $6, 'active', $7, $8)`,
		publicID, tenantID, userID, conversationID, turnID, make([]byte, 32), created, created.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	return tenantID, publicID
}

func applyDeliveryMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, path string) {
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

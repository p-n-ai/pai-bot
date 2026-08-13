//go:build integration

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

func TestPostgresInboundDeliveryStorePersistsLifecycleAcrossRestart(t *testing.T) {
	ctx := context.Background()
	pool := startInboundDeliveryPostgres(t, ctx)
	tenantID := seedInboundDeliveryTenant(t, ctx, pool, "inbound-lifecycle")
	store := newPostgresInboundDeliveryStore(pool)
	now := time.Date(2026, 8, 13, 8, 0, 0, 0, time.UTC)
	input := postgresInboundDeliveryInput(tenantID, "delivery-1", "hello")

	var insertedCount atomic.Int32
	var workers sync.WaitGroup
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, inserted, err := store.Accept(ctx, input, now)
			if err != nil {
				t.Errorf("Accept() error = %v", err)
				return
			}
			if inserted {
				insertedCount.Add(1)
			}
		}()
	}
	workers.Wait()
	if insertedCount.Load() != 1 {
		t.Fatalf("inserted duplicates = %d, want one", insertedCount.Load())
	}
	changed := input
	changed.Message.Text = "changed duplicate"
	if _, _, err := store.Accept(ctx, changed, now); !errors.Is(err, ErrInboundDeliveryConflict) {
		t.Fatalf("Accept(changed duplicate) error = %v, want conflict", err)
	}

	restarted := newPostgresInboundDeliveryStore(pool)
	processing, ok, err := restarted.ClaimDue(ctx, "processor", now, now.Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("ClaimDue(processing) = %#v, %t, %v", processing, ok, err)
	}
	if processing.Status != inboundDeliveryProcessing || processing.Message != input.Message {
		t.Fatalf("recovered processing payload = %#v, want %#v", processing, input.Message)
	}
	if err := restarted.RenewLease(
		ctx, processing.ID, "stale", inboundDeliveryProcessing, now, now.Add(2*time.Minute),
	); !errors.Is(err, ErrInboundDeliveryLeaseLost) {
		t.Fatalf("RenewLease(stale) error = %v, want lease lost", err)
	}
	if err := restarted.RenewLease(
		ctx, processing.ID, "processor", inboundDeliveryProcessing, now, now.Add(2*time.Minute),
	); err != nil {
		t.Fatalf("RenewLease(current) error = %v", err)
	}

	result := agent.TurnResult{
		Text: "durable reply",
		FocusedPage: &focusedpage.Artifact{
			PublicID: "page-1", URL: "https://example.com/a/page-1",
			ExpiresAt: now.Add(time.Hour), TenantID: tenantID, TurnID: "turn-1",
		},
	}
	completeAt := now.Add(30 * time.Second)
	if err := restarted.CompleteProcessing(ctx, processing.ID, "processor", result, completeAt); err != nil {
		t.Fatalf("CompleteProcessing() error = %v", err)
	}
	delivering, ok, err := newPostgresInboundDeliveryStore(pool).ClaimDue(
		ctx, "deliverer-1", completeAt, completeAt.Add(time.Minute),
	)
	if err != nil || !ok {
		t.Fatalf("ClaimDue(delivery) = %#v, %t, %v", delivering, ok, err)
	}
	if delivering.Status != inboundDeliveryDelivering || delivering.Result.Text != result.Text ||
		delivering.Result.FocusedPage == nil || *delivering.Result.FocusedPage != *result.FocusedPage {
		t.Fatalf("durable result = %#v, want %#v", delivering.Result, result)
	}

	retryAt := completeAt.Add(2 * time.Minute)
	if err := restarted.ScheduleDeliveryRetry(ctx, delivering.ID, "deliverer-1", retryAt, completeAt); err != nil {
		t.Fatalf("ScheduleDeliveryRetry() error = %v", err)
	}
	if _, ok, err := restarted.ClaimDue(ctx, "early", retryAt.Add(-time.Second), retryAt.Add(time.Minute)); err != nil || ok {
		t.Fatalf("ClaimDue(before retry) = %t, %v; want false, nil", ok, err)
	}
	retried, ok, err := restarted.ClaimDue(ctx, "deliverer-2", retryAt, retryAt.Add(time.Minute))
	if err != nil || !ok || retried.Result.Text != result.Text || retried.ProcessingAttemptCount != 1 {
		t.Fatalf("ClaimDue(retry) = %#v, %t, %v", retried, ok, err)
	}
	if err := restarted.MarkDelivered(ctx, retried.ID, "deliverer-2", retryAt); err != nil {
		t.Fatalf("MarkDelivered() error = %v", err)
	}
	duplicate, inserted, err := restarted.Accept(ctx, input, retryAt.Add(time.Hour))
	if err != nil || inserted || duplicate.Status != inboundDeliveryDelivered {
		t.Fatalf("Accept(delivered duplicate) = %#v, %t, %v", duplicate, inserted, err)
	}
	deleted, err := restarted.DeleteTerminalBefore(ctx, retryAt, 10)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteTerminalBefore() = %d, %v; want one", deleted, err)
	}
}

func TestPostgresInboundDeliveryStoreRecoversCrashWithoutReprocessing(t *testing.T) {
	ctx := context.Background()
	pool := startInboundDeliveryPostgres(t, ctx)
	tenantID := seedInboundDeliveryTenant(t, ctx, pool, "inbound-recovery")
	store := newPostgresInboundDeliveryStore(pool)
	now := time.Date(2026, 8, 13, 9, 0, 0, 0, time.UTC)
	firstInput := postgresInboundDeliveryInput(tenantID, "delivery-1", "first")
	secondInput := postgresInboundDeliveryInput(tenantID, "delivery-2", "second")
	secondInput.Message.ThreadID = "thread-2"
	secondInput.DestinationKey = inboundCompositeKey("slack", "thread-2")
	if _, _, err := store.Accept(ctx, firstInput, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Accept(ctx, secondInput, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	first, ok, err := store.ClaimDue(ctx, "crashed", now, now.Add(time.Minute))
	if err != nil || !ok || first.DeliveryID != firstInput.DeliveryID {
		t.Fatalf("ClaimDue(first) = %#v, %t, %v", first, ok, err)
	}
	if _, ok, err := store.ClaimDue(ctx, "blocked", now.Add(30*time.Second), now.Add(2*time.Minute)); err != nil || ok {
		t.Fatalf("ClaimDue(blocked learner) = %t, %v; want false, nil", ok, err)
	}
	recoveryAt := now.Add(time.Minute)
	recovered, err := newPostgresInboundDeliveryStore(pool).RecoverExpiredProcessing(
		ctx, interruptedTurnResult, recoveryAt, 10,
	)
	if err != nil || recovered != 1 {
		t.Fatalf("RecoverExpiredProcessing() = %d, %v", recovered, err)
	}
	recovery, ok, err := store.ClaimDue(ctx, "recovery-delivery", recoveryAt, recoveryAt.Add(time.Minute))
	if err != nil || !ok || recovery.DeliveryID != firstInput.DeliveryID ||
		recovery.Status != inboundDeliveryDelivering || recovery.Result.Text != interruptedTurnResult(firstInput.Message).Text {
		t.Fatalf("ClaimDue(recovery result) = %#v, %t, %v", recovery, ok, err)
	}
	if err := store.MarkDelivered(ctx, recovery.ID, "recovery-delivery", recoveryAt); err != nil {
		t.Fatal(err)
	}
	second, ok, err := store.ClaimDue(ctx, "second-processor", recoveryAt, recoveryAt.Add(time.Minute))
	if err != nil || !ok || second.DeliveryID != secondInput.DeliveryID ||
		second.Status != inboundDeliveryProcessing {
		t.Fatalf("ClaimDue(second) = %#v, %t, %v", second, ok, err)
	}
}

func TestPostgresInboundDeliveryStoreFailedDeliveryUnblocksNextMessage(t *testing.T) {
	ctx := context.Background()
	pool := startInboundDeliveryPostgres(t, ctx)
	tenantID := seedInboundDeliveryTenant(t, ctx, pool, "inbound-failed-delivery")
	store := newPostgresInboundDeliveryStore(pool)
	now := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	first := postgresInboundDeliveryInput(tenantID, "delivery-failed", "first")
	second := postgresInboundDeliveryInput(tenantID, "delivery-next", "second")
	if _, _, err := store.Accept(ctx, first, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Accept(ctx, second, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	processing, ok, err := store.ClaimDue(ctx, "processor", now, now.Add(time.Minute))
	if err != nil || !ok || processing.DeliveryID != first.DeliveryID {
		t.Fatalf("ClaimDue(processing) = %#v, %t, %v", processing, ok, err)
	}
	if err := store.CompleteProcessing(ctx, processing.ID, processing.LeaseToken, agent.TurnResult{Text: "reply"}, now); err != nil {
		t.Fatal(err)
	}
	delivering, ok, err := store.ClaimDue(ctx, "deliverer", now, now.Add(time.Minute))
	if err != nil || !ok || delivering.Status != inboundDeliveryDelivering {
		t.Fatalf("ClaimDue(delivery) = %#v, %t, %v", delivering, ok, err)
	}
	if err := store.MarkDeliveryFailed(ctx, delivering.ID, delivering.LeaseToken, now); err != nil {
		t.Fatal(err)
	}
	next, ok, err := store.ClaimDue(ctx, "next-processor", now.Add(time.Second), now.Add(time.Minute))
	if err != nil || !ok || next.DeliveryID != second.DeliveryID || next.Status != inboundDeliveryProcessing {
		t.Fatalf("ClaimDue(next) = %#v, %t, %v", next, ok, err)
	}
	deleted, err := store.DeleteTerminalBefore(ctx, now, 10)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteTerminalBefore() = %d, %v; want failed row", deleted, err)
	}
}

func postgresInboundDeliveryInput(tenantID, deliveryID, text string) inboundDeliveryAcceptInput {
	message := chat.InboundMessage{
		Channel: "slack", UserID: "learner-1", TenantID: tenantID,
		InternalUserID: "internal-1", IdentityChannel: "slack", ExternalID: "learner-1",
		ThreadID: "thread-1", MessageID: "message-" + deliveryID, DeliveryID: deliveryID,
		Text: text, Caption: "caption", HasImage: true, ImageFileID: "image-1",
		ImageDataURL: "data:image/png;base64,AA==", ReplyToText: "previous",
		Username: "learner", FirstName: "First", LastName: "Last", Language: "en",
		CallbackQueryID: "callback-1", CallbackMessageID: 42,
	}
	return inboundDeliveryAcceptInput{
		TenantID: tenantID, Channel: message.Channel, DeliveryID: deliveryID,
		LearnerKey:     inboundLearnerKey(message),
		DestinationKey: inboundCompositeKey(message.Channel, message.DestinationID()),
		Message:        message,
	}
}

func startInboundDeliveryPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
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
	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	for _, name := range []string{
		"20260318100000_initial.sql",
		"20260813100000_chat_inbound_deliveries.sql",
	} {
		applyInboundDeliveryMigration(t, ctx, pool, filepath.Join("..", "..", "migrations", name))
	}
	return pool
}

func seedInboundDeliveryTenant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var tenantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO tenants (name, slug) VALUES ($1, $1) RETURNING id::text`, slug).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	return tenantID
}

func applyInboundDeliveryMigration(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	path string,
) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	up := strings.Index(text, "-- +goose Up")
	down := strings.Index(text, "-- +goose Down")
	if up < 0 || down < 0 || down <= up {
		t.Fatalf("invalid Goose migration %s", path)
	}
	if _, err := pool.Exec(ctx, text[up+len("-- +goose Up"):down]); err != nil {
		t.Fatalf("apply %s: %v", path, err)
	}
}

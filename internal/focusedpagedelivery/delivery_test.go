// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpagedelivery

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryStoreEnqueueIsIdempotentAndTenantScoped(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	firstInput := deliveryInput("tenant-1", "turn-1")
	first, err := store.Enqueue(context.Background(), firstInput, now)
	if err != nil {
		t.Fatal(err)
	}
	changed := firstInput
	changed.RecipientID = "attacker"
	changed.FinalText = "changed payload"
	changed.FocusedPagePublicID = "changed-page"
	duplicate, err := store.Enqueue(context.Background(), changed, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if duplicate != first {
		t.Fatalf("duplicate enqueue changed delivery: got %#v want %#v", duplicate, first)
	}
	otherTenant, err := store.Enqueue(context.Background(), deliveryInput("tenant-2", "turn-1"), now)
	if err != nil {
		t.Fatal(err)
	}
	if otherTenant.ID == first.ID {
		t.Fatal("same turn and channel collided across tenants")
	}
}

func TestMemoryStoreAllowsOnlyOneConcurrentClaimer(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	delivery, err := store.Enqueue(context.Background(), deliveryInput("tenant-1", "turn-1"), now)
	if err != nil {
		t.Fatal(err)
	}
	var claims atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok, claimErr := store.Claim(context.Background(), delivery.ID, randomToken(), now, now.Add(time.Minute)); claimErr != nil {
				t.Errorf("claim: %v", claimErr)
			} else if ok {
				claims.Add(1)
			}
		}()
	}
	wg.Wait()
	if claims.Load() != 1 {
		t.Fatalf("successful claims = %d, want 1", claims.Load())
	}
}

func TestProcessorRecoversPendingAndExpiredLeasesWithoutChangingPayload(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	current := now
	input := deliveryInput("tenant-1", "turn-1")
	pending, err := store.Enqueue(context.Background(), input, now)
	if err != nil {
		t.Fatal(err)
	}
	sender := &recordingSender{}
	processor := newTestProcessor(t, store, sender, func() time.Time { return current })
	if err := processor.deliverPendingForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.deliveries) != 1 || sender.deliveries[0].FinalText != input.FinalText ||
		sender.deliveries[0].FocusedPagePublicID != input.FocusedPagePublicID {
		t.Fatalf("recovered deliveries = %#v", sender.deliveries)
	}
	delivered, _ := store.Get(pending.ID)
	if delivered.Status != StatusDelivered {
		t.Fatalf("status = %q, want delivered", delivered.Status)
	}

	second, err := store.Enqueue(context.Background(), deliveryInput("tenant-1", "turn-2"), now)
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := store.Claim(context.Background(), second.ID, "crashed-worker", now, now.Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("initial claim ok = %t, err = %v", ok, err)
	}
	if _, ok, err := store.ClaimDue(context.Background(), "new-worker", now.Add(30*time.Second), now.Add(90*time.Second)); err != nil || ok {
		t.Fatalf("claim before lease expiry ok = %t, err = %v", ok, err)
	}
	current = now.Add(time.Minute)
	reclaimed, ok, err := store.ClaimDue(context.Background(), "new-worker", current, current.Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("expired lease reclaim ok = %t, err = %v", ok, err)
	}
	if reclaimed.ID != claimed.ID || reclaimed.FinalText != claimed.FinalText ||
		reclaimed.FocusedPagePublicID != claimed.FocusedPagePublicID {
		t.Fatal("lease reclaim changed delivery identity or payload")
	}
}

func TestProcessorRetriesWithBoundedBackoffThenSucceeds(t *testing.T) {
	store := NewMemoryStore()
	current := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	sender := &recordingSender{failures: 2}
	processor := newTestProcessor(t, store, sender, func() time.Time { return current })
	input := deliveryInput("tenant-1", "turn-retry")

	if err := processor.EnqueueAndDeliver(context.Background(), input); err == nil {
		t.Fatal("first channel failure was not returned")
	}
	delivery := onlyDelivery(t, store)
	if delivery.Status != StatusPending || delivery.AttemptCount != 1 ||
		!delivery.NextAttemptAt.Equal(current.Add(time.Second)) {
		t.Fatalf("first retry state = %#v", delivery)
	}
	current = current.Add(500 * time.Millisecond)
	if err := processor.deliverPendingForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.deliveries) != 1 {
		t.Fatalf("send count before retry due = %d, want 1", len(sender.deliveries))
	}
	current = current.Add(500 * time.Millisecond)
	if err := processor.deliverPendingForTest(context.Background()); err == nil {
		t.Fatal("second channel failure was not returned")
	}
	delivery = onlyDelivery(t, store)
	if delivery.AttemptCount != 2 || !delivery.NextAttemptAt.Equal(current.Add(2*time.Second)) {
		t.Fatalf("second retry state = %#v", delivery)
	}
	current = current.Add(2 * time.Second)
	if err := processor.deliverPendingForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	delivery = onlyDelivery(t, store)
	if delivery.Status != StatusDelivered || len(sender.deliveries) != 3 {
		t.Fatalf("final state = %#v, sends = %d", delivery, len(sender.deliveries))
	}
	if got := processor.backoff(100); got != 8*time.Second {
		t.Fatalf("bounded backoff = %s, want 8s", got)
	}
	for _, attempt := range sender.deliveries {
		if attempt.FinalText != input.FinalText || attempt.FocusedPagePublicID != input.FocusedPagePublicID {
			t.Fatal("retry changed immutable payload")
		}
	}
}

func TestSendBeforeAckFailureReplaysAfterLeaseExpiry(t *testing.T) {
	base := NewMemoryStore()
	current := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	store := &failFirstAckStore{MemoryStore: base}
	sender := &recordingSender{}
	processor := newTestProcessor(t, store, sender, func() time.Time { return current })
	if err := processor.EnqueueAndDeliver(context.Background(), deliveryInput("tenant-1", "turn-ack")); err == nil {
		t.Fatal("ack failure was not returned")
	}
	if len(sender.deliveries) != 1 {
		t.Fatalf("first send count = %d, want 1", len(sender.deliveries))
	}
	current = current.Add(time.Minute)
	if err := processor.deliverPendingForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.deliveries) != 2 {
		t.Fatalf("send-before-ack replay count = %d, want 2", len(sender.deliveries))
	}
	if sender.deliveries[0].FinalText != sender.deliveries[1].FinalText ||
		sender.deliveries[0].FocusedPagePublicID != sender.deliveries[1].FocusedPagePublicID {
		t.Fatal("send-before-ack replay changed payload")
	}
}

func TestProcessorCancellationLeavesLeaseForRestartAndStopsWorker(t *testing.T) {
	store := NewMemoryStore()
	current := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	sender := &cancelingSender{entered: make(chan struct{})}
	processor := newTestProcessor(t, store, sender, func() time.Time { return current })
	if _, err := store.Enqueue(context.Background(), deliveryInput("tenant-1", "turn-cancel"), current); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		processor.Run(ctx)
	}()
	<-sender.entered
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
	delivery := onlyDelivery(t, store)
	if delivery.Status != StatusLeased || delivery.AttemptCount != 0 {
		t.Fatalf("cancelled delivery state = %#v", delivery)
	}
}

func TestProcessorLogsOnlySafeDeliveryMetadata(t *testing.T) {
	store := NewMemoryStore()
	current := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	sender := &recordingSender{failures: 1, err: errors.New("secret-token private page content")}
	processor := newTestProcessor(t, store, sender, func() time.Time { return current })
	var logs bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })

	input := deliveryInput("tenant-1", "turn-safe-logs")
	input.FinalText = "private tutor response"
	input.FocusedPagePublicID = "private-public-id"
	_ = processor.EnqueueAndDeliver(context.Background(), input)
	output := logs.String()
	for _, secret := range []string{input.FinalText, input.FocusedPagePublicID, input.RecipientID, "secret-token", "private page content"} {
		if strings.Contains(output, secret) {
			t.Fatalf("log contains private value %q: %s", secret, output)
		}
	}
	for _, safe := range []string{"delivery_id", "status", "attempt_count", "error_category"} {
		if !strings.Contains(output, safe) {
			t.Fatalf("log missing safe field %q: %s", safe, output)
		}
	}
}

func (p *Processor) deliverPendingForTest(ctx context.Context) error {
	delivery, ok, err := p.claimDue(ctx)
	if err != nil || !ok {
		return err
	}
	return p.sendClaimed(ctx, delivery)
}

func newTestProcessor(t *testing.T, store Store, sender Sender, now func() time.Time) *Processor {
	t.Helper()
	processor, err := NewProcessor(store, sender, Config{
		LeaseDuration: time.Minute,
		PollInterval:  time.Millisecond,
		BaseBackoff:   time.Second,
		MaxBackoff:    8 * time.Second,
		Now:           now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return processor
}

func deliveryInput(tenantID, turnID string) EnqueueInput {
	return EnqueueInput{
		TenantID: tenantID, TurnID: turnID, Channel: "telegram", RecipientID: "learner-1",
		FinalText: "Your report is ready.", FocusedPagePublicID: "page-" + turnID,
	}
}

func onlyDelivery(t *testing.T, store *MemoryStore) Delivery {
	t.Helper()
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.deliveries) != 1 {
		t.Fatalf("deliveries = %d, want 1", len(store.deliveries))
	}
	for _, delivery := range store.deliveries {
		return cloneDelivery(delivery)
	}
	return Delivery{}
}

type recordingSender struct {
	failures   int
	err        error
	deliveries []Delivery
}

func (s *recordingSender) SendFocusedPage(_ context.Context, delivery Delivery) error {
	s.deliveries = append(s.deliveries, delivery)
	if len(s.deliveries) <= s.failures {
		if s.err != nil {
			return s.err
		}
		return errors.New("channel unavailable")
	}
	return nil
}

type cancelingSender struct {
	entered chan struct{}
	once    sync.Once
}

func (s *cancelingSender) SendFocusedPage(ctx context.Context, _ Delivery) error {
	s.once.Do(func() { close(s.entered) })
	<-ctx.Done()
	return ctx.Err()
}

type failFirstAckStore struct {
	*MemoryStore
	failed bool
}

func (s *failFirstAckStore) MarkDelivered(ctx context.Context, id, token string, now time.Time) error {
	if !s.failed {
		s.failed = true
		return errors.New("database unavailable")
	}
	return s.MemoryStore.MarkDelivered(ctx, id, token, now)
}

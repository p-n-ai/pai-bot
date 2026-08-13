// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

type memoryInboundDeliveryStore struct {
	mu           sync.Mutex
	deliveries   map[string]inboundDelivery
	byIdentity   map[string]string
	nextSequence int64
	failRenewal  bool
	renewals     int
}

func newMemoryInboundDeliveryStore() *memoryInboundDeliveryStore {
	return &memoryInboundDeliveryStore{
		deliveries: make(map[string]inboundDelivery),
		byIdentity: make(map[string]string),
	}
}

func (s *memoryInboundDeliveryStore) Accept(
	ctx context.Context,
	input inboundDeliveryAcceptInput,
	now time.Time,
) (inboundDelivery, bool, error) {
	if err := ctx.Err(); err != nil {
		return inboundDelivery{}, false, err
	}
	if err := validateInboundDeliveryAcceptInput(input); err != nil {
		return inboundDelivery{}, false, err
	}
	identity := input.TenantID + "\x00" + input.Channel + "\x00" + input.DeliveryID
	s.mu.Lock()
	defer s.mu.Unlock()
	if id := s.byIdentity[identity]; id != "" {
		delivery := s.deliveries[id]
		if delivery.LearnerKey != input.LearnerKey ||
			delivery.DestinationKey != input.DestinationKey ||
			!reflect.DeepEqual(delivery.Message, input.Message) {
			return inboundDelivery{}, false, ErrInboundDeliveryConflict
		}
		return delivery, false, nil
	}
	s.nextSequence++
	id := fmt.Sprintf("delivery-%d", s.nextSequence)
	delivery := inboundDelivery{
		ID: id, TenantID: input.TenantID, Channel: input.Channel, DeliveryID: input.DeliveryID,
		LearnerKey: input.LearnerKey, DestinationKey: input.DestinationKey,
		AcceptedSequence: s.nextSequence, Message: input.Message, Status: inboundDeliveryReceived,
		NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
	}
	s.deliveries[id] = delivery
	s.byIdentity[identity] = id
	return delivery, true, nil
}

func (s *memoryInboundDeliveryStore) ClaimDue(
	ctx context.Context,
	token string,
	now time.Time,
	leaseExpiresAt time.Time,
) (inboundDelivery, bool, error) {
	if err := ctx.Err(); err != nil {
		return inboundDelivery{}, false, err
	}
	if err := validateLeaseInput(token, now, leaseExpiresAt); err != nil {
		return inboundDelivery{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ordered := make([]inboundDelivery, 0, len(s.deliveries))
	for _, delivery := range s.deliveries {
		ordered = append(ordered, delivery)
	}
	sort.Slice(ordered, func(a, b int) bool {
		if ordered[a].NextAttemptAt.Equal(ordered[b].NextAttemptAt) {
			return ordered[a].AcceptedSequence < ordered[b].AcceptedSequence
		}
		return ordered[a].NextAttemptAt.Before(ordered[b].NextAttemptAt)
	})
	for _, candidate := range ordered {
		if !s.claimableLocked(candidate, now) {
			continue
		}
		if candidate.Status == inboundDeliveryReceived {
			candidate.Status = inboundDeliveryProcessing
			candidate.ProcessingAttemptCount++
		} else {
			candidate.Status = inboundDeliveryDelivering
			candidate.DeliveryAttemptCount++
		}
		candidate.LeaseToken = token
		candidate.LeaseExpiresAt = timePointer(leaseExpiresAt)
		candidate.UpdatedAt = now
		s.deliveries[candidate.ID] = candidate
		return candidate, true, nil
	}
	return inboundDelivery{}, false, nil
}

func (s *memoryInboundDeliveryStore) claimableLocked(candidate inboundDelivery, now time.Time) bool {
	switch candidate.Status {
	case inboundDeliveryReceived:
		if candidate.NextAttemptAt.After(now) {
			return false
		}
		for _, earlier := range s.deliveries {
			if earlier.TenantID == candidate.TenantID &&
				earlier.LearnerKey == candidate.LearnerKey &&
				earlier.AcceptedSequence < candidate.AcceptedSequence &&
				earlier.Status != inboundDeliveryDelivered &&
				earlier.Status != inboundDeliveryFailed {
				return false
			}
		}
		return true
	case inboundDeliveryDeliveryPending:
		if candidate.NextAttemptAt.After(now) {
			return false
		}
		for _, earlier := range s.deliveries {
			if earlier.TenantID == candidate.TenantID &&
				earlier.DestinationKey == candidate.DestinationKey &&
				earlier.AcceptedSequence < candidate.AcceptedSequence &&
				earlier.Status != inboundDeliveryDelivered &&
				earlier.Status != inboundDeliveryFailed {
				return false
			}
		}
		return true
	case inboundDeliveryDelivering:
		return candidate.LeaseExpiresAt != nil && !candidate.LeaseExpiresAt.After(now)
	default:
		return false
	}
}

func (s *memoryInboundDeliveryStore) RenewLease(
	ctx context.Context,
	id string,
	token string,
	status inboundDeliveryStatus,
	now time.Time,
	leaseExpiresAt time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failRenewal {
		return ErrInboundDeliveryLeaseLost
	}
	delivery, ok := s.deliveries[id]
	if !ok || delivery.Status != status || delivery.LeaseToken != token ||
		delivery.LeaseExpiresAt == nil || !delivery.LeaseExpiresAt.After(now) {
		return ErrInboundDeliveryLeaseLost
	}
	delivery.LeaseExpiresAt = timePointer(leaseExpiresAt)
	delivery.UpdatedAt = now
	s.deliveries[id] = delivery
	s.renewals++
	return nil
}

func (s *memoryInboundDeliveryStore) CompleteProcessing(
	ctx context.Context,
	id string,
	token string,
	result agent.TurnResult,
	now time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok || delivery.Status != inboundDeliveryProcessing || delivery.LeaseToken != token ||
		delivery.LeaseExpiresAt == nil || !delivery.LeaseExpiresAt.After(now) {
		return ErrInboundDeliveryLeaseLost
	}
	delivery.Status = inboundDeliveryDeliveryPending
	delivery.Result = result
	delivery.NextAttemptAt = now
	delivery.LeaseToken = ""
	delivery.LeaseExpiresAt = nil
	delivery.UpdatedAt = now
	s.deliveries[id] = delivery
	return nil
}

func (s *memoryInboundDeliveryStore) ScheduleDeliveryRetry(
	ctx context.Context,
	id string,
	token string,
	nextAttempt time.Time,
	now time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok || delivery.Status != inboundDeliveryDelivering || delivery.LeaseToken != token ||
		delivery.LeaseExpiresAt == nil || !delivery.LeaseExpiresAt.After(now) {
		return ErrInboundDeliveryLeaseLost
	}
	delivery.Status = inboundDeliveryDeliveryPending
	delivery.NextAttemptAt = nextAttempt
	delivery.LeaseToken = ""
	delivery.LeaseExpiresAt = nil
	delivery.UpdatedAt = now
	s.deliveries[id] = delivery
	return nil
}

func (s *memoryInboundDeliveryStore) MarkDelivered(
	ctx context.Context,
	id string,
	token string,
	now time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok || delivery.Status != inboundDeliveryDelivering || delivery.LeaseToken != token ||
		delivery.LeaseExpiresAt == nil || !delivery.LeaseExpiresAt.After(now) {
		return ErrInboundDeliveryLeaseLost
	}
	delivery.Status = inboundDeliveryDelivered
	delivery.LeaseToken = ""
	delivery.LeaseExpiresAt = nil
	delivery.DeliveredAt = timePointer(now)
	delivery.UpdatedAt = now
	s.deliveries[id] = delivery
	return nil
}

func (s *memoryInboundDeliveryStore) MarkDeliveryFailed(
	ctx context.Context,
	id string,
	token string,
	now time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok || delivery.Status != inboundDeliveryDelivering || delivery.LeaseToken != token ||
		delivery.LeaseExpiresAt == nil || !delivery.LeaseExpiresAt.After(now) {
		return ErrInboundDeliveryLeaseLost
	}
	delivery.Status = inboundDeliveryFailed
	delivery.LeaseToken = ""
	delivery.LeaseExpiresAt = nil
	delivery.FailedAt = timePointer(now)
	delivery.UpdatedAt = now
	s.deliveries[id] = delivery
	return nil
}

func (s *memoryInboundDeliveryStore) RecoverExpiredProcessing(
	ctx context.Context,
	resultFor func(chat.InboundMessage) agent.TurnResult,
	now time.Time,
	limit int,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if limit <= 0 {
		return 0, errors.New("inbound delivery recovery limit must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var recovered int64
	for id, delivery := range s.deliveries {
		if recovered >= int64(limit) {
			break
		}
		if delivery.Status != inboundDeliveryProcessing || delivery.LeaseExpiresAt == nil ||
			delivery.LeaseExpiresAt.After(now) {
			continue
		}
		delivery.Status = inboundDeliveryDeliveryPending
		delivery.Result = resultFor(delivery.Message)
		delivery.NextAttemptAt = now
		delivery.LeaseToken = ""
		delivery.LeaseExpiresAt = nil
		delivery.UpdatedAt = now
		s.deliveries[id] = delivery
		recovered++
	}
	return recovered, nil
}

func (s *memoryInboundDeliveryStore) DeleteTerminalBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if limit <= 0 {
		return 0, errors.New("inbound delivery cleanup limit must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted int64
	for id, delivery := range s.deliveries {
		if deleted >= int64(limit) {
			break
		}
		terminalAt := delivery.DeliveredAt
		if delivery.Status == inboundDeliveryFailed {
			terminalAt = delivery.FailedAt
		}
		if terminalAt == nil || terminalAt.After(before) {
			continue
		}
		delete(s.deliveries, id)
		delete(s.byIdentity, delivery.TenantID+"\x00"+delivery.Channel+"\x00"+delivery.DeliveryID)
		deleted++
	}
	return deleted, nil
}

func (s *memoryInboundDeliveryStore) snapshotByDeliveryID(deliveryID string) (inboundDelivery, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, delivery := range s.deliveries {
		if delivery.DeliveryID == deliveryID {
			return delivery, true
		}
	}
	return inboundDelivery{}, false
}

func (s *memoryInboundDeliveryStore) renewalCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.renewals
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func TestMemoryInboundDeliveryStoreRejectsChangedDuplicatePayload(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	now := time.Date(2026, 8, 13, 8, 0, 0, 0, time.UTC)
	input := testInboundDeliveryInput("delivery-1", "first")
	first, inserted, err := store.Accept(t.Context(), input, now)
	if err != nil || !inserted {
		t.Fatalf("Accept(first) = %#v, %t, %v; want inserted", first, inserted, err)
	}
	duplicate, inserted, err := store.Accept(t.Context(), input, now.Add(time.Hour))
	if err != nil || inserted || duplicate.ID != first.ID {
		t.Fatalf("Accept(duplicate) = %#v, %t, %v; want original", duplicate, inserted, err)
	}
	changed := input
	changed.Message.Text = "changed"
	if _, _, err := store.Accept(t.Context(), changed, now); !errors.Is(err, ErrInboundDeliveryConflict) {
		t.Fatalf("Accept(changed duplicate) error = %v, want conflict", err)
	}
}

func TestMemoryInboundDeliveryStoreRenewsOnlyCurrentLease(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	now := time.Date(2026, 8, 13, 8, 0, 0, 0, time.UTC)
	if _, _, err := store.Accept(t.Context(), testInboundDeliveryInput("delivery-1", "hello"), now); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := store.ClaimDue(t.Context(), "current", now, now.Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("ClaimDue() = %#v, %t, %v", claimed, ok, err)
	}
	if err := store.RenewLease(
		t.Context(), claimed.ID, "stale", inboundDeliveryProcessing, now, now.Add(2*time.Minute),
	); !errors.Is(err, ErrInboundDeliveryLeaseLost) {
		t.Fatalf("RenewLease(stale) error = %v, want lease lost", err)
	}
	if err := store.RenewLease(
		t.Context(), claimed.ID, "current", inboundDeliveryProcessing, now, now.Add(2*time.Minute),
	); err != nil {
		t.Fatalf("RenewLease(current) error = %v", err)
	}
	if _, ok, err := store.ClaimDue(t.Context(), "reclaim", now.Add(time.Minute), now.Add(3*time.Minute)); err != nil || ok {
		t.Fatalf("ClaimDue(before renewed expiry) = %t, %v; want false, nil", ok, err)
	}
}

func TestMemoryInboundDeliveryStoreRecoversProcessingAsStoredResult(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	now := time.Date(2026, 8, 13, 8, 0, 0, 0, time.UTC)
	if _, _, err := store.Accept(t.Context(), testInboundDeliveryInput("delivery-1", "hello"), now); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.ClaimDue(t.Context(), "worker", now, now.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("ClaimDue() = %t, %v", ok, err)
	}
	recovered, err := store.RecoverExpiredProcessing(
		t.Context(), func(chat.InboundMessage) agent.TurnResult {
			return agent.TurnResult{Text: "safe result"}
		}, now.Add(time.Minute), 10,
	)
	if err != nil || recovered != 1 {
		t.Fatalf("RecoverExpiredProcessing() = %d, %v", recovered, err)
	}
	claimed, ok, err := store.ClaimDue(
		t.Context(), "deliverer", now.Add(time.Minute), now.Add(2*time.Minute),
	)
	if err != nil || !ok || claimed.Status != inboundDeliveryDelivering || claimed.Result.Text != "safe result" {
		t.Fatalf("recovered ClaimDue() = %#v, %t, %v", claimed, ok, err)
	}
}

func testInboundDeliveryInput(deliveryID, text string) inboundDeliveryAcceptInput {
	message := testInboundMessage(deliveryID, text)
	message.TenantID = testChatIngressTenantID
	return inboundDeliveryAcceptInput{
		TenantID: testChatIngressTenantID, Channel: message.Channel,
		DeliveryID: deliveryID, LearnerKey: inboundCompositeKey("external", "slack", "learner-1"),
		DestinationKey: inboundCompositeKey("slack", "thread-1"), Message: message,
	}
}

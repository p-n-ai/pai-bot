// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpagedelivery

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MemoryStore struct {
	mu         sync.Mutex
	nextID     int
	deliveries map[string]Delivery
	keys       map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{deliveries: make(map[string]Delivery), keys: make(map[string]string)}
}

func (s *MemoryStore) Enqueue(ctx context.Context, input EnqueueInput, now time.Time) (Delivery, error) {
	if err := ctx.Err(); err != nil {
		return Delivery{}, err
	}
	if err := validateInput(input); err != nil {
		return Delivery{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := input.TenantID + "\x00" + input.TurnID + "\x00" + input.Channel
	if id, ok := s.keys[key]; ok {
		return cloneDelivery(s.deliveries[id]), nil
	}
	s.nextID++
	id := fmt.Sprintf("delivery-%d", s.nextID)
	delivery := Delivery{
		ID: id, TenantID: input.TenantID, TurnID: input.TurnID, Channel: input.Channel,
		RecipientID: input.RecipientID, FinalText: input.FinalText,
		FocusedPagePublicID: input.FocusedPagePublicID, Status: StatusPending,
		NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
	}
	s.deliveries[id] = delivery
	s.keys[key] = id
	return cloneDelivery(delivery), nil
}

func (s *MemoryStore) Claim(ctx context.Context, id, token string, now, leaseExpiry time.Time) (Delivery, bool, error) {
	if err := ctx.Err(); err != nil {
		return Delivery{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok || !claimable(delivery, now) {
		return Delivery{}, false, nil
	}
	return s.lease(delivery, token, now, leaseExpiry), true, nil
}

func (s *MemoryStore) ClaimDue(ctx context.Context, token string, now, leaseExpiry time.Time) (Delivery, bool, error) {
	if err := ctx.Err(); err != nil {
		return Delivery{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var selected *Delivery
	for _, delivery := range s.deliveries {
		if !claimable(delivery, now) {
			continue
		}
		if selected == nil || delivery.NextAttemptAt.Before(selected.NextAttemptAt) ||
			(delivery.NextAttemptAt.Equal(selected.NextAttemptAt) && delivery.CreatedAt.Before(selected.CreatedAt)) {
			copy := delivery
			selected = &copy
		}
	}
	if selected == nil {
		return Delivery{}, false, nil
	}
	return s.lease(*selected, token, now, leaseExpiry), true, nil
}

func (s *MemoryStore) MarkDelivered(ctx context.Context, id, token string, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok || delivery.Status != StatusLeased || delivery.LeaseToken != token {
		return ErrLeaseLost
	}
	delivery.Status = StatusDelivered
	delivery.LeaseToken = ""
	delivery.LeaseExpiresAt = nil
	delivery.DeliveredAt = timePointer(now)
	delivery.UpdatedAt = now
	s.deliveries[id] = delivery
	return nil
}

func (s *MemoryStore) ScheduleRetry(ctx context.Context, id, token string, nextAttempt, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok || delivery.Status != StatusLeased || delivery.LeaseToken != token {
		return ErrLeaseLost
	}
	delivery.Status = StatusPending
	delivery.AttemptCount++
	delivery.NextAttemptAt = nextAttempt
	delivery.LeaseToken = ""
	delivery.LeaseExpiresAt = nil
	delivery.UpdatedAt = now
	s.deliveries[id] = delivery
	return nil
}

func (s *MemoryStore) Get(id string) (Delivery, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	return cloneDelivery(delivery), ok
}

func (s *MemoryStore) lease(delivery Delivery, token string, now, leaseExpiry time.Time) Delivery {
	delivery.Status = StatusLeased
	delivery.LeaseToken = token
	delivery.LeaseExpiresAt = timePointer(leaseExpiry)
	delivery.UpdatedAt = now
	s.deliveries[delivery.ID] = delivery
	return cloneDelivery(delivery)
}

func claimable(delivery Delivery, now time.Time) bool {
	switch delivery.Status {
	case StatusPending:
		return !delivery.NextAttemptAt.After(now)
	case StatusLeased:
		return delivery.LeaseExpiresAt != nil && !delivery.LeaseExpiresAt.After(now)
	default:
		return false
	}
}

func cloneDelivery(delivery Delivery) Delivery {
	if delivery.LeaseExpiresAt != nil {
		delivery.LeaseExpiresAt = timePointer(*delivery.LeaseExpiresAt)
	}
	if delivery.DeliveredAt != nil {
		delivery.DeliveredAt = timePointer(*delivery.DeliveredAt)
	}
	return delivery
}

func timePointer(value time.Time) *time.Time {
	return &value
}

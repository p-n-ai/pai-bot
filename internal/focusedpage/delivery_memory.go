// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpage

import (
	"context"
	"sync"
	"time"
)

type MemoryDeliveryStore struct {
	mu         sync.Mutex
	deliveries map[string]Delivery
}

func NewMemoryDeliveryStore() *MemoryDeliveryStore {
	return &MemoryDeliveryStore{deliveries: make(map[string]Delivery)}
}

func (s *MemoryDeliveryStore) Enqueue(_ context.Context, delivery Delivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := delivery.TenantID + "\x00" + delivery.PublicID
	if existing, ok := s.deliveries[key]; ok {
		if !sameDeliveryPayload(existing, delivery) {
			return ErrDeliveryConflict
		}
		return nil
	}
	delivery.Status = DeliveryPending
	s.deliveries[key] = delivery
	return nil
}

func (s *MemoryDeliveryStore) ClaimDue(_ context.Context, now time.Time, lease time.Duration, limit int) ([]Delivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		return nil, nil
	}
	claimed := make([]Delivery, 0, limit)
	for key, delivery := range s.deliveries {
		if delivery.Status != DeliveryPending {
			continue
		}
		if !now.Before(delivery.ExpiresAt) {
			delivery.Status = DeliveryExpired
			s.deliveries[key] = delivery
			continue
		}
		if delivery.NextAttempt.After(now) {
			continue
		}
		delivery.Attempts++
		delivery.NextAttempt = now.Add(lease)
		s.deliveries[key] = delivery
		claimed = append(claimed, delivery)
		if len(claimed) == limit {
			break
		}
	}
	return claimed, nil
}

func (s *MemoryDeliveryStore) MarkSent(_ context.Context, tenantID, publicID string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenantID + "\x00" + publicID
	delivery, ok := s.deliveries[key]
	if !ok {
		return ErrNotFound
	}
	delivery.Status = DeliverySent
	s.deliveries[key] = delivery
	return nil
}

func (s *MemoryDeliveryStore) Reschedule(_ context.Context, tenantID, publicID string, nextAttempt, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenantID + "\x00" + publicID
	delivery, ok := s.deliveries[key]
	if !ok {
		return ErrNotFound
	}
	if !now.Before(delivery.ExpiresAt) || !nextAttempt.Before(delivery.ExpiresAt) {
		delivery.Status = DeliveryExpired
	} else {
		delivery.Status = DeliveryPending
		delivery.NextAttempt = nextAttempt
	}
	s.deliveries[key] = delivery
	return nil
}

func sameDeliveryPayload(left, right Delivery) bool {
	return left.TenantID == right.TenantID && left.PublicID == right.PublicID && left.TurnID == right.TurnID &&
		left.Channel == right.Channel && left.RecipientID == right.RecipientID && left.TutorText == right.TutorText &&
		left.ExpiresAt.Equal(right.ExpiresAt)
}

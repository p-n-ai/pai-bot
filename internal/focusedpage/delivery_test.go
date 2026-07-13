// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryDeliveryStoreIsIdempotentAndStopsClaimingAtExpiry(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	store := NewMemoryDeliveryStore()
	delivery := Delivery{
		TenantID: "tenant-1", PublicID: "page-1", TurnID: "turn-1", Channel: "telegram",
		RecipientID: "learner-1", TutorText: "Ready", NextAttempt: now, ExpiresAt: now.Add(Lifetime),
	}
	if err := store.Enqueue(context.Background(), delivery); err != nil {
		t.Fatal(err)
	}
	if err := store.Enqueue(context.Background(), delivery); err != nil {
		t.Fatalf("idempotent enqueue = %v", err)
	}
	conflict := delivery
	conflict.RecipientID = "other-learner"
	if err := store.Enqueue(context.Background(), conflict); !errors.Is(err, ErrDeliveryConflict) {
		t.Fatalf("conflicting enqueue = %v", err)
	}
	claimed, err := store.ClaimDue(context.Background(), now, time.Minute, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].Attempts != 1 {
		t.Fatalf("claimed deliveries = %d, want one first attempt", len(claimed))
	}
	if err := store.Reschedule(context.Background(), delivery.TenantID, delivery.PublicID, delivery.ExpiresAt, now); err != nil {
		t.Fatal(err)
	}
	claimed, err = store.ClaimDue(context.Background(), delivery.ExpiresAt, time.Minute, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 0 {
		t.Fatalf("expired deliveries claimed = %d, want 0", len(claimed))
	}
}

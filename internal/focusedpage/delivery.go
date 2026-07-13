// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpage

import (
	"context"
	"errors"
	"time"
)

var ErrDeliveryConflict = errors.New("focused page delivery conflict")

type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliverySent      DeliveryStatus = "sent"
	DeliveryExpired   DeliveryStatus = "expired"
	DeliveryCancelled DeliveryStatus = "cancelled"
)

type Delivery struct {
	TenantID    string
	PublicID    string
	TurnID      string
	Channel     string
	RecipientID string
	TutorText   string
	Status      DeliveryStatus
	Attempts    int
	NextAttempt time.Time
	ExpiresAt   time.Time
}

type DeliveryStore interface {
	Enqueue(context.Context, Delivery) error
	ClaimDue(context.Context, time.Time, time.Duration, int) ([]Delivery, error)
	MarkSent(context.Context, string, string, time.Time) error
	Reschedule(context.Context, string, string, time.Time, time.Time) error
}

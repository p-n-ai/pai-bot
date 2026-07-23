// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package focusedpagedelivery owns durable delivery of focused-page turns.
package focusedpagedelivery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

var ErrLeaseLost = errors.New("focused-page delivery lease lost")

type Status string

const (
	StatusPending   Status = "pending"
	StatusLeased    Status = "leased"
	StatusDelivered Status = "delivered"
)

type Delivery struct {
	ID                  string
	TenantID            string
	TurnID              string
	Channel             string
	RecipientID         string
	FinalText           string
	FocusedPagePublicID string
	Status              Status
	AttemptCount        int
	NextAttemptAt       time.Time
	LeaseToken          string
	LeaseExpiresAt      *time.Time
	DeliveredAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type EnqueueInput struct {
	TenantID            string
	TurnID              string
	Channel             string
	RecipientID         string
	FinalText           string
	FocusedPagePublicID string
}

type Store interface {
	Enqueue(context.Context, EnqueueInput, time.Time) (Delivery, error)
	Claim(context.Context, string, string, time.Time, time.Time) (Delivery, bool, error)
	ClaimDue(context.Context, string, time.Time, time.Time) (Delivery, bool, error)
	MarkDelivered(context.Context, string, string, time.Time) error
	ScheduleRetry(context.Context, string, string, time.Time, time.Time) error
}

type Sender interface {
	SendFocusedPage(context.Context, Delivery) error
}

type Config struct {
	LeaseDuration time.Duration
	PollInterval  time.Duration
	BaseBackoff   time.Duration
	MaxBackoff    time.Duration
	Now           func() time.Time
}

func DefaultConfig() Config {
	return Config{
		LeaseDuration: 30 * time.Second,
		PollInterval:  time.Second,
		BaseBackoff:   time.Second,
		MaxBackoff:    5 * time.Minute,
		Now:           time.Now,
	}
}

type Processor struct {
	store Store
	send  Sender
	cfg   Config
	wake  chan struct{}
}

func NewProcessor(store Store, sender Sender, cfg Config) (*Processor, error) {
	if store == nil {
		return nil, fmt.Errorf("focused-page delivery store is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("focused-page delivery sender is required")
	}
	defaults := DefaultConfig()
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = defaults.LeaseDuration
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaults.PollInterval
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = defaults.BaseBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = defaults.MaxBackoff
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		return nil, fmt.Errorf("focused-page delivery max backoff must not be shorter than base backoff")
	}
	if cfg.Now == nil {
		cfg.Now = defaults.Now
	}
	return &Processor{store: store, send: sender, cfg: cfg, wake: make(chan struct{}, 1)}, nil
}

func (p *Processor) EnqueueAndDeliver(ctx context.Context, input EnqueueInput) error {
	now := p.now()
	delivery, err := p.store.Enqueue(ctx, input, now)
	if err != nil {
		return fmt.Errorf("enqueue focused-page delivery: %w", err)
	}
	p.notify()
	if delivery.Status == StatusDelivered {
		return nil
	}
	claimed, ok, err := p.claim(ctx, delivery.ID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return p.sendClaimed(ctx, claimed)
}

func (p *Processor) Run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.wake:
		case <-timer.C:
		}

		for ctx.Err() == nil {
			claimed, ok, err := p.claimDue(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.Error("focused-page delivery claim failed", "error_category", "store")
				}
				break
			}
			if !ok {
				break
			}
			_ = p.sendClaimed(ctx, claimed)
		}
		timer.Reset(p.cfg.PollInterval)
	}
}

func (p *Processor) claim(ctx context.Context, deliveryID string) (Delivery, bool, error) {
	now := p.now()
	delivery, ok, err := p.store.Claim(ctx, deliveryID, randomToken(), now, now.Add(p.cfg.LeaseDuration))
	if err != nil {
		return Delivery{}, false, fmt.Errorf("claim focused-page delivery: %w", err)
	}
	return delivery, ok, nil
}

func (p *Processor) claimDue(ctx context.Context) (Delivery, bool, error) {
	now := p.now()
	delivery, ok, err := p.store.ClaimDue(ctx, randomToken(), now, now.Add(p.cfg.LeaseDuration))
	if err != nil {
		return Delivery{}, false, fmt.Errorf("claim due focused-page delivery: %w", err)
	}
	return delivery, ok, nil
}

func (p *Processor) sendClaimed(ctx context.Context, delivery Delivery) error {
	err := p.send.SendFocusedPage(ctx, delivery)
	if err != nil {
		if isCancellation(ctx, err) {
			return err
		}
		nextAttempt := p.now().Add(p.backoff(delivery.AttemptCount + 1))
		if retryErr := p.store.ScheduleRetry(ctx, delivery.ID, delivery.LeaseToken, nextAttempt, p.now()); retryErr != nil {
			slog.Error("focused-page delivery retry update failed",
				"delivery_id", delivery.ID,
				"status", StatusLeased,
				"attempt_count", delivery.AttemptCount,
				"error_category", "store",
			)
			return safeDeliveryError{category: "store", cause: retryErr}
		}
		slog.Warn("focused-page delivery scheduled for retry",
			"delivery_id", delivery.ID,
			"status", StatusPending,
			"attempt_count", delivery.AttemptCount+1,
			"error_category", "channel",
		)
		return safeDeliveryError{category: "channel", cause: err}
	}
	if err := p.store.MarkDelivered(ctx, delivery.ID, delivery.LeaseToken, p.now()); err != nil {
		slog.Error("focused-page delivery acknowledgement failed",
			"delivery_id", delivery.ID,
			"status", StatusLeased,
			"attempt_count", delivery.AttemptCount,
			"error_category", "store",
		)
		return safeDeliveryError{category: "store", cause: err}
	}
	slog.Info("focused-page delivery completed",
		"delivery_id", delivery.ID,
		"status", StatusDelivered,
		"attempt_count", delivery.AttemptCount,
	)
	return nil
}

func (p *Processor) backoff(attempt int) time.Duration {
	delay := p.cfg.BaseBackoff
	for currentAttempt := 1; currentAttempt < attempt; currentAttempt++ {
		if delay >= p.cfg.MaxBackoff/2 {
			return p.cfg.MaxBackoff
		}
		delay *= 2
	}
	if delay > p.cfg.MaxBackoff {
		return p.cfg.MaxBackoff
	}
	return delay
}

func (p *Processor) now() time.Time {
	return p.cfg.Now().UTC()
}

func (p *Processor) notify() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func isCancellation(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

type safeDeliveryError struct {
	category string
	cause    error
}

func (e safeDeliveryError) Error() string {
	return "focused-page delivery failed: " + e.category
}

func (e safeDeliveryError) Unwrap() error {
	return e.cause
}

func randomToken() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Sprintf("generate focused-page delivery lease token: %v", err))
	}
	return hex.EncodeToString(raw[:])
}

func validateInput(input EnqueueInput) error {
	if strings.TrimSpace(input.TenantID) == "" ||
		strings.TrimSpace(input.TurnID) == "" ||
		strings.TrimSpace(input.Channel) == "" ||
		strings.TrimSpace(input.RecipientID) == "" ||
		strings.TrimSpace(input.FinalText) == "" ||
		strings.TrimSpace(input.FocusedPagePublicID) == "" {
		return fmt.Errorf("focused-page delivery identity and payload are required")
	}
	return nil
}

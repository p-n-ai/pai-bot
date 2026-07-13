// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

const (
	focusedPageFirstRetry = 5 * time.Second
	focusedPagePoll       = 5 * time.Second
	focusedPageLease      = 30 * time.Second
	focusedPageBatchSize  = 20
	focusedPageMaxBackoff = 5 * time.Minute
)

// FocusedPageTurnDeliverer persists focused-page delivery ownership before sending.
type FocusedPageTurnDeliverer struct {
	direct       GatewayTurnDeliverer
	store        focusedpage.DeliveryStore
	pages        *focusedpage.Service
	now          func() time.Time
	firstRetry   time.Duration
	pollInterval time.Duration
	lease        time.Duration
}

func NewFocusedPageTurnDeliverer(gw *chat.Gateway, conversations agent.ConversationStore, store focusedpage.DeliveryStore, pages *focusedpage.Service) *FocusedPageTurnDeliverer {
	return &FocusedPageTurnDeliverer{
		direct:       NewGatewayTurnDeliverer(gw, conversations),
		store:        store,
		pages:        pages,
		now:          time.Now,
		firstRetry:   focusedPageFirstRetry,
		pollInterval: focusedPagePoll,
		lease:        focusedPageLease,
	}
}

func (d *FocusedPageTurnDeliverer) DeliverTurn(ctx context.Context, inbound chat.InboundMessage, result agent.TurnResult) error {
	if result.FocusedPage == nil {
		return d.direct.DeliverTurn(ctx, inbound, result)
	}
	now := d.now().UTC()
	delivery := focusedpage.Delivery{
		TenantID:    result.FocusedPage.TenantID,
		PublicID:    result.FocusedPage.PublicID,
		TurnID:      result.FocusedPage.TurnID,
		Channel:     inbound.Channel,
		RecipientID: inbound.UserID,
		TutorText:   result.Text,
		NextAttempt: now.Add(d.firstRetry),
		ExpiresAt:   result.FocusedPage.ExpiresAt,
	}
	if err := d.store.Enqueue(ctx, delivery); err != nil {
		return err
	}
	if err := d.direct.DeliverTurn(ctx, inbound, result); err != nil {
		return err
	}
	return d.store.MarkSent(ctx, delivery.TenantID, delivery.PublicID, now)
}

func (d *FocusedPageTurnDeliverer) Run(ctx context.Context) {
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()
	for {
		if err := d.retryDue(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("focused page delivery retry failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *FocusedPageTurnDeliverer) retryDue(ctx context.Context) error {
	now := d.now().UTC()
	deliveries, err := d.store.ClaimDue(ctx, now, d.lease, focusedPageBatchSize)
	if err != nil {
		return err
	}
	var retryErrors []error
	for _, delivery := range deliveries {
		artifact, err := d.pages.ArtifactForDelivery(delivery)
		if err == nil {
			err = d.direct.DeliverTurn(ctx,
				chat.InboundMessage{Channel: delivery.Channel, UserID: delivery.RecipientID},
				agent.TurnResult{Text: delivery.TutorText, FocusedPage: &artifact})
		}
		if err == nil {
			err = d.store.MarkSent(ctx, delivery.TenantID, delivery.PublicID, now)
		} else {
			nextAttempt := now.Add(focusedPageRetryBackoff(delivery.Attempts))
			if rescheduleErr := d.store.Reschedule(ctx, delivery.TenantID, delivery.PublicID, nextAttempt, now); rescheduleErr != nil {
				err = errors.Join(err, rescheduleErr)
			}
		}
		if err != nil {
			retryErrors = append(retryErrors, err)
		}
	}
	return errors.Join(retryErrors...)
}

func focusedPageRetryBackoff(attempt int) time.Duration {
	backoff := focusedPageFirstRetry
	for i := 1; i < attempt && backoff < focusedPageMaxBackoff; i++ {
		backoff *= 2
	}
	if backoff > focusedPageMaxBackoff {
		return focusedPageMaxBackoff
	}
	return backoff
}

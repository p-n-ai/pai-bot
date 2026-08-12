// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

const (
	chatIngressLeaseDuration   = 30 * time.Second
	chatIngressPollInterval    = time.Second
	chatIngressBaseBackoff     = time.Second
	chatIngressMaxBackoff      = 5 * time.Minute
	chatIngressRecordRetention = 24 * time.Hour
	chatIngressCleanupInterval = 10 * time.Minute
	chatIngressBatchSize       = 1000
	chatIngressWorkerCount     = 16
)

const chatIngressInterruptedResponse = "I hit a technical issue while processing that message. Please try again."

type InboundTurnProcessor interface {
	ProcessTurn(context.Context, chat.InboundMessage) (agent.TurnResult, error)
	DeliverTurn(context.Context, chat.InboundMessage, agent.TurnResult) error
}

type chatIngressConfig struct {
	LeaseDuration time.Duration
	PollInterval  time.Duration
	BaseBackoff   time.Duration
	MaxBackoff    time.Duration
	WorkerCount   int
	Now           func() time.Time
}

func defaultChatIngressConfig() chatIngressConfig {
	return chatIngressConfig{
		LeaseDuration: chatIngressLeaseDuration,
		PollInterval:  chatIngressPollInterval,
		BaseBackoff:   chatIngressBaseBackoff,
		MaxBackoff:    chatIngressMaxBackoff,
		WorkerCount:   chatIngressWorkerCount,
		Now:           time.Now,
	}
}

type ChatIngress struct {
	defaultTenantID string
	store           inboundDeliveryStore
	processor       InboundTurnProcessor
	cfg             chatIngressConfig
	wake            chan struct{}
}

func newChatIngressWithDefaults(
	defaultTenantID string,
	store inboundDeliveryStore,
	processor InboundTurnProcessor,
) (*ChatIngress, error) {
	return newChatIngress(defaultTenantID, store, processor, defaultChatIngressConfig())
}

func newChatIngress(
	defaultTenantID string,
	store inboundDeliveryStore,
	processor InboundTurnProcessor,
	cfg chatIngressConfig,
) (*ChatIngress, error) {
	if strings.TrimSpace(defaultTenantID) == "" {
		return nil, errors.New("chat ingress default tenant is required")
	}
	if store == nil {
		return nil, errors.New("chat ingress store is required")
	}
	if processor == nil {
		return nil, errors.New("chat ingress processor is required")
	}
	if cfg.LeaseDuration <= 0 || cfg.PollInterval <= 0 || cfg.BaseBackoff <= 0 || cfg.MaxBackoff <= 0 {
		return nil, errors.New("chat ingress durations must be positive")
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		return nil, errors.New("chat ingress max backoff must not be shorter than base backoff")
	}
	if cfg.WorkerCount <= 0 {
		return nil, errors.New("chat ingress worker count must be positive")
	}
	if cfg.Now == nil {
		return nil, errors.New("chat ingress clock is required")
	}
	return &ChatIngress{
		defaultTenantID: strings.TrimSpace(defaultTenantID),
		store:           store,
		processor:       processor,
		cfg:             cfg,
		wake:            make(chan struct{}, 1),
	}, nil
}

func (i *ChatIngress) Accept(ctx context.Context, message chat.InboundMessage) error {
	message, input, err := i.acceptInput(message)
	if err != nil {
		return err
	}
	input.Message = message
	if _, _, err := i.store.Accept(ctx, input, i.now()); err != nil {
		return fmt.Errorf("persist inbound chat message: %w", err)
	}
	i.notify()
	return nil
}

func (i *ChatIngress) Run(ctx context.Context) {
	recovery := time.NewTicker(i.cfg.PollInterval)
	cleanup := time.NewTicker(chatIngressCleanupInterval)
	defer recovery.Stop()
	defer cleanup.Stop()

	jobs := make(chan inboundDelivery)
	completed := make(chan struct{}, i.cfg.WorkerCount)
	var workers sync.WaitGroup
	for range i.cfg.WorkerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			i.runWorker(ctx, jobs, completed)
		}()
	}
	defer func() {
		close(jobs)
		workers.Wait()
	}()

	active := 0
	i.recoverExpiredProcessing(ctx)
	claimWork := true
	for {
		if claimWork && active < i.cfg.WorkerCount {
			delivery, ok, err := i.claimDue(ctx)
			if err != nil {
				if ctx.Err() == nil {
					slog.Error("claim inbound delivery failed", "error_category", "store")
				}
				claimWork = false
			} else if ok {
				select {
				case jobs <- delivery:
					active++
					continue
				case <-ctx.Done():
					return
				}
			} else {
				claimWork = false
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-i.wake:
			claimWork = true
		case <-recovery.C:
			i.recoverExpiredProcessing(ctx)
			claimWork = true
		case <-cleanup.C:
			if _, err := i.store.DeleteDeliveredBefore(
				ctx, i.now().Add(-chatIngressRecordRetention), chatIngressBatchSize,
			); err != nil && ctx.Err() == nil {
				slog.Warn("clean delivered inbound records failed", "error_category", "store")
			}
		case <-completed:
			active--
			claimWork = true
		}
	}
}

func (i *ChatIngress) runWorker(
	ctx context.Context,
	jobs <-chan inboundDelivery,
	completed chan<- struct{},
) {
	for delivery := range jobs {
		switch delivery.Status {
		case inboundDeliveryProcessing:
			i.process(ctx, delivery)
		case inboundDeliveryDelivering:
			i.deliver(ctx, delivery)
		}
		select {
		case completed <- struct{}{}:
		case <-ctx.Done():
			return
		}
	}
}

func (i *ChatIngress) process(ctx context.Context, delivery inboundDelivery) {
	result, processErr, leaseErr := withInboundLease(
		ctx, i, delivery, inboundDeliveryProcessing,
		func(phaseCtx context.Context) (agent.TurnResult, error) {
			return i.processor.ProcessTurn(phaseCtx, delivery.Message)
		},
	)
	if leaseErr != nil {
		if ctx.Err() == nil {
			slog.Warn("inbound processing lease lost", "delivery_id", delivery.ID, "error_category", "store")
		}
		return
	}
	if processErr != nil {
		slog.Error("process inbound turn failed", "delivery_id", delivery.ID, "error_category", "turn")
		result = interruptedTurnResult()
	}
	if err := i.store.CompleteProcessing(ctx, delivery.ID, delivery.LeaseToken, result, i.now()); err != nil {
		if ctx.Err() == nil {
			slog.Error("persist inbound turn result failed", "delivery_id", delivery.ID, "error_category", "store")
		}
		return
	}
	i.notify()
}

func (i *ChatIngress) deliver(ctx context.Context, delivery inboundDelivery) {
	_, deliveryErr, leaseErr := withInboundLease(
		ctx, i, delivery, inboundDeliveryDelivering,
		func(phaseCtx context.Context) (struct{}, error) {
			return struct{}{}, i.processor.DeliverTurn(phaseCtx, delivery.Message, delivery.Result)
		},
	)
	if leaseErr != nil {
		if ctx.Err() == nil {
			slog.Warn("inbound delivery lease lost", "delivery_id", delivery.ID, "error_category", "store")
		}
		return
	}
	if deliveryErr != nil {
		if ctx.Err() != nil {
			return
		}
		nextAttempt := i.now().Add(i.backoff(delivery.DeliveryAttemptCount))
		if err := i.store.ScheduleDeliveryRetry(
			ctx, delivery.ID, delivery.LeaseToken, nextAttempt, i.now(),
		); err != nil {
			slog.Error("schedule inbound delivery retry failed", "delivery_id", delivery.ID, "error_category", "store")
			return
		}
		slog.Warn("inbound turn delivery scheduled for retry",
			"delivery_id", delivery.ID,
			"attempt_count", delivery.DeliveryAttemptCount,
			"error_category", "channel",
		)
		return
	}
	if err := i.store.MarkDelivered(ctx, delivery.ID, delivery.LeaseToken, i.now()); err != nil {
		if ctx.Err() == nil {
			slog.Error("acknowledge inbound turn delivery failed", "delivery_id", delivery.ID, "error_category", "store")
		}
	}
}

func withInboundLease[T any](
	ctx context.Context,
	ingress *ChatIngress,
	delivery inboundDelivery,
	status inboundDeliveryStatus,
	work func(context.Context) (T, error),
) (T, error, error) {
	phaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	type outcome struct {
		value T
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		value, err := work(phaseCtx)
		done <- outcome{value: value, err: err}
	}()
	ticker := time.NewTicker(ingress.cfg.LeaseDuration / 3)
	defer ticker.Stop()
	for {
		select {
		case result := <-done:
			return result.value, result.err, nil
		case <-ticker.C:
			now := ingress.now()
			if err := ingress.store.RenewLease(
				ctx, delivery.ID, delivery.LeaseToken, status, now, now.Add(ingress.cfg.LeaseDuration),
			); err != nil {
				cancel()
				result := <-done
				return result.value, result.err, err
			}
		case <-ctx.Done():
			cancel()
			result := <-done
			return result.value, result.err, ctx.Err()
		}
	}
}

func (i *ChatIngress) claimDue(ctx context.Context) (inboundDelivery, bool, error) {
	now := i.now()
	return i.store.ClaimDue(ctx, newInboundDeliveryToken(), now, now.Add(i.cfg.LeaseDuration))
}

func (i *ChatIngress) recoverExpiredProcessing(ctx context.Context) {
	_, err := i.store.RecoverExpiredProcessing(
		ctx,
		agent.TurnResult{Text: chatIngressInterruptedResponse},
		i.now(),
		chatIngressBatchSize,
	)
	if err != nil {
		if ctx.Err() == nil {
			slog.Error("recover interrupted inbound processing failed", "error_category", "store")
		}
	}
}

func (i *ChatIngress) acceptInput(message chat.InboundMessage) (
	chat.InboundMessage,
	inboundDeliveryAcceptInput,
	error,
) {
	message.Channel = strings.TrimSpace(message.Channel)
	if message.Channel == "" {
		return chat.InboundMessage{}, inboundDeliveryAcceptInput{}, errors.New("inbound chat channel is required")
	}
	message.TenantID = strings.TrimSpace(message.TenantID)
	if message.TenantID == "" {
		message.TenantID = i.defaultTenantID
	}
	message.DeliveryID = strings.TrimSpace(message.DeliveryID)
	if message.DeliveryID == "" {
		switch message.Channel {
		case "websocket", "embed":
			message.DeliveryID = "generated:" + newInboundDeliveryToken()
		default:
			return chat.InboundMessage{}, inboundDeliveryAcceptInput{}, errors.New("inbound chat delivery ID is required")
		}
	}
	learnerKey := inboundLearnerKey(message)
	destination := strings.TrimSpace(message.DestinationID())
	if learnerKey == "" || destination == "" {
		return chat.InboundMessage{}, inboundDeliveryAcceptInput{}, errors.New("inbound chat learner and destination are required")
	}
	return message, inboundDeliveryAcceptInput{
		TenantID: message.TenantID, Channel: message.Channel, DeliveryID: message.DeliveryID,
		LearnerKey: learnerKey, DestinationKey: inboundCompositeKey(message.Channel, destination),
	}, nil
}

func inboundLearnerKey(message chat.InboundMessage) string {
	if internalID := strings.TrimSpace(message.InternalUserID); internalID != "" {
		return inboundCompositeKey("internal", internalID)
	}
	channel := strings.TrimSpace(message.IdentityChannel)
	if channel == "" {
		channel = strings.TrimSpace(message.Channel)
	}
	externalID := strings.TrimSpace(message.ExternalID)
	if externalID == "" {
		externalID = strings.TrimSpace(message.UserID)
	}
	if channel == "" || externalID == "" {
		return ""
	}
	return inboundCompositeKey("external", channel, externalID)
}

func inboundCompositeKey(parts ...string) string {
	var key strings.Builder
	for _, part := range parts {
		key.WriteString(strconv.Itoa(len(part)))
		key.WriteByte(':')
		key.WriteString(part)
	}
	return key.String()
}

func interruptedTurnResult() agent.TurnResult {
	return agent.TurnResult{Text: chatIngressInterruptedResponse}
}

func (i *ChatIngress) backoff(attempt int) time.Duration {
	delay := i.cfg.BaseBackoff
	for current := 1; current < attempt; current++ {
		if delay >= i.cfg.MaxBackoff/2 {
			return i.cfg.MaxBackoff
		}
		delay *= 2
	}
	if delay > i.cfg.MaxBackoff {
		return i.cfg.MaxBackoff
	}
	return delay
}

func (i *ChatIngress) now() time.Time {
	return i.cfg.Now().UTC()
}

func (i *ChatIngress) notify() {
	select {
	case i.wake <- struct{}{}:
	default:
	}
}

func newInboundDeliveryToken() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Sprintf("generate inbound delivery token: %v", err))
	}
	return hex.EncodeToString(raw[:])
}

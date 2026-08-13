// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

const testChatIngressTenantID = "11111111-1111-1111-1111-111111111111"

type chatIngressProcessorStub struct {
	process func(context.Context, chat.InboundMessage) (agent.TurnResult, error)
	deliver func(context.Context, chat.InboundMessage, agent.TurnResult) error
}

func (s chatIngressProcessorStub) ProcessTurn(
	ctx context.Context,
	message chat.InboundMessage,
) (agent.TurnResult, error) {
	return s.process(ctx, message)
}

func (s chatIngressProcessorStub) DeliverTurn(
	ctx context.Context,
	message chat.InboundMessage,
	result agent.TurnResult,
) error {
	return s.deliver(ctx, message, result)
}

func newTestChatIngress(
	t *testing.T,
	workers int,
	store inboundDeliveryStore,
	processor InboundTurnProcessor,
) *ChatIngress {
	t.Helper()
	ingress, err := newChatIngress(
		testChatIngressTenantID,
		store,
		processor,
		chatIngressConfig{
			LeaseDuration:       90 * time.Millisecond,
			PollInterval:        5 * time.Millisecond,
			BaseBackoff:         100 * time.Millisecond,
			MaxBackoff:          200 * time.Millisecond,
			WorkerCount:         workers,
			MaxDeliveryAttempts: chatIngressMaxDeliveryAttempts,
			Now:                 time.Now,
		},
	)
	if err != nil {
		t.Fatalf("newChatIngress() error = %v", err)
	}
	return ingress
}

func TestChatIngressDeduplicatesExplicitDeliveryPerChannel(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	processed := make(chan chat.InboundMessage, 3)
	ingress := newTestChatIngress(t, 2, store, chatIngressProcessorStub{
		process: func(_ context.Context, message chat.InboundMessage) (agent.TurnResult, error) {
			processed <- message
			return agent.TurnResult{Text: message.Text}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error { return nil },
	})
	ctx, cancel, done := runChatIngress(t, ingress)
	defer stopChatIngress(cancel, done)

	slack := testInboundMessage("delivery-1", "first")
	if err := ingress.Accept(ctx, slack); err != nil {
		t.Fatalf("Accept(first Slack delivery) error = %v", err)
	}
	if err := ingress.Accept(ctx, slack); err != nil {
		t.Fatalf("Accept(duplicate Slack delivery) error = %v", err)
	}
	discord := slack
	discord.Channel = "discord"
	discord.ThreadID = "discord-thread"
	discord.Text = "same ID, other channel"
	if err := ingress.Accept(ctx, discord); err != nil {
		t.Fatalf("Accept(Discord delivery) error = %v", err)
	}

	for count := 0; count < 2; count++ {
		select {
		case <-processed:
		case <-time.After(time.Second):
			t.Fatalf("processed %d messages, want two channel-scoped deliveries", count)
		}
	}
	select {
	case extra := <-processed:
		t.Fatalf("duplicate delivery processed: %#v", extra)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestChatIngressRoutesTenantlessMessageThroughDefaultProcessor(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	defaultProcessor := &turnProcessorStub{}
	router, err := NewTenantTurnRouter(defaultProcessor, func(string) (TurnProcessor, error) {
		return nil, errors.New("tenant factory must not be called")
	})
	if err != nil {
		t.Fatal(err)
	}
	ingress := newTestChatIngress(t, 1, store, router)
	ctx, cancel, done := runChatIngress(t, ingress)
	defer stopChatIngress(cancel, done)
	message := testInboundMessage("default-tenant-delivery", "hello")
	if err := ingress.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}
	waitForInboundStatus(t, store, message.DeliveryID, inboundDeliveryDelivered)
	if len(defaultProcessor.messages) != 1 || len(defaultProcessor.deliveries) != 1 {
		t.Fatalf("default processor calls = process:%d deliver:%d, want one each", len(defaultProcessor.messages), len(defaultProcessor.deliveries))
	}
	if got := defaultProcessor.messages[0].TenantID; got != "" {
		t.Fatalf("routing tenant = %q, want blank default route", got)
	}
}

func TestChatIngressAssignsDistinctIDsToLegacyWebSocketMessages(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	processed := make(chan chat.InboundMessage, 2)
	ingress := newTestChatIngress(t, 2, store, chatIngressProcessorStub{
		process: func(_ context.Context, message chat.InboundMessage) (agent.TurnResult, error) {
			processed <- message
			return agent.TurnResult{Text: "ok"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error { return nil },
	})
	ctx, cancel, done := runChatIngress(t, ingress)
	defer stopChatIngress(cancel, done)

	message := chat.InboundMessage{
		Channel: "websocket", UserID: "learner-1", ExternalID: "learner-1", Text: "same text",
	}
	if err := ingress.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}
	if err := ingress.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}
	first := waitForProcessedMessage(t, processed)
	second := waitForProcessedMessage(t, processed)
	if first.DeliveryID == "" || second.DeliveryID == "" || first.DeliveryID == second.DeliveryID {
		t.Fatalf("generated delivery IDs = %q and %q, want distinct non-empty IDs", first.DeliveryID, second.DeliveryID)
	}
}

func TestChatIngressStoresFailureResultWithoutReprocessing(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	var processCalls atomic.Int32
	delivered := make(chan agent.TurnResult, 1)
	ingress := newTestChatIngress(t, 1, store, chatIngressProcessorStub{
		process: func(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
			processCalls.Add(1)
			return agent.TurnResult{Text: "unsafe partial reply"}, errors.New("model failed after an uncertain mutation")
		},
		deliver: func(_ context.Context, _ chat.InboundMessage, result agent.TurnResult) error {
			delivered <- result
			return nil
		},
	})
	ctx, cancel, done := runChatIngress(t, ingress)
	defer stopChatIngress(cancel, done)
	message := testInboundMessage("delivery-failed-processing", "hello")
	if err := ingress.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}
	if err := ingress.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-delivered:
		if result.Text != interruptedTurnResult(message).Text {
			t.Fatalf("delivered text = %q, want technical response", result.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("stored failure response was not delivered")
	}
	if calls := processCalls.Load(); calls != 1 {
		t.Fatalf("processing calls = %d, want one", calls)
	}
}

func TestChatIngressRetriesStoredOutputWithoutReprocessing(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	var processCalls atomic.Int32
	firstDelivery := make(chan struct{}, 1)
	first := newTestChatIngress(t, 1, store, chatIngressProcessorStub{
		process: func(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
			processCalls.Add(1)
			return agent.TurnResult{Text: "durable reply"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error {
			firstDelivery <- struct{}{}
			return errors.New("temporary channel failure")
		},
	})
	ctx, cancel, done := runChatIngress(t, first)
	message := testInboundMessage("delivery-retry", "hello")
	if err := first.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstDelivery:
	case <-time.After(time.Second):
		t.Fatal("first delivery attempt did not run")
	}
	waitForInboundStatus(t, store, message.DeliveryID, inboundDeliveryDeliveryPending)
	stopChatIngress(cancel, done)

	retried := make(chan agent.TurnResult, 1)
	restarted := newTestChatIngress(t, 1, store, chatIngressProcessorStub{
		process: func(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
			processCalls.Add(1)
			return agent.TurnResult{}, errors.New("must not reprocess stored output")
		},
		deliver: func(_ context.Context, _ chat.InboundMessage, result agent.TurnResult) error {
			retried <- result
			return nil
		},
	})
	_, cancelRestarted, restartedDone := runChatIngress(t, restarted)
	defer stopChatIngress(cancelRestarted, restartedDone)
	select {
	case result := <-retried:
		if result.Text != "durable reply" {
			t.Fatalf("retried result = %#v, want exact durable reply", result)
		}
	case <-time.After(time.Second):
		t.Fatal("fresh worker did not retry stored output")
	}
	if calls := processCalls.Load(); calls != 1 {
		t.Fatalf("processing calls = %d, want one across restart", calls)
	}
}

func TestChatIngressSerializesOneLearnerThroughDelivery(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	firstDeliveryStarted := make(chan struct{})
	releaseFirstDelivery := make(chan struct{})
	secondProcessed := make(chan struct{})
	ingress := newTestChatIngress(t, 2, store, chatIngressProcessorStub{
		process: func(_ context.Context, message chat.InboundMessage) (agent.TurnResult, error) {
			if message.Text == "second" {
				close(secondProcessed)
			}
			return agent.TurnResult{Text: message.Text}, nil
		},
		deliver: func(_ context.Context, message chat.InboundMessage, _ agent.TurnResult) error {
			if message.Text == "first" {
				close(firstDeliveryStarted)
				<-releaseFirstDelivery
			}
			return nil
		},
	})
	ctx, cancel, done := runChatIngress(t, ingress)
	defer stopChatIngress(cancel, done)
	first := testInboundMessage("delivery-1", "first")
	first.ThreadID = "thread-1"
	second := testInboundMessage("delivery-2", "second")
	second.ThreadID = "thread-2"
	if err := ingress.Accept(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := ingress.Accept(ctx, second); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstDeliveryStarted:
	case <-time.After(time.Second):
		t.Fatal("first delivery did not start")
	}
	select {
	case <-secondProcessed:
		t.Fatal("second thread processed before the learner's first delivery completed")
	case <-time.After(40 * time.Millisecond):
	}
	close(releaseFirstDelivery)
	select {
	case <-secondProcessed:
	case <-time.After(time.Second):
		t.Fatal("second thread did not process after the first delivery completed")
	}
}

func TestChatIngressDoesNotBlockDifferentLearners(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondProcessed := make(chan struct{})
	ingress := newTestChatIngress(t, 2, store, chatIngressProcessorStub{
		process: func(_ context.Context, message chat.InboundMessage) (agent.TurnResult, error) {
			if message.UserID == "learner-1" {
				close(firstStarted)
				<-releaseFirst
			} else {
				close(secondProcessed)
			}
			return agent.TurnResult{Text: "ok"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error { return nil },
	})
	ctx, cancel, done := runChatIngress(t, ingress)
	defer stopChatIngress(cancel, done)
	first := testInboundMessage("delivery-1", "first")
	second := testInboundMessage("delivery-2", "second")
	second.UserID = "learner-2"
	second.ExternalID = "learner-2"
	second.ThreadID = "thread-2"
	if err := ingress.Accept(ctx, first); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first learner did not start")
	}
	if err := ingress.Accept(ctx, second); err != nil {
		t.Fatal(err)
	}
	select {
	case <-secondProcessed:
	case <-time.After(time.Second):
		t.Fatal("different learner was blocked")
	}
	close(releaseFirst)
}

func TestChatIngressRenewsClaimWhileProcessing(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	release := make(chan struct{})
	delivered := make(chan struct{})
	ingress := newTestChatIngress(t, 1, store, chatIngressProcessorStub{
		process: func(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
			<-release
			return agent.TurnResult{Text: "ok"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error {
			close(delivered)
			return nil
		},
	})
	ingress.cfg.LeaseDuration = 30 * time.Millisecond
	ctx, cancel, done := runChatIngress(t, ingress)
	defer stopChatIngress(cancel, done)
	message := testInboundMessage("delivery-renew", "hello")
	if err := ingress.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}
	waitForCondition(t, time.Second, func() bool { return store.renewalCount() >= 2 }, "lease renewal")
	if _, ok, err := store.ClaimDue(
		t.Context(), "competing-worker", time.Now(), time.Now().Add(time.Second),
	); err != nil || ok {
		t.Fatalf("competing ClaimDue() = %t, %v; want no claim while renewed", ok, err)
	}
	close(release)
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("renewed turn did not finish")
	}
	if delivery, ok := store.snapshotByDeliveryID(message.DeliveryID); !ok || delivery.ProcessingAttemptCount != 1 {
		t.Fatalf("processing state = %#v, want one attempt", delivery)
	}
}

func TestChatIngressCancelsAndDoesNotPersistAfterLeaseLoss(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	store.failRenewal = true
	processingCanceled := make(chan struct{})
	var deliveryCalls atomic.Int32
	ingress := newTestChatIngress(t, 1, store, chatIngressProcessorStub{
		process: func(ctx context.Context, _ chat.InboundMessage) (agent.TurnResult, error) {
			<-ctx.Done()
			close(processingCanceled)
			return agent.TurnResult{Text: "unsafe late result"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error {
			deliveryCalls.Add(1)
			return nil
		},
	})
	ingress.cfg.LeaseDuration = 60 * time.Millisecond
	ctx, cancel, done := runChatIngress(t, ingress)
	message := testInboundMessage("delivery-lost", "hello")
	if err := ingress.Accept(ctx, message); err != nil {
		t.Fatal(err)
	}
	select {
	case <-processingCanceled:
	case <-time.After(time.Second):
		t.Fatal("lease loss did not cancel processing")
	}
	stopChatIngress(cancel, done)
	delivery, ok := store.snapshotByDeliveryID(message.DeliveryID)
	if !ok || delivery.Status != inboundDeliveryProcessing || delivery.Result.Text != "" {
		t.Fatalf("state after lease loss = %#v, want uncommitted processing claim", delivery)
	}
	if calls := deliveryCalls.Load(); calls != 0 {
		t.Fatalf("delivery calls after lease loss = %d, want zero", calls)
	}
}

func TestChatIngressRunDoesNotWaitForeverForUncooperativeWorker(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	started := make(chan struct{})
	release := make(chan struct{})
	ingress := newTestChatIngress(t, 1, store, chatIngressProcessorStub{
		process: func(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
			close(started)
			<-release
			return agent.TurnResult{Text: "late result"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error { return nil },
	})
	ctx, cancel, done := runChatIngress(t, ingress)
	if err := ingress.Accept(ctx, testInboundMessage("delivery-cancel", "hello")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	cancel()
	select {
	case <-done:
		close(release)
	case <-time.After(250 * time.Millisecond):
		close(release)
		t.Fatal("Run() waited indefinitely for work that ignored cancellation")
	}
}

func TestChatIngressExhaustedDeliveryDoesNotBlockLaterMessage(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	ingress := newTestChatIngress(t, 1, store, chatIngressProcessorStub{
		process: func(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
			return agent.TurnResult{Text: "reply"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error {
			return errors.New("permanent channel failure")
		},
	})
	now := time.Now().UTC()
	firstInput := testInboundDeliveryInput("poison", "first")
	if _, _, err := store.Accept(t.Context(), firstInput, now); err != nil {
		t.Fatal(err)
	}
	processing, ok, err := store.ClaimDue(t.Context(), "process", now, now.Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("claim processing = %t, %v", ok, err)
	}
	if err := store.CompleteProcessing(t.Context(), processing.ID, processing.LeaseToken, agent.TurnResult{Text: "reply"}, now); err != nil {
		t.Fatal(err)
	}
	delivering, ok, err := store.ClaimDue(t.Context(), "deliver", now, now.Add(time.Minute))
	if err != nil || !ok {
		t.Fatalf("claim delivery = %t, %v", ok, err)
	}
	delivering.DeliveryAttemptCount = ingress.cfg.MaxDeliveryAttempts
	ingress.deliver(t.Context(), delivering)
	failed, _ := store.snapshotByDeliveryID("poison")
	if failed.Status != inboundDeliveryFailed {
		t.Fatalf("exhausted status = %q, want failed", failed.Status)
	}
	if _, _, err := store.Accept(t.Context(), testInboundDeliveryInput("next", "second"), now); err != nil {
		t.Fatal(err)
	}
	next, ok, err := store.ClaimDue(t.Context(), "next-process", now, now.Add(time.Minute))
	if err != nil || !ok || next.DeliveryID != "next" || next.Status != inboundDeliveryProcessing {
		t.Fatalf("next claim = %#v, %t, %v", next, ok, err)
	}
}

func TestInterruptedTurnResultUsesMessageLanguage(t *testing.T) {
	english := interruptedTurnResult(chat.InboundMessage{Language: "en"})
	malay := interruptedTurnResult(chat.InboundMessage{Language: "ms"})
	chinese := interruptedTurnResult(chat.InboundMessage{Language: "zh"})
	if english.Text == malay.Text || english.Text == chinese.Text || malay.Text == chinese.Text {
		t.Fatalf("localized failure results are not distinct: %q, %q, %q", english.Text, malay.Text, chinese.Text)
	}
}

func testInboundMessage(deliveryID, text string) chat.InboundMessage {
	return chat.InboundMessage{
		Channel: "slack", UserID: "learner-1", ExternalID: "learner-1",
		ThreadID: "thread-1", DeliveryID: deliveryID, Text: text,
	}
}

func runChatIngress(t *testing.T, ingress *ChatIngress) (context.Context, context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ingress.Run(ctx)
	}()
	return ctx, cancel, done
}

func stopChatIngress(cancel context.CancelFunc, done <-chan struct{}) {
	cancel()
	<-done
}

func waitForProcessedMessage(t *testing.T, messages <-chan chat.InboundMessage) chat.InboundMessage {
	t.Helper()
	select {
	case message := <-messages:
		return message
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for processed message")
		return chat.InboundMessage{}
	}
}

func waitForInboundStatus(
	t *testing.T,
	store *memoryInboundDeliveryStore,
	deliveryID string,
	status inboundDeliveryStatus,
) {
	t.Helper()
	waitForCondition(t, time.Second, func() bool {
		delivery, ok := store.snapshotByDeliveryID(deliveryID)
		return ok && delivery.Status == status
	}, "inbound status "+string(status))
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestChatIngressAcceptIsSafeUnderConcurrentDuplicateRequests(t *testing.T) {
	store := newMemoryInboundDeliveryStore()
	processor := chatIngressProcessorStub{
		process: func(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
			return agent.TurnResult{Text: "ok"}, nil
		},
		deliver: func(context.Context, chat.InboundMessage, agent.TurnResult) error { return nil },
	}
	ingress := newTestChatIngress(t, 1, store, processor)
	message := testInboundMessage("delivery-concurrent", "hello")
	var workers sync.WaitGroup
	var failures atomic.Int32
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := ingress.Accept(t.Context(), message); err != nil {
				failures.Add(1)
			}
		}()
	}
	workers.Wait()
	if failures.Load() != 0 {
		t.Fatalf("concurrent duplicate acceptance failures = %d", failures.Load())
	}
	store.mu.Lock()
	count := len(store.deliveries)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("stored deliveries = %d, want one", count)
	}
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestChatIngressDeduplicatesDeliveriesPerChannel(t *testing.T) {
	processed := make(chan chat.InboundMessage, 3)
	ingress, err := newChatIngress(4, func(_ context.Context, message chat.InboundMessage) {
		processed <- message
	})
	if err != nil {
		t.Fatalf("newChatIngress() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ingress.Run(ctx)
	}()

	slack := chat.InboundMessage{Channel: "slack", DeliveryID: "delivery-1", Text: "first"}
	if err := ingress.Enqueue(ctx, slack); err != nil {
		t.Fatalf("Enqueue(first Slack delivery) error = %v", err)
	}
	if err := ingress.Enqueue(ctx, slack); err != nil {
		t.Fatalf("Enqueue(duplicate Slack delivery) error = %v", err)
	}
	if err := ingress.Enqueue(ctx, chat.InboundMessage{
		Channel: "discord", DeliveryID: "delivery-1", Text: "same ID, other channel",
	}); err != nil {
		t.Fatalf("Enqueue(Discord delivery) error = %v", err)
	}

	got := make([]chat.InboundMessage, 0, 2)
	for len(got) < 2 {
		select {
		case message := <-processed:
			got = append(got, message)
		case <-time.After(time.Second):
			t.Fatalf("processed %d messages, want 2 unique channel deliveries", len(got))
		}
	}
	select {
	case extra := <-processed:
		t.Fatalf("duplicate delivery processed: %#v", extra)
	case <-time.After(25 * time.Millisecond):
	}

	cancel()
	<-done
}

func TestChatIngressProcessesMessagesWithoutDeliveryID(t *testing.T) {
	processed := make(chan chat.InboundMessage, 2)
	ingress, err := newChatIngress(2, func(_ context.Context, message chat.InboundMessage) {
		processed <- message
	})
	if err != nil {
		t.Fatalf("newChatIngress() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go ingress.Run(ctx)

	message := chat.InboundMessage{Channel: "websocket", UserID: "learner-1", Text: "hello"}
	if err := ingress.Enqueue(ctx, message); err != nil {
		t.Fatalf("Enqueue(first message) error = %v", err)
	}
	if err := ingress.Enqueue(ctx, message); err != nil {
		t.Fatalf("Enqueue(second message) error = %v", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-processed:
		case <-time.After(time.Second):
			t.Fatalf("processed %d messages without delivery IDs, want 2", i)
		}
	}
}

func TestChatIngressDoesNotBlockIndependentThreads(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondProcessed := make(chan struct{})
	ingress, err := newChatIngress(4, func(_ context.Context, message chat.InboundMessage) {
		switch message.ThreadID {
		case "slack:C1:T1":
			close(firstStarted)
			<-releaseFirst
		case "slack:C2:T2":
			close(secondProcessed)
		}
	})
	if err != nil {
		t.Fatalf("newChatIngress() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go ingress.Run(ctx)

	if err := ingress.Enqueue(ctx, chat.InboundMessage{
		Channel: "slack", ThreadID: "slack:C1:T1", DeliveryID: "D1",
	}); err != nil {
		t.Fatalf("Enqueue(first thread) error = %v", err)
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first thread did not start")
	}
	if err := ingress.Enqueue(ctx, chat.InboundMessage{
		Channel: "slack", ThreadID: "slack:C2:T2", DeliveryID: "D2",
	}); err != nil {
		t.Fatalf("Enqueue(second thread) error = %v", err)
	}
	select {
	case <-secondProcessed:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("independent thread was blocked by an in-flight turn")
	}
	close(releaseFirst)
}

func TestChatIngressSerializesOneThread(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	order := make(chan string, 2)
	ingress, err := newChatIngress(4, func(_ context.Context, message chat.InboundMessage) {
		if message.Text == "first" {
			close(firstStarted)
			<-releaseFirst
		}
		order <- message.Text
	})
	if err != nil {
		t.Fatalf("newChatIngress() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go ingress.Run(ctx)

	for index, text := range []string{"first", "second"} {
		if err := ingress.Enqueue(ctx, chat.InboundMessage{
			Channel: "slack", ThreadID: "slack:C1:T1", DeliveryID: string(rune('1' + index)), Text: text,
		}); err != nil {
			t.Fatalf("Enqueue(%s) error = %v", text, err)
		}
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first turn did not start")
	}
	select {
	case got := <-order:
		t.Fatalf("same-thread turn completed before release: %q", got)
	case <-time.After(25 * time.Millisecond):
	}
	close(releaseFirst)
	for _, want := range []string{"first", "second"} {
		select {
		case got := <-order:
			if got != want {
				t.Fatalf("processed %q, want %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}
}

func TestChatIngressCapacityBoundsQueuedAndProcessingMessages(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	ingress, err := newChatIngress(1, func(_ context.Context, _ chat.InboundMessage) {
		close(started)
		<-release
	})
	if err != nil {
		t.Fatalf("newChatIngress() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go ingress.Run(ctx)

	if err := ingress.Enqueue(ctx, chat.InboundMessage{Channel: "slack", ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Enqueue(first) error = %v", err)
	}
	<-started
	enqueueCtx, cancelEnqueue := context.WithTimeout(ctx, 25*time.Millisecond)
	defer cancelEnqueue()
	if err := ingress.Enqueue(enqueueCtx, chat.InboundMessage{
		Channel: "slack", ThreadID: "thread-2",
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Enqueue(over capacity) error = %v, want deadline exceeded", err)
	}
	close(release)
}

func TestChatIngressRunWaitsForCanceledWorker(t *testing.T) {
	started := make(chan struct{})
	workerDone := make(chan struct{})
	ingress, err := newChatIngress(1, func(ctx context.Context, _ chat.InboundMessage) {
		close(started)
		<-ctx.Done()
		close(workerDone)
	})
	if err != nil {
		t.Fatalf("newChatIngress() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	runDone := make(chan struct{})
	go func() {
		ingress.Run(ctx)
		close(runDone)
	}()
	if err := ingress.Enqueue(ctx, chat.InboundMessage{Channel: "slack", ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	<-started
	cancel()
	select {
	case <-runDone:
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after cancellation")
	}
	select {
	case <-workerDone:
	default:
		t.Fatal("Run() returned before its worker stopped")
	}
}

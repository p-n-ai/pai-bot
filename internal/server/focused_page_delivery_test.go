// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

func TestFocusedPageTurnDelivererRetriesPersistedArtifactWithoutRerunningTurn(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	pages, err := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := pages.Create(context.Background(), focusedpage.CreateInput{
		TenantID: "tenant-1", OwnerUserID: "owner-1", ConversationID: "conversation-1",
		TurnID: "turn-1", RecipientName: "Aina", Message: "Goal report",
	})
	if err != nil {
		t.Fatal(err)
	}
	channel := &controlledChannel{fail: true}
	gateway := chat.NewGateway()
	gateway.Register("telegram", channel)
	queue := focusedpage.NewMemoryDeliveryStore()
	deliverer := NewFocusedPageTurnDeliverer(gateway, agent.NewMemoryStore(), queue, pages)
	deliverer.now = func() time.Time { return now }
	deliverer.firstRetry = 0

	inbound := chat.InboundMessage{Channel: "telegram", UserID: "learner-1"}
	result := agent.TurnResult{Text: "Your report is ready.", FocusedPage: &artifact}
	if err := deliverer.DeliverTurn(context.Background(), inbound, result); err == nil {
		t.Fatal("initial delivery unexpectedly succeeded")
	}
	channel.setFailure(false)
	if err := deliverer.retryDue(context.Background()); err != nil {
		t.Fatal(err)
	}

	messages := channel.messagesSnapshot()
	if len(messages) != 2 {
		t.Fatalf("delivery attempts = %d, want 2", len(messages))
	}
	for _, message := range messages {
		if message.Text != result.Text || len(message.InlineKeyboard) == 0 {
			t.Fatal("retry changed the tutor text or omitted the focused-page button")
		}
		button := message.InlineKeyboard[len(message.InlineKeyboard)-1][0]
		if button.URL != artifact.URL {
			t.Fatal("retry changed the private focused-page URL")
		}
	}
	claimed, err := queue.ClaimDue(context.Background(), now.Add(time.Minute), time.Second, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 0 {
		t.Fatalf("sent deliveries claimed = %d, want 0", len(claimed))
	}
}

type controlledChannel struct {
	mu       sync.Mutex
	fail     bool
	messages []chat.OutboundMessage
}

func (c *controlledChannel) SendMessage(_ context.Context, _ string, message chat.OutboundMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, message)
	if c.fail {
		return errors.New("telegram unavailable")
	}
	return nil
}

func (*controlledChannel) SendTyping(context.Context, string) error               { return nil }
func (*controlledChannel) Start(context.Context, func(chat.InboundMessage)) error { return nil }
func (*controlledChannel) Stop() error                                            { return nil }
func (c *controlledChannel) setFailure(fail bool)                                 { c.mu.Lock(); c.fail = fail; c.mu.Unlock() }
func (c *controlledChannel) messagesSnapshot() []chat.OutboundMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]chat.OutboundMessage(nil), c.messages...)
}

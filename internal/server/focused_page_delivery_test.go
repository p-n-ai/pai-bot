// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
	"github.com/p-n-ai/pai-bot/internal/focusedpagedelivery"
)

func TestGatewayTurnDelivererPersistsFocusedPageBeforeSendingStoredPayload(t *testing.T) {
	ctx := context.Background()
	conversations := agent.NewMemoryStore()
	_ = conversations.SetUserName("learner-1", "Aina")
	pages, err := focusedpage.NewService(
		focusedpage.NewMemoryStore(),
		"https://pages.example",
		[]byte("0123456789abcdef0123456789abcdef"),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := pages.Create(ctx, focusedpage.CreateInput{
		TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conversation-1",
		TurnID: "turn-1", RecipientName: "Aina", Message: "Private goal report",
	})
	if err != nil {
		t.Fatal(err)
	}
	gateway := chat.NewGateway()
	channel := &chat.MockChannel{}
	gateway.Register("telegram", channel)
	outbox := focusedpagedelivery.NewMemoryStore()
	processor, err := focusedpagedelivery.NewProcessor(
		outbox,
		NewGatewayFocusedPageSender(gateway, conversations, pages),
		focusedpagedelivery.DefaultConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
	deliverer := NewGatewayTurnDeliverer(gateway, conversations, processor)
	result := agent.TurnResult{Text: "Your report is ready.", FocusedPage: &artifact}
	if err := deliverer.DeliverTurn(ctx, chat.InboundMessage{Channel: "telegram", UserID: "learner-1"}, result); err != nil {
		t.Fatal(err)
	}
	if len(channel.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(channel.SentMessages))
	}
	sent := channel.SentMessages[0]
	if sent.Text != result.Text {
		t.Fatalf("sent text = %q, want %q", sent.Text, result.Text)
	}
	var focusedURL string
	for _, row := range sent.InlineKeyboard {
		for _, button := range row {
			if button.URL != "" {
				focusedURL = button.URL
			}
		}
	}
	if focusedURL != artifact.URL {
		t.Fatalf("focused-page URL = %q, want %q", focusedURL, artifact.URL)
	}
}

func TestFocusedPageOutboxLeavesPlainTurnsRemindersAndNotificationsOnDirectGatewayPaths(t *testing.T) {
	ctx := context.Background()
	conversations := agent.NewMemoryStore()
	_ = conversations.SetUserName("learner-1", "Aina")
	gateway := chat.NewGateway()
	channel := &chat.MockChannel{}
	gateway.Register("telegram", channel)

	turns := NewGatewayTurnDeliverer(gateway, conversations, nil)
	if err := turns.DeliverTurn(ctx,
		chat.InboundMessage{Channel: "telegram", UserID: "learner-1"},
		agent.TurnResult{Text: "Plain tutor response"},
	); err != nil {
		t.Fatal(err)
	}
	if err := NewGatewaySender(gateway).Send(ctx, outboundMessage{
		Channel: "telegram", UserID: "learner-1", Text: "Reminder",
	}); err != nil {
		t.Fatal(err)
	}
	NewGatewayNotifier(gateway, conversations).Notify(ctx, "telegram", "learner-1", "Notification")

	if len(channel.SentMessages) != 3 {
		t.Fatalf("direct gateway messages = %d, want 3", len(channel.SentMessages))
	}
	got := []string{
		channel.SentMessages[0].Text,
		channel.SentMessages[1].Text,
		channel.SentMessages[2].Text,
	}
	want := []string{"Plain tutor response", "Reminder", "Notification"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("message %d = %q, want %q", i, got[i], want[i])
		}
	}
}

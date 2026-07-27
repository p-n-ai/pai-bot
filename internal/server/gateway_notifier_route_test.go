// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestGatewayNotifierSendsToSavedRouteForExplicitChannel(t *testing.T) {
	ctx := context.Background()
	conversations := agent.NewMemoryStore()
	gateway := chat.NewGateway()

	routes := map[string]string{
		"slack":   "slack:C123:1700000000.000001",
		"discord": "discord:G123:C123",
		"teams":   "teams:Y29udmVyc2F0aW9u:aHR0cHM6Ly9zbS5iYS5saW5r",
	}
	channels := make(map[string]*chat.MockChannel, len(routes))
	for channel, threadID := range routes {
		channel := channel
		adapter := &chat.MockChannel{}
		channels[channel] = adapter
		gateway.Register(channel, adapter)

		identity, err := agent.NewLearnerIdentity(channel, "shared-external-user")
		if err != nil {
			t.Fatalf("NewLearnerIdentity(%s) error = %v", channel, err)
		}
		if _, err := conversations.CreateConversationForThread(identity, threadID, agent.Conversation{
			State: "teaching",
		}); err != nil {
			t.Fatalf("CreateConversationForThread(%s) error = %v", channel, err)
		}
	}

	notifier := NewGatewayNotifier(gateway, conversations)
	for channel, wantThreadID := range routes {
		notifier.Notify(ctx, channel, "shared-external-user", "Challenge ready")

		sent := channels[channel].SentMessages
		if len(sent) != 1 {
			t.Fatalf("%s sent messages = %d, want 1", channel, len(sent))
		}
		if sent[0].ThreadID != wantThreadID {
			t.Fatalf("%s thread ID = %q, want %q", channel, sent[0].ThreadID, wantThreadID)
		}
		if sent[0].UserID != "shared-external-user" {
			t.Fatalf("%s user ID = %q, want shared-external-user", channel, sent[0].UserID)
		}
	}
}

func TestGatewayNotifierUsesLatestActiveSavedRoute(t *testing.T) {
	ctx := context.Background()
	conversations := agent.NewMemoryStore()
	identity, err := agent.NewLearnerIdentity("slack", "U123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conversations.CreateConversationForThread(identity, "slack:C-old:1", agent.Conversation{
		State: "teaching",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := conversations.CreateConversationForThread(identity, "slack:C-new:2", agent.Conversation{
		State: "teaching",
	}); err != nil {
		t.Fatal(err)
	}

	gateway := chat.NewGateway()
	slack := &chat.MockChannel{}
	gateway.Register("slack", slack)

	NewGatewayNotifier(gateway, conversations).Notify(ctx, "slack", "U123", "Latest route")

	if len(slack.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(slack.SentMessages))
	}
	if got := slack.SentMessages[0].ThreadID; got != "slack:C-new:2" {
		t.Fatalf("thread ID = %q, want latest active route", got)
	}
}

func TestGatewayNotifierDoesNotSendWhenSavedRouteIsUnresolved(t *testing.T) {
	ctx := context.Background()
	conversations := agent.NewMemoryStore()
	slackIdentity, err := agent.NewLearnerIdentity("slack", "known-without-route")
	if err != nil {
		t.Fatal(err)
	}
	if err := conversations.SetUserNameFor(slackIdentity, "Aina"); err != nil {
		t.Fatal(err)
	}

	gateway := chat.NewGateway()
	slack := &chat.MockChannel{}
	discord := &chat.MockChannel{}
	teams := &chat.MockChannel{}
	gateway.Register("slack", slack)
	gateway.Register("discord", discord)
	gateway.Register("teams", teams)

	notifier := NewGatewayNotifier(gateway, conversations)
	notifier.Notify(ctx, "slack", "known-without-route", "No route")
	notifier.Notify(ctx, "discord", "missing", "No identity")

	if len(slack.SentMessages)+len(discord.SentMessages)+len(teams.SentMessages) != 0 {
		t.Fatalf(
			"unresolved sends = slack:%d discord:%d teams:%d, want zero",
			len(slack.SentMessages),
			len(discord.SentMessages),
			len(teams.SentMessages),
		)
	}
}

func TestGatewayNotifierAllowsDirectChannelWithoutThreadRoute(t *testing.T) {
	conversations := agent.NewMemoryStore()
	identity, err := agent.NewLearnerIdentity("websocket", "learner-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conversations.CreateConversationFor(identity, agent.Conversation{State: "teaching"}); err != nil {
		t.Fatal(err)
	}
	gateway := chat.NewGateway()
	websocket := &chat.MockChannel{}
	gateway.Register("websocket", websocket)

	NewGatewayNotifier(gateway, conversations).Notify(
		context.Background(),
		"websocket",
		"learner-1",
		"Challenge ready",
	)

	if len(websocket.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(websocket.SentMessages))
	}
	if websocket.SentMessages[0].ThreadID != "" {
		t.Fatalf("ThreadID = %q, want empty direct route", websocket.SentMessages[0].ThreadID)
	}
}

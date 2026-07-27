// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestEngine_IsolatesActiveConversationsForSameExternalUserAcrossChannels(t *testing.T) {
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(ai.NewMockProvider("reply")),
		Store:    store,
	})

	for _, msg := range []chat.InboundMessage{
		{Channel: "telegram", UserID: "shared-user", Text: "telegram question"},
		{Channel: "slack", UserID: "shared-user", Text: "slack question"},
	} {
		if _, err := engine.ProcessMessage(context.Background(), msg); err != nil {
			t.Fatalf("ProcessMessage(%s) error = %v", msg.Channel, err)
		}
	}

	telegram := mustLearnerIdentity(t, "telegram", "shared-user")
	slack := mustLearnerIdentity(t, "slack", "shared-user")
	telegramConversation, ok := store.GetActiveConversationFor(telegram)
	if !ok {
		t.Fatal("telegram active conversation not found")
	}
	slackConversation, ok := store.GetActiveConversationFor(slack)
	if !ok {
		t.Fatal("slack active conversation not found")
	}
	if telegramConversation.ID == slackConversation.ID {
		t.Fatalf("active conversation ID = %q for both channels, want isolated conversations", telegramConversation.ID)
	}
	if got := telegramConversation.Messages[0].Content; got != "telegram question" {
		t.Fatalf("telegram first message = %q, want telegram question", got)
	}
	if got := slackConversation.Messages[0].Content; got != "slack question" {
		t.Fatalf("slack first message = %q, want slack question", got)
	}
}

func TestEngine_IsolatesLearnerProfilesForSameExternalUserAcrossChannels(t *testing.T) {
	store := agent.NewMemoryStore()
	provider := ai.NewMockProvider("reply")
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(provider),
		Store:    store,
	})

	for _, msg := range []chat.InboundMessage{
		{Channel: "telegram", UserID: "shared-profile", FirstName: "Tasha", Text: "telegram question"},
		{Channel: "slack", UserID: "shared-profile", FirstName: "Sam", Text: "slack question"},
	} {
		if _, err := engine.ProcessMessage(context.Background(), msg); err != nil {
			t.Fatalf("ProcessMessage(%s) error = %v", msg.Channel, err)
		}
	}

	telegram := mustLearnerIdentity(t, "telegram", "shared-profile")
	slack := mustLearnerIdentity(t, "slack", "shared-profile")
	if name, ok := store.GetUserNameFor(telegram); !ok || name != "Tasha" {
		t.Fatalf("telegram learner name = %q, %v, want Tasha, true", name, ok)
	}
	if name, ok := store.GetUserNameFor(slack); !ok || name != "Sam" {
		t.Fatalf("slack learner name = %q, %v, want Sam, true", name, ok)
	}
	var prompt strings.Builder
	for _, message := range provider.LastRequest.Messages {
		prompt.WriteString(message.Content)
		prompt.WriteByte('\n')
	}
	if !strings.Contains(prompt.String(), "Sam") {
		t.Fatalf("slack tutor context does not contain Slack learner name: %q", prompt.String())
	}
	if strings.Contains(prompt.String(), "Tasha") {
		t.Fatalf("slack tutor context contains Telegram learner name: %q", prompt.String())
	}
}

func TestEngine_IsolatesConversationHistoryForSameAuthorAcrossThreads(t *testing.T) {
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(ai.NewMockProvider("reply")),
		Store:    store,
	})

	for _, msg := range []chat.InboundMessage{
		{
			Channel:  "slack",
			UserID:   "U123",
			ThreadID: "slack:C123:1700000000.000001",
			Text:     "question in first thread",
		},
		{
			Channel:  "slack",
			UserID:   "U123",
			ThreadID: "slack:C123:1700000000.000002",
			Text:     "question in second thread",
		},
	} {
		if _, err := engine.ProcessMessage(context.Background(), msg); err != nil {
			t.Fatalf("ProcessMessage(%s) error = %v", msg.ThreadID, err)
		}
	}

	identity := mustLearnerIdentity(t, "slack", "U123")
	first, ok := store.GetActiveConversationForThread(identity, "slack:C123:1700000000.000001")
	if !ok {
		t.Fatal("first Slack thread conversation not found")
	}
	second, ok := store.GetActiveConversationForThread(identity, "slack:C123:1700000000.000002")
	if !ok {
		t.Fatal("second Slack thread conversation not found")
	}
	if first.ID == second.ID {
		t.Fatalf("conversation ID = %q for both Slack threads, want isolated histories", first.ID)
	}
	if got := first.Messages[0].Content; got != "question in first thread" {
		t.Fatalf("first thread first message = %q", got)
	}
	if got := second.Messages[0].Content; got != "question in second thread" {
		t.Fatalf("second thread first message = %q", got)
	}
}

func TestEngine_RecoversWhenAnotherInstanceCreatesConversationFirst(t *testing.T) {
	store := &activeConversationRaceStore{MemoryStore: agent.NewMemoryStore()}
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(ai.NewMockProvider("reply")),
		Store:    store,
	})

	response, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:  "slack",
		UserID:   "U-race",
		ThreadID: "slack:C-race:1700000000.000001",
		Text:     "question after race",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if response != "reply" {
		t.Fatalf("ProcessMessage() response = %q, want reply", response)
	}

	identity := mustLearnerIdentity(t, "slack", "U-race")
	conversation, ok := store.GetActiveConversationForThread(identity, "slack:C-race:1700000000.000001")
	if !ok || len(conversation.Messages) != 2 {
		t.Fatalf("active conversation = %#v, %v, want recovered conversation with two messages", conversation, ok)
	}
}

type activeConversationRaceStore struct {
	*agent.MemoryStore
	simulated bool
}

func (s *activeConversationRaceStore) UserExistsFor(agent.LearnerIdentity) bool {
	return true
}

func (s *activeConversationRaceStore) CreateConversationForThread(identity agent.LearnerIdentity, threadID string, conversation agent.Conversation) (string, error) {
	if !s.simulated {
		s.simulated = true
		if _, err := s.MemoryStore.CreateConversationForThread(identity, threadID, conversation); err != nil {
			return "", err
		}
		return "", agent.ErrActiveConversationExists
	}
	return s.MemoryStore.CreateConversationForThread(identity, threadID, conversation)
}

func mustLearnerIdentity(t *testing.T, channel, externalID string) agent.LearnerIdentity {
	t.Helper()
	identity, err := agent.NewLearnerIdentity(channel, externalID)
	if err != nil {
		t.Fatalf("NewLearnerIdentity(%q, %q) error = %v", channel, externalID, err)
	}
	return identity
}

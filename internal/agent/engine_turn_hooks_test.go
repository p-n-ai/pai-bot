// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

func TestEngine_ProcessMessage_TurnHooksMoveRatingPromptBehindFlag(t *testing.T) {
	features, err := featureflags.Parse("turn_hooks")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	mockAI := ai.NewMockProvider("AI response")
	tracker := &callTracker{provider: mockAI}
	var notices []agent.TurnHookCallNotice
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:          mockRouter(tracker),
		Store:             agent.NewMemoryStore(),
		RatingPromptEvery: 1,
		FeatureFlags:      func() featureflags.Features { return features },
		DevMode:           true,
		TurnHookNotice: func(notice agent.TurnHookCallNotice) {
			notices = append(notices, notice)
		},
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-turn-hooks-rating",
		Text:    "question 1",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "[[PAI_REVIEW:") {
		t.Fatalf("expected rating prompt on first tutoring reply, got: %q", resp)
	}
	requests := tracker.Requests()
	if len(requests) != 1 {
		t.Fatalf("AI request count = %d, want 1", len(requests))
	}
	if got := countMessagesContaining(requests[0].Messages, "system", "quick 1-5 rating"); got != 1 {
		t.Fatalf("rating instruction count = %d, want 1", got)
	}
	if len(notices) != 1 {
		t.Fatalf("notice count = %d, want 1", len(notices))
	}
	if notices[0].Name != "rate_convo_hook" || notices[0].Outcome != "inject" {
		t.Fatalf("notice = %#v, want rate_convo_hook inject", notices[0])
	}
}

func TestEngine_ProcessMessage_TurnHooksNoticeContinueOutcome(t *testing.T) {
	features, err := featureflags.Parse("turn_hooks")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var notices []agent.TurnHookCallNotice
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:          mockRouter(ai.NewMockProvider("AI response")),
		Store:             agent.NewMemoryStore(),
		RatingPromptEvery: 5,
		FeatureFlags:      func() featureflags.Features { return features },
		DevMode:           true,
		TurnHookNotice: func(notice agent.TurnHookCallNotice) {
			notices = append(notices, notice)
		},
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-turn-hooks-continue",
		Text:    "question 1",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if contains(resp, "[[PAI_REVIEW:") {
		t.Fatalf("did not expect rating prompt, got: %q", resp)
	}
	if len(notices) != 1 {
		t.Fatalf("notice count = %d, want 1", len(notices))
	}
	if notices[0].Name != "rate_convo_hook" || notices[0].Outcome != "continue" {
		t.Fatalf("notice = %#v, want rate_convo_hook continue", notices[0])
	}
}

func TestEngine_ProcessMessage_TurnHookNoticeDisabledWithoutFeatureFlag(t *testing.T) {
	var notices []agent.TurnHookCallNotice
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:          mockRouter(ai.NewMockProvider("AI response")),
		Store:             agent.NewMemoryStore(),
		RatingPromptEvery: 1,
		DevMode:           true,
		TurnHookNotice: func(notice agent.TurnHookCallNotice) {
			notices = append(notices, notice)
		},
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-turn-hooks-flag-off",
		Text:    "question 1",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "[[PAI_REVIEW:") {
		t.Fatalf("expected legacy rating prompt with flag off, got: %q", resp)
	}
	if len(notices) != 0 {
		t.Fatalf("notice count = %d, want 0", len(notices))
	}
}

func TestEngine_ProcessMessage_TurnHookNoticeDisabledWithoutDevMode(t *testing.T) {
	features, err := featureflags.Parse("turn_hooks")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var notices []agent.TurnHookCallNotice
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:          mockRouter(ai.NewMockProvider("AI response")),
		Store:             agent.NewMemoryStore(),
		RatingPromptEvery: 1,
		FeatureFlags:      func() featureflags.Features { return features },
		TurnHookNotice: func(notice agent.TurnHookCallNotice) {
			notices = append(notices, notice)
		},
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-turn-hooks-dev-off",
		Text:    "question 1",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "[[PAI_REVIEW:") {
		t.Fatalf("expected rating prompt with hooks enabled, got: %q", resp)
	}
	if len(notices) != 0 {
		t.Fatalf("notice count = %d, want 0", len(notices))
	}
}

func countMessagesContaining(messages []ai.Message, role, content string) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == role && contains(msg.Content, content) {
			count++
		}
	}
	return count
}

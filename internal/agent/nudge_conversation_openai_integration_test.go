// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package agent

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestOpenAILiveNudgeConversationDemo(t *testing.T) {
	if liveDemoRunningInCI() {
		t.Skip("skipping live OpenAI nudge conversation demo on CI")
	}

	apiKey := strings.TrimSpace(os.Getenv("LEARN_AI_OPENAI_API_KEY"))
	if apiKey == "" {
		t.Skip("LEARN_AI_OPENAI_API_KEY is not set; skipping live OpenAI nudge conversation demo")
	}

	timeout := 45 * time.Second
	provider := ai.NewOpenAIProvider(apiKey, ai.WithHTTPClient(&http.Client{Timeout: timeout}))
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{250 * time.Millisecond, 500 * time.Millisecond},
		BreakerFailureThreshold: 2,
		BreakerCooldown:         2 * time.Second,
	})
	router.Register("openai", provider)

	store := NewMemoryStore()
	userID := "live-nudge-conversation-demo"
	if err := store.SetUserPreferredLanguage(userID, "en"); err != nil {
		t.Fatalf("SetUserPreferredLanguage() error = %v", err)
	}

	memoryTracker := progress.NewMemoryTracker()
	if err := memoryTracker.UpdateMastery(userID, "kssm-form1", "linear-equations", 0.58); err != nil {
		t.Fatalf("UpdateMastery() error = %v", err)
	}
	dueItem := progress.ProgressItem{
		UserID:       userID,
		SyllabusID:   "kssm-form1",
		TopicID:      "linear-equations",
		MasteryScore: 0.58,
		NextReviewAt: time.Now().Add(-56 * time.Hour),
	}
	tracker := &liveDemoTracker{
		Tracker: memoryTracker,
		due:     []progress.ProgressItem{dueItem},
	}

	streaks := progress.NewMemoryStreakTracker()
	if err := streaks.RecordActivity(userID, time.Now().Add(-48*time.Hour)); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if err := streaks.RecordActivity(userID, time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if err := streaks.RecordActivity(userID, time.Now()); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}

	xp := progress.NewMemoryXPTracker()
	if err := xp.Award(userID, progress.XPSourceSession, 135, map[string]any{"topic_id": "linear-equations"}); err != nil {
		t.Fatalf("Award() error = %v", err)
	}

	gateway := chat.NewGateway()
	channel := &chat.MockChannel{}
	gateway.Register("telegram", channel)

	scheduler := NewScheduler(
		DefaultSchedulerConfig(),
		tracker,
		streaks,
		xp,
		nil,
		NewMemoryNudgeTracker(),
		gateway,
		router,
		store,
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := scheduler.checkUser(ctx, userID, activeMYTTime(time.Now())); err != nil {
		t.Fatalf("checkUser() error = %v", err)
	}
	if len(channel.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(channel.SentMessages))
	}

	nudge := channel.SentMessages[0].Text
	t.Logf("turn 0 assistant nudge:\n%s", nudge)

	engine := NewEngine(EngineConfig{
		AIRouter:             router,
		Store:                store,
		EventLogger:          NewMemoryEventLogger(),
		DisableMultiLanguage: false,
		RatingPromptEvery:    5,
		Tracker:              tracker,
		Streaks:              streaks,
		XP:                   xp,
	})

	userTurn1 := "Okay, let's do it. Why do we subtract 4 first?"
	resp1, err := engine.ProcessMessage(ctx, chat.InboundMessage{
		Channel:  "terminal",
		UserID:   userID,
		Text:     userTurn1,
		Language: "en",
	})
	if err != nil {
		t.Fatalf("turn 1 ProcessMessage error: %v", err)
	}
	t.Logf("turn 1 user:\n%s", userTurn1)
	t.Logf("turn 1 assistant:\n%s", resp1)

	userTurn2 := "I think that gives 2x = 6, right?"
	resp2, err := engine.ProcessMessage(ctx, chat.InboundMessage{
		Channel:  "terminal",
		UserID:   userID,
		Text:     userTurn2,
		Language: "en",
	})
	if err != nil {
		t.Fatalf("turn 2 ProcessMessage error: %v", err)
	}
	t.Logf("turn 2 user:\n%s", userTurn2)
	t.Logf("turn 2 assistant:\n%s", resp2)
}

type liveDemoTracker struct {
	progress.Tracker
	due []progress.ProgressItem
}

func (t *liveDemoTracker) GetDueReviews(userID string) ([]progress.ProgressItem, error) {
	var items []progress.ProgressItem
	for _, item := range t.due {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	return items, nil
}

func liveDemoRunningInCI() bool {
	return strings.EqualFold(os.Getenv("CI"), "true") || strings.EqualFold(os.Getenv("GITHUB_ACTIONS"), "true")
}

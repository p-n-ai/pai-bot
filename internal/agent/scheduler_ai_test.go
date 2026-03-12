package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestSchedulerUsesAIPersonalizedNudgeWhenEnabled(t *testing.T) {
	store := NewMemoryStore()
	if err := store.SetUserPreferredLanguage("user-1", "en"); err != nil {
		t.Fatalf("SetUserPreferredLanguage() error = %v", err)
	}

	mockAI := ai.NewMockProvider("You've built a 4-day streak. Let's revisit linear equations for five focused minutes. Start now.")
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{}})
	router.Register("mock", mockAI)

	scheduler := NewScheduler(
		SchedulerConfig{
			CheckInterval:               5 * time.Minute,
			MaxNudgesPerDay:             MaxNudgesPerDay,
			AIPersonalizedNudgesEnabled: true,
		},
		progress.NewMemoryTracker(),
		progress.NewMemoryStreakTracker(),
		progress.NewMemoryXPTracker(),
		NewMemoryNudgeTracker(),
		chat.NewGateway(),
		router,
		store,
	)

	if err := scheduler.streaks.RecordActivity("user-1", time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if err := scheduler.streaks.RecordActivity("user-1", time.Now()); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if err := scheduler.xp.Award("user-1", progress.XPSourceSession, 80, map[string]any{"topic_id": "linear-equations"}); err != nil {
		t.Fatalf("Award() error = %v", err)
	}

	now := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	item := progress.ProgressItem{
		UserID:       "user-1",
		TopicID:      "linear-equations",
		MasteryScore: 0.62,
		NextReviewAt: now.Add(-48 * time.Hour),
	}

	msg := scheduler.buildNudgeMessage(context.Background(), "user-1", item, now)
	want := "You've built a 4-day streak.\nLet's revisit linear equations for five focused minutes.\nStart now."
	if msg != want {
		t.Fatalf("buildNudgeMessage() = %q, want formatted AI response %q", msg, want)
	}
	if mockAI.LastRequest == nil {
		t.Fatal("expected AI request to be captured")
	}
	if mockAI.LastRequest.Task != ai.TaskNudge {
		t.Fatalf("AI task = %v, want %v", mockAI.LastRequest.Task, ai.TaskNudge)
	}
	if len(mockAI.LastRequest.Messages) != 2 {
		t.Fatalf("AI messages = %d, want 2", len(mockAI.LastRequest.Messages))
	}
	if !strings.Contains(mockAI.LastRequest.Messages[0].Content, "clear invitation to continue learning now") {
		t.Fatalf("AI system prompt = %q, want invitation requirement", mockAI.LastRequest.Messages[0].Content)
	}
	if !strings.Contains(mockAI.LastRequest.Messages[0].Content, "Keep it short: 1 to 3 short sentences") {
		t.Fatalf("AI system prompt = %q, want brevity requirement", mockAI.LastRequest.Messages[0].Content)
	}
	if !strings.Contains(mockAI.LastRequest.Messages[0].Content, "Use line breaks") {
		t.Fatalf("AI system prompt = %q, want readability requirement", mockAI.LastRequest.Messages[0].Content)
	}
	if !strings.Contains(mockAI.LastRequest.Messages[1].Content, "Preferred language: en") {
		t.Fatalf("AI prompt = %q, want preferred language", mockAI.LastRequest.Messages[1].Content)
	}
	if !strings.Contains(mockAI.LastRequest.Messages[1].Content, "Current streak: 2 days") {
		t.Fatalf("AI prompt = %q, want streak context", mockAI.LastRequest.Messages[1].Content)
	}
	if !strings.Contains(mockAI.LastRequest.Messages[1].Content, "Total XP: 80") {
		t.Fatalf("AI prompt = %q, want xp context", mockAI.LastRequest.Messages[1].Content)
	}
}

func TestSchedulerFallsBackWhenAINudgeFails(t *testing.T) {
	store := NewMemoryStore()
	if err := store.SetUserPreferredLanguage("user-2", "en"); err != nil {
		t.Fatalf("SetUserPreferredLanguage() error = %v", err)
	}

	mockAI := ai.NewMockProvider("")
	mockAI.Err = context.DeadlineExceeded
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{}})
	router.Register("mock", mockAI)

	scheduler := NewScheduler(
		SchedulerConfig{
			CheckInterval:               5 * time.Minute,
			MaxNudgesPerDay:             MaxNudgesPerDay,
			AIPersonalizedNudgesEnabled: true,
		},
		progress.NewMemoryTracker(),
		progress.NewMemoryStreakTracker(),
		progress.NewMemoryXPTracker(),
		NewMemoryNudgeTracker(),
		chat.NewGateway(),
		router,
		store,
	)

	now := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	item := progress.ProgressItem{
		UserID:       "user-2",
		TopicID:      "linear-equations",
		MasteryScore: 0.62,
		NextReviewAt: now.Add(-48 * time.Hour),
	}

	msg := scheduler.buildNudgeMessage(context.Background(), "user-2", item, now)
	if strings.TrimSpace(msg) == "" {
		t.Fatal("buildNudgeMessage() should return fallback text")
	}
	if !strings.Contains(msg, "Topic: linear-equations") {
		t.Fatalf("fallback nudge = %q, want English localized topic label", msg)
	}
	if !strings.Contains(msg, "Mastery: 62%") {
		t.Fatalf("fallback nudge = %q, want mastery percentage", msg)
	}
}

func TestFormatAINudgeMessageSplitsDenseParagraph(t *testing.T) {
	raw := "Hey there! You've got a solid base in linear equations with a 57% mastery score. Let's boost your skills and get you back on track. Dive into some practice now and start your learning streak!"

	got := formatAINudgeMessage(raw)
	want := "Hey there!\nYou've got a solid base in linear equations with a 57% mastery score.\nLet's boost your skills and get you back on track.\nDive into some practice now and start your learning streak!"

	if got != want {
		t.Fatalf("formatAINudgeMessage() = %q, want %q", got, want)
	}
}

//go:build integration
// +build integration

package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestSchedulerOpenAILiveNudge(t *testing.T) {
	if strings.EqualFold(os.Getenv("CI"), "true") {
		t.Skip("skipping live OpenAI integration test on CI")
	}

	apiKey := strings.TrimSpace(os.Getenv("LEARN_AI_OPENAI_API_KEY"))
	if apiKey == "" {
		t.Skip("LEARN_AI_OPENAI_API_KEY is not set; skipping live OpenAI nudge test")
	}

	store := NewMemoryStore()
	if err := store.SetUserPreferredLanguage("live-nudge-user", "en"); err != nil {
		t.Fatalf("SetUserPreferredLanguage() error = %v", err)
	}

	provider := ai.NewOpenAIProvider(apiKey)
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{250 * time.Millisecond, 500 * time.Millisecond},
		BreakerFailureThreshold: 2,
		BreakerCooldown:         2 * time.Second,
	})
	router.Register("openai", provider)

	streaks := progress.NewMemoryStreakTracker()
	xp := progress.NewMemoryXPTracker()
	if err := streaks.RecordActivity("live-nudge-user", time.Now().Add(-48*time.Hour)); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if err := streaks.RecordActivity("live-nudge-user", time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if err := streaks.RecordActivity("live-nudge-user", time.Now()); err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if err := xp.Award("live-nudge-user", progress.XPSourceSession, 135, map[string]any{"topic_id": "algebra-linear-equations"}); err != nil {
		t.Fatalf("Award() error = %v", err)
	}

	scheduler := NewScheduler(
		DefaultSchedulerConfig(),
		progress.NewMemoryTracker(),
		streaks,
		xp,
		nil,
		NewMemoryNudgeTracker(),
		chat.NewGateway(),
		router,
		store,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	now := time.Now()
	item := progress.ProgressItem{
		UserID:       "live-nudge-user",
		SyllabusID:   "kssm-form1",
		TopicID:      "algebra-linear-equations",
		MasteryScore: 0.58,
		NextReviewAt: now.Add(-56 * time.Hour),
	}

	msg := scheduler.buildNudgeMessage(ctx, "live-nudge-user", item, now)
	if strings.TrimSpace(msg) == "" {
		t.Fatal("buildNudgeMessage() returned empty output")
	}

	t.Logf("live ai nudge output:\n%s", msg)
}

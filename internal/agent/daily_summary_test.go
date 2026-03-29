package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestComputeDailySummary(t *testing.T) {
	tracker := progress.NewMemoryTracker()
	streaks := progress.NewMemoryStreakTracker()
	xp := progress.NewMemoryXPTracker()

	_ = tracker.UpdateMastery("user1", "default", "F1-01", 0.6)
	_ = tracker.UpdateMastery("user1", "default", "F1-02", 0.8)
	_ = streaks.RecordActivity("user1", time.Now())
	_ = xp.Award("user1", progress.XPSourceSession, 30, nil)
	_ = xp.Award("user1", progress.XPSourceQuiz, 20, nil)

	summary := ComputeDailySummary("user1", tracker, streaks, xp)

	if summary.UserID != "user1" {
		t.Errorf("UserID = %q, want user1", summary.UserID)
	}
	if summary.TopicsStudied < 1 {
		t.Errorf("TopicsStudied = %d, want >= 1", summary.TopicsStudied)
	}
	if summary.TotalXP != 50 {
		t.Errorf("TotalXP = %d, want 50", summary.TotalXP)
	}
	if summary.CurrentStreak < 1 {
		t.Errorf("CurrentStreak = %d, want >= 1", summary.CurrentStreak)
	}
}

func TestComputeDailySummary_NoProgress(t *testing.T) {
	tracker := progress.NewMemoryTracker()
	streaks := progress.NewMemoryStreakTracker()
	xp := progress.NewMemoryXPTracker()

	summary := ComputeDailySummary("nobody", tracker, streaks, xp)
	if summary.TopicsStudied != 0 {
		t.Errorf("TopicsStudied = %d, want 0", summary.TopicsStudied)
	}
	if summary.TotalXP != 0 {
		t.Errorf("TotalXP = %d, want 0", summary.TotalXP)
	}
}

func TestFormatDailySummary(t *testing.T) {
	summary := DailySummary{
		UserID:         "user1",
		TopicsStudied:  3,
		MasteredTopics: 1,
		TotalXP:        250,
		CurrentStreak:  5,
		BestTopic:      "Linear Equations",
		BestMastery:    0.92,
	}

	msg := FormatDailySummary(summary, "en")
	if msg == "" {
		t.Error("FormatDailySummary returned empty")
	}
	if !containsAll(msg, "3", "250", "5", "Linear Equations") {
		t.Errorf("FormatDailySummary missing expected content: %s", msg)
	}
}

func TestFormatDailySummary_ZeroTopics(t *testing.T) {
	summary := DailySummary{UserID: "user1"}
	msg := FormatDailySummary(summary, "ms")
	if msg != "" {
		t.Errorf("expected empty summary for inactive day, got: %s", msg)
	}
}

func TestTimeUntilNext(t *testing.T) {
	d := timeUntilNext(22, 0)
	if d <= 0 || d > 24*time.Hour {
		t.Errorf("timeUntilNext(22,0) = %v, want 0 < d <= 24h", d)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

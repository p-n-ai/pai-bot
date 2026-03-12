package progress_test

import (
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestStreakTracker_FirstActivity(t *testing.T) {
	tracker := progress.NewMemoryStreakTracker()

	if err := tracker.RecordActivity("user1", time.Now()); err != nil {
		t.Fatalf("RecordActivity: %v", err)
	}
	streak, err := tracker.GetStreak("user1")
	if err != nil {
		t.Fatalf("GetStreak: %v", err)
	}

	if streak.CurrentStreak != 1 {
		t.Errorf("CurrentStreak = %d, want 1", streak.CurrentStreak)
	}
	if streak.LongestStreak != 1 {
		t.Errorf("LongestStreak = %d, want 1", streak.LongestStreak)
	}
}

func TestStreakTracker_SameDayNoop(t *testing.T) {
	tracker := progress.NewMemoryStreakTracker()
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	_ = tracker.RecordActivity("user1", now)
	_ = tracker.RecordActivity("user1", now.Add(2*time.Hour))

	streak, _ := tracker.GetStreak("user1")
	if streak.CurrentStreak != 1 {
		t.Errorf("Same-day duplicate: CurrentStreak = %d, want 1", streak.CurrentStreak)
	}
}

func TestStreakTracker_ConsecutiveDays(t *testing.T) {
	tracker := progress.NewMemoryStreakTracker()
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	_ = tracker.RecordActivity("user1", now.Add(-48*time.Hour))
	_ = tracker.RecordActivity("user1", now.Add(-24*time.Hour))
	_ = tracker.RecordActivity("user1", now)

	streak, _ := tracker.GetStreak("user1")
	if streak.CurrentStreak != 3 {
		t.Errorf("ConsecutiveDays: CurrentStreak = %d, want 3", streak.CurrentStreak)
	}
	if streak.LongestStreak != 3 {
		t.Errorf("ConsecutiveDays: LongestStreak = %d, want 3", streak.LongestStreak)
	}
}

func TestStreakTracker_BrokenStreak(t *testing.T) {
	tracker := progress.NewMemoryStreakTracker()
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	// Build 3-day streak
	_ = tracker.RecordActivity("user1", now.Add(-72*time.Hour))
	_ = tracker.RecordActivity("user1", now.Add(-48*time.Hour))
	_ = tracker.RecordActivity("user1", now.Add(-24*time.Hour))
	// Skip a day, then come back
	_ = tracker.RecordActivity("user1", now.Add(24*time.Hour))

	streak, _ := tracker.GetStreak("user1")
	if streak.CurrentStreak != 1 {
		t.Errorf("BrokenStreak: CurrentStreak = %d, want 1", streak.CurrentStreak)
	}
	if streak.LongestStreak != 3 {
		t.Errorf("BrokenStreak: LongestStreak = %d, want 3 (preserved)", streak.LongestStreak)
	}
}

func TestStreakTracker_UnknownUser(t *testing.T) {
	tracker := progress.NewMemoryStreakTracker()

	streak, err := tracker.GetStreak("nobody")
	if err != nil {
		t.Fatalf("GetStreak: %v", err)
	}
	if streak.CurrentStreak != 0 {
		t.Errorf("Unknown user: CurrentStreak = %d, want 0", streak.CurrentStreak)
	}
}

func TestIsStreakMilestone(t *testing.T) {
	tests := []struct {
		days        int
		isMilestone bool
	}{
		{1, false},
		{2, false},
		{3, true},
		{5, false},
		{7, true},
		{10, false},
		{14, true},
		{30, true},
		{60, true},
		{100, true},
	}

	for _, tt := range tests {
		got := progress.IsStreakMilestone(tt.days)
		if got != tt.isMilestone {
			t.Errorf("IsStreakMilestone(%d) = %v, want %v", tt.days, got, tt.isMilestone)
		}
	}
}

func TestStreakMilestoneMessage(t *testing.T) {
	tests := []struct {
		days    int
		wantLen bool // should produce a non-empty message
	}{
		{3, true},
		{7, true},
		{14, true},
		{30, true},
		{5, false},
	}

	for _, tt := range tests {
		msg := progress.StreakMilestoneMessage(tt.days)
		if tt.wantLen && msg == "" {
			t.Errorf("StreakMilestoneMessage(%d) = empty, want message", tt.days)
		}
		if !tt.wantLen && msg != "" {
			t.Errorf("StreakMilestoneMessage(%d) = %q, want empty", tt.days, msg)
		}
	}
}

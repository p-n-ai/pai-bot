package progress_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestXPTracker_AwardAndTotal(t *testing.T) {
	xp := progress.NewMemoryXPTracker()

	if err := xp.Award("user1", progress.XPSourceSession, 10, nil); err != nil {
		t.Fatalf("Award: %v", err)
	}

	total, err := xp.GetTotal("user1")
	if err != nil {
		t.Fatalf("GetTotal: %v", err)
	}
	if total != 10 {
		t.Errorf("GetTotal = %d, want 10", total)
	}
}

func TestXPTracker_MultipleAwards(t *testing.T) {
	xp := progress.NewMemoryXPTracker()

	_ = xp.Award("user1", progress.XPSourceSession, 10, nil)
	_ = xp.Award("user1", progress.XPSourceQuiz, 20, nil)
	_ = xp.Award("user1", progress.XPSourceMastery, 50, nil)
	_ = xp.Award("user1", progress.XPSourceStreak, 100, nil)

	total, _ := xp.GetTotal("user1")
	if total != 180 {
		t.Errorf("GetTotal = %d, want 180", total)
	}
}

func TestXPTracker_SeparateUsers(t *testing.T) {
	xp := progress.NewMemoryXPTracker()

	_ = xp.Award("user1", progress.XPSourceSession, 10, nil)
	_ = xp.Award("user2", progress.XPSourceSession, 25, nil)

	t1, _ := xp.GetTotal("user1")
	t2, _ := xp.GetTotal("user2")

	if t1 != 10 {
		t.Errorf("user1 total = %d, want 10", t1)
	}
	if t2 != 25 {
		t.Errorf("user2 total = %d, want 25", t2)
	}
}

func TestXPTracker_UnknownUser(t *testing.T) {
	xp := progress.NewMemoryXPTracker()

	total, err := xp.GetTotal("nobody")
	if err != nil {
		t.Fatalf("GetTotal: %v", err)
	}
	if total != 0 {
		t.Errorf("Unknown user total = %d, want 0", total)
	}
}

func TestXPTracker_ZeroAmount(t *testing.T) {
	xp := progress.NewMemoryXPTracker()

	err := xp.Award("user1", progress.XPSourceSession, 0, nil)
	if err == nil {
		t.Error("Expected error for zero amount")
	}
}

func TestXPTracker_NegativeAmount(t *testing.T) {
	xp := progress.NewMemoryXPTracker()

	err := xp.Award("user1", progress.XPSourceSession, -5, nil)
	if err == nil {
		t.Error("Expected error for negative amount")
	}
}

func TestXPValues(t *testing.T) {
	// Verify XP constants match design spec
	tests := []struct {
		name   string
		source progress.XPSource
		amount int
	}{
		{"session", progress.XPSourceSession, progress.XPSession},
		{"quiz_correct", progress.XPSourceQuiz, progress.XPQuizCorrect},
		{"mastery", progress.XPSourceMastery, progress.XPMasteryUp},
		{"streak_milestone", progress.XPSourceStreak, progress.XPStreakMilestone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.amount <= 0 {
				t.Errorf("%s XP = %d, want > 0", tt.name, tt.amount)
			}
		})
	}
}

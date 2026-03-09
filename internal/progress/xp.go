package progress

import (
	"fmt"
	"sync"
)

// XPSource identifies where XP was earned.
type XPSource string

const (
	XPSourceSession   XPSource = "session"
	XPSourceQuiz      XPSource = "quiz"
	XPSourceMastery   XPSource = "mastery"
	XPSourceStreak    XPSource = "streak"
	XPSourceChallenge XPSource = "challenge"
	XPSourceReview    XPSource = "review"
)

// XP award amounts.
const (
	XPSession         = 10  // per teaching session message exchange
	XPQuizCorrect     = 20  // per correct quiz answer
	XPMasteryUp       = 50  // when mastery threshold crossed for a topic
	XPStreakMilestone  = 100 // on streak milestones (3, 7, 14, 30, etc.)
	XPChallengeWin    = 30  // winning a peer challenge
	XPReviewCompleted = 15  // completing post-challenge review
)

// XPEntry represents a single XP award.
type XPEntry struct {
	UserID   string
	Source   XPSource
	Amount   int
	Metadata map[string]any
}

// XPTracker defines the interface for XP tracking.
type XPTracker interface {
	Award(userID string, source XPSource, amount int, metadata map[string]any) error
	GetTotal(userID string) (int, error)
}

// MemoryXPTracker is an in-memory implementation for testing.
type MemoryXPTracker struct {
	mu      sync.RWMutex
	entries []XPEntry
}

// NewMemoryXPTracker creates a new in-memory XP tracker.
func NewMemoryXPTracker() *MemoryXPTracker {
	return &MemoryXPTracker{}
}

func (t *MemoryXPTracker) Award(userID string, source XPSource, amount int, metadata map[string]any) error {
	if amount <= 0 {
		return fmt.Errorf("XP amount must be positive, got %d", amount)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries = append(t.entries, XPEntry{
		UserID:   userID,
		Source:   source,
		Amount:   amount,
		Metadata: metadata,
	})
	return nil
}

func (t *MemoryXPTracker) GetTotal(userID string) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := 0
	for _, e := range t.entries {
		if e.UserID == userID {
			total += e.Amount
		}
	}
	return total, nil
}

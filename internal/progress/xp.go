// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"fmt"
	"sync"

	"github.com/p-n-ai/pai-bot/internal/jsonobject"
)

type XPField = jsonobject.Field
type XPMetadata = jsonobject.Object

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
	XPStreakMilestone = 100 // on streak milestones (3, 7, 14, 30, etc.)
	XPChallengeWin    = 30  // winning a peer challenge
	XPReviewCompleted = 50  // completing post-challenge review
)

// XPEntry represents a single XP award.
type XPEntry struct {
	UserID   string
	Source   XPSource
	Amount   int
	Metadata *XPMetadata
}

// XPTracker defines the interface for XP tracking.
type XPTracker interface {
	Award(userID string, source XPSource, amount int, metadata *XPMetadata) error
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

func (t *MemoryXPTracker) Award(userID string, source XPSource, amount int, metadata *XPMetadata) error {
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

func NewXPField[T any](name string, value T) XPField {
	return jsonobject.Member(name, value)
}

func NewXPMetadata(fields ...XPField) *XPMetadata {
	metadata := jsonobject.New(fields...)
	return &metadata
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

// ResetAll removes all XP entries for a user.
func (t *MemoryXPTracker) ResetAll(userID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	filtered := t.entries[:0]
	for _, e := range t.entries {
		if e.UserID != userID {
			filtered = append(filtered, e)
		}
	}
	t.entries = filtered
	return nil
}

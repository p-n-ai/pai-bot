// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"sync"
	"time"
)

// Streak holds a user's streak data.
type Streak struct {
	UserID        string
	CurrentStreak int
	LongestStreak int
	LastActiveDate time.Time // truncated to date
}

// StreakTracker defines the interface for streak tracking.
type StreakTracker interface {
	RecordActivity(userID string, at time.Time) error
	GetStreak(userID string) (Streak, error)
}

// IsStreakMilestone returns true if the streak count is a milestone worth celebrating.
func IsStreakMilestone(days int) bool {
	switch days {
	case 3, 7, 14, 30, 60, 100:
		return true
	}
	return false
}

// StreakMilestoneMessage returns a celebration message for milestone streaks.
func StreakMilestoneMessage(days int) string {
	switch days {
	case 3:
		return "3 hari berturut-turut! Teruskan momentum! (3-day streak!)"
	case 7:
		return "Seminggu tanpa henti! Hebat! (7-day streak!)"
	case 14:
		return "2 minggu konsisten! Luar biasa! (14-day streak!)"
	case 30:
		return "Sebulan penuh! Anda seorang pejuang matematik! (30-day streak!)"
	case 60:
		return "60 hari! Dedikasi yang mengagumkan! (60-day streak!)"
	case 100:
		return "100 HARI! Legenda! (100-day streak!)"
	}
	return ""
}

// MemoryStreakTracker is an in-memory implementation for testing.
type MemoryStreakTracker struct {
	mu      sync.RWMutex
	streaks map[string]*Streak
}

// NewMemoryStreakTracker creates a new in-memory streak tracker.
func NewMemoryStreakTracker() *MemoryStreakTracker {
	return &MemoryStreakTracker{
		streaks: make(map[string]*Streak),
	}
}

func (t *MemoryStreakTracker) RecordActivity(userID string, at time.Time) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	today := truncateToDate(at)

	s, ok := t.streaks[userID]
	if !ok {
		t.streaks[userID] = &Streak{
			UserID:         userID,
			CurrentStreak:  1,
			LongestStreak:  1,
			LastActiveDate: today,
		}
		return nil
	}

	lastDate := truncateToDate(s.LastActiveDate)
	diff := today.Sub(lastDate)

	switch {
	case diff < 24*time.Hour:
		// Same day — no change.
	case diff < 48*time.Hour:
		// Next day — increment streak.
		s.CurrentStreak++
		if s.CurrentStreak > s.LongestStreak {
			s.LongestStreak = s.CurrentStreak
		}
	default:
		// Missed day(s) — reset streak.
		s.CurrentStreak = 1
	}

	s.LastActiveDate = today
	return nil
}

func (t *MemoryStreakTracker) GetStreak(userID string) (Streak, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	s, ok := t.streaks[userID]
	if !ok {
		return Streak{UserID: userID}, nil
	}
	return *s, nil
}

// ResetAll removes all streak data for a user.
func (t *MemoryStreakTracker) ResetAll(userID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.streaks, userID)
	return nil
}

// truncateToDate truncates a time to the start of the day in UTC.
func truncateToDate(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

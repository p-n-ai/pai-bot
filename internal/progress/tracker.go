// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"strings"
	"sync"
	"time"
)

// MasteryThreshold is the score at or above which a topic is considered mastered.
const MasteryThreshold = 0.75

// ProgressItem represents a student's progress on a single topic.
type ProgressItem struct {
	UserID       string
	SyllabusID   string
	TopicID      string
	MasteryScore float64
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	NextReviewAt time.Time
	LastStudied  time.Time
}

// Tracker defines the interface for mastery progress tracking.
type Tracker interface {
	UpdateMastery(userID, syllabusID, topicID string, delta float64) error
	GetMastery(userID, syllabusID, topicID string) (float64, error)
	GetAllProgress(userID string) ([]ProgressItem, error)
	GetDueReviews(userID string) ([]ProgressItem, error)
}

// IsMastered returns true if the score meets or exceeds MasteryThreshold.
func IsMastered(score float64) bool {
	return score >= MasteryThreshold
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func progressKey(userID, syllabusID, topicID string) string {
	return userID + "|" + syllabusID + "|" + topicID
}

// MemoryTracker is an in-memory implementation of Tracker for testing and development.
type MemoryTracker struct {
	mu    sync.RWMutex
	items map[string]*ProgressItem
}

// NewMemoryTracker creates a new in-memory tracker.
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		items: make(map[string]*ProgressItem),
	}
}

func (m *MemoryTracker) UpdateMastery(userID, syllabusID, topicID string, delta float64) error {
	delta = clamp(delta, 0.0, 1.0)

	m.mu.Lock()
	defer m.mu.Unlock()

	key := progressKey(userID, syllabusID, topicID)
	item, exists := m.items[key]
	now := time.Now()

	if !exists {
		// Seed on first update: set score directly to delta.
		quality := DeltaToQuality(delta)
		sm2 := SM2Calculate(quality, 0, 2.5, 1)
		m.items[key] = &ProgressItem{
			UserID:       userID,
			SyllabusID:   syllabusID,
			TopicID:      topicID,
			MasteryScore: delta,
			EaseFactor:   sm2.EaseFactor,
			IntervalDays: sm2.IntervalDays,
			Repetitions:  sm2.Repetitions,
			NextReviewAt: now.Add(time.Duration(sm2.IntervalDays*24) * time.Hour),
			LastStudied:  now,
		}
		return nil
	}

	// Weighted blend on subsequent updates.
	item.MasteryScore = clamp(item.MasteryScore*0.7+delta*0.3, 0.0, 1.0)

	quality := DeltaToQuality(delta)
	sm2 := SM2Calculate(quality, item.Repetitions, item.EaseFactor, item.IntervalDays)
	item.EaseFactor = sm2.EaseFactor
	item.IntervalDays = sm2.IntervalDays
	item.Repetitions = sm2.Repetitions
	item.LastStudied = now
	item.NextReviewAt = now.Add(time.Duration(sm2.IntervalDays*24) * time.Hour)

	return nil
}

func (m *MemoryTracker) GetMastery(userID, syllabusID, topicID string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := progressKey(userID, syllabusID, topicID)
	item, exists := m.items[key]
	if !exists {
		return 0, nil
	}
	return item.MasteryScore, nil
}

func (m *MemoryTracker) GetAllProgress(userID string) ([]ProgressItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ProgressItem
	for _, item := range m.items {
		if item.UserID == userID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *MemoryTracker) GetDueReviews(userID string) ([]ProgressItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	var result []ProgressItem
	for _, item := range m.items {
		if item.UserID == userID && !item.NextReviewAt.After(now) {
			result = append(result, *item)
		}
	}
	return result, nil
}

// SetMastery directly sets a topic's mastery score (dev/testing only).
func (m *MemoryTracker) SetMastery(userID, syllabusID, topicID string, score float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := progressKey(userID, syllabusID, topicID)
	if item, exists := m.items[key]; exists {
		item.MasteryScore = score
	} else {
		m.items[key] = &ProgressItem{
			UserID:       userID,
			SyllabusID:   syllabusID,
			TopicID:      topicID,
			MasteryScore: score,
			EaseFactor:   2.5,
			IntervalDays: 1,
		}
	}
	return nil
}

// ResetAll removes all progress data for a user.
func (m *MemoryTracker) ResetAll(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key := range m.items {
		if strings.HasPrefix(key, userID+"|") {
			delete(m.items, key)
		}
	}
	return nil
}

// setNextReviewAt is a test helper to override the next review time.
func (m *MemoryTracker) setNextReviewAt(userID, syllabusID, topicID string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := progressKey(userID, syllabusID, topicID)
	if item, exists := m.items[key]; exists {
		item.NextReviewAt = t
	}
}

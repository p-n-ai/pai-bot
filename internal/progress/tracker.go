package progress

import (
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
		m.items[key] = &ProgressItem{
			UserID:       userID,
			SyllabusID:   syllabusID,
			TopicID:      topicID,
			MasteryScore: delta,
			EaseFactor:   2.5,
			IntervalDays: 1,
			Repetitions:  1,
			NextReviewAt: now.Add(24 * time.Hour),
			LastStudied:  now,
		}
		return nil
	}

	// Weighted blend on subsequent updates.
	item.MasteryScore = clamp(item.MasteryScore*0.7+delta*0.3, 0.0, 1.0)
	item.Repetitions++
	item.LastStudied = now
	item.NextReviewAt = now.Add(time.Duration(item.IntervalDays*24) * time.Hour)

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

// setNextReviewAt is a test helper to override the next review time.
func (m *MemoryTracker) setNextReviewAt(userID, syllabusID, topicID string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := progressKey(userID, syllabusID, topicID)
	if item, exists := m.items[key]; exists {
		item.NextReviewAt = t
	}
}

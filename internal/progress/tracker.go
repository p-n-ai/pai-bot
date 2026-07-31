// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// MasteryThreshold is the score at or above which a topic is considered mastered.
const MasteryThreshold = 0.75

// LearnerID identifies one internal user record. Chat-provider identifiers must
// be resolved to this value before entering learner progress persistence.
type LearnerID struct {
	value string
}

// NewLearnerID creates a refined internal learner identifier.
func NewLearnerID(value string) (LearnerID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return LearnerID{}, fmt.Errorf("learner ID is required")
	}
	return LearnerID{value: value}, nil
}

// String returns the persistence-safe learner identifier.
func (id LearnerID) String() string {
	return id.value
}

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

// LearnerTracker exposes progress operations keyed by an internal learner ID.
// It lets runtime callers avoid ambiguous provider-owned external IDs while the
// legacy Tracker interface remains available to existing scheduler callers.
type LearnerTracker interface {
	UpdateMasteryForLearner(learnerID LearnerID, syllabusID, topicID string, delta float64) error
	GetMasteryForLearner(learnerID LearnerID, syllabusID, topicID string) (float64, error)
	GetAllProgressForLearner(learnerID LearnerID) ([]ProgressItem, error)
	GetDueReviewsForLearner(learnerID LearnerID) ([]ProgressItem, error)
}

// ErrMasteryEvidenceConflict means an idempotency key was reused for different
// learner evidence.
var ErrMasteryEvidenceConflict = errors.New("mastery evidence attempt ID conflicts with an existing attempt")

// MasteryEvidence is one gradeable observation that may update mastery once.
type MasteryEvidence struct {
	AttemptID      string
	LearnerID      LearnerID
	SyllabusID     string
	TopicID        string
	SourceKind     string
	SourceID       string
	SourceRevision string
	Score          float64
	PayloadHash    [32]byte
}

// MasteryEvidenceResult describes the stable transition produced by evidence.
// Applied is false when the exact attempt was already recorded.
type MasteryEvidenceResult struct {
	Applied       bool
	MasteryBefore float64
	MasteryAfter  float64
}

// MasteryEvidenceCount is the number of distinct recorded attempts for a topic.
type MasteryEvidenceCount struct {
	SyllabusID string
	TopicID    string
	Count      int
}

// MasterySnapshotItem combines one topic's current score and durable evidence
// count from the same read boundary.
type MasterySnapshotItem struct {
	SyllabusID     string
	TopicID        string
	MasteryScore   float64
	MasteryKnown   bool
	EvidenceCount  int
	LatestEvidence *MasteryEvidenceSummary
}

// MasteryEvidenceSummary identifies the latest graded source for a topic.
type MasteryEvidenceSummary struct {
	SourceKind string
	SourceID   string
	Score      float64
}

// MasteryEvidenceStore atomically records evidence and reports its coverage.
type MasteryEvidenceStore interface {
	RecordMasteryEvidence(context.Context, MasteryEvidence) (MasteryEvidenceResult, error)
	GetMasteryEvidenceCounts(context.Context, LearnerID) ([]MasteryEvidenceCount, error)
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
	mu               sync.RWMutex
	items            map[string]*ProgressItem
	evidence         map[string]recordedMasteryEvidence
	evidenceSequence uint64
}

type recordedMasteryEvidence struct {
	evidence MasteryEvidence
	result   MasteryEvidenceResult
	sequence uint64
}

// NewMemoryTracker creates a new in-memory tracker.
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		items:    make(map[string]*ProgressItem),
		evidence: make(map[string]recordedMasteryEvidence),
	}
}

func (m *MemoryTracker) UpdateMastery(userID, syllabusID, topicID string, delta float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateMasteryLocked(userID, syllabusID, topicID, delta, time.Now())
	return nil
}

func (m *MemoryTracker) updateMasteryLocked(userID, syllabusID, topicID string, delta float64, now time.Time) (float64, float64) {
	key := progressKey(userID, syllabusID, topicID)
	var existing *ProgressItem
	if item, found := m.items[key]; found {
		existing = item
	}
	before, next := calculateMasteryTransition(userID, syllabusID, topicID, delta, existing, now)
	m.items[key] = &next
	return before, next.MasteryScore
}

func calculateMasteryTransition(userID, syllabusID, topicID string, delta float64, existing *ProgressItem, now time.Time) (float64, ProgressItem) {
	delta = clamp(delta, 0.0, 1.0)
	before := 0.0
	repetitions := 0
	easeFactor := 2.5
	intervalDays := 1
	score := delta
	if existing != nil {
		before = existing.MasteryScore
		repetitions = existing.Repetitions
		easeFactor = existing.EaseFactor
		intervalDays = existing.IntervalDays
		score = clamp(before*0.7+delta*0.3, 0.0, 1.0)
	}
	quality := DeltaToQuality(delta)
	sm2 := SM2Calculate(quality, repetitions, easeFactor, intervalDays)
	return before, ProgressItem{
		UserID:       userID,
		SyllabusID:   syllabusID,
		TopicID:      topicID,
		MasteryScore: score,
		EaseFactor:   sm2.EaseFactor,
		IntervalDays: sm2.IntervalDays,
		Repetitions:  sm2.Repetitions,
		NextReviewAt: now.Add(time.Duration(sm2.IntervalDays*24) * time.Hour),
		LastStudied:  now,
	}
}

func (m *MemoryTracker) UpdateMasteryForLearner(learnerID LearnerID, syllabusID, topicID string, delta float64) error {
	return m.UpdateMastery(learnerID.String(), syllabusID, topicID, delta)
}

// RecordMasteryEvidence records one idempotent mastery transition.
func (m *MemoryTracker) RecordMasteryEvidence(ctx context.Context, evidence MasteryEvidence) (MasteryEvidenceResult, error) {
	if err := ctx.Err(); err != nil {
		return MasteryEvidenceResult{}, err
	}
	evidence, err := validateMasteryEvidence(evidence)
	if err != nil {
		return MasteryEvidenceResult{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.evidence == nil {
		m.evidence = make(map[string]recordedMasteryEvidence)
	}
	key := evidence.LearnerID.String() + "\x00" + evidence.AttemptID
	if recorded, found := m.evidence[key]; found {
		if !sameMasteryEvidence(recorded.evidence, evidence) {
			return MasteryEvidenceResult{}, ErrMasteryEvidenceConflict
		}
		replay := recorded.result
		replay.Applied = false
		return replay, nil
	}

	before, after := m.updateMasteryLocked(
		evidence.LearnerID.String(),
		evidence.SyllabusID,
		evidence.TopicID,
		evidence.Score,
		time.Now(),
	)
	result := MasteryEvidenceResult{
		Applied:       true,
		MasteryBefore: before,
		MasteryAfter:  after,
	}
	m.evidenceSequence++
	m.evidence[key] = recordedMasteryEvidence{
		evidence: evidence,
		result:   result,
		sequence: m.evidenceSequence,
	}
	return result, nil
}

// GetMasteryEvidenceCounts returns distinct recorded attempts grouped by topic.
func (m *MemoryTracker) GetMasteryEvidenceCounts(ctx context.Context, learnerID LearnerID) ([]MasteryEvidenceCount, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if learnerID.String() == "" {
		return nil, fmt.Errorf("mastery evidence learner ID is required")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[string]MasteryEvidenceCount)
	for _, recorded := range m.evidence {
		if recorded.evidence.LearnerID.String() != learnerID.String() {
			continue
		}
		key := progressKey("", recorded.evidence.SyllabusID, recorded.evidence.TopicID)
		count := counts[key]
		count.SyllabusID = recorded.evidence.SyllabusID
		count.TopicID = recorded.evidence.TopicID
		count.Count++
		counts[key] = count
	}
	result := make([]MasteryEvidenceCount, 0, len(counts))
	for _, count := range counts {
		result = append(result, count)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SyllabusID == result[j].SyllabusID {
			return result[i].TopicID < result[j].TopicID
		}
		return result[i].SyllabusID < result[j].SyllabusID
	})
	return result, nil
}

// GetMasterySnapshot returns scores and evidence counts under one read lock.
func (m *MemoryTracker) GetMasterySnapshot(ctx context.Context, learnerID LearnerID) ([]MasterySnapshotItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if learnerID.String() == "" {
		return nil, fmt.Errorf("mastery snapshot learner ID is required")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make(map[string]MasterySnapshotItem)
	latestSequence := make(map[string]uint64)
	for _, progressItem := range m.items {
		if progressItem.UserID != learnerID.String() {
			continue
		}
		key := progressKey("", progressItem.SyllabusID, progressItem.TopicID)
		items[key] = MasterySnapshotItem{
			SyllabusID:   progressItem.SyllabusID,
			TopicID:      progressItem.TopicID,
			MasteryScore: progressItem.MasteryScore,
			MasteryKnown: true,
		}
	}
	for _, recorded := range m.evidence {
		if recorded.evidence.LearnerID.String() != learnerID.String() {
			continue
		}
		key := progressKey("", recorded.evidence.SyllabusID, recorded.evidence.TopicID)
		item := items[key]
		item.SyllabusID = recorded.evidence.SyllabusID
		item.TopicID = recorded.evidence.TopicID
		item.EvidenceCount++
		if latestSequence[key] < recorded.sequence {
			item.LatestEvidence = &MasteryEvidenceSummary{
				SourceKind: recorded.evidence.SourceKind,
				SourceID:   recorded.evidence.SourceID,
				Score:      recorded.evidence.Score,
			}
			latestSequence[key] = recorded.sequence
		}
		items[key] = item
	}

	result := make([]MasterySnapshotItem, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SyllabusID == result[j].SyllabusID {
			return result[i].TopicID < result[j].TopicID
		}
		return result[i].SyllabusID < result[j].SyllabusID
	})
	return result, nil
}

func validateMasteryEvidence(evidence MasteryEvidence) (MasteryEvidence, error) {
	evidence.AttemptID = strings.TrimSpace(evidence.AttemptID)
	evidence.SyllabusID = strings.TrimSpace(evidence.SyllabusID)
	evidence.TopicID = strings.TrimSpace(evidence.TopicID)
	evidence.SourceKind = strings.TrimSpace(evidence.SourceKind)
	evidence.SourceID = strings.TrimSpace(evidence.SourceID)
	evidence.SourceRevision = strings.TrimSpace(evidence.SourceRevision)
	switch {
	case evidence.AttemptID == "":
		return MasteryEvidence{}, fmt.Errorf("mastery evidence attempt ID is required")
	case len(evidence.AttemptID) > 256:
		return MasteryEvidence{}, fmt.Errorf("mastery evidence attempt ID must not exceed 256 bytes")
	case evidence.LearnerID.String() == "":
		return MasteryEvidence{}, fmt.Errorf("mastery evidence learner ID is required")
	case evidence.SyllabusID == "":
		return MasteryEvidence{}, fmt.Errorf("mastery evidence syllabus ID is required")
	case evidence.TopicID == "":
		return MasteryEvidence{}, fmt.Errorf("mastery evidence topic ID is required")
	case evidence.SourceKind == "":
		return MasteryEvidence{}, fmt.Errorf("mastery evidence source kind is required")
	case evidence.SourceID == "":
		return MasteryEvidence{}, fmt.Errorf("mastery evidence source ID is required")
	case evidence.SourceRevision == "":
		return MasteryEvidence{}, fmt.Errorf("mastery evidence source revision is required")
	case len(evidence.SourceRevision) > 256:
		return MasteryEvidence{}, fmt.Errorf("mastery evidence source revision must not exceed 256 bytes")
	case evidence.PayloadHash == [32]byte{}:
		return MasteryEvidence{}, fmt.Errorf("mastery evidence payload hash is required")
	case math.IsNaN(evidence.Score), math.IsInf(evidence.Score, 0), evidence.Score < 0, evidence.Score > 1:
		return MasteryEvidence{}, fmt.Errorf("mastery evidence score must be between 0 and 1")
	default:
		return evidence, nil
	}
}

func sameMasteryEvidence(left, right MasteryEvidence) bool {
	return left.AttemptID == right.AttemptID &&
		left.LearnerID.String() == right.LearnerID.String() &&
		left.SyllabusID == right.SyllabusID &&
		left.TopicID == right.TopicID &&
		left.SourceKind == right.SourceKind &&
		left.SourceID == right.SourceID &&
		left.SourceRevision == right.SourceRevision &&
		left.Score == right.Score &&
		left.PayloadHash == right.PayloadHash
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

func (m *MemoryTracker) GetMasteryForLearner(learnerID LearnerID, syllabusID, topicID string) (float64, error) {
	return m.GetMastery(learnerID.String(), syllabusID, topicID)
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

func (m *MemoryTracker) GetAllProgressForLearner(learnerID LearnerID) ([]ProgressItem, error) {
	return m.GetAllProgress(learnerID.String())
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

func (m *MemoryTracker) GetDueReviewsForLearner(learnerID LearnerID) ([]ProgressItem, error) {
	return m.GetDueReviews(learnerID.String())
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

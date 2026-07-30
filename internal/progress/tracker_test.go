// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"
)

func TestTracker_UpdateMastery(t *testing.T) {
	tracker := NewMemoryTracker()

	err := tracker.UpdateMastery("user1", "kssm-form1", "algebra-linear-eq", 0.8)
	if err != nil {
		t.Fatalf("UpdateMastery() error = %v", err)
	}

	score, err := tracker.GetMastery("user1", "kssm-form1", "algebra-linear-eq")
	if err != nil {
		t.Fatalf("GetMastery() error = %v", err)
	}
	if score < 0.7 || score > 0.9 {
		t.Errorf("expected score in [0.7, 0.9], got %f", score)
	}
}

func TestLearnerTracker_IsolatesInternalLearnerIDs(t *testing.T) {
	tracker := NewMemoryTracker()
	telegramLearner, err := NewLearnerID("9f832956-e935-41a3-b984-df499398a1b8")
	if err != nil {
		t.Fatal(err)
	}
	slackLearner, err := NewLearnerID("232b2439-61d1-4b41-9468-0ddaa6295b84")
	if err != nil {
		t.Fatal(err)
	}

	if err := tracker.UpdateMasteryForLearner(telegramLearner, "syllabus", "topic", 0.9); err != nil {
		t.Fatal(err)
	}
	if err := tracker.UpdateMasteryForLearner(slackLearner, "syllabus", "topic", 0.2); err != nil {
		t.Fatal(err)
	}

	telegramScore, _ := tracker.GetMasteryForLearner(telegramLearner, "syllabus", "topic")
	slackScore, _ := tracker.GetMasteryForLearner(slackLearner, "syllabus", "topic")
	if telegramScore != 0.9 || slackScore != 0.2 {
		t.Fatalf("scores = %v, %v, want 0.9 and 0.2", telegramScore, slackScore)
	}
}

func TestNewLearnerIDRejectsEmptyValue(t *testing.T) {
	if _, err := NewLearnerID(" "); err == nil {
		t.Fatal("NewLearnerID(empty) error = nil, want error")
	}
}

func TestMemoryTracker_RecordMasteryEvidenceIsIdempotent(t *testing.T) {
	tracker := NewMemoryTracker()
	learnerID, err := NewLearnerID("learner-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.UpdateMasteryForLearner(learnerID, "syllabus", "topic", 0.8); err != nil {
		t.Fatal(err)
	}
	evidence := MasteryEvidence{
		AttemptID:      "message-42/Q1",
		LearnerID:      learnerID,
		SyllabusID:     "syllabus",
		TopicID:        "topic",
		SourceKind:     "oss_assessment",
		SourceID:       "Q1",
		SourceRevision: "sha256:fixture",
		Score:          0,
		PayloadHash:    sha256.Sum256([]byte("wrong answer")),
	}

	first, err := tracker.RecordMasteryEvidence(context.Background(), evidence)
	if err != nil {
		t.Fatalf("first RecordMasteryEvidence() error = %v", err)
	}
	replay, err := tracker.RecordMasteryEvidence(context.Background(), evidence)
	if err != nil {
		t.Fatalf("replayed RecordMasteryEvidence() error = %v", err)
	}

	if !first.Applied || replay.Applied {
		t.Fatalf("applied flags = %v, %v, want true then false", first.Applied, replay.Applied)
	}
	if replay.MasteryBefore != first.MasteryBefore || replay.MasteryAfter != first.MasteryAfter {
		t.Fatalf("replay result = %#v, want original %#v", replay, first)
	}
	score, err := tracker.GetMasteryForLearner(learnerID, "syllabus", "topic")
	if err != nil {
		t.Fatal(err)
	}
	if score != first.MasteryAfter {
		t.Fatalf("mastery after replay = %v, want %v", score, first.MasteryAfter)
	}
	counts, err := tracker.GetMasteryEvidenceCounts(context.Background(), learnerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 1 || counts[0].SyllabusID != "syllabus" || counts[0].TopicID != "topic" || counts[0].Count != 1 {
		t.Fatalf("evidence counts = %#v, want one distinct topic attempt", counts)
	}
}

func TestMemoryTracker_RecordMasteryEvidenceRejectsAttemptIDCollision(t *testing.T) {
	tracker := NewMemoryTracker()
	learnerID, err := NewLearnerID("learner-1")
	if err != nil {
		t.Fatal(err)
	}
	evidence := MasteryEvidence{
		AttemptID:      "message-42/Q1",
		LearnerID:      learnerID,
		SyllabusID:     "syllabus",
		TopicID:        "topic",
		SourceKind:     "oss_assessment",
		SourceID:       "Q1",
		SourceRevision: "sha256:fixture",
		Score:          1,
		PayloadHash:    sha256.Sum256([]byte("correct answer")),
	}
	if _, err := tracker.RecordMasteryEvidence(context.Background(), evidence); err != nil {
		t.Fatal(err)
	}
	evidence.PayloadHash = sha256.Sum256([]byte("different answer"))

	if _, err := tracker.RecordMasteryEvidence(context.Background(), evidence); !errors.Is(err, ErrMasteryEvidenceConflict) {
		t.Fatalf("RecordMasteryEvidence() error = %v, want ErrMasteryEvidenceConflict", err)
	}
	score, err := tracker.GetMasteryForLearner(learnerID, "syllabus", "topic")
	if err != nil {
		t.Fatal(err)
	}
	if score != 1 {
		t.Fatalf("mastery after collision = %v, want unchanged 1", score)
	}
}

func TestMemoryTracker_RecordMasteryEvidenceRejectsSourceRevisionCollision(t *testing.T) {
	tracker := NewMemoryTracker()
	learnerID, err := NewLearnerID("learner-1")
	if err != nil {
		t.Fatal(err)
	}
	evidence := MasteryEvidence{
		AttemptID:      "message-42/Q1",
		LearnerID:      learnerID,
		SyllabusID:     "syllabus",
		TopicID:        "topic",
		SourceKind:     "oss_assessment",
		SourceID:       "Q1",
		SourceRevision: "sha256:first",
		Score:          1,
		PayloadHash:    sha256.Sum256([]byte("correct answer")),
	}
	if _, err := tracker.RecordMasteryEvidence(context.Background(), evidence); err != nil {
		t.Fatal(err)
	}
	evidence.SourceRevision = "sha256:changed"

	if _, err := tracker.RecordMasteryEvidence(context.Background(), evidence); !errors.Is(err, ErrMasteryEvidenceConflict) {
		t.Fatalf("RecordMasteryEvidence() error = %v, want ErrMasteryEvidenceConflict", err)
	}
}

func TestMemoryTracker_GetMasterySnapshotStaysCoherentDuringEvidenceWrites(t *testing.T) {
	tracker := NewMemoryTracker()
	learnerID, err := NewLearnerID("learner-1")
	if err != nil {
		t.Fatal(err)
	}

	const attempts = 500
	expectedScores := make([]float64, attempts+1)
	for attempt := 1; attempt <= attempts; attempt++ {
		score := float64(attempt % 2)
		if attempt == 1 {
			expectedScores[attempt] = score
			continue
		}
		expectedScores[attempt] = expectedScores[attempt-1]*0.7 + score*0.3
	}

	start := make(chan struct{})
	writerDone := make(chan struct{})
	writeErr := make(chan error, 1)
	go func() {
		defer close(writerDone)
		<-start
		for attempt := 1; attempt <= attempts; attempt++ {
			attemptID := fmt.Sprintf("attempt-%d", attempt)
			_, err := tracker.RecordMasteryEvidence(context.Background(), MasteryEvidence{
				AttemptID:      attemptID,
				LearnerID:      learnerID,
				SyllabusID:     "syllabus",
				TopicID:        "topic",
				SourceKind:     "oss_assessment",
				SourceID:       "question",
				SourceRevision: "sha256:fixture",
				Score:          float64(attempt % 2),
				PayloadHash:    sha256.Sum256([]byte(attemptID)),
			})
			if err != nil {
				writeErr <- err
				return
			}
		}
	}()

	close(start)
	for {
		snapshot, err := tracker.GetMasterySnapshot(context.Background(), learnerID)
		if err != nil {
			t.Fatalf("GetMasterySnapshot() error = %v", err)
		}
		if len(snapshot) > 1 {
			t.Fatalf("snapshot = %#v, want at most one topic", snapshot)
		}
		if len(snapshot) == 1 {
			item := snapshot[0]
			if !item.MasteryKnown {
				t.Fatalf("snapshot item = %#v, recorded evidence must have known mastery", item)
			}
			if item.EvidenceCount < 1 || item.EvidenceCount > attempts {
				t.Fatalf("snapshot evidence count = %d, want 1..%d", item.EvidenceCount, attempts)
			}
			wantScore := expectedScores[item.EvidenceCount]
			if math.Abs(item.MasteryScore-wantScore) > 1e-12 {
				t.Fatalf(
					"snapshot mixed score %v with evidence count %d; want score %v",
					item.MasteryScore,
					item.EvidenceCount,
					wantScore,
				)
			}
		}

		select {
		case <-writerDone:
			if len(writeErr) != 0 {
				t.Fatal(<-writeErr)
			}
			final, err := tracker.GetMasterySnapshot(context.Background(), learnerID)
			if err != nil {
				t.Fatal(err)
			}
			if len(final) != 1 || final[0].EvidenceCount != attempts {
				t.Fatalf("final snapshot = %#v, want %d attempts", final, attempts)
			}
			return
		default:
		}
	}
}

func TestMemoryTracker_GetMasterySnapshotIncludesProgressWithoutEvidence(t *testing.T) {
	tracker := NewMemoryTracker()
	learnerID, err := NewLearnerID("learner-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.UpdateMasteryForLearner(learnerID, "syllabus", "topic", 0.4); err != nil {
		t.Fatal(err)
	}

	snapshot, err := tracker.GetMasterySnapshot(context.Background(), learnerID)
	if err != nil {
		t.Fatalf("GetMasterySnapshot() error = %v", err)
	}
	if len(snapshot) != 1 {
		t.Fatalf("snapshot = %#v, want one item", snapshot)
	}
	item := snapshot[0]
	if item.SyllabusID != "syllabus" ||
		item.TopicID != "topic" ||
		item.MasteryScore != 0.4 ||
		!item.MasteryKnown ||
		item.EvidenceCount != 0 {
		t.Fatalf("snapshot item = %#v, want known 0.4 mastery without evidence", item)
	}
}

func TestTracker_UpdateMastery_WeightedBlend(t *testing.T) {
	tracker := NewMemoryTracker()

	// First update: seeds score directly.
	if err := tracker.UpdateMastery("user1", "syl1", "topic1", 0.6); err != nil {
		t.Fatal(err)
	}

	// Second update: weighted blend = existing*0.7 + delta*0.3
	if err := tracker.UpdateMastery("user1", "syl1", "topic1", 1.0); err != nil {
		t.Fatal(err)
	}

	score, err := tracker.GetMastery("user1", "syl1", "topic1")
	if err != nil {
		t.Fatal(err)
	}

	// Expected: 0.6*0.7 + 1.0*0.3 = 0.42 + 0.30 = 0.72
	if score < 0.70 || score > 0.74 {
		t.Errorf("expected weighted blend ~0.72, got %f", score)
	}
}

func TestTracker_UpdateMastery_Clamp(t *testing.T) {
	tracker := NewMemoryTracker()

	// Delta > 1.0 should be clamped to 1.0.
	if err := tracker.UpdateMastery("user1", "syl1", "topic1", 1.5); err != nil {
		t.Fatal(err)
	}

	score, err := tracker.GetMastery("user1", "syl1", "topic1")
	if err != nil {
		t.Fatal(err)
	}
	if score > 1.0 {
		t.Errorf("expected clamped score <= 1.0, got %f", score)
	}

	// Delta < 0.0 should be clamped to 0.0.
	tracker2 := NewMemoryTracker()
	if err := tracker2.UpdateMastery("user1", "syl1", "topic1", -0.5); err != nil {
		t.Fatal(err)
	}

	score2, err := tracker2.GetMastery("user1", "syl1", "topic1")
	if err != nil {
		t.Fatal(err)
	}
	if score2 < 0.0 {
		t.Errorf("expected clamped score >= 0.0, got %f", score2)
	}
}

func TestTracker_GetAllProgress(t *testing.T) {
	tracker := NewMemoryTracker()

	if err := tracker.UpdateMastery("user1", "syl1", "topic1", 0.5); err != nil {
		t.Fatal(err)
	}
	if err := tracker.UpdateMastery("user1", "syl1", "topic2", 0.9); err != nil {
		t.Fatal(err)
	}

	items, err := tracker.GetAllProgress("user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 progress items, got %d", len(items))
	}
}

func TestTracker_GetAllProgress_Empty(t *testing.T) {
	tracker := NewMemoryTracker()

	items, err := tracker.GetAllProgress("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 progress items for unknown user, got %d", len(items))
	}
}

func TestTracker_GetDueReviews(t *testing.T) {
	tracker := NewMemoryTracker()

	// Create an item with a past review date.
	if err := tracker.UpdateMastery("user1", "syl1", "topic-past", 0.5); err != nil {
		t.Fatal(err)
	}
	// Manually set NextReviewAt to the past.
	tracker.setNextReviewAt("user1", "syl1", "topic-past", time.Now().Add(-1*time.Hour))

	// Create an item with a future review date.
	if err := tracker.UpdateMastery("user1", "syl1", "topic-future", 0.5); err != nil {
		t.Fatal(err)
	}
	tracker.setNextReviewAt("user1", "syl1", "topic-future", time.Now().Add(24*time.Hour))

	due, err := tracker.GetDueReviews("user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due review, got %d", len(due))
	}
	if due[0].TopicID != "topic-past" {
		t.Errorf("expected due topic 'topic-past', got %q", due[0].TopicID)
	}
}

func TestMasteryThreshold(t *testing.T) {
	tests := []struct {
		score    float64
		expected bool
	}{
		{0.74, false},
		{0.75, true},
		{0.80, true},
		{0.0, false},
		{1.0, true},
	}

	for _, tc := range tests {
		got := IsMastered(tc.score)
		if got != tc.expected {
			t.Errorf("IsMastered(%f) = %v, want %v", tc.score, got, tc.expected)
		}
	}
}

func TestTracker_GetMastery_NotFound(t *testing.T) {
	tracker := NewMemoryTracker()

	score, err := tracker.GetMastery("user1", "syl1", "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if score != 0 {
		t.Errorf("expected 0 for unknown topic, got %f", score)
	}
}

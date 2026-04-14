// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
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

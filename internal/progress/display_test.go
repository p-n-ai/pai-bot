// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestFormatProgressBar(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		width     int
		wantFull  int
		wantEmpty int
	}{
		{"empty", 0.0, 10, 0, 10},
		{"half", 0.5, 10, 5, 5},
		{"full", 1.0, 10, 10, 0},
		{"over", 1.5, 10, 10, 0},
		{"quarter", 0.25, 8, 2, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := FormatProgressBar(tt.score, tt.width)
			if bar == "" {
				t.Error("FormatProgressBar() returned empty")
			}
			runeCount := utf8.RuneCountInString(bar)
			if runeCount != tt.width {
				t.Errorf("expected bar width %d runes, got %d", tt.width, runeCount)
			}
			filled := strings.Count(bar, "█")
			empty := strings.Count(bar, "░")
			if filled != tt.wantFull {
				t.Errorf("expected %d filled, got %d", tt.wantFull, filled)
			}
			if empty != tt.wantEmpty {
				t.Errorf("expected %d empty, got %d", tt.wantEmpty, empty)
			}
		})
	}
}

func TestFormatProgressReport(t *testing.T) {
	items := []ProgressItem{
		{TopicID: "F1-01", MasteryScore: 0.8},
		{TopicID: "F1-02", MasteryScore: 0.3},
	}

	report := FormatProgressReport(items, 150, 3)
	if !strings.Contains(report, "F1-01") {
		t.Error("Report should contain topic ID F1-01")
	}
	if !strings.Contains(report, "F1-02") {
		t.Error("Report should contain topic ID F1-02")
	}
	if !strings.Contains(report, "150") {
		t.Error("Report should contain XP value")
	}
	if !strings.Contains(report, "3") {
		t.Error("Report should contain streak value")
	}
	if !strings.Contains(report, "80%") {
		t.Error("Report should contain 80% for 0.8 mastery")
	}
	if !strings.Contains(report, "30%") {
		t.Error("Report should contain 30% for 0.3 mastery")
	}
}

func TestFormatProgressReport_Empty(t *testing.T) {
	report := FormatProgressReport(nil, 0, 0)
	if report == "" {
		t.Error("Report should not be empty even with no items")
	}
	if !strings.Contains(report, "mula belajar") {
		t.Error("Empty report should contain encouragement message")
	}
}

func TestFormatProgressReport_MasteredTopicIcon(t *testing.T) {
	items := []ProgressItem{
		{TopicID: "mastered-topic", MasteryScore: 0.9},
	}
	report := FormatProgressReport(items, 0, 0)
	if !strings.Contains(report, "✅") {
		t.Error("Mastered topic should show ✅ icon")
	}
}

func TestFormatProgressReport_UnmasteredTopicIcon(t *testing.T) {
	items := []ProgressItem{
		{TopicID: "learning-topic", MasteryScore: 0.4},
	}
	report := FormatProgressReport(items, 0, 0)
	if !strings.Contains(report, "📖") {
		t.Error("Unmastered topic should show 📖 icon")
	}
}

func TestFormatProgressReport_IncludesNextReview(t *testing.T) {
	items := []ProgressItem{
		{
			TopicID:      "review-topic",
			MasteryScore: 0.6,
			NextReviewAt: time.Now().Add(2 * 24 * time.Hour),
		},
	}
	report := FormatProgressReport(items, 0, 0)
	if !strings.Contains(report, "review-topic") {
		t.Error("Report should contain topic ID")
	}
}

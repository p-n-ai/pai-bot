package agent

import (
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestAdaptiveDepthBlock_Beginner(t *testing.T) {
	block := adaptiveDepthBlock(0.2, nil)

	if block == "" {
		t.Fatal("expected non-empty block for beginner")
	}
	mustContain(t, block, "simple")
	mustContain(t, block, "example")
}

func TestAdaptiveDepthBlock_Developing(t *testing.T) {
	block := adaptiveDepthBlock(0.45, nil)

	mustContain(t, block, "formal notation")
}

func TestAdaptiveDepthBlock_Proficient(t *testing.T) {
	block := adaptiveDepthBlock(0.75, nil)

	mustContain(t, block, "concise")
	mustContain(t, block, "edge case")
}

func TestAdaptiveDepthBlock_ZeroMastery(t *testing.T) {
	block := adaptiveDepthBlock(0.0, nil)
	mustContain(t, block, "simple")
}

func TestAdaptiveDepthBlock_WithProgressContext(t *testing.T) {
	items := []progress.ProgressItem{
		{TopicID: "F1-05", MasteryScore: 0.85},
		{TopicID: "F1-06", MasteryScore: 0.45},
		{TopicID: "F1-07", MasteryScore: 0.15},
	}

	block := adaptiveDepthBlock(0.45, items)

	mustContain(t, block, "F1-05") // mastered
	mustContain(t, block, "F1-06") // working on
	mustContain(t, block, "F1-07") // struggles
}

func TestAdaptiveDepthBlock_NoProgressItems(t *testing.T) {
	block := adaptiveDepthBlock(0.3, nil)

	// Should still produce depth instructions, just no progress context.
	if block == "" {
		t.Fatal("expected non-empty block even without progress items")
	}
}

func TestProgressContextSummary(t *testing.T) {
	tests := []struct {
		name  string
		items []progress.ProgressItem
		want  struct {
			mastered  int
			working   int
			struggles int
		}
	}{
		{
			name: "mixed",
			items: []progress.ProgressItem{
				{TopicID: "A", MasteryScore: 0.9},
				{TopicID: "B", MasteryScore: 0.5},
				{TopicID: "C", MasteryScore: 0.1},
			},
			want: struct {
				mastered  int
				working   int
				struggles int
			}{1, 1, 1},
		},
		{
			name: "all mastered",
			items: []progress.ProgressItem{
				{TopicID: "A", MasteryScore: 0.8},
				{TopicID: "B", MasteryScore: 0.9},
			},
			want: struct {
				mastered  int
				working   int
				struggles int
			}{2, 0, 0},
		},
		{
			name:  "empty",
			items: nil,
			want: struct {
				mastered  int
				working   int
				struggles int
			}{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mastered, working, struggles := categorizeProgress(tt.items)
			if len(mastered) != tt.want.mastered {
				t.Errorf("mastered = %d, want %d", len(mastered), tt.want.mastered)
			}
			if len(working) != tt.want.working {
				t.Errorf("working = %d, want %d", len(working), tt.want.working)
			}
			if len(struggles) != tt.want.struggles {
				t.Errorf("struggles = %d, want %d", len(struggles), tt.want.struggles)
			}
		})
	}
}

func mustContain(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

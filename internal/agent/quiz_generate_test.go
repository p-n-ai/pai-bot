package agent

import "testing"

func TestSelectExemplars_FiltersByDifficulty(t *testing.T) {
	questions := []QuizQuestion{
		{ID: "Q1", Difficulty: "easy"},
		{ID: "Q2", Difficulty: "medium"},
		{ID: "Q3", Difficulty: "medium"},
		{ID: "Q4", Difficulty: "hard"},
		{ID: "Q5", Difficulty: "medium"},
	}
	exemplars := selectExemplars(questions, "medium")
	if len(exemplars) < 2 || len(exemplars) > 3 {
		t.Fatalf("len(exemplars) = %d, want 2-3", len(exemplars))
	}
	for _, ex := range exemplars {
		if normalizeQuizIntensity(ex.Difficulty) != "medium" {
			t.Errorf("exemplar %s difficulty %q, want medium", ex.ID, ex.Difficulty)
		}
	}
}

func TestSelectExemplars_FallbackWhenFewerThanTwo(t *testing.T) {
	questions := []QuizQuestion{
		{ID: "Q1", Difficulty: "easy"},
		{ID: "Q2", Difficulty: "medium"},
		{ID: "Q3", Difficulty: "hard"},
	}
	exemplars := selectExemplars(questions, "hard")
	if len(exemplars) < 2 {
		t.Fatalf("len(exemplars) = %d, want >= 2", len(exemplars))
	}
}

func TestSelectExemplars_EmptyPool(t *testing.T) {
	exemplars := selectExemplars(nil, "medium")
	if len(exemplars) != 0 {
		t.Fatalf("len(exemplars) = %d, want 0", len(exemplars))
	}
}

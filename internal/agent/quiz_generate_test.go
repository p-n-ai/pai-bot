// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

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

func TestBuildExamMimicryPrompt_ContainsRequiredParts(t *testing.T) {
	exemplars := []QuizQuestion{
		{ID: "Q1", Text: "Solve 2x = 6", Difficulty: "medium", AnswerType: "exact", Answer: "3"},
	}
	prompt := buildExamMimicryPrompt(examMimicryPromptInput{
		N: 3, TopicName: "Linear Equations", TopicID: "F1-02",
		SyllabusID: "kssm-f1", Intensity: "medium",
		TeachingNotes: "Focus on isolating the variable.",
		Exemplars:     exemplars,
	})
	for _, want := range []string{"Linear Equations", "F1-02", "medium", "Focus on isolating", "Solve 2x = 6"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestParseGeneratedQuestions_ValidJSON(t *testing.T) {
	raw := `[{"text":"Solve: $3x + 5 = 20$","difficulty":"medium","answer_type":"exact","answer":"5","working":"3x=15, x=5","hints":[{"level":1,"text":"Subtract 5"},{"level":2,"text":"Divide by 3"}],"distractors":[]}]`
	questions, err := parseGeneratedQuestions([]byte(raw))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("len = %d, want 1", len(questions))
	}
	if questions[0].Answer != "5" {
		t.Errorf("Answer = %q, want 5", questions[0].Answer)
	}
	if questions[0].ID == "" {
		t.Error("expected non-empty generated ID")
	}
	if len(questions[0].Hints) != 2 {
		t.Errorf("len(Hints) = %d, want 2", len(questions[0].Hints))
	}
}

func TestParseGeneratedQuestions_InvalidJSON(t *testing.T) {
	_, err := parseGeneratedQuestions([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseGeneratedQuestions_EmptyArray(t *testing.T) {
	questions, err := parseGeneratedQuestions([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(questions) != 0 {
		t.Fatalf("len = %d, want 0", len(questions))
	}
}

func TestQuizQuestionGenerator_Generate_Success(t *testing.T) {
	fakeResp := `[{"text":"What is 2+2?","difficulty":"easy","answer_type":"exact","answer":"4","working":"2+2=4","hints":[{"level":1,"text":"count"}],"distractors":[]}]`
	mockAI := ai.NewMockProvider(fakeResp)
	router := ai.NewRouter()
	router.Register("fake", mockAI)
	gen := quizQuestionGenerator{aiRouter: router}

	questions, err := gen.Generate(context.Background(), quizGenerateInput{
		TopicID: "F1-01", TopicName: "Basics", SyllabusID: "kssm-f1",
		Intensity: "easy", N: 3, TeachingNotes: "Basic arithmetic.",
		AllQuestions: []QuizQuestion{{ID: "Q1", Difficulty: "easy", AnswerType: "exact", Answer: "1"}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("len = %d, want 1", len(questions))
	}
	if questions[0].Answer != "4" {
		t.Errorf("Answer = %q, want 4", questions[0].Answer)
	}
}

func TestQuizQuestionGenerator_Generate_AIFailure(t *testing.T) {
	mockAI := ai.NewMockProvider("invalid json")
	router := ai.NewRouter()
	router.Register("fake", mockAI)
	gen := quizQuestionGenerator{aiRouter: router}

	_, err := gen.Generate(context.Background(), quizGenerateInput{
		TopicID: "F1-01", TopicName: "Basics", SyllabusID: "kssm-f1",
		Intensity: "easy", N: 3,
		AllQuestions: []QuizQuestion{{ID: "Q1", Difficulty: "easy", AnswerType: "exact", Answer: "1"}},
	})
	if err == nil {
		t.Fatal("expected error on invalid AI response")
	}
}

func TestQuizQuestionGenerator_Generate_NoExemplars(t *testing.T) {
	router := ai.NewRouter()
	gen := quizQuestionGenerator{aiRouter: router}

	_, err := gen.Generate(context.Background(), quizGenerateInput{
		TopicID: "F1-01", TopicName: "Basics", SyllabusID: "kssm-f1",
		Intensity: "easy", N: 3,
		AllQuestions: nil,
	})
	if err == nil {
		t.Fatal("expected error with no exemplars")
	}
}

package agent

import (
	"fmt"
	"testing"
)

func TestQuizSession_SubmitAnswer_ExactAdvances(t *testing.T) {
	session := NewQuizSession("user-1", "F1-01", []QuizQuestion{
		{
			ID:         "Q1",
			Text:       "What is 3x when x = 2?",
			AnswerType: "exact",
			Answer:     "6",
			Marks:      1,
		},
		{
			ID:         "Q2",
			Text:       "What is 5 + 5?",
			AnswerType: "exact",
			Answer:     "10",
			Marks:      1,
		},
	})

	question, ok := session.NextQuestion()
	if !ok || question.ID != "Q1" {
		t.Fatalf("NextQuestion() = %#v, %v, want Q1, true", question, ok)
	}

	result := session.SubmitAnswer("6")
	if !result.Correct {
		t.Fatal("SubmitAnswer() should mark exact answer as correct")
	}
	if session.CurrentIndex != 1 {
		t.Fatalf("CurrentIndex = %d, want 1", session.CurrentIndex)
	}
	if session.CorrectAnswers != 1 {
		t.Fatalf("CorrectAnswers = %d, want 1", session.CorrectAnswers)
	}
}

func TestQuizSession_SubmitAnswer_FreeTextContainsExpectedValue(t *testing.T) {
	session := NewQuizSession("user-1", "F1-01", []QuizQuestion{
		{
			ID:         "Q1",
			Text:       "Is the value fixed or varied?",
			AnswerType: "free_text",
			Answer:     "varied value",
			Marks:      2,
		},
	})

	result := session.SubmitAnswer("It is a varied value because it can change.")
	if !result.Correct {
		t.Fatal("SubmitAnswer() should accept free-text containing expected value")
	}
	if !session.IsComplete() {
		t.Fatal("session should be complete after the only question")
	}
}

func TestQuizSession_SubmitAnswer_FreeTextRejectsNegatedAnswer(t *testing.T) {
	session := NewQuizSession("user-1", "F1-01", []QuizQuestion{
		{
			ID:         "Q1",
			Text:       "Is the value fixed or varied?",
			AnswerType: "free_text",
			Answer:     "varied",
			Marks:      2,
		},
	})

	result := session.SubmitAnswer("It is not varied.")
	if result.Correct {
		t.Fatal("SubmitAnswer() should reject negated free-text answers")
	}
}

func TestQuizSession_SubmitAnswer_FreeTextRejectsLongerExpressionSuperstring(t *testing.T) {
	session := NewQuizSession("user-1", "F1-01", []QuizQuestion{
		{
			ID:         "Q1",
			Text:       "Substitute x = 4 into x + 3 = 7.",
			AnswerType: "free_text",
			Answer:     "4+3=7",
			Marks:      2,
		},
	})

	result := session.SubmitAnswer("Because 4+3=70, it is correct.")
	if result.Correct {
		t.Fatal("SubmitAnswer() should reject superstrings of the expected expression")
	}
}

func TestQuizSession_SubmitAnswer_StructuredAnswersAllowEquivalentFormatting(t *testing.T) {
	tests := []struct {
		name       string
		answerType string
		expected   string
		actual     string
	}{
		{
			name:       "indices prompt accepts unlabeled ordered parts",
			answerType: "exact",
			expected:   "5^4; base = 5; index = 4",
			actual:     "5^4, 5, 4",
		},
		{
			name:       "gradient prompt accepts symbolic labels",
			answerType: "exact",
			expected:   "gradient = 3; y-intercept = -4",
			actual:     "m=3, c=-4",
		},
		{
			name:       "subjective free text accepts equivalent expression formatting",
			answerType: "free_text",
			expected:   "(a) 1500x + 700y\n(b) 800x + 1600y",
			actual:     "1500x+700y and 800x+1600y",
		},
		{
			name:       "multi part exact accepts answers on separate lines",
			answerType: "exact",
			expected:   "(i) x + 5  (ii) 11",
			actual:     "x + 5\n11",
		},
		{
			name:       "ordinal range prompt accepts ordered lines without labels",
			answerType: "exact",
			expected:   "(i) -1 <= x <= 3, (ii) 1 <= x < 3, (iii) -1 < x < 3",
			actual:     "-1 <= x <= 3\n1 <= x < 3\n-1 < x < 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewQuizSession("user-1", "F1-01", []QuizQuestion{
				{
					ID:         "Q1",
					Text:       "Structured answer",
					AnswerType: tt.answerType,
					Answer:     tt.expected,
					Marks:      1,
				},
			})

			result := session.SubmitAnswer(tt.actual)
			if !result.Correct {
				t.Fatalf("SubmitAnswer() should accept %q for %q", tt.actual, tt.expected)
			}
		})
	}
}

func TestQuizSession_SubmitAnswer_StructuredAnswersRejectContradictoryExtraParts(t *testing.T) {
	session := NewQuizSession("user-1", "F3-09", []QuizQuestion{
		{
			ID:         "Q1",
			Text:       "State the gradient and y-intercept.",
			AnswerType: "exact",
			Answer:     "gradient = 3; y-intercept = -4",
			Marks:      2,
		},
	})

	result := session.SubmitAnswer("gradient = 0, gradient = 3, y-intercept = -4")
	if result.Correct {
		t.Fatal("SubmitAnswer() should reject structured answers with contradictory extra parts")
	}
}

func TestQuizSession_SubmitAnswer_StructuredAnswersRejectWrongAssignmentLabels(t *testing.T) {
	session := NewQuizSession("user-1", "F3-09", []QuizQuestion{
		{
			ID:         "Q1",
			Text:       "State the gradient and y-intercept.",
			AnswerType: "exact",
			Answer:     "gradient = 3; y-intercept = -4",
			Marks:      2,
		},
	})

	result := session.SubmitAnswer("x=3, y=-4")
	if result.Correct {
		t.Fatal("SubmitAnswer() should reject unrelated assignment labels for structured answers")
	}
}

func TestQuizSession_AppendQuestions(t *testing.T) {
	session := NewQuizSession("user-1", "F1-01", []QuizQuestion{
		{ID: "Q1", Text: "Q1", AnswerType: "exact", Answer: "1"},
	})
	extra := []QuizQuestion{
		{ID: "gen-1", Text: "Gen1", AnswerType: "exact", Answer: "2"},
		{ID: "gen-2", Text: "Gen2", AnswerType: "exact", Answer: "3"},
	}
	session.AppendQuestions(extra)
	if len(session.Questions) != 3 {
		t.Fatalf("len(Questions) = %d, want 3", len(session.Questions))
	}
}

func TestQuizSession_AppendQuestions_RespectsMaxCap(t *testing.T) {
	initial := make([]QuizQuestion, 8)
	for i := range initial {
		initial[i] = QuizQuestion{ID: fmt.Sprintf("Q%d", i), AnswerType: "exact", Answer: "x"}
	}
	session := NewQuizSession("user-1", "F1-01", initial)
	extra := make([]QuizQuestion, 5)
	for i := range extra {
		extra[i] = QuizQuestion{ID: fmt.Sprintf("gen-%d", i), AnswerType: "exact", Answer: "y"}
	}
	session.AppendQuestions(extra)
	if len(session.Questions) != QuizMaxQuestions {
		t.Fatalf("len(Questions) = %d, want %d", len(session.Questions), QuizMaxQuestions)
	}
}

func TestQuizSession_SubmitAnswer_WrongKeepsCurrentQuestion(t *testing.T) {
	session := NewQuizSession("user-1", "F1-01", []QuizQuestion{
		{
			ID:         "Q1",
			Text:       "What is 3x when x = 2?",
			AnswerType: "exact",
			Answer:     "6",
			Hints:      []QuizHint{{Level: 1, Text: "Substitute x with 2 first."}},
			Marks:      1,
		},
	})

	result := session.SubmitAnswer("7")
	if result.Correct {
		t.Fatal("SubmitAnswer() should mark wrong answer as incorrect")
	}
	if result.Hint != "Substitute x with 2 first." {
		t.Fatalf("Hint = %q, want first hint", result.Hint)
	}
	if session.CurrentIndex != 0 {
		t.Fatalf("CurrentIndex = %d, want 0", session.CurrentIndex)
	}
}

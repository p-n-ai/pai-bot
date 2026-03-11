package agent

import "testing"

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

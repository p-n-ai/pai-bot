package agent

import (
	"sort"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

// QuizQuestion is the agent-facing question shape used by the quiz runtime.
type QuizQuestion struct {
	ID                string
	Text              string
	Difficulty        string
	LearningObjective string
	AnswerType        string
	Answer            string
	Working           string
	Marks             int
	Hints             []QuizHint
	Distractors       []QuizDistractor
}

// QuizHint is a progressive hint for a question.
type QuizHint struct {
	Level int
	Text  string
}

// QuizDistractor is an incorrect option with feedback.
type QuizDistractor struct {
	Value    string
	Feedback string
}

// QuizAnswerResult is the deterministic grading result for a question.
type QuizAnswerResult struct {
	Correct          bool
	Feedback         string
	Hint             string
	ExpectedAnswer   string
	Explanation      string
	QuestionComplete bool
}

// QuizSummary is the final quiz result snapshot.
type QuizSummary struct {
	TopicID         string
	TotalQuestions  int
	CorrectAnswers  int
	ScorePercentage int
}

// QuizSession holds deterministic quiz progress.
type QuizSession struct {
	UserID         string
	TopicID        string
	Intensity      string
	Questions      []QuizQuestion
	CurrentIndex   int
	CorrectAnswers int
}

// NewQuizSession creates a new in-memory quiz session.
func NewQuizSession(userID, topicID string, questions []QuizQuestion) *QuizSession {
	return &QuizSession{
		UserID:    userID,
		TopicID:   topicID,
		Questions: questions,
	}
}

// NextQuestion returns the current question without advancing.
func (s *QuizSession) NextQuestion() (QuizQuestion, bool) {
	if s == nil || s.CurrentIndex < 0 || s.CurrentIndex >= len(s.Questions) {
		return QuizQuestion{}, false
	}
	return s.Questions[s.CurrentIndex], true
}

// IsComplete returns true when all questions are answered correctly or exhausted.
func (s *QuizSession) IsComplete() bool {
	return s == nil || s.CurrentIndex >= len(s.Questions)
}

// SubmitAnswer grades the current question and advances on correct answers.
func (s *QuizSession) SubmitAnswer(answer string) QuizAnswerResult {
	question, ok := s.NextQuestion()
	if !ok {
		return QuizAnswerResult{}
	}

	correct := gradeQuizAnswer(question, answer)
	result := QuizAnswerResult{
		Correct:          correct,
		ExpectedAnswer:   question.Answer,
		Explanation:      question.Working,
		QuestionComplete: correct,
	}

	if correct {
		result.Feedback = "Correct."
		s.CorrectAnswers++
		s.CurrentIndex++
		return result
	}

	result.Feedback = matchingDistractorFeedback(question, answer)
	if result.Feedback == "" {
		result.Feedback = "Not quite."
	}
	if len(question.Hints) > 0 {
		hints := append([]QuizHint(nil), question.Hints...)
		sort.Slice(hints, func(i, j int) bool {
			return hints[i].Level < hints[j].Level
		})
		result.Hint = hints[0].Text
	}
	return result
}

// Summary returns the current score snapshot.
func (s *QuizSession) Summary() QuizSummary {
	total := len(s.Questions)
	score := 0
	if total > 0 {
		score = (s.CorrectAnswers * 100) / total
	}
	return QuizSummary{
		TopicID:         s.TopicID,
		TotalQuestions:  total,
		CorrectAnswers:  s.CorrectAnswers,
		ScorePercentage: score,
	}
}

func gradeQuizAnswer(question QuizQuestion, answer string) bool {
	expected := normalizeQuizAnswer(question.Answer)
	actual := normalizeQuizAnswer(answer)
	if expected == "" || actual == "" {
		return false
	}

	switch question.AnswerType {
	case "free_text":
		return strings.Contains(actual, expected) || strings.Contains(expected, actual)
	case "range":
		return actual == expected
	case "multiple_choice":
		return actual == expected
	default:
		return actual == expected
	}
}

func matchingDistractorFeedback(question QuizQuestion, answer string) string {
	actual := normalizeQuizAnswer(answer)
	for _, distractor := range question.Distractors {
		if normalizeQuizAnswer(distractor.Value) == actual {
			return distractor.Feedback
		}
	}
	return ""
}

func normalizeQuizAnswer(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		" ", "",
		"\n", "",
		"\t", "",
		"−", "-",
		"–", "-",
		"=", "",
	)
	return replacer.Replace(normalized)
}

func questionsFromAssessment(assessment curriculum.Assessment) []QuizQuestion {
	questions := make([]QuizQuestion, 0, len(assessment.Questions))
	for _, question := range assessment.Questions {
		quizQuestion := QuizQuestion{
			ID:                question.ID,
			Text:              question.Text,
			Difficulty:        question.Difficulty,
			LearningObjective: question.LearningObjective,
			AnswerType:        question.Answer.Type,
			Answer:            question.Answer.Value,
			Working:           question.Answer.Working,
			Marks:             question.Marks,
		}
		for _, hint := range question.Hints {
			quizQuestion.Hints = append(quizQuestion.Hints, QuizHint{
				Level: hint.Level,
				Text:  hint.Text,
			})
		}
		for _, distractor := range question.Distractors {
			quizQuestion.Distractors = append(quizQuestion.Distractors, QuizDistractor{
				Value:    distractor.Value,
				Feedback: distractor.Feedback,
			})
		}
		questions = append(questions, quizQuestion)
	}
	return questions
}

func filterQuizQuestionsByIntensity(questions []QuizQuestion, intensity string) []QuizQuestion {
	normalized := normalizeQuizIntensity(intensity)
	if normalized == "" || normalized == "mixed" {
		return append([]QuizQuestion(nil), questions...)
	}

	filtered := make([]QuizQuestion, 0, len(questions))
	for _, question := range questions {
		if normalizeQuizIntensity(question.Difficulty) == normalized {
			filtered = append(filtered, question)
		}
	}
	if len(filtered) == 0 {
		return append([]QuizQuestion(nil), questions...)
	}
	return filtered
}

func normalizeQuizIntensity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "easy", "senang", "mudah", "ringan":
		return "easy"
	case "medium", "sederhana", "normal":
		return "medium"
	case "hard", "susah", "sukar", "intense", "intensive", "challenging":
		return "hard"
	case "mixed", "campur", "any", "auto":
		return "mixed"
	default:
		return ""
	}
}

func inferQuizStartIntensity(text string) string {
	normalized := " " + strings.ToLower(strings.TrimSpace(text)) + " "

	switch {
	case strings.Contains(normalized, " hard "), strings.Contains(normalized, " susah "), strings.Contains(normalized, " sukar "), strings.Contains(normalized, " challenging "):
		return "hard"
	case strings.Contains(normalized, " medium "), strings.Contains(normalized, " sederhana "), strings.Contains(normalized, " normal "):
		return "medium"
	case strings.Contains(normalized, " easy "), strings.Contains(normalized, " senang "), strings.Contains(normalized, " mudah "), strings.Contains(normalized, " ringan "):
		return "easy"
	case strings.Contains(normalized, " mixed "), strings.Contains(normalized, " campur "):
		return "mixed"
	default:
		return ""
	}
}

func defaultQuizIntensity() string {
	return "mixed"
}

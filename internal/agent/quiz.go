// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"regexp"
	"sort"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

// QuizMaxQuestions is the maximum number of questions allowed in a single quiz session.
const QuizMaxQuestions = 10

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

// AppendQuestions adds extra questions to the session up to QuizMaxQuestions total.
func (s *QuizSession) AppendQuestions(questions []QuizQuestion) {
	remaining := QuizMaxQuestions - len(s.Questions)
	if remaining <= 0 {
		return
	}
	if len(questions) > remaining {
		questions = questions[:remaining]
	}
	s.Questions = append(s.Questions, questions...)
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

var (
	freeTextTokenPattern            = regexp.MustCompile(`[a-z0-9+\-*/=^]+`)
	structuredOrdinalLabelPattern   = regexp.MustCompile(`(?i)\(([ivx]+|[a-z])\)`)
	structuredOrdinalPrefixPattern  = regexp.MustCompile(`(?i)^\s*\(?([ivx]+|[a-z])\)?[.)]\s*`)
	structuredAndSeparatorPattern   = regexp.MustCompile(`(?i)\s+\band\b\s+`)
	structuredAssignmentPattern     = regexp.MustCompile(`(?i)^\s*([a-z])\s*=\s*`)
	structuredAssignmentListPattern = regexp.MustCompile(`(?i)\b[a-z]\s*=\s*[^,;\n]+,\s*[a-z]\s*=`)
	structuredNamedFieldPattern     = regexp.MustCompile(`(?i)^\s*(base|index|gradient|y[\s-]?intercept|intercept)\s*(?:is|=|:)?\s*`)
)

type structuredQuizFragment struct {
	label string
	value string
}

func gradeQuizAnswer(question QuizQuestion, answer string) bool {
	expected := normalizeQuizAnswer(question.Answer)
	actual := normalizeQuizAnswer(answer)
	if expected == "" || actual == "" {
		return false
	}
	if actual == expected {
		return true
	}

	switch question.AnswerType {
	case "free_text":
		if gradeStructuredQuizAnswer(question.Answer, answer) {
			return true
		}
		return gradeQuizFreeText(question.Answer, answer)
	case "range":
		return actual == expected
	case "multiple_choice":
		return actual == expected
	case "exact":
		if gradeStructuredQuizAnswer(question.Answer, answer) {
			return true
		}
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

func gradeQuizFreeText(expected, answer string) bool {
	expectedTokens := tokenizeQuizFreeText(expected)
	actualTokens := tokenizeQuizFreeText(answer)
	if len(expectedTokens) == 0 || len(actualTokens) == 0 {
		return false
	}

	matchIndex := containsTokenSequence(actualTokens, expectedTokens)
	if matchIndex < 0 {
		return false
	}

	return !hasFreeTextNegation(actualTokens, matchIndex)
}

func tokenizeQuizFreeText(value string) []string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer(
		"’", "",
		"'", "",
		"−", "-",
		"–", "-",
	).Replace(normalized)
	return freeTextTokenPattern.FindAllString(normalized, -1)
}

func containsTokenSequence(tokens, sequence []string) int {
	if len(sequence) == 0 || len(tokens) < len(sequence) {
		return -1
	}
	for start := 0; start <= len(tokens)-len(sequence); start++ {
		matched := true
		for i := range sequence {
			if tokens[start+i] != sequence[i] {
				matched = false
				break
			}
		}
		if matched {
			return start
		}
	}
	return -1
}

func hasFreeTextNegation(tokens []string, matchIndex int) bool {
	if matchIndex <= 0 {
		return false
	}

	negations := map[string]struct{}{
		"no":     {},
		"not":    {},
		"never":  {},
		"tak":    {},
		"takkan": {},
		"tidak":  {},
		"bukan":  {},
		"jangan": {},
		"aint":   {},
		"isnt":   {},
		"arent":  {},
		"dont":   {},
		"doesnt": {},
		"didnt":  {},
		"cannot": {},
		"cant":   {},
		"wont":   {},
		"wrong":  {},
	}

	start := matchIndex - 2
	if start < 0 {
		start = 0
	}
	for i := start; i < matchIndex; i++ {
		if _, found := negations[tokens[i]]; found {
			return true
		}
	}
	return false
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

func gradeStructuredQuizAnswer(expected, answer string) bool {
	if !looksStructuredQuizAnswer(expected) {
		return false
	}

	expectedParts := splitStructuredQuizAnswer(expected)
	actualParts := splitStructuredQuizAnswer(answer)
	if len(expectedParts) < 2 || len(actualParts) != len(expectedParts) {
		return false
	}

	for i := range expectedParts {
		if !structuredQuizFragmentMatches(expectedParts[i], actualParts[i]) {
			return false
		}
	}

	return true
}

func looksStructuredQuizAnswer(value string) bool {
	normalized := strings.ToLower(value)
	if strings.Contains(normalized, ";") || strings.Contains(normalized, "\n") {
		return true
	}
	if structuredOrdinalLabelPattern.MatchString(normalized) {
		return true
	}
	if strings.Contains(normalized, "base") || strings.Contains(normalized, "index") ||
		strings.Contains(normalized, "gradient") || strings.Contains(normalized, "intercept") {
		return true
	}
	return structuredAssignmentListPattern.MatchString(normalized)
}

func splitStructuredQuizAnswer(value string) []structuredQuizFragment {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	if labeled := splitStructuredQuizAnswerByLabels(value); len(labeled) >= 2 {
		return labeled
	}

	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\n", ";")
	normalized = structuredAndSeparatorPattern.ReplaceAllString(normalized, ";")

	if structuredAssignmentListPattern.MatchString(normalized) {
		normalized = strings.ReplaceAll(normalized, ",", ";")
	}

	rawParts := strings.Split(normalized, ";")
	parts := make([]structuredQuizFragment, 0, len(rawParts))
	for _, raw := range rawParts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.Contains(raw, ",") && !strings.ContainsAny(raw, "()") {
			for _, nested := range strings.Split(raw, ",") {
				appendStructuredQuizPart(&parts, nested)
			}
			continue
		}
		appendStructuredQuizPart(&parts, raw)
	}
	return parts
}

func splitStructuredQuizAnswerByLabels(value string) []structuredQuizFragment {
	matches := structuredOrdinalLabelPattern.FindAllStringIndex(value, -1)
	if len(matches) < 2 {
		return nil
	}

	parts := make([]structuredQuizFragment, 0, len(matches))
	for i, match := range matches {
		start := match[1]
		end := len(value)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		appendStructuredQuizPart(&parts, value[start:end])
	}
	return parts
}

func appendStructuredQuizPart(parts *[]structuredQuizFragment, raw string) {
	part := parseStructuredQuizFragment(raw)
	if part.value != "" {
		*parts = append(*parts, part)
	}
}

func parseStructuredQuizFragment(value string) structuredQuizFragment {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, ";,")
	value = structuredOrdinalPrefixPattern.ReplaceAllString(value, "")

	label := ""
	if matches := structuredNamedFieldPattern.FindStringSubmatch(value); len(matches) > 1 {
		label = canonicalStructuredQuizLabel(matches[1])
		value = value[len(matches[0]):]
	} else if matches := structuredAssignmentPattern.FindStringSubmatch(value); len(matches) > 1 {
		label = canonicalStructuredQuizAssignmentLabel(matches[1])
		value = value[len(matches[0]):]
	}

	value = strings.TrimSpace(value)
	value = strings.Trim(value, "[]()")
	value = normalizeStructuredQuizValue(value)
	if value == "" {
		return structuredQuizFragment{}
	}
	return structuredQuizFragment{
		label: label,
		value: value,
	}
}

func canonicalStructuredQuizLabel(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "gradient":
		return "gradient"
	case "base":
		return "base"
	case "index":
		return "index"
	case "intercept", "y-intercept", "y intercept":
		return "intercept"
	default:
		return strings.ToLower(strings.TrimSpace(label))
	}
}

func canonicalStructuredQuizAssignmentLabel(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "m":
		return "gradient"
	case "c":
		return "intercept"
	default:
		return strings.ToLower(strings.TrimSpace(label))
	}
}

func normalizeStructuredQuizValue(value string) string {
	replacer := strings.NewReplacer(
		" ", "",
		"\n", "",
		"\t", "",
		"=", "",
		":", "",
		"−", "-",
		"–", "-",
		"≤", "<=",
		"≥", ">=",
		"\\leq", "<=",
		"\\geq", ">=",
	)
	return strings.ToLower(replacer.Replace(value))
}

func structuredQuizFragmentMatches(expected, actual structuredQuizFragment) bool {
	if expected.value != actual.value {
		return false
	}
	if expected.label == "" {
		return actual.label == ""
	}
	return actual.label == "" || actual.label == expected.label
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

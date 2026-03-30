package agent

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	mathrand "math/rand/v2"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

// quizGenerateInput holds the parameters for generating quiz questions via AI.
type quizGenerateInput struct {
	TopicID       string
	TopicName     string
	SyllabusID    string
	Intensity     string
	N             int
	TeachingNotes string
	AllQuestions  []QuizQuestion
}

// quizQuestionGenerator generates quiz questions using an AI router.
type quizQuestionGenerator struct {
	aiRouter *ai.Router
}

// Generate produces N new quiz questions for the given topic using exam-style mimicry.
func (g *quizQuestionGenerator) Generate(ctx context.Context, input quizGenerateInput) ([]QuizQuestion, error) {
	if g.aiRouter == nil || !g.aiRouter.HasProvider() {
		return nil, fmt.Errorf("no AI provider available")
	}
	exemplars := selectExemplars(input.AllQuestions, input.Intensity)
	if len(exemplars) == 0 {
		return nil, fmt.Errorf("no exemplar questions available")
	}
	prompt := buildExamMimicryPrompt(examMimicryPromptInput{
		N:             input.N,
		TopicName:     input.TopicName,
		TopicID:       input.TopicID,
		SyllabusID:    input.SyllabusID,
		Intensity:     input.Intensity,
		TeachingNotes: input.TeachingNotes,
		Exemplars:     exemplars,
	})
	resp, err := g.aiRouter.Complete(ctx, ai.CompletionRequest{
		Task:        ai.TaskGrading,
		MaxTokens:   2000,
		Temperature: 0.7,
		Messages:    []ai.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return nil, fmt.Errorf("AI question generation: %w", err)
	}
	slog.Debug("AI question generation completed",
		"topic_id", input.TopicID,
		"model", resp.Model,
		"tokens", resp.TotalTokens(),
	)
	return parseGeneratedQuestions([]byte(resp.Content))
}

func selectExemplars(questions []QuizQuestion, intensity string) []QuizQuestion {
	if len(questions) == 0 {
		return nil
	}
	normalized := normalizeQuizIntensity(intensity)

	var matched []QuizQuestion
	for _, q := range questions {
		if normalizeQuizIntensity(q.Difficulty) == normalized {
			matched = append(matched, q)
		}
	}
	if len(matched) < 2 {
		matched = append([]QuizQuestion(nil), questions...)
	}
	mathrand.Shuffle(len(matched), func(i, j int) {
		matched[i], matched[j] = matched[j], matched[i]
	})
	count := 3
	if len(matched) < count {
		count = len(matched)
	}
	return matched[:count]
}

type examMimicryPromptInput struct {
	N             int
	TopicName     string
	TopicID       string
	SyllabusID    string
	Intensity     string
	TeachingNotes string
	Exemplars     []QuizQuestion
}

func buildExamMimicryPrompt(input examMimicryPromptInput) string {
	exemplarJSON, _ := json.MarshalIndent(input.Exemplars, "", "  ")
	return fmt.Sprintf(`You are a KSSM Mathematics exam question writer for Malaysian secondary students.

Generate %d new questions for:
- Topic: %s (%s)
- Syllabus: %s
- Difficulty: %s

Curriculum context:
%s

Use these real exam questions as style and format references:
%s

Requirements:
- Match the style, format, and difficulty of the examples
- Each question must have: text, answer (type + value + working), difficulty, hints (2 levels), and distractors (for multiple_choice)
- Use Bahasa Melayu or English matching the exemplar language
- Include LaTeX math notation where appropriate
- Do not duplicate any of the example questions

Return a JSON array of questions.`,
		input.N, input.TopicName, input.TopicID, input.SyllabusID,
		input.Intensity, input.TeachingNotes, string(exemplarJSON))
}

type generatedQuestionJSON struct {
	Text       string `json:"text"`
	Difficulty string `json:"difficulty"`
	AnswerType string `json:"answer_type"`
	Answer     string `json:"answer"`
	Working    string `json:"working"`
	Hints      []struct {
		Level int    `json:"level"`
		Text  string `json:"text"`
	} `json:"hints"`
	Distractors []struct {
		Value    string `json:"value"`
		Feedback string `json:"feedback"`
	} `json:"distractors"`
}

func parseGeneratedQuestions(raw []byte) ([]QuizQuestion, error) {
	var parsed []generatedQuestionJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse generated questions: %w", err)
	}
	questions := make([]QuizQuestion, 0, len(parsed))
	for i, p := range parsed {
		b := make([]byte, 4)
		_, _ = rand.Read(b)
		q := QuizQuestion{
			ID:         fmt.Sprintf("gen-%d-%x", i+1, b),
			Text:       p.Text,
			Difficulty: p.Difficulty,
			AnswerType: p.AnswerType,
			Answer:     p.Answer,
			Working:    p.Working,
		}
		for _, h := range p.Hints {
			q.Hints = append(q.Hints, QuizHint{Level: h.Level, Text: h.Text})
		}
		for _, d := range p.Distractors {
			q.Distractors = append(q.Distractors, QuizDistractor{Value: d.Value, Feedback: d.Feedback})
		}
		questions = append(questions, q)
	}
	return questions, nil
}

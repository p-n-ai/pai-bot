package agent

import (
	mathrand "math/rand/v2"
)

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

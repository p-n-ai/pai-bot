// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

// DetectTopic scans user text for keywords matching loaded topics.
// Returns the best matching topic ID and true, or empty string and false.
func DetectTopic(text string, topics []curriculum.Topic) (string, bool) {
	normalizedTokens := tokenize(text)
	if len(normalizedTokens) == 0 {
		return "", false
	}
	tokenSet := makeTokenSet(normalizedTokens)

	bestID := ""
	bestScore := 0

	for _, topic := range topics {
		score := topicMatchScore(tokenSet, topic)
		if score > bestScore || (score == bestScore && score > 0 && (bestID == "" || topic.ID < bestID)) {
			bestScore = score
			bestID = topic.ID
		}
	}

	if bestScore == 0 {
		return "", false
	}
	return bestID, true
}

func topicMatchScore(tokenSet map[string]struct{}, topic curriculum.Topic) int {
	score := 0

	// Topic name words are the strongest hints.
	for _, word := range tokenize(topic.Name) {
		if len(word) < 3 || isStopWord(word) {
			continue
		}
		if _, ok := tokenSet[word]; ok {
			score += 2
		}
	}

	// Learning objective words add supporting context.
	for _, lo := range topic.LearningObjectives {
		for _, word := range tokenize(lo.Text) {
			if len(word) < 4 || isStopWord(word) {
				continue
			}
			if _, ok := tokenSet[word]; ok {
				score++
			}
		}
	}

	return score
}

func makeTokenSet(tokens []string) map[string]struct{} {
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set
}

func tokenize(text string) []string {
	lower := strings.ToLower(text)
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return ' '
	}, lower)
	return strings.Fields(clean)
}

func isStopWord(word string) bool {
	switch word {
	case "and", "the", "for", "with", "that", "this", "what", "how", "from":
		return true
	default:
		return false
	}
}

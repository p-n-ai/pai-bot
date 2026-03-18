package agent

import (
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

// adaptiveDepthBlock returns system prompt instructions adapted to the student's
// mastery level on the current topic, plus a progress context summary.
//
// Mastery levels:
//   - Beginner (<0.3): simple language, more examples, smaller steps
//   - Developing (0.3–0.6): standard explanations, introduce formal notation gradually
//   - Proficient (>0.6): concise, focus on edge cases and cross-topic connections
func adaptiveDepthBlock(topicMastery float64, allProgress []progress.ProgressItem) string {
	var b strings.Builder

	b.WriteString("\n========================================\n")
	b.WriteString("ADAPTIVE EXPLANATION DEPTH\n")
	b.WriteString("========================================\n\n")

	switch {
	case topicMastery < 0.3:
		b.WriteString(`Student mastery level: BEGINNER (below 30%)
- Use simple, everyday language. Avoid formal math jargon unless defining it.
- Give more examples — at least one worked example before asking the student to try.
- Break problems into smaller steps than usual.
- Use concrete numbers and relatable Malaysian contexts (e.g., prices in RM, distances).
- Be extra patient and encouraging.`)

	case topicMastery < 0.6:
		b.WriteString(`Student mastery level: DEVELOPING (30%–60%)
- Use standard explanations with a mix of everyday and formal notation.
- Introduce formal notation gradually — define it once, then use it.
- Expect the student to handle 2-3 step problems with some guidance.
- Give one example, then ask the student to try a similar one.
- Start connecting this topic to previously learned concepts.`)

	default:
		b.WriteString(`Student mastery level: PROFICIENT (above 60%)
- Be concise — skip basics the student already knows.
- Focus on edge cases, tricky variations, and common exam pitfalls.
- Emphasize cross-topic connections (e.g., how algebra relates to geometry).
- Challenge with harder problems and less scaffolding.
- Use formal notation freely.`)
	}

	// Add progress context if available.
	if len(allProgress) > 0 {
		mastered, working, struggles := categorizeProgress(allProgress)

		b.WriteString("\n\nSTUDENT PROGRESS CONTEXT:\n")
		if len(mastered) > 0 {
			fmt.Fprintf(&b, "- Mastered: %s\n", strings.Join(mastered, ", "))
		}
		if len(working) > 0 {
			fmt.Fprintf(&b, "- Working on: %s\n", strings.Join(working, ", "))
		}
		if len(struggles) > 0 {
			fmt.Fprintf(&b, "- Struggles with: %s\n", strings.Join(struggles, ", "))
		}
	}

	return b.String()
}

// categorizeProgress splits progress items into mastered (≥0.75), working (0.3–0.75),
// and struggles (<0.3) buckets, returning topic IDs for each.
func categorizeProgress(items []progress.ProgressItem) (mastered, working, struggles []string) {
	for _, item := range items {
		switch {
		case item.MasteryScore >= progress.MasteryThreshold:
			mastered = append(mastered, item.TopicID)
		case item.MasteryScore >= 0.3:
			working = append(working, item.TopicID)
		default:
			struggles = append(struggles, item.TopicID)
		}
	}
	return
}

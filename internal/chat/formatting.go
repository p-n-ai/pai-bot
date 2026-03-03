package chat

import (
	"regexp"
	"strings"
)

var doubleAsteriskBoldPattern = regexp.MustCompile(`\*\*([^*\n][^*\n]*?)\*\*`)
var markdownHeadingPattern = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.+?)\s*$`)

// NormalizeTelegramMarkdown converts common markdown patterns from LLM output
// into Telegram's markdown-compatible subset.
func NormalizeTelegramMarkdown(text string) string {
	if text == "" {
		return text
	}
	// Telegram markdown expects *bold* instead of **bold**.
	normalized := doubleAsteriskBoldPattern.ReplaceAllString(text, `*$1*`)

	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		match := markdownHeadingPattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		title := strings.TrimSpace(match[1])
		if title == "" {
			continue
		}
		lines[i] = "*" + title + "*"
	}
	return strings.Join(lines, "\n")
}

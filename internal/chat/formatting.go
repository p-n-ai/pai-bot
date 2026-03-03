package chat

import (
	"regexp"
	"strings"
)

var doubleAsteriskBoldPattern = regexp.MustCompile(`\*\*([^*\n][^*\n]*?)\*\*`)
var markdownHeadingPattern = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.+?)\s*$`)
var inlineMultiplyPattern = regexp.MustCompile(`\b[0-9A-Za-z]+(?:\s*\*\s*[0-9A-Za-z]+)+(?:\s*=\s*[-]?[0-9A-Za-z]+)?\b`)

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
			lines[i] = wrapInlineMath(line)
			continue
		}
		title := strings.TrimSpace(match[1])
		if title == "" {
			lines[i] = wrapInlineMath(line)
			continue
		}
		lines[i] = "*" + wrapInlineMath(title) + "*"
	}
	return strings.Join(lines, "\n")
}

func wrapInlineMath(line string) string {
	if line == "" {
		return line
	}
	indices := inlineMultiplyPattern.FindAllStringIndex(line, -1)
	if len(indices) == 0 {
		return line
	}

	var b strings.Builder
	last := 0
	for _, idx := range indices {
		start, end := idx[0], idx[1]
		if start < last {
			continue
		}
		candidate := line[start:end]
		if !containsDigit(candidate) {
			b.WriteString(line[last:end])
			last = end
			continue
		}
		alreadyCode := (start > 0 && line[start-1] == '`') || (end < len(line) && line[end] == '`')
		b.WriteString(line[last:start])
		if alreadyCode {
			b.WriteString(candidate)
		} else {
			b.WriteByte('`')
			b.WriteString(candidate)
			b.WriteByte('`')
		}
		last = end
	}
	b.WriteString(line[last:])
	return b.String()
}

func containsDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return true
		}
	}
	return false
}

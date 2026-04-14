// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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

// --- LaTeX to Unicode conversion for Telegram ---

var (
	latexInlinePattern = regexp.MustCompile(`\$([^$]+)\$`)
	latexFracPattern   = regexp.MustCompile(`\\frac\{([^}]*)\}\{([^}]*)\}`)
	latexSqrtPattern   = regexp.MustCompile(`\\sqrt\{([^}]*)\}`)
	latexTextPattern   = regexp.MustCompile(`\\text\{([^}]*)\}`)
	latexPowerPattern  = regexp.MustCompile(`\^(\{[^}]+\}|[0-9a-zA-Z])`)
	latexSubPattern    = regexp.MustCompile(`_(\{[^}]+\}|[0-9a-zA-Z])`)
)

var latexSymbols = []struct {
	from string
	to   string
}{
	{`\times`, "×"},
	{`\div`, "÷"},
	{`\cdot`, "·"},
	{`\pm`, "±"},
	{`\mp`, "∓"},
	{`\leq`, "≤"},
	{`\geq`, "≥"},
	{`\neq`, "≠"},
	{`\approx`, "≈"},
	{`\infty`, "∞"},
	{`\pi`, "π"},
	{`\theta`, "θ"},
	{`\alpha`, "α"},
	{`\beta`, "β"},
	{`\gamma`, "γ"},
	{`\delta`, "δ"},
	{`\lambda`, "λ"},
	{`\sigma`, "σ"},
	{`\rightarrow`, "→"},
	{`\leftarrow`, "←"},
	{`\Rightarrow`, "⇒"},
	{`\left`, ""},
	{`\right`, ""},
	{`\,`, " "},
	{`\;`, " "},
	{`\quad`, "  "},
	{`\qquad`, "    "},
	{`\ `, " "},
}

var superscriptMap = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
	'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'+': '⁺', '-': '⁻', '=': '⁼', '(': '⁽', ')': '⁾',
	'n': 'ⁿ', 'i': 'ⁱ',
}

var subscriptMap = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
	'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎',
}

// ConvertLaTeXToUnicode converts LaTeX math notation to Unicode text suitable
// for Telegram. Handles $...$ inline math, fractions, sqrt, powers, etc.
func ConvertLaTeXToUnicode(text string) string {
	return latexInlinePattern.ReplaceAllStringFunc(text, func(match string) string {
		// Strip the $ delimiters.
		inner := match[1 : len(match)-1]
		return convertLaTeXInner(inner)
	})
}

func convertLaTeXInner(s string) string {
	// Apply symbol replacements first.
	for _, sym := range latexSymbols {
		s = strings.ReplaceAll(s, sym.from, sym.to)
	}

	// \text{...} → just the text (must run before \frac to handle nested \text).
	s = latexTextPattern.ReplaceAllString(s, "$1")

	// \frac{a}{b} → a/b or a⁄b
	s = latexFracPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := latexFracPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		num := convertLaTeXInner(parts[1])
		den := convertLaTeXInner(parts[2])
		// Simple single-char fractions
		if len(num) <= 2 && len(den) <= 2 {
			return num + "/" + den
		}
		return "(" + num + ")/(" + den + ")"
	})

	// \sqrt{x} → √x or √(x)
	s = latexSqrtPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := latexSqrtPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		inner := convertLaTeXInner(parts[1])
		if len(inner) <= 3 {
			return "√" + inner
		}
		return "√(" + inner + ")"
	})

	// ^{2} or ^2 → superscript
	s = latexPowerPattern.ReplaceAllStringFunc(s, func(match string) string {
		exp := match[1:]
		exp = strings.TrimPrefix(exp, "{")
		exp = strings.TrimSuffix(exp, "}")
		// Try Unicode superscripts.
		if sup := toSuperscript(exp); sup != "" {
			return sup
		}
		return "^" + exp
	})

	// _{2} or _2 → subscript
	s = latexSubPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := match[1:]
		sub = strings.TrimPrefix(sub, "{")
		sub = strings.TrimSuffix(sub, "}")
		if result := toSubscript(sub); result != "" {
			return result
		}
		return "_" + sub
	})

	// ^\circ → °
	s = strings.ReplaceAll(s, "^°", "°")
	s = strings.ReplaceAll(s, `^\circ`, "°")

	// Strip remaining \command sequences (e.g., \sin → sin, \cos → cos).
	s = stripBackslashCommands(s)

	// Clean up remaining braces.
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")

	return s
}

var backslashCommandPattern = regexp.MustCompile(`\\([a-zA-Z]+)`)

// stripBackslashCommands removes \command but keeps the command name (e.g., \sin → sin).
func stripBackslashCommands(s string) string {
	return backslashCommandPattern.ReplaceAllString(s, "$1")
}

func toSuperscript(s string) string {
	var b strings.Builder
	for _, r := range s {
		if sup, ok := superscriptMap[r]; ok {
			b.WriteRune(sup)
		} else {
			return ""
		}
	}
	return b.String()
}

func toSubscript(s string) string {
	var b strings.Builder
	for _, r := range s {
		if sub, ok := subscriptMap[r]; ok {
			b.WriteRune(sub)
		} else {
			return ""
		}
	}
	return b.String()
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestNormalizeTelegramMarkdown_ConvertsDoubleAsteriskBold(t *testing.T) {
	in := "1. **Isolate the variable term** first."
	out := chat.NormalizeTelegramMarkdown(in)
	want := "1. *Isolate the variable term* first."
	if out != want {
		t.Fatalf("NormalizeTelegramMarkdown() = %q, want %q", out, want)
	}
}

func TestNormalizeTelegramMarkdown_LeavesPlainTextUntouched(t *testing.T) {
	in := "x + 2 = 5"
	out := chat.NormalizeTelegramMarkdown(in)
	if out != in {
		t.Fatalf("NormalizeTelegramMarkdown() changed plain text: got %q", out)
	}
}

func TestNormalizeTelegramMarkdown_ConvertsHeadingToBold(t *testing.T) {
	in := "# Basics of Linear Equations"
	out := chat.NormalizeTelegramMarkdown(in)
	want := "*Basics of Linear Equations*"
	if out != want {
		t.Fatalf("NormalizeTelegramMarkdown() = %q, want %q", out, want)
	}
}

func TestNormalizeTelegramMarkdown_WrapsInlineMathWithCode(t *testing.T) {
	in := "Penyelesaian: 2 * 2 = 4"
	out := chat.NormalizeTelegramMarkdown(in)
	want := "Penyelesaian: `2 * 2 = 4`"
	if out != want {
		t.Fatalf("NormalizeTelegramMarkdown() = %q, want %q", out, want)
	}
}

func TestNormalizeTelegramMarkdown_WrapsEquationWithPlus(t *testing.T) {
	in := "Semuanya: 2 + 2 = 4"
	out := chat.NormalizeTelegramMarkdown(in)
	want := "Semuanya: 2 + 2 = 4"
	if out != want {
		t.Fatalf("NormalizeTelegramMarkdown() = %q, want %q", out, want)
	}
}

func TestNormalizeTelegramMarkdown_WrapsMultiplicationWithVariable(t *testing.T) {
	in := "Contoh: 2*x = 8"
	out := chat.NormalizeTelegramMarkdown(in)
	want := "Contoh: `2*x = 8`"
	if out != want {
		t.Fatalf("NormalizeTelegramMarkdown() = %q, want %q", out, want)
	}
}

func TestNormalizeTelegramMarkdown_DoesNotDoubleWrapExistingCode(t *testing.T) {
	in := "Sudah code: `2 * 2 = 4`"
	out := chat.NormalizeTelegramMarkdown(in)
	if out != in {
		t.Fatalf("NormalizeTelegramMarkdown() = %q, want unchanged %q", out, in)
	}
}

func TestConvertLaTeXToUnicode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple power", in: "$x^2 = 49$", want: "x² = 49"},
		{name: "frac simple", in: "$\\frac{2}{5}$", want: "2/5"},
		{name: "frac complex", in: "$\\frac{2}{5b}(15a + 25b)$", want: "2/5b(15a + 25b)"},
		{name: "sqrt", in: "$\\sqrt{144}$", want: "√144"},
		{name: "sqrt long", in: "$\\sqrt{x + 1}$", want: "√(x + 1)"},
		{name: "times", in: "$3 \\times 4$", want: "3 × 4"},
		{name: "pm", in: "$x = \\pm 7$", want: "x = ± 7"},
		{name: "approx", in: "$x \\approx 3.14$", want: "x ≈ 3.14"},
		{name: "degree", in: "$30^\\circ$", want: "30°"},
		{name: "text", in: "$\\sin \\theta = \\frac{\\text{opposite}}{\\text{hypotenuse}}$", want: "sin θ = (opposite)/(hypotenuse)"},
		{name: "no latex", in: "Hello world", want: "Hello world"},
		{name: "mixed", in: "Solve $x^2 = 49$ for $x$.", want: "Solve x² = 49 for x."},
		{name: "subscript", in: "$a_1 + a_2$", want: "a₁ + a₂"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chat.ConvertLaTeXToUnicode(tt.in)
			if got != tt.want {
				t.Errorf("ConvertLaTeXToUnicode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

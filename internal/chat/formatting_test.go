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

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

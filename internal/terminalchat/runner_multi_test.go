package terminalchat

import (
	"context"
	"strings"
	"testing"
)

func TestRunMulti_RoutesMessagesToDifferentUsers(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("1:hello from user 1\n2:hello from user 2\n/exit\n")
	processor := &stubProcessor{
		responses: []string{"reply to 1", "reply to 2"},
	}

	err := RunMulti(context.Background(), input, &output, processor, MultiConfig{
		UserCount:  2,
		UserPrefix: "player",
		Channel:    "test",
	})
	if err != nil {
		t.Fatalf("RunMulti() error = %v", err)
	}

	if len(processor.messages) != 2 {
		t.Fatalf("processed messages = %d, want 2", len(processor.messages))
	}
	if processor.messages[0].UserID != "player-1" {
		t.Errorf("first UserID = %q, want player-1", processor.messages[0].UserID)
	}
	if processor.messages[0].Text != "hello from user 1" {
		t.Errorf("first Text = %q, want 'hello from user 1'", processor.messages[0].Text)
	}
	if processor.messages[1].UserID != "player-2" {
		t.Errorf("second UserID = %q, want player-2", processor.messages[1].UserID)
	}
	if processor.messages[1].Text != "hello from user 2" {
		t.Errorf("second Text = %q, want 'hello from user 2'", processor.messages[1].Text)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "[player-1] P&AI> reply to 1") {
		t.Errorf("output missing user 1 reply, got: %s", rendered)
	}
	if !strings.Contains(rendered, "[player-2] P&AI> reply to 2") {
		t.Errorf("output missing user 2 reply, got: %s", rendered)
	}
}

func TestRunMulti_DefaultsToUser1WhenNoPrefix(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("no prefix message\n/exit\n")
	processor := &stubProcessor{
		responses: []string{"ok"},
	}

	err := RunMulti(context.Background(), input, &output, processor, MultiConfig{UserCount: 2})
	if err != nil {
		t.Fatalf("RunMulti() error = %v", err)
	}

	if len(processor.messages) != 1 {
		t.Fatalf("processed messages = %d, want 1", len(processor.messages))
	}
	if processor.messages[0].UserID != "terminal-user-1" {
		t.Errorf("UserID = %q, want terminal-user-1", processor.messages[0].UserID)
	}
	if processor.messages[0].Text != "no prefix message" {
		t.Errorf("Text = %q, want 'no prefix message'", processor.messages[0].Text)
	}
}

func TestRunMulti_InvalidPrefixDefaultsToUser1(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("9:out of range\n/exit\n")
	processor := &stubProcessor{
		responses: []string{"ok"},
	}

	err := RunMulti(context.Background(), input, &output, processor, MultiConfig{UserCount: 2})
	if err != nil {
		t.Fatalf("RunMulti() error = %v", err)
	}

	if processor.messages[0].UserID != "terminal-user-1" {
		t.Errorf("UserID = %q, want terminal-user-1 for invalid prefix", processor.messages[0].UserID)
	}
	// The full "9:out of range" should be the message text
	if processor.messages[0].Text != "9:out of range" {
		t.Errorf("Text = %q, want '9:out of range'", processor.messages[0].Text)
	}
}

func TestRunMulti_ShowsWelcomeWithUserList(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("/exit\n")
	processor := &stubProcessor{}

	err := RunMulti(context.Background(), input, &output, processor, MultiConfig{
		UserCount:  3,
		UserPrefix: "p",
	})
	if err != nil {
		t.Fatalf("RunMulti() error = %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "1=p-1") {
		t.Errorf("output missing user 1 in welcome, got: %s", rendered)
	}
	if !strings.Contains(rendered, "3=p-3") {
		t.Errorf("output missing user 3 in welcome, got: %s", rendered)
	}
}

func TestRunMulti_EOFEndsSession(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("1:hello\n")
	processor := &stubProcessor{responses: []string{"hi"}}

	err := RunMulti(context.Background(), input, &output, processor, MultiConfig{UserCount: 2})
	if err != nil {
		t.Fatalf("RunMulti() error = %v", err)
	}

	if !strings.Contains(output.String(), "Session ended.") {
		t.Errorf("output missing 'Session ended.', got: %s", output.String())
	}
}

func TestParseMultiInput(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		maxUsers int
		wantIdx  int
		wantText string
	}{
		{name: "user 1", raw: "1:hello", maxUsers: 2, wantIdx: 0, wantText: "hello"},
		{name: "user 2", raw: "2:world", maxUsers: 2, wantIdx: 1, wantText: "world"},
		{name: "no prefix", raw: "just text", maxUsers: 2, wantIdx: 0, wantText: "just text"},
		{name: "out of range", raw: "5:hello", maxUsers: 2, wantIdx: 0, wantText: "5:hello"},
		{name: "zero", raw: "0:hello", maxUsers: 2, wantIdx: 0, wantText: "0:hello"},
		{name: "negative", raw: "-1:hello", maxUsers: 2, wantIdx: 0, wantText: "-1:hello"},
		{name: "empty after colon", raw: "1:", maxUsers: 2, wantIdx: 0, wantText: "1:"},
		{name: "spaces around text", raw: "2:  hello  ", maxUsers: 3, wantIdx: 1, wantText: "hello"},
		{name: "command", raw: "1:/challenge invite algebra", maxUsers: 2, wantIdx: 0, wantText: "/challenge invite algebra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIdx, gotText := parseMultiInput(tt.raw, tt.maxUsers)
			if gotIdx != tt.wantIdx {
				t.Errorf("parseMultiInput(%q) idx = %d, want %d", tt.raw, gotIdx, tt.wantIdx)
			}
			if gotText != tt.wantText {
				t.Errorf("parseMultiInput(%q) text = %q, want %q", tt.raw, gotText, tt.wantText)
			}
		})
	}
}

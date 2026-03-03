package chat_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLen    int
		wantParts int
	}{
		{"short", "Hello", 4096, 1},
		{"exact", "Hello", 5, 1},
		{"split-needed", "Hello World, this is a test", 10, 4},
		{"empty", "", 4096, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := chat.SplitMessage(tt.text, tt.maxLen)
			if len(parts) != tt.wantParts {
				t.Errorf("SplitMessage() = %d parts, want %d", len(parts), tt.wantParts)
			}
		})
	}
}

func TestSplitMessage_PartsNotExceedMax(t *testing.T) {
	text := "This is a longer message that needs to be split into multiple parts for Telegram delivery."
	maxLen := 20
	parts := chat.SplitMessage(text, maxLen)

	for i, part := range parts {
		if len(part) > maxLen {
			t.Errorf("part[%d] len=%d exceeds maxLen=%d: %q", i, len(part), maxLen, part)
		}
	}
}

func TestNewTelegramChannel_NoToken(t *testing.T) {
	_, err := chat.NewTelegramChannel("")
	if err == nil {
		t.Error("NewTelegramChannel() should error with empty token")
	}
}

func TestNewTelegramChannel_ValidToken(t *testing.T) {
	ch, err := chat.NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	if ch == nil {
		t.Error("NewTelegramChannel() returned nil")
	}
}

func TestMapTelegramInbound_CallbackQuery(t *testing.T) {
	update := map[string]any{
		"update_id": 1001,
		"callback_query": map[string]any{
			"id":   "cb-1",
			"data": "4",
			"from": map[string]any{
				"id":         777,
				"username":   "u",
				"first_name": "A",
				"last_name":  "B",
			},
			"message": map[string]any{
				"message_id": 88,
				"chat": map[string]any{
					"id": 123456,
				},
			},
		},
	}
	msg, ok := chat.MapTelegramInboundForTest(update)
	if !ok {
		t.Fatal("expected callback query to map as inbound")
	}
	if msg.Text != "4" {
		t.Fatalf("Text = %q, want 4", msg.Text)
	}
	if msg.CallbackQueryID != "cb-1" {
		t.Fatalf("CallbackQueryID = %q, want cb-1", msg.CallbackQueryID)
	}
	if msg.CallbackMessageID != 88 {
		t.Fatalf("CallbackMessageID = %d, want 88", msg.CallbackMessageID)
	}
	if msg.UserID != "123456" {
		t.Fatalf("UserID = %q, want 123456", msg.UserID)
	}
}

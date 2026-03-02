package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestMemoryEventLogger_LogEvent(t *testing.T) {
	logger := agent.NewMemoryEventLogger()

	err := logger.LogEvent(agent.Event{
		ConversationID: "conv-1",
		UserID:         "user-1",
		EventType:      "message_sent",
		Data: map[string]any{
			"text_len": 42,
		},
	})
	if err != nil {
		t.Fatalf("LogEvent() error = %v", err)
	}

	events := logger.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].EventType != "message_sent" {
		t.Errorf("EventType = %q, want message_sent", events[0].EventType)
	}
	if events[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestPostgresEventLogger_LogEvent_NilPool(t *testing.T) {
	logger := agent.NewPostgresEventLogger(nil)

	err := logger.LogEvent(agent.Event{
		ConversationID: "conv-1",
		EventType:      "session_started",
	})
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
}

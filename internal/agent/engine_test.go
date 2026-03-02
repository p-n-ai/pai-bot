package agent_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestEngine_ProcessMessage(t *testing.T) {
	mockAI := ai.NewMockProvider("This is the AI response about algebra.")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "What is algebra?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp == "" {
		t.Error("ProcessMessage() returned empty response")
	}
}

func TestEngine_ProcessMessage_StartCommand(t *testing.T) {
	mockAI := ai.NewMockProvider("Welcome!")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:   "telegram",
		UserID:    "123",
		Text:      "/start",
		FirstName: "Ali",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp == "" {
		t.Error("ProcessMessage() returned empty response for /start")
	}
}

func TestEngine_ProcessMessage_StartCommand_UsesFirstName(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(ai.NewMockProvider("")),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:   "telegram",
		UserID:    "123",
		Text:      "/start",
		FirstName: "Ali",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp == "" {
		t.Error("Expected non-empty welcome message")
	}
	// Should contain the user's first name
	if !contains(resp, "Ali") {
		t.Errorf("Welcome message should contain user's name 'Ali', got: %s", resp)
	}
}

func TestEngine_ProcessMessage_StartCommand_FallbackName(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(ai.NewMockProvider("")),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "/start",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp == "" {
		t.Error("Expected non-empty welcome message even without name")
	}
}

func TestEngine_ProcessMessage_UnknownCommand(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(ai.NewMockProvider("")),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "/unknown",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp == "" {
		t.Error("Expected non-empty response for unknown command")
	}
}

func TestEngine_ProcessMessage_AIError(t *testing.T) {
	mockAI := &ai.MockProvider{Err: context.DeadlineExceeded}

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "What is x+1?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() should not return error on AI failure, got: %v", err)
	}
	if resp == "" {
		t.Error("Should return a fallback message when AI fails")
	}
}

// mockRouter creates an AI router with a single mock provider.
func mockRouter(provider ai.Provider) *ai.Router {
	r := ai.NewRouter()
	r.Register("mock", provider)
	return r
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

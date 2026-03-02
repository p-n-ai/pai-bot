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

func TestEngine_ConversationHistory(t *testing.T) {
	mockAI := ai.NewMockProvider("Response 2")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	// First message
	mockAI.Response = "Response 1"
	engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "What is x?",
	})

	// Second message — should include history
	mockAI.Response = "Response 2"
	engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "What about y?",
	})

	// The last request should have: system + user("What is x?") + assistant("Response 1") + user("What about y?")
	if mockAI.LastRequest == nil {
		t.Fatal("LastRequest is nil")
	}
	msgs := mockAI.LastRequest.Messages
	if len(msgs) < 4 {
		t.Fatalf("Expected at least 4 messages (system + 2 user + 1 assistant), got %d", len(msgs))
	}
	// First should be system
	if msgs[0].Role != "system" {
		t.Errorf("msgs[0].Role = %q, want system", msgs[0].Role)
	}
	// Second should be user's first message
	if msgs[1].Role != "user" || msgs[1].Content != "What is x?" {
		t.Errorf("msgs[1] = {%q, %q}, want {user, What is x?}", msgs[1].Role, msgs[1].Content)
	}
	// Third should be assistant's first response
	if msgs[2].Role != "assistant" || msgs[2].Content != "Response 1" {
		t.Errorf("msgs[2] = {%q, %q}, want {assistant, Response 1}", msgs[2].Role, msgs[2].Content)
	}
	// Fourth should be user's second message
	if msgs[3].Role != "user" || msgs[3].Content != "What about y?" {
		t.Errorf("msgs[3] = {%q, %q}, want {user, What about y?}", msgs[3].Role, msgs[3].Content)
	}
}

func TestEngine_StartClearsHistory(t *testing.T) {
	mockAI := ai.NewMockProvider("Response")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	// Build some history
	engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "Hello",
	})

	// /start should clear it
	engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "/start",
	})

	// Next message should have only system + this user message (no old history)
	engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "Fresh start",
	})

	msgs := mockAI.LastRequest.Messages
	// system + user("Fresh start") = 2 messages (no old history)
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages after /start, got %d", len(msgs))
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

package agent_test

import (
	"context"
	"fmt"
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
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "What is x?",
	})

	// Second message — should include history
	mockAI.Response = "Response 2"
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
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
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "Hello",
	})

	// /start should clear it
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "/start",
	})

	// Next message should have only system + this user message (no old history)
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "Fresh start",
	})

	msgs := mockAI.LastRequest.Messages
	// system + user("Fresh start") = 2 messages (no old history)
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages after /start, got %d", len(msgs))
	}
}

func TestEngine_ProcessMessage_ReplyToText(t *testing.T) {
	mockAI := ai.NewMockProvider("Let me explain that step again.")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:     "telegram",
		UserID:      "123",
		Text:        "I don't understand this part",
		ReplyToText: "Step 2: Move x to the left side of the equation",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	// The user message sent to AI should include the replied text as context.
	msgs := mockAI.LastRequest.Messages
	lastUserMsg := msgs[len(msgs)-1]
	if lastUserMsg.Role != "user" {
		t.Fatalf("last message role = %q, want user", lastUserMsg.Role)
	}
	if !contains(lastUserMsg.Content, "Replying to") {
		t.Errorf("user message should contain reply context, got: %s", lastUserMsg.Content)
	}
	if !contains(lastUserMsg.Content, "Step 2") {
		t.Errorf("user message should contain original text, got: %s", lastUserMsg.Content)
	}
	if !contains(lastUserMsg.Content, "I don't understand") {
		t.Errorf("user message should contain user's text, got: %s", lastUserMsg.Content)
	}
}

func TestEngine_Compaction(t *testing.T) {
	mockAI := ai.NewMockProvider("response")

	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CompactThreshold: 6, // compact after 6 messages
		KeepRecent:       2, // keep last 2 messages
	})

	// Send 4 exchanges (8 messages total, exceeds threshold of 6)
	for i := 0; i < 4; i++ {
		mockAI.Response = fmt.Sprintf("response %d", i)
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram", UserID: "123", Text: fmt.Sprintf("question %d", i),
		})
	}

	// The summarization AI call should have happened.
	// Next message should get: system + summary + recent messages (not all 8).
	mockAI.Response = "final response"
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "another question",
	})

	msgs := mockAI.LastRequest.Messages
	// Without compaction: system + 9 conversation messages = 10.
	// With compaction: system(1) + summary pair(2) + recent messages — should be well under 10.
	if len(msgs) >= 10 {
		t.Errorf("Expected compacted messages (< 10), got %d", len(msgs))
	}
	// First should be system
	if msgs[0].Role != "system" {
		t.Errorf("msgs[0].Role = %q, want system", msgs[0].Role)
	}
	// Second should be the summary context
	if !contains(msgs[1].Content, "Previous conversation summary") {
		t.Errorf("msgs[1] should contain summary, got: %s", msgs[1].Content)
	}
}

func TestEngine_Compaction_NoRecompressEveryTurn(t *testing.T) {
	summarizeCount := 0
	mockAI := &ai.MockProvider{}
	mockAI.Response = "response"

	// We'll track summarization calls by checking the task type.
	// The summarization uses TaskAnalysis, teaching uses TaskTeaching.
	tracker := &callTracker{provider: mockAI}

	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(tracker),
		Store:            store,
		CompactThreshold: 6,
		KeepRecent:       2,
	})

	// Send 4 exchanges (8 messages) — should trigger ONE compaction.
	for i := 0; i < 4; i++ {
		mockAI.Response = fmt.Sprintf("response %d", i)
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram", UserID: "123", Text: fmt.Sprintf("q%d", i),
		})
	}

	// Count summarization calls (TaskAnalysis).
	for _, req := range tracker.requests {
		if req.Task == ai.TaskAnalysis {
			summarizeCount++
		}
	}

	firstSummarizeCount := summarizeCount

	// Send 2 more messages — should NOT trigger another compaction
	// because we haven't accumulated enough new messages past the threshold.
	for i := 0; i < 2; i++ {
		mockAI.Response = fmt.Sprintf("more response %d", i)
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram", UserID: "123", Text: fmt.Sprintf("more q%d", i),
		})
	}

	summarizeCount = 0
	for _, req := range tracker.requests {
		if req.Task == ai.TaskAnalysis {
			summarizeCount++
		}
	}

	if summarizeCount != firstSummarizeCount {
		t.Errorf("Should not re-compact, but summarization calls went from %d to %d",
			firstSummarizeCount, summarizeCount)
	}
}

func TestEngine_Compaction_LongMessages(t *testing.T) {
	mockAI := ai.NewMockProvider("short reply")

	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:              mockRouter(mockAI),
		Store:                 store,
		CompactThreshold:      100,  // high message threshold — won't trigger by count
		CompactTokenThreshold: 200,  // low token threshold — triggers by content size
		KeepRecent:            2,
	})

	// Send 3 messages with long content (~100 tokens each = ~400 chars).
	longText := string(make([]byte, 400))
	for i := range longText {
		longText = longText[:i] + "a" + longText[i+1:]
	}
	for i := 0; i < 3; i++ {
		mockAI.Response = longText
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram", UserID: "token-user", Text: longText,
		})
	}

	// Should have compacted despite only 6 messages (3 user + 3 assistant),
	// because token estimate exceeds 200.
	conv, found := store.GetActiveConversation("token-user")
	if !found {
		t.Fatal("conversation not found")
	}
	if conv.Summary == "" {
		t.Error("Expected compaction to trigger based on token count, but no summary found")
	}
}

func TestEngine_NoCompaction_UnderThreshold(t *testing.T) {
	mockAI := ai.NewMockProvider("response")

	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CompactThreshold: 20, // high threshold
		KeepRecent:       6,
	})

	// Send 3 messages — well under threshold.
	for i := 0; i < 3; i++ {
		mockAI.Response = fmt.Sprintf("response %d", i)
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram", UserID: "123", Text: fmt.Sprintf("q%d", i),
		})
	}

	// All messages should be in the prompt (no compaction).
	msgs := mockAI.LastRequest.Messages
	// system + 3 user + 2 assistant (from prior turns) + 1 user (current) = ...
	// Actually: after 3 turns: system + user0 + asst0 + user1 + asst1 + user2 = 6
	if len(msgs) != 6 {
		t.Errorf("Expected 6 messages (no compaction), got %d", len(msgs))
	}
}

// callTracker wraps a provider to record all requests.
type callTracker struct {
	provider ai.Provider
	requests []ai.CompletionRequest
}

func (c *callTracker) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	c.requests = append(c.requests, req)
	return c.provider.Complete(ctx, req)
}

func (c *callTracker) StreamComplete(ctx context.Context, req ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return c.provider.StreamComplete(ctx, req)
}

func (c *callTracker) Models() []ai.ModelInfo {
	return c.provider.Models()
}

func (c *callTracker) HealthCheck(ctx context.Context) error {
	return c.provider.HealthCheck(ctx)
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

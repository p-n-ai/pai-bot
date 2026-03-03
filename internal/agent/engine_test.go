package agent_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
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

func TestEngine_ProcessMessage_StartCommand_CreatesOnboardingConversation(t *testing.T) {
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(ai.NewMockProvider("")),
		Store:    store,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-start-1",
		Text:    "/start",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	conv, found := store.GetActiveConversation("u-start-1")
	if !found {
		t.Fatal("expected active conversation after /start")
	}
	if conv.State != "onboarding_language" {
		t.Fatalf("conversation state = %q, want onboarding_language", conv.State)
	}
}

func TestEngine_ProcessMessage_AutoStartForNewUser(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := newAutoStartStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:   "telegram",
		UserID:    "new-user-1",
		Text:      "Hi tutor",
		FirstName: "Aina",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Bahasa pilihan anda") {
		t.Fatalf("expected onboarding prompt, got: %q", resp)
	}

	conv, found := store.GetActiveConversation("new-user-1")
	if !found {
		t.Fatal("expected active conversation after auto-start")
	}
	if conv.State != "onboarding_language" {
		t.Fatalf("conversation state = %q, want onboarding_language", conv.State)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called when auto-start onboarding is triggered")
	}
}

func TestEngine_ProcessMessage_ExistingUserDoesNotAutoStart(t *testing.T) {
	mockAI := ai.NewMockProvider("teaching response")
	store := newAutoStartStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	_, err := store.CreateConversation(agent.Conversation{
		UserID: "existing-user-1",
		State:  "teaching",
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "existing-user-1",
		Text:    "Explain linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if contains(resp, "Tingkatan berapa") {
		t.Fatalf("should not auto-start onboarding for existing user, got: %q", resp)
	}
	if mockAI.LastRequest == nil {
		t.Fatal("AI should be called for existing user teaching flow")
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

func TestEngine_ProcessMessage_RetryThenSuccess(t *testing.T) {
	flaky := &flakyProvider{
		failuresBeforeSuccess: 2,
		response:              "Recovered after retries",
	}

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(flaky),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "retry-user",
		Text:    "Explain x + 2 = 5",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp != "Recovered after retries" {
		t.Fatalf("unexpected response: %q", resp)
	}
	if flaky.calls != 3 {
		t.Fatalf("calls = %d, want 3", flaky.calls)
	}
}

func TestEngine_ProcessMessage_RetryExhausted(t *testing.T) {
	flaky := &flakyProvider{
		failuresBeforeSuccess: 99,
		response:              "should not happen",
	}

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(flaky),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "retry-fail-user",
		Text:    "Explain x + 2 = 5",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() should not return error, got: %v", err)
	}
	if !contains(resp, "masalah teknikal") {
		t.Fatalf("expected friendly fallback message, got: %q", resp)
	}
	if flaky.calls != 4 {
		t.Fatalf("calls = %d, want 4", flaky.calls)
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

	// Complete onboarding first.
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "English",
	})
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "1",
	})

	// Next teaching message should not include old pre-/start history.
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "Fresh start",
	})

	msgs := mockAI.LastRequest.Messages
	for _, m := range msgs {
		if m.Content == "Hello" {
			t.Errorf("Expected old history to be cleared after /start, but found message %q", m.Content)
		}
	}
}

func TestEngine_Onboarding_InvalidSelection_AsksClarification(t *testing.T) {
	mockAI := ai.NewMockProvider("unknown")
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    agent.NewMemoryStore(),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user", Text: "/start",
	})
	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user", Text: "saya tak pasti",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Bahasa Melayu") {
		t.Fatalf("unexpected onboarding invalid response: %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called for onboarding language clarification")
	}
}

func TestEngine_Onboarding_SelectionTransitionsToTeaching(t *testing.T) {
	mockAI := ai.NewMockProvider("AI teaching response")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-2", Text: "/start",
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-2", Text: "Bahasa Melayu",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-2", Text: "Tingkatan 2",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Tingkatan 2") {
		t.Fatalf("unexpected onboarding completion response: %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called when selecting onboarding form")
	}

	conv, found := store.GetActiveConversation("onboard-user-2")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "teaching" {
		t.Fatalf("conversation state = %q, want teaching", conv.State)
	}

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-2", Text: "Apa itu algebra?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if mockAI.LastRequest == nil {
		t.Fatal("expected AI to be called after onboarding completes")
	}
}

func TestEngine_Onboarding_FreeTextSelection_ParsesRuleBased(t *testing.T) {
	mockAI := ai.NewMockProvider("AI teaching response")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-3", Text: "/start",
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-3", Text: "English",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-3", Text: "I am in form two",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Form 2") {
		t.Fatalf("unexpected onboarding completion response: %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called when free-text can be parsed by rules")
	}
}

func TestEngine_Onboarding_AIFallbackSelection_TransitionsToTeaching(t *testing.T) {
	mockAI := ai.NewMockProvider("2")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-4", Text: "/start",
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-4", Text: "English",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "onboard-user-4", Text: "middle school level",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Form 2") {
		t.Fatalf("unexpected onboarding completion response: %q", resp)
	}
	if mockAI.LastRequest == nil {
		t.Fatal("expected AI fallback classification call")
	}
}

func TestEngine_LanguageCommand_InteractiveSelection_SendsConfirmation(t *testing.T) {
	mockAI := ai.NewMockProvider("AI teaching response")
	tracker := &callTracker{provider: mockAI}
	store := agent.NewMemoryStore()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(tracker),
		Store:    store,
	})

	userID := "lang-cmd-user"
	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "Explain linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if len(tracker.requests) != 1 {
		t.Fatalf("AI requests = %d, want 1", len(tracker.requests))
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "/language",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Choose your language") {
		t.Fatalf("expected language chooser response, got: %q", resp)
	}

	conv, found := store.GetActiveConversation(userID)
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "language_selection" {
		t.Fatalf("conversation state = %q, want language_selection", conv.State)
	}

	resp, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:         "telegram",
		UserID:          userID,
		Text:            "lang:en",
		CallbackQueryID: "cb-lang-1",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Language updated to English.") {
		t.Fatalf("expected language changed confirmation, got: %q", resp)
	}
	if len(tracker.requests) != 1 {
		t.Fatalf("AI should not be called for language callback; requests = %d, want 1", len(tracker.requests))
	}

	conv, found = store.GetActiveConversation(userID)
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "teaching" {
		t.Fatalf("conversation state = %q, want teaching", conv.State)
	}
}

func TestEngine_OnboardingLanguageSelection_IncludesConfirmation(t *testing.T) {
	mockAI := ai.NewMockProvider("unused")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	userID := "lang-onboarding-user"
	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "/start",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "English",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Language updated to English.") {
		t.Fatalf("expected language changed confirmation, got: %q", resp)
	}
	if !contains(resp, "Which form are you in now?") {
		t.Fatalf("expected onboarding form prompt, got: %q", resp)
	}

	conv, found := store.GetActiveConversation(userID)
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "onboarding_form" {
		t.Fatalf("conversation state = %q, want onboarding_form", conv.State)
	}
}

func TestEngine_OnboardingLanguageCommandWithArgs_ContinuesToFormStep(t *testing.T) {
	mockAI := ai.NewMockProvider("unused")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	userID := "lang-onboarding-user-command"
	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "/start",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "/language en",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Language updated to English.") {
		t.Fatalf("expected language changed confirmation, got: %q", resp)
	}
	if !contains(resp, "Which form are you in now?") {
		t.Fatalf("expected onboarding form prompt, got: %q", resp)
	}

	conv, found := store.GetActiveConversation(userID)
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "onboarding_form" {
		t.Fatalf("conversation state = %q, want onboarding_form", conv.State)
	}
}

func TestEngine_ProcessMessage_ClearCommand(t *testing.T) {
	mockAI := ai.NewMockProvider("ok")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "/clear",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "dikosongkan") {
		t.Fatalf("unexpected /clear response: %q", resp)
	}
}

func TestEngine_SystemPrompt_HasImageFollowUpReplyGuidance(t *testing.T) {
	mockAI := ai.NewMockProvider("ok")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-system-prompt",
		Text:    "what is 2x + 1 = 5?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	if mockAI.LastRequest == nil || len(mockAI.LastRequest.Messages) == 0 {
		t.Fatal("expected request messages to be sent to AI")
	}
	systemPrompt := mockAI.LastRequest.Messages[0]
	if systemPrompt.Role != "system" {
		t.Fatalf("first message role = %q, want system", systemPrompt.Role)
	}
	if !contains(systemPrompt.Content, "did not reply to that image") {
		t.Fatalf("system prompt missing image follow-up reply guidance")
	}
}

func TestEngine_ProcessMessage_InjectsCurriculumContextWhenTopicMatched(t *testing.T) {
	mockAI := ai.NewMockProvider("ok")
	loader := createTestCurriculumLoader(t)

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		CurriculumLoader: loader,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-curriculum",
		Text:    "Please teach me linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	systemPrompt := mockAI.LastRequest.Messages[0].Content
	if !contains(systemPrompt, "TOPIC CONTEXT") {
		t.Fatalf("expected TOPIC CONTEXT in system prompt, got: %s", systemPrompt)
	}
	if !contains(systemPrompt, "F1-02") {
		t.Fatalf("expected topic ID in system prompt, got: %s", systemPrompt)
	}
	if !contains(systemPrompt, "subtract 5 on both sides") {
		t.Fatalf("expected teaching notes in system prompt, got: %s", systemPrompt)
	}
}

func TestEngine_ProcessMessage_NoCurriculumContextWhenNoTopicMatch(t *testing.T) {
	mockAI := ai.NewMockProvider("ok")
	loader := createTestCurriculumLoader(t)

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		CurriculumLoader: loader,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-curriculum-no-match",
		Text:    "What is your favorite color?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	systemPrompt := mockAI.LastRequest.Messages[0].Content
	if contains(systemPrompt, "TOPIC CONTEXT") {
		t.Fatalf("did not expect TOPIC CONTEXT in system prompt when no topic matches, got: %s", systemPrompt)
	}
}

func TestEngine_ProcessMessage_UsesConfiguredContextResolver(t *testing.T) {
	mockAI := ai.NewMockProvider("ok")
	resolver := &stubContextResolver{
		topic: &curriculum.Topic{
			ID:         "X-01",
			Name:       "Custom Interface Topic",
			SubjectID:  "math",
			SyllabusID: "custom-syllabus",
			LearningObjectives: []curriculum.LearningObjective{
				{ID: "LO1", Text: "Explain variables in simple terms"},
			},
		},
		notes: "Use a scale analogy and isolate one variable first.",
	}

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		ContextResolver:  resolver,
		CurriculumLoader: nil,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-context-resolver",
		Text:    "hello there",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	if !resolver.called {
		t.Fatal("expected configured context resolver to be called")
	}
	if resolver.lastText != "hello there" {
		t.Fatalf("resolver.lastText = %q, want %q", resolver.lastText, "hello there")
	}

	systemPrompt := mockAI.LastRequest.Messages[0].Content
	if !contains(systemPrompt, "TOPIC CONTEXT") {
		t.Fatalf("expected TOPIC CONTEXT in system prompt, got: %s", systemPrompt)
	}
	if !contains(systemPrompt, "X-01") {
		t.Fatalf("expected resolver topic ID in system prompt, got: %s", systemPrompt)
	}
	if !contains(systemPrompt, "scale analogy") {
		t.Fatalf("expected resolver notes in system prompt, got: %s", systemPrompt)
	}
}

func TestEngine_ClearClearsHistory(t *testing.T) {
	mockAI := ai.NewMockProvider("Response")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	// Build some history
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "Hello",
	})

	// /clear should clear it
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "/clear",
	})

	// Next message should have only system + this user message (no old history)
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram", UserID: "123", Text: "Fresh start",
	})

	msgs := mockAI.LastRequest.Messages
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages after /clear, got %d", len(msgs))
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

func TestEngine_ProcessMessage_StripsMarkdownFormattingFromAIResponse(t *testing.T) {
	mockAI := ai.NewMockProvider("1. **Faham**: Ini konsep asas.\n2. **Rancangan**: Cuba selesaikan langkah demi langkah.\n- **Tip**: Semak jawapan.\n`x = 6`")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-no-markdown",
		Text:    "Tolong ajar persamaan linear",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	if contains(resp, "**") || contains(resp, "`") {
		t.Fatalf("response should not contain markdown formatting, got: %q", resp)
	}
	if contains(resp, "1. ") || contains(resp, "- ") {
		t.Fatalf("response should not keep markdown list markers, got: %q", resp)
	}
	if !contains(resp, "Faham: Ini konsep asas.") {
		t.Fatalf("response should preserve content text, got: %q", resp)
	}
	if !contains(resp, "Tip: Semak jawapan.") {
		t.Fatalf("response should preserve bullet text, got: %q", resp)
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
		CompactThreshold:      100, // high message threshold — won't trigger by count
		CompactTokenThreshold: 200, // low token threshold — triggers by content size
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

func TestEngine_ProcessMessage_LogsCoreEvents(t *testing.T) {
	mockAI := ai.NewMockProvider("AI response")
	eventLogger := agent.NewMemoryEventLogger()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:    mockRouter(mockAI),
		EventLogger: eventLogger,
		Store:       agent.NewMemoryStore(),
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u-1",
		Text:    "Explain linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for len(eventLogger.Events()) < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	events := eventLogger.Events()
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}

	var sessionStarted, messageSent, aiResponse bool
	for _, e := range events {
		switch e.EventType {
		case "session_started":
			sessionStarted = true
		case "message_sent":
			messageSent = true
		case "ai_response":
			aiResponse = true
		}
	}

	if !sessionStarted || !messageSent || !aiResponse {
		t.Fatalf("missing expected events: session_started=%v message_sent=%v ai_response=%v", sessionStarted, messageSent, aiResponse)
	}
}

func TestEngine_ProcessMessage_EventLoggingNonBlocking(t *testing.T) {
	mockAI := ai.NewMockProvider("AI response")
	blockingLogger := &blockingEventLogger{
		started: make(chan struct{}, 8),
		release: make(chan struct{}),
	}

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:    mockRouter(mockAI),
		EventLogger: blockingLogger,
		Store:       agent.NewMemoryStore(),
	})

	done := make(chan struct{})
	go func() {
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram",
			UserID:  "u-2",
			Text:    "What is algebra?",
		})
		close(done)
	}()

	select {
	case <-blockingLogger.started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected event logger to be called")
	}

	select {
	case <-done:
		// expected: ProcessMessage should return even while logger is blocked.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("ProcessMessage should not block on event logger")
	}

	close(blockingLogger.release)
}

func TestEngine_ProcessMessage_PromptsForRatingOnEveryTutoringReply(t *testing.T) {
	mockAI := ai.NewMockProvider("AI response")
	tracker := &callTracker{provider: mockAI}
	eventLogger := agent.NewMemoryEventLogger()
	store := agent.NewMemoryStore()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:          mockRouter(tracker),
		EventLogger:       eventLogger,
		Store:             store,
		RatingPromptEvery: 1,
	})

	userID := "u-rating-prompt"
	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "question 1",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "[[PAI_REVIEW:") {
		t.Fatalf("expected rating prompt on first tutoring reply, got: %q", resp)
	}
	if len(tracker.requests) != 1 {
		t.Fatalf("AI request count = %d, want 1", len(tracker.requests))
	}

	conv, found := store.GetActiveConversation(userID)
	if !found {
		t.Fatal("expected active conversation")
	}
	if len(conv.Messages) == 0 {
		t.Fatal("expected stored messages")
	}
	last := conv.Messages[len(conv.Messages)-1]
	if last.Role != "assistant" || !contains(last.Content, agent.ReviewActionCode) {
		t.Fatalf("expected final message to be rating prompt, got role=%q content=%q", last.Role, last.Content)
	}
	if last.Model == "" {
		t.Fatalf("assistant tutoring reply should retain model metadata")
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var promptLogged bool
	for time.Now().Before(deadline) {
		events := eventLogger.Events()
		for _, e := range events {
			if e.EventType == "answer_rating_requested" {
				promptLogged = true
				break
			}
		}
		if promptLogged {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !promptLogged {
		t.Fatal("expected answer_rating_requested event")
	}
}

func TestEngine_ProcessMessage_ConsumesValidRatingWithoutAICall(t *testing.T) {
	mockAI := ai.NewMockProvider("AI response")
	tracker := &callTracker{provider: mockAI}
	eventLogger := agent.NewMemoryEventLogger()
	store := agent.NewMemoryStore()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:    mockRouter(tracker),
		EventLogger: eventLogger,
		Store:       store,
	})

	userID := "u-rating-submit"
	for i := 1; i <= 5; i++ {
		_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram",
			UserID:  userID,
			Text:    fmt.Sprintf("question %d", i),
		})
		if err != nil {
			t.Fatalf("ProcessMessage() error = %v", err)
		}
	}
	beforeCalls := len(tracker.requests)

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "4",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Terima kasih") {
		t.Fatalf("expected thank-you response for rating, got: %q", resp)
	}
	if len(tracker.requests) != beforeCalls {
		t.Fatalf("AI should not be called for valid rating; calls = %d, want %d", len(tracker.requests), beforeCalls)
	}

	conv, found := store.GetActiveConversation(userID)
	if !found {
		t.Fatal("expected active conversation")
	}
	last := conv.Messages[len(conv.Messages)-1]
	if last.Role != "assistant" || !contains(last.Content, "Terima kasih") {
		t.Fatalf("expected assistant thank-you stored, got role=%q content=%q", last.Role, last.Content)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var submitted *agent.Event
	for time.Now().Before(deadline) {
		for _, e := range eventLogger.Events() {
			if e.EventType == "answer_rating_submitted" {
				eventCopy := e
				submitted = &eventCopy
				break
			}
		}
		if submitted != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if submitted == nil {
		t.Fatal("expected answer_rating_submitted event")
	}
	if got, ok := submitted.Data["rating"].(int); !ok || got != 4 {
		t.Fatalf("rating event payload = %#v, want rating=4", submitted.Data)
	}
}

func TestEngine_ProcessMessage_InvalidNumericRating_FallsBackToTutoring(t *testing.T) {
	mockAI := ai.NewMockProvider("AI response")
	tracker := &callTracker{provider: mockAI}
	eventLogger := agent.NewMemoryEventLogger()
	store := agent.NewMemoryStore()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:    mockRouter(tracker),
		EventLogger: eventLogger,
		Store:       store,
	})

	userID := "u-rating-invalid"
	for i := 1; i <= 5; i++ {
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram",
			UserID:  userID,
			Text:    fmt.Sprintf("question %d", i),
		})
	}
	beforeCalls := len(tracker.requests)

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "7",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if contains(resp, "Terima kasih atas rating anda") {
		t.Fatalf("did not expect rating thanks for invalid numeric rating, got: %q", resp)
	}
	if len(tracker.requests) != beforeCalls+1 {
		t.Fatalf("AI should be called for invalid numeric rating fallback; calls = %d, want %d", len(tracker.requests), beforeCalls+1)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var submitted bool
	var skipped *agent.Event
	for time.Now().Before(deadline) {
		for _, e := range eventLogger.Events() {
			if e.EventType == "answer_rating_submitted" {
				submitted = true
				break
			}
			if e.EventType == "answer_rating_skipped" {
				eventCopy := e
				skipped = &eventCopy
			}
		}
		if submitted || skipped != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if submitted {
		t.Fatal("did not expect answer_rating_submitted event for invalid numeric rating")
	}
	if skipped == nil {
		t.Fatal("expected answer_rating_skipped event for invalid numeric rating")
	}
	if got, ok := skipped.Data["rating_input_kind"].(string); !ok || got != "numeric_out_of_range" {
		t.Fatalf("rating skipped payload = %#v, want rating_input_kind=numeric_out_of_range", skipped.Data)
	}
}

func TestEngine_ProcessMessage_NonRatingAfterPrompt_FallsBackToTutoring(t *testing.T) {
	mockAI := ai.NewMockProvider("AI response")
	tracker := &callTracker{provider: mockAI}
	eventLogger := agent.NewMemoryEventLogger()
	store := agent.NewMemoryStore()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:    mockRouter(tracker),
		EventLogger: eventLogger,
		Store:       store,
	})

	userID := "u-rating-normal-chat"
	for i := 1; i <= 5; i++ {
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram",
			UserID:  userID,
			Text:    fmt.Sprintf("question %d", i),
		})
	}
	beforeCalls := len(tracker.requests)

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "boleh teruskan topik ini?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if contains(resp, "Terima kasih atas rating anda") {
		t.Fatalf("expected normal tutoring response for non-rating text, got: %q", resp)
	}
	if len(tracker.requests) != beforeCalls+1 {
		t.Fatalf("AI should be called for non-rating text; calls = %d, want %d", len(tracker.requests), beforeCalls+1)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var submitted bool
	var skipped *agent.Event
	for time.Now().Before(deadline) {
		for _, e := range eventLogger.Events() {
			if e.EventType == "answer_rating_submitted" {
				submitted = true
				break
			}
			if e.EventType == "answer_rating_skipped" {
				eventCopy := e
				skipped = &eventCopy
			}
		}
		if submitted || skipped != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if submitted {
		t.Fatal("did not expect answer_rating_submitted event for non-rating text")
	}
	if skipped == nil {
		t.Fatal("expected answer_rating_skipped event for non-rating text")
	}
	if got, ok := skipped.Data["rating_input_kind"].(string); !ok || got != "non_rating_text" {
		t.Fatalf("rating skipped payload = %#v, want rating_input_kind=non_rating_text", skipped.Data)
	}
}

func TestEngine_ProcessMessage_DelayedTelegramCallbackRating_SubmitsWithoutAICall(t *testing.T) {
	mockAI := ai.NewMockProvider("AI response")
	tracker := &callTracker{provider: mockAI}
	eventLogger := agent.NewMemoryEventLogger()
	store := agent.NewMemoryStore()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:    mockRouter(tracker),
		EventLogger: eventLogger,
		Store:       store,
	})

	userID := "u-rating-delayed-callback"
	for i := 1; i <= 5; i++ {
		_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram",
			UserID:  userID,
			Text:    fmt.Sprintf("question %d", i),
		})
	}
	conv, found := store.GetActiveConversation(userID)
	if !found || len(conv.Messages) == 0 {
		t.Fatal("expected active conversation with messages")
	}
	ratedMessageID := conv.Messages[len(conv.Messages)-1].ID
	if ratedMessageID == "" {
		t.Fatal("expected rating prompt message to have ID")
	}

	// User continues chatting first; this should consume the rating prompt and call AI.
	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    "teruskan topik ini",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	beforeCallbackCalls := len(tracker.requests)

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:         "telegram",
		UserID:          userID,
		Text:            fmt.Sprintf("rating:%s:4", ratedMessageID),
		CallbackQueryID: "cb-delayed-1",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Terima kasih") {
		t.Fatalf("expected thank-you response for delayed callback rating, got: %q", resp)
	}
	if len(tracker.requests) != beforeCallbackCalls {
		t.Fatalf("AI should not be called for delayed callback rating; calls = %d, want %d", len(tracker.requests), beforeCallbackCalls)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var submitted *agent.Event
	for time.Now().Before(deadline) {
		for _, e := range eventLogger.Events() {
			if e.EventType == "answer_rating_submitted" {
				eventCopy := e
				submitted = &eventCopy
			}
		}
		if submitted != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if submitted == nil {
		t.Fatal("expected answer_rating_submitted event for delayed callback rating")
	}
	if got, ok := submitted.Data["source"].(string); !ok || got != "telegram_inline_button" {
		t.Fatalf("submitted payload = %#v, want source=telegram_inline_button", submitted.Data)
	}
	if got, ok := submitted.Data["delayed_submit"].(bool); !ok || !got {
		t.Fatalf("submitted payload = %#v, want delayed_submit=true", submitted.Data)
	}
	if got, ok := submitted.Data["rated_message_id"].(string); !ok || got != ratedMessageID {
		t.Fatalf("submitted payload = %#v, want rated_message_id=%s", submitted.Data, ratedMessageID)
	}
}

func TestEngine_ImageDataURL_NotPersistedInConversationHistory(t *testing.T) {
	mockAI := ai.NewMockProvider("image response")
	store := agent.NewMemoryStore()

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
		Store:    store,
	})

	imageDataURL := "data:image/jpeg;base64,AAAAABBBBBCCCCCDDDDDEEEEE"
	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:      "telegram",
		UserID:       "img-user",
		Text:         "whats this",
		HasImage:     true,
		ImageDataURL: imageDataURL,
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}

	conv, found := store.GetActiveConversation("img-user")
	if !found {
		t.Fatal("expected active conversation")
	}

	for _, m := range conv.Messages {
		if contains(m.Content, "data:image") || contains(m.Content, "base64,") {
			t.Fatalf("stored message should not contain raw image data URL, got: %q", m.Content)
		}
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
	r := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond},
		BreakerFailureThreshold: 3,
		BreakerCooldown:         10 * time.Millisecond,
	})
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

type blockingEventLogger struct {
	started chan struct{}
	release chan struct{}
}

func (l *blockingEventLogger) LogEvent(_ agent.Event) error {
	select {
	case l.started <- struct{}{}:
	default:
	}
	<-l.release
	return nil
}

type autoStartStore struct {
	*agent.MemoryStore
	known map[string]bool
}

func newAutoStartStore() *autoStartStore {
	return &autoStartStore{
		MemoryStore: agent.NewMemoryStore(),
		known:       map[string]bool{},
	}
}

func (s *autoStartStore) UserExists(userID string) bool {
	return s.known[userID]
}

func (s *autoStartStore) CreateConversation(conv agent.Conversation) (string, error) {
	id, err := s.MemoryStore.CreateConversation(conv)
	if err == nil {
		s.known[conv.UserID] = true
	}
	return id, err
}

type flakyProvider struct {
	failuresBeforeSuccess int
	calls                 int
	response              string
}

func (f *flakyProvider) Complete(_ context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	f.calls++
	if f.calls <= f.failuresBeforeSuccess {
		return ai.CompletionResponse{}, fmt.Errorf("transient provider error")
	}
	return ai.CompletionResponse{
		Content:      f.response,
		Model:        "flaky",
		InputTokens:  1,
		OutputTokens: len(f.response),
	}, nil
}

func (f *flakyProvider) StreamComplete(_ context.Context, _ ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *flakyProvider) Models() []ai.ModelInfo {
	return []ai.ModelInfo{{ID: "flaky", Name: "Flaky", MaxTokens: 1024}}
}

func (f *flakyProvider) HealthCheck(_ context.Context) error {
	return nil
}

type stubContextResolver struct {
	topic    *curriculum.Topic
	notes    string
	called   bool
	lastText string
}

func (r *stubContextResolver) Resolve(text string) (*curriculum.Topic, string) {
	r.called = true
	r.lastText = text
	return r.topic, r.notes
}

func createTestCurriculumLoader(t *testing.T) *curriculum.Loader {
	t.Helper()

	dir := t.TempDir()
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	yamlPath := filepath.Join(topicsDir, "01-linear-equations.yaml")
	yamlData := `id: F1-02
name: Linear Equations
subject_id: math
syllabus_id: kssm-f1
difficulty: beginner
learning_objectives:
  - id: LO1
    text: Solve linear equations in one variable
    bloom: apply
`
	if err := os.WriteFile(yamlPath, []byte(yamlData), 0o644); err != nil {
		t.Fatalf("WriteFile(yaml) error = %v", err)
	}

	notesPath := filepath.Join(topicsDir, "01-linear-equations.teaching.md")
	notes := "# Linear Equations Teaching Notes\nUse balance method and subtract 5 on both sides."
	if err := os.WriteFile(notesPath, []byte(notes), 0o644); err != nil {
		t.Fatalf("WriteFile(notes) error = %v", err)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

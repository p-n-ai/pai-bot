package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestEngine_ProcessMessage_QuizIntentStartsQuizWithoutSlashCommand(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	loader := createTestCurriculumLoader(t)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: loader,
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-1",
		Text:    "quiz me on linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Question 1/3") {
		t.Fatalf("expected first quiz question, got %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called when quiz starts through intent routing")
	}

	conv, found := store.GetActiveConversation("quiz-user-1")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "quiz_active" {
		t.Fatalf("conversation state = %q, want quiz_active", conv.State)
	}
	if conv.QuizState == nil {
		t.Fatal("expected active quiz state")
	}
	if conv.QuizState.TopicID != "F1-02" || conv.QuizState.Intensity != "mixed" || conv.QuizState.CurrentIndex != 0 || conv.QuizState.CorrectAnswers != 0 {
		t.Fatalf("QuizState = %#v, want topic/intensity/index/correct = F1-02/mixed/0/0", conv.QuizState)
	}
}

func TestEngine_ProcessMessage_QuizStartPersistsKnownNameAndFormWithoutBlockingOnIntensity(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	if err := store.SetUserName("quiz-user-profile", "Aina"); err != nil {
		t.Fatalf("SetUserName() error = %v", err)
	}
	if err := store.SetUserForm("quiz-user-profile", "2"); err != nil {
		t.Fatalf("SetUserForm() error = %v", err)
	}
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-profile",
		Text:    "quiz me on linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Question 1/3") {
		t.Fatalf("expected immediate quiz question, got %q", resp)
	}
	name, ok := store.GetUserName("quiz-user-profile")
	if !ok || name != "Aina" {
		t.Fatalf("GetUserName() = %q, %v, want Aina, true", name, ok)
	}
	form, ok := store.GetUserForm("quiz-user-profile")
	if !ok || form != "2" {
		t.Fatalf("GetUserForm() = %q, %v, want 2, true", form, ok)
	}
}

func TestEngine_ProcessMessage_QuizIntensityReplyStoresPreferenceAndStartsQuiz(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	conversationID, err := store.CreateConversation(agent.Conversation{
		UserID: "quiz-user-intensity",
		State:  "teaching",
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationPendingQuiz(conversationID, "quiz_intensity", "F1-02"); err != nil {
		t.Fatalf("UpdateConversationPendingQuiz() error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-intensity",
		Text:    "hard",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Question 1/1") {
		t.Fatalf("expected hard-filtered first quiz question, got %q", resp)
	}
	if !contains(resp, "Solve the linear equation") {
		t.Fatalf("expected hard question content, got %q", resp)
	}

	intensity, ok := store.GetUserPreferredQuizIntensity("quiz-user-intensity")
	if !ok || intensity != "hard" {
		t.Fatalf("stored intensity = %q, %v, want hard, true", intensity, ok)
	}

	conv, found := store.GetActiveConversation("quiz-user-intensity")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "quiz_active" {
		t.Fatalf("conversation state = %q, want quiz_active", conv.State)
	}
	if conv.QuizState == nil {
		t.Fatal("expected active quiz metadata")
	}
	if conv.QuizState.TopicID != "F1-02" || conv.QuizState.Intensity != "hard" || conv.QuizState.CurrentIndex != 0 || conv.QuizState.CorrectAnswers != 0 {
		t.Fatalf("QuizState = %#v, want topic/intensity/index/correct = F1-02/hard/0/0", conv.QuizState)
	}
}

func TestEngine_ProcessMessage_QuizAnswerAdvancesWithoutAICall(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	progressTracker := progress.NewMemoryTracker()
	xpTracker := progress.NewMemoryXPTracker()
	if err := store.SetUserPreferredQuizIntensity("quiz-user-2", "mixed"); err != nil {
		t.Fatalf("SetUserPreferredQuizIntensity() error = %v", err)
	}
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
		Tracker:          progressTracker,
		XP:               xpTracker,
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-2",
		Text:    "give me a quiz on linear equations",
	})
	mockAI.LastRequest = nil

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-2",
		Text:    "4",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Correct") {
		t.Fatalf("expected correct feedback, got %q", resp)
	}
	if !contains(resp, "Question 2/3") {
		t.Fatalf("expected next quiz question, got %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called for deterministic quiz grading")
	}
	time.Sleep(100 * time.Millisecond)

	conv, found := store.GetActiveConversation("quiz-user-2")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "quiz_active" {
		t.Fatalf("conversation state = %q, want quiz_active", conv.State)
	}
	if conv.QuizState == nil {
		t.Fatal("expected active quiz metadata")
	}
	if conv.QuizState.TopicID != "F1-02" || conv.QuizState.Intensity != "mixed" || conv.QuizState.CurrentIndex != 1 || conv.QuizState.CorrectAnswers != 1 {
		t.Fatalf("QuizState = %#v, want topic/intensity/index/correct = F1-02/mixed/1/1", conv.QuizState)
	}
	totalXP, err := xpTracker.GetTotal("quiz-user-2")
	if err != nil {
		t.Fatalf("GetTotal() error = %v", err)
	}
	if totalXP != progress.XPQuizCorrect {
		t.Fatalf("quiz XP total = %d, want %d", totalXP, progress.XPQuizCorrect)
	}
	mastery, err := progressTracker.GetMastery("quiz-user-2", "kssm-f1", "F1-02")
	if err != nil {
		t.Fatalf("GetMastery() error = %v", err)
	}
	if mastery <= 0 {
		t.Fatalf("expected mastery > 0 after correct quiz answer, got %f", mastery)
	}
}

func TestEngine_ProcessMessage_QuizWrongAnswerReturnsHint(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	progressTracker := progress.NewMemoryTracker()
	xpTracker := progress.NewMemoryXPTracker()
	if err := store.SetUserPreferredQuizIntensity("quiz-user-3", "mixed"); err != nil {
		t.Fatalf("SetUserPreferredQuizIntensity() error = %v", err)
	}
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
		Tracker:          progressTracker,
		XP:               xpTracker,
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-3",
		Text:    "uji saya tentang linear equations",
	})
	mockAI.LastRequest = nil

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-3",
		Text:    "10",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Not quite") {
		t.Fatalf("expected incorrect-answer feedback, got %q", resp)
	}
	if !contains(resp, "subtracting 3 from both sides") {
		t.Fatalf("expected hint in response, got %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called for deterministic wrong-answer feedback")
	}
	time.Sleep(100 * time.Millisecond)

	conv, found := store.GetActiveConversation("quiz-user-3")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "quiz_active" {
		t.Fatalf("conversation state = %q, want quiz_active", conv.State)
	}
	if conv.QuizState == nil {
		t.Fatal("expected active quiz metadata")
	}
	if conv.QuizState.TopicID != "F1-02" || conv.QuizState.Intensity != "mixed" || conv.QuizState.CurrentIndex != 0 || conv.QuizState.CorrectAnswers != 0 {
		t.Fatalf("QuizState = %#v, want topic/intensity/index/correct = F1-02/mixed/0/0", conv.QuizState)
	}
	totalXP, err := xpTracker.GetTotal("quiz-user-3")
	if err != nil {
		t.Fatalf("GetTotal() error = %v", err)
	}
	if totalXP != 0 {
		t.Fatalf("quiz XP total = %d, want 0", totalXP)
	}
	mastery, err := progressTracker.GetMastery("quiz-user-3", "kssm-f1", "F1-02")
	if err != nil {
		t.Fatalf("GetMastery() error = %v", err)
	}
	if mastery <= 0 {
		t.Fatalf("expected low-but-present mastery signal after wrong quiz answer, got %f", mastery)
	}
}

func TestEngine_ProcessMessage_QuizCallbackUsesSameRouter(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	if err := store.SetUserPreferredQuizIntensity("quiz-user-4", "mixed"); err != nil {
		t.Fatalf("SetUserPreferredQuizIntensity() error = %v", err)
	}
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:         "telegram",
		UserID:          "quiz-user-4",
		Text:            "quiz:start:F1-02",
		CallbackQueryID: "cb-quiz-start",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Question 1/3") {
		t.Fatalf("expected callback start to use quiz router, got %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called for quiz start callback")
	}
}

func TestEngine_ProcessMessage_RemembersQuizIntensityOnNextStart(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	if err := store.SetUserPreferredQuizIntensity("quiz-user-remember", "hard"); err != nil {
		t.Fatalf("SetUserPreferredQuizIntensity() error = %v", err)
	}
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-remember",
		Text:    "give me a quiz on linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if contains(resp, "What intensity do you want") {
		t.Fatalf("did not expect intensity reprompt, got %q", resp)
	}
	if !contains(resp, "Question 1/1") {
		t.Fatalf("expected remembered hard quiz start, got %q", resp)
	}
}

func TestEngine_ProcessMessage_QuizIntentWithExplicitIntensityStoresPreference(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-explicit",
		Text:    "give me a hard quiz on linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Question 1/1") {
		t.Fatalf("expected hard-filtered first quiz question, got %q", resp)
	}
	intensity, ok := store.GetUserPreferredQuizIntensity("quiz-user-explicit")
	if !ok || intensity != "hard" {
		t.Fatalf("stored intensity = %q, %v, want hard, true", intensity, ok)
	}
}

func TestEngine_ProcessMessage_SideQuestionDuringQuizPausesWithoutGrading(t *testing.T) {
	mockAI := ai.NewMockProvider("The weather looks clear today. Want to continue your quiz after this?")
	store := agent.NewMemoryStore()
	progressTracker := progress.NewMemoryTracker()
	xpTracker := progress.NewMemoryXPTracker()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
		Tracker:          progressTracker,
		XP:               xpTracker,
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-weather",
		Text:    "quiz me on linear equations",
	})
	mockAI.LastRequest = nil

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-weather",
		Text:    "how is the weather today?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "weather") {
		t.Fatalf("expected AI side-conversation reply, got %q", resp)
	}
	if mockAI.LastRequest == nil {
		t.Fatal("AI should be called for side conversation during quiz")
	}

	conv, found := store.GetActiveConversation("quiz-user-weather")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "teaching" {
		t.Fatalf("conversation state = %q, want teaching", conv.State)
	}
	if conv.QuizState == nil {
		t.Fatal("expected paused quiz state")
	}
	if conv.QuizState.RunState != "paused" || conv.QuizState.SuspendedBy != "side_question" {
		t.Fatalf("QuizState = %#v, want paused side-question state", conv.QuizState)
	}
	if conv.QuizState.CurrentIndex != 0 || conv.QuizState.CorrectAnswers != 0 {
		t.Fatalf("QuizState progress = %#v, want unchanged index/correct", conv.QuizState)
	}

	totalXP, err := xpTracker.GetTotal("quiz-user-weather")
	if err != nil {
		t.Fatalf("GetTotal() error = %v", err)
	}
	if totalXP != 0 {
		t.Fatalf("quiz XP total = %d, want 0", totalXP)
	}
}

func TestEngine_ProcessMessage_ContinueQuizResumesPausedQuestionWithoutAICall(t *testing.T) {
	mockAI := ai.NewMockProvider("The weather looks clear today.")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-resume",
		Text:    "quiz me on linear equations",
	})
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-resume",
		Text:    "how is the weather today?",
	})
	mockAI.LastRequest = nil

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-resume",
		Text:    "continue quiz",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Resuming your quiz.") {
		t.Fatalf("expected resume message, got %q", resp)
	}
	if !contains(resp, "Question 1/3") {
		t.Fatalf("expected original question on resume, got %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called when resuming paused quiz")
	}

	conv, found := store.GetActiveConversation("quiz-user-resume")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "quiz_active" {
		t.Fatalf("conversation state = %q, want quiz_active", conv.State)
	}
	if conv.QuizState == nil || conv.QuizState.RunState != "active" {
		t.Fatalf("QuizState = %#v, want resumed active quiz state", conv.QuizState)
	}
}

func TestEngine_ProcessMessage_NewQuizIntentDuringQuizRestartsInsteadOfGrading(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-restart",
		Text:    "give me a hard quiz on linear equations",
	})
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-restart",
		Text:    "19",
	})
	mockAI.LastRequest = nil

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-restart",
		Text:    "give me another quiz on linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Question 1/1") {
		t.Fatalf("expected restarted quiz question, got %q", resp)
	}
	if contains(resp, "Not quite") {
		t.Fatalf("quiz restart should not be graded as wrong answer, got %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called for quiz restart intent")
	}

	conv, found := store.GetActiveConversation("quiz-user-restart")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.QuizState == nil {
		t.Fatal("expected active quiz state")
	}
	if conv.QuizState.CurrentIndex != 0 || conv.QuizState.CorrectAnswers != 0 {
		t.Fatalf("QuizState = %#v, want restarted quiz progress", conv.QuizState)
	}
}

func TestEngine_ProcessMessage_StopExitsQuizWithoutAICall(t *testing.T) {
	mockAI := ai.NewMockProvider("should-not-be-used")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-stop",
		Text:    "quiz me on linear equations",
	})
	mockAI.LastRequest = nil

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-stop",
		Text:    "stop",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "stop the quiz") {
		t.Fatalf("expected quiz exit response, got %q", resp)
	}
	if mockAI.LastRequest != nil {
		t.Fatal("AI should not be called when student exits quiz naturally")
	}

	conv, found := store.GetActiveConversation("quiz-user-stop")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "teaching" {
		t.Fatalf("conversation state = %q, want teaching", conv.State)
	}
	if conv.QuizState != nil {
		t.Fatalf("QuizState = %#v, want nil after exit", conv.QuizState)
	}
}

func TestEngine_ProcessMessage_DontGetItPausesQuizForTeaching(t *testing.T) {
	mockAI := ai.NewMockProvider("Let’s slow down. Think of the equation like a balance first.")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-teach",
		Text:    "quiz me on linear equations",
	})
	mockAI.LastRequest = nil

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "quiz-user-teach",
		Text:    "I don't get it",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "balance") {
		t.Fatalf("expected teaching reply, got %q", resp)
	}
	if mockAI.LastRequest == nil {
		t.Fatal("AI should be called when student asks for teaching help during quiz")
	}

	conv, found := store.GetActiveConversation("quiz-user-teach")
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.State != "teaching" {
		t.Fatalf("conversation state = %q, want teaching", conv.State)
	}
	if conv.QuizState == nil {
		t.Fatal("expected paused quiz state")
	}
	if conv.QuizState.RunState != "paused" || conv.QuizState.SuspendedBy != "teach_first" {
		t.Fatalf("QuizState = %#v, want paused teaching detour", conv.QuizState)
	}
}

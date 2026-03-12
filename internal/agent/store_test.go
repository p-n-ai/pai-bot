package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestConversationStore_Interface(t *testing.T) {
	store := agent.NewMemoryStore()

	conv := agent.Conversation{
		UserID:   "123",
		TopicID:  "F1-01",
		State:    "teaching",
		Messages: []agent.StoredMessage{},
	}

	id, err := store.CreateConversation(conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if id == "" {
		t.Error("CreateConversation() returned empty ID")
	}

	// Add a message
	msgID, err := store.AddMessage(id, agent.StoredMessage{
		Role:    "user",
		Content: "What is algebra?",
	})
	if err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}
	if msgID == "" {
		t.Fatal("AddMessage() returned empty message ID")
	}

	// Get conversation
	got, err := store.GetConversation(id)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if len(got.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(got.Messages))
	}
}

func TestConversationStore_GetActiveForUser(t *testing.T) {
	store := agent.NewMemoryStore()

	conv := agent.Conversation{
		UserID: "123",
		State:  "teaching",
	}
	_, err := store.CreateConversation(conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	active, found := store.GetActiveConversation("123")
	if !found {
		t.Error("GetActiveConversation() should find active conversation")
	}
	if active.UserID != "123" {
		t.Errorf("UserID = %q, want 123", active.UserID)
	}
}

func TestConversationStore_GetActiveForUser_NotFound(t *testing.T) {
	store := agent.NewMemoryStore()

	_, found := store.GetActiveConversation("nonexistent")
	if found {
		t.Error("GetActiveConversation() should not find non-existent user")
	}
}

func TestConversationStore_UserExists(t *testing.T) {
	store := agent.NewMemoryStore()

	_, err := store.CreateConversation(agent.Conversation{
		UserID: "u-100",
		State:  "teaching",
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	if !store.UserExists("u-100") {
		t.Fatal("UserExists() = false, want true")
	}
}

func TestConversationStore_UserExists_NotFound(t *testing.T) {
	store := agent.NewMemoryStore()

	if store.UserExists("missing-user") {
		t.Fatal("UserExists() = true, want false")
	}
}

func TestConversationStore_QuizIntensityPreference(t *testing.T) {
	store := agent.NewMemoryStore()

	if err := store.SetUserPreferredQuizIntensity("u-quiz-pref", "hard"); err != nil {
		t.Fatalf("SetUserPreferredQuizIntensity() error = %v", err)
	}

	got, ok := store.GetUserPreferredQuizIntensity("u-quiz-pref")
	if !ok {
		t.Fatal("GetUserPreferredQuizIntensity() = false, want true")
	}
	if got != "hard" {
		t.Fatalf("quiz intensity = %q, want hard", got)
	}
}

func TestConversationStore_UserProfileNameAndForm(t *testing.T) {
	store := agent.NewMemoryStore()

	if err := store.SetUserName("u-profile", "Aina"); err != nil {
		t.Fatalf("SetUserName() error = %v", err)
	}
	if err := store.SetUserForm("u-profile", "2"); err != nil {
		t.Fatalf("SetUserForm() error = %v", err)
	}

	name, ok := store.GetUserName("u-profile")
	if !ok || name != "Aina" {
		t.Fatalf("GetUserName() = %q, %v, want Aina, true", name, ok)
	}
	form, ok := store.GetUserForm("u-profile")
	if !ok || form != "2" {
		t.Fatalf("GetUserForm() = %q, %v, want 2, true", form, ok)
	}
}

func TestConversationStore_EndConversation(t *testing.T) {
	store := agent.NewMemoryStore()

	id, _ := store.CreateConversation(agent.Conversation{
		UserID: "123",
		State:  "teaching",
	})

	err := store.EndConversation(id)
	if err != nil {
		t.Fatalf("EndConversation() error = %v", err)
	}

	// Should no longer be active
	_, found := store.GetActiveConversation("123")
	if found {
		t.Error("GetActiveConversation() should not find ended conversation")
	}
}

func TestConversationStore_MultipleMessages(t *testing.T) {
	store := agent.NewMemoryStore()

	id, _ := store.CreateConversation(agent.Conversation{
		UserID: "123",
		State:  "teaching",
	})

	_, _ = store.AddMessage(id, agent.StoredMessage{Role: "user", Content: "Hello"})
	_, _ = store.AddMessage(id, agent.StoredMessage{Role: "assistant", Content: "Hi!"})
	_, _ = store.AddMessage(id, agent.StoredMessage{Role: "user", Content: "What is x?"})

	got, _ := store.GetConversation(id)
	if len(got.Messages) != 3 {
		t.Errorf("Messages count = %d, want 3", len(got.Messages))
	}
}

func TestConversationStore_SetSummary(t *testing.T) {
	store := agent.NewMemoryStore()

	id, _ := store.CreateConversation(agent.Conversation{
		UserID: "123",
		State:  "teaching",
	})

	err := store.SetSummary(id, "Student learned about algebra basics.", 10)
	if err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}

	got, _ := store.GetConversation(id)
	if got.Summary != "Student learned about algebra basics." {
		t.Errorf("Summary = %q, want 'Student learned about algebra basics.'", got.Summary)
	}
	if got.CompactedAt != 10 {
		t.Errorf("CompactedAt = %d, want 10", got.CompactedAt)
	}
}

func TestConversationStore_SetSummary_NotFound(t *testing.T) {
	store := agent.NewMemoryStore()

	err := store.SetSummary("nonexistent", "summary", 5)
	if err == nil {
		t.Error("SetSummary() should error for non-existent conversation")
	}
}

func TestConversationStore_UpdateConversationState(t *testing.T) {
	store := agent.NewMemoryStore()
	id, _ := store.CreateConversation(agent.Conversation{
		UserID: "123",
		State:  "onboarding",
	})

	if err := store.UpdateConversationState(id, "teaching"); err != nil {
		t.Fatalf("UpdateConversationState() error = %v", err)
	}

	got, err := store.GetConversation(id)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if got.State != "teaching" {
		t.Fatalf("State = %q, want teaching", got.State)
	}
}

func TestConversationStore_UpdateConversationState_NotFound(t *testing.T) {
	store := agent.NewMemoryStore()

	err := store.UpdateConversationState("nonexistent", "teaching")
	if err == nil {
		t.Error("UpdateConversationState() should error for non-existent conversation")
	}
}

func TestConversationStore_UpdateConversationQuizState_PreservesPausedQuizOutsideQuizMode(t *testing.T) {
	store := agent.NewMemoryStore()
	id, _ := store.CreateConversation(agent.Conversation{
		UserID: "u-paused-quiz",
		State:  "quiz_active",
	})

	err := store.UpdateConversationQuizState(id, "teaching", agent.ConversationQuizState{
		TopicID:        "F1-02",
		Intensity:      "mixed",
		CurrentIndex:   1,
		CorrectAnswers: 1,
		RunState:       "paused",
		SuspendedBy:    "side_question",
	})
	if err != nil {
		t.Fatalf("UpdateConversationQuizState() error = %v", err)
	}

	got, err := store.GetConversation(id)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if got.State != "teaching" {
		t.Fatalf("State = %q, want teaching", got.State)
	}
	if got.QuizState == nil {
		t.Fatal("QuizState = nil, want preserved paused quiz state")
	}
	if got.QuizState.RunState != "paused" || got.QuizState.SuspendedBy != "side_question" {
		t.Fatalf("QuizState = %#v, want paused side-question state", got.QuizState)
	}
}

func TestConversationStore_AddMessage_NotFound(t *testing.T) {
	store := agent.NewMemoryStore()

	_, err := store.AddMessage("nonexistent", agent.StoredMessage{Role: "user", Content: "Hello"})
	if err == nil {
		t.Error("AddMessage() should error for non-existent conversation")
	}
}

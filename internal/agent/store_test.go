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
	err = store.AddMessage(id, agent.StoredMessage{
		Role:    "user",
		Content: "What is algebra?",
	})
	if err != nil {
		t.Fatalf("AddMessage() error = %v", err)
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

	_ = store.AddMessage(id, agent.StoredMessage{Role: "user", Content: "Hello"})
	_ = store.AddMessage(id, agent.StoredMessage{Role: "assistant", Content: "Hi!"})
	_ = store.AddMessage(id, agent.StoredMessage{Role: "user", Content: "What is x?"})

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

func TestConversationStore_AddMessage_NotFound(t *testing.T) {
	store := agent.NewMemoryStore()

	err := store.AddMessage("nonexistent", agent.StoredMessage{Role: "user", Content: "Hello"})
	if err == nil {
		t.Error("AddMessage() should error for non-existent conversation")
	}
}

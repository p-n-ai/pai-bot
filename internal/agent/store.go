package agent

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// StoredMessage represents a single message in a conversation.
type StoredMessage struct {
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	Model        string    `json:"model,omitempty"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Conversation represents a teaching conversation session.
type Conversation struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	TopicID         string          `json:"topic_id,omitempty"`
	State           string          `json:"state"`
	Messages        []StoredMessage `json:"messages"`
	Summary         string          `json:"summary,omitempty"`
	CompactedAt     int             `json:"compacted_at,omitempty"` // number of messages included in Summary
	StartedAt       time.Time       `json:"started_at"`
	EndedAt         *time.Time      `json:"ended_at,omitempty"`
}

// ConversationStore persists conversation state and message history.
type ConversationStore interface {
	CreateConversation(conv Conversation) (string, error)
	GetConversation(id string) (*Conversation, error)
	GetActiveConversation(userID string) (*Conversation, bool)
	AddMessage(conversationID string, msg StoredMessage) error
	SetSummary(conversationID string, summary string, compactedAt int) error
	EndConversation(id string) error
}

// MemoryStore is an in-memory implementation of ConversationStore.
type MemoryStore struct {
	conversations map[string]*Conversation
	mu            sync.RWMutex
}

// NewMemoryStore creates a new in-memory conversation store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(map[string]*Conversation),
	}
}

func (s *MemoryStore) CreateConversation(conv Conversation) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateID()
	conv.ID = id
	conv.StartedAt = time.Now()
	if conv.Messages == nil {
		conv.Messages = []StoredMessage{}
	}
	s.conversations[id] = &conv
	return id, nil
}

func (s *MemoryStore) GetConversation(id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[id]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}
	return conv, nil
}

func (s *MemoryStore) GetActiveConversation(userID string) (*Conversation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conv := range s.conversations {
		if conv.UserID == userID && conv.EndedAt == nil {
			return conv, true
		}
	}
	return nil, false
}

func (s *MemoryStore) AddMessage(conversationID string, msg StoredMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	conv.Messages = append(conv.Messages, msg)
	return nil
}

func (s *MemoryStore) SetSummary(conversationID string, summary string, compactedAt int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	conv.Summary = summary
	conv.CompactedAt = compactedAt
	return nil
}

func (s *MemoryStore) EndConversation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[id]
	if !ok {
		return fmt.Errorf("conversation not found: %s", id)
	}
	now := time.Now()
	conv.EndedAt = &now
	return nil
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

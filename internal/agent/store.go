package agent

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"
)

// StoredMessage represents a single message in a conversation.
type StoredMessage struct {
	ID           string    `json:"id,omitempty"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	Model        string    `json:"model,omitempty"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ConversationQuizState is the persisted runtime state for an active quiz.
type ConversationQuizState struct {
	TopicID        string `json:"topic_id"`
	Intensity      string `json:"intensity"`
	CurrentIndex   int    `json:"current_index"`
	CorrectAnswers int    `json:"correct_answers"`
	RunState       string `json:"run_state,omitempty"`
	SuspendedBy    string `json:"suspended_by,omitempty"`
}

// PendingGoalDraft stores a suggested goal awaiting confirmation.
type PendingGoalDraft struct {
	Summary       string  `json:"summary"`
	TopicID       string  `json:"topic_id"`
	TopicName     string  `json:"topic_name"`
	SyllabusID    string  `json:"syllabus_id"`
	TargetMastery float64 `json:"target_mastery"`
}

// Conversation represents a teaching conversation session.
type Conversation struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	TopicID            string                 `json:"topic_id,omitempty"`
	State              string                 `json:"state"`
	Messages           []StoredMessage        `json:"messages"`
	Summary            string                 `json:"summary,omitempty"`
	CompactedAt        int                    `json:"compacted_at,omitempty"` // number of messages included in Summary
	PendingQuizTopicID string                 `json:"pending_quiz_topic_id,omitempty"`
	QuizState          *ConversationQuizState `json:"quiz_state,omitempty"`
	PendingGoal        *PendingGoalDraft      `json:"pending_goal,omitempty"`
	StartedAt          time.Time              `json:"started_at"`
	EndedAt            *time.Time             `json:"ended_at,omitempty"`
}

// ConversationStore persists conversation state and message history.
type ConversationStore interface {
	UserExists(userID string) bool
	GetUserName(userID string) (string, bool)
	SetUserName(userID, name string) error
	GetUserForm(userID string) (string, bool)
	SetUserForm(userID, form string) error
	GetUserPreferredLanguage(userID string) (string, bool)
	SetUserPreferredLanguage(userID, lang string) error
	GetUserPreferredQuizIntensity(userID string) (string, bool)
	SetUserPreferredQuizIntensity(userID, intensity string) error
	CreateConversation(conv Conversation) (string, error)
	GetConversation(id string) (*Conversation, error)
	GetActiveConversation(userID string) (*Conversation, bool)
	AddMessage(conversationID string, msg StoredMessage) (string, error)
	SetSummary(conversationID string, summary string, compactedAt int) error
	UpdateConversationState(conversationID string, state string) error
	UpdateConversationTopicID(conversationID, topicID string) error
	UpdateConversationPendingQuiz(conversationID, state, topicID string) error
	UpdateConversationQuizState(conversationID, state string, quizState ConversationQuizState) error
	ClearConversationQuizState(conversationID, state string) error
	SetConversationPendingGoal(conversationID string, goal PendingGoalDraft) error
	ClearConversationPendingGoal(conversationID string) error
	EndConversation(id string) error
}

// MemoryStore is an in-memory implementation of ConversationStore.
type MemoryStore struct {
	conversations map[string]*Conversation
	userName      map[string]string
	userForm      map[string]string
	userLang      map[string]string
	userQuizLevel map[string]string
	mu            sync.RWMutex
}

// NewMemoryStore creates a new in-memory conversation store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(map[string]*Conversation),
		userName:      make(map[string]string),
		userForm:      make(map[string]string),
		userLang:      make(map[string]string),
		userQuizLevel: make(map[string]string),
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

func (s *MemoryStore) UserExists(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.userName[userID]; ok {
		return true
	}
	if _, ok := s.userForm[userID]; ok {
		return true
	}
	if _, ok := s.userLang[userID]; ok {
		return true
	}
	if _, ok := s.userQuizLevel[userID]; ok {
		return true
	}
	for _, conv := range s.conversations {
		if conv.UserID == userID {
			return true
		}
	}
	return false
}

func (s *MemoryStore) GetUserName(userID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	name, ok := s.userName[userID]
	return name, ok
}

func (s *MemoryStore) SetUserName(userID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		delete(s.userName, userID)
		return nil
	}
	s.userName[userID] = name
	return nil
}

func (s *MemoryStore) GetUserForm(userID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	form, ok := s.userForm[userID]
	return form, ok
}

func (s *MemoryStore) SetUserForm(userID, form string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	form = strings.TrimSpace(form)
	if form == "" {
		delete(s.userForm, userID)
		return nil
	}
	s.userForm[userID] = form
	return nil
}

func (s *MemoryStore) GetUserPreferredLanguage(userID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lang, ok := s.userLang[userID]
	return lang, ok
}

func (s *MemoryStore) SetUserPreferredLanguage(userID, lang string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if lang == "" {
		delete(s.userLang, userID)
		return nil
	}
	s.userLang[userID] = lang
	return nil
}

func (s *MemoryStore) GetUserPreferredQuizIntensity(userID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	intensity, ok := s.userQuizLevel[userID]
	return intensity, ok
}

func (s *MemoryStore) SetUserPreferredQuizIntensity(userID, intensity string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if intensity == "" {
		delete(s.userQuizLevel, userID)
		return nil
	}
	s.userQuizLevel[userID] = intensity
	return nil
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

func (s *MemoryStore) AddMessage(conversationID string, msg StoredMessage) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return "", fmt.Errorf("conversation not found: %s", conversationID)
	}
	if msg.ID == "" {
		msg.ID = generateID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	conv.Messages = append(conv.Messages, msg)
	return msg.ID, nil
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

func (s *MemoryStore) UpdateConversationState(conversationID string, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	if state == "" {
		return fmt.Errorf("state is required")
	}
	conv.State = state
	return nil
}

func (s *MemoryStore) UpdateConversationTopicID(conversationID, topicID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	conv.TopicID = topicID
	return nil
}

func (s *MemoryStore) UpdateConversationPendingQuiz(conversationID, state, topicID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	if state == "" {
		return fmt.Errorf("state is required")
	}
	conv.State = state
	conv.PendingQuizTopicID = strings.TrimSpace(topicID)
	conv.QuizState = nil
	conv.PendingGoal = nil
	return nil
}

func (s *MemoryStore) UpdateConversationQuizState(conversationID, state string, quizState ConversationQuizState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	if state == "" {
		return fmt.Errorf("state is required")
	}
	conv.State = state
	conv.PendingQuizTopicID = ""
	stateCopy := quizState
	conv.QuizState = &stateCopy
	conv.PendingGoal = nil
	return nil
}

func (s *MemoryStore) ClearConversationQuizState(conversationID, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	if state == "" {
		return fmt.Errorf("state is required")
	}
	conv.State = state
	conv.PendingQuizTopicID = ""
	conv.QuizState = nil
	return nil
}

func (s *MemoryStore) SetConversationPendingGoal(conversationID string, goal PendingGoalDraft) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	draft := goal
	conv.PendingGoal = &draft
	return nil
}

func (s *MemoryStore) ClearConversationPendingGoal(conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	conv.PendingGoal = nil
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
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

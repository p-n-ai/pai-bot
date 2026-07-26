// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ErrActiveConversationExists reports that a learner already has an active
// conversation in the requested provider thread.
var ErrActiveConversationExists = errors.New("active conversation already exists")

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
	TopicID            string         `json:"topic_id"`
	Intensity          string         `json:"intensity"`
	CurrentIndex       int            `json:"current_index"`
	CorrectAnswers     int            `json:"correct_answers"`
	RunState           string         `json:"run_state,omitempty"`
	SuspendedBy        string         `json:"suspended_by,omitempty"`
	GeneratedQuestions []QuizQuestion `json:"generated_questions,omitempty"`
}

// ConversationChallengeState is the persisted runtime state for an active challenge.
type ConversationChallengeState struct {
	ChallengeID   string                  `json:"challenge_id"`
	TopicID       string                  `json:"topic_id,omitempty"`
	Phase         string                  `json:"phase"` // "playing", "review_offered", "reviewing"
	Questions     []QuizQuestion          `json:"questions"`
	CurrentIndex  int                     `json:"current_index"`
	CorrectCount  int                     `json:"correct_count"`
	Answers       []ChallengeAnswerRecord `json:"answers"`
	MissedIndices []int                   `json:"missed_indices,omitempty"`
	ReviewIndex   int                     `json:"review_index,omitempty"`
	ReviewCorrect int                     `json:"review_correct,omitempty"`
}

// ChallengeAnswerRecord records one answer attempt during a challenge.
type ChallengeAnswerRecord struct {
	QuestionIndex int    `json:"question_index"`
	UserAnswer    string `json:"user_answer"`
	Correct       bool   `json:"correct"`
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
	ID                 string                      `json:"id"`
	UserID             string                      `json:"user_id"`
	Channel            string                      `json:"channel,omitempty"`
	ThreadID           string                      `json:"thread_id,omitempty"`
	TopicID            string                      `json:"topic_id,omitempty"`
	State              string                      `json:"state"`
	Messages           []StoredMessage             `json:"messages"`
	Summary            string                      `json:"summary,omitempty"`
	CompactedAt        int                         `json:"compacted_at,omitempty"` // number of messages included in Summary
	PendingQuizTopicID string                      `json:"pending_quiz_topic_id,omitempty"`
	QuizState          *ConversationQuizState      `json:"quiz_state,omitempty"`
	PendingGoal        *PendingGoalDraft           `json:"pending_goal,omitempty"`
	ChallengeState     *ConversationChallengeState `json:"challenge_state,omitempty"`
	StartedAt          time.Time                   `json:"started_at"`
	EndedAt            *time.Time                  `json:"ended_at,omitempty"`
}

// LearnerIdentity identifies one learner within one chat provider.
type LearnerIdentity struct {
	channel    string
	externalID string
}

// NewLearnerIdentity creates a channel-qualified external learner identity.
func NewLearnerIdentity(channel, externalID string) (LearnerIdentity, error) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return LearnerIdentity{}, fmt.Errorf("channel is required")
	}
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return LearnerIdentity{}, fmt.Errorf("external_id is required")
	}
	return LearnerIdentity{channel: channel, externalID: externalID}, nil
}

// Channel returns the chat provider name.
func (i LearnerIdentity) Channel() string { return i.channel }

// ExternalID returns the provider-owned learner ID.
func (i LearnerIdentity) ExternalID() string { return i.externalID }

func (i LearnerIdentity) validate() error {
	if i.channel == "" {
		return fmt.Errorf("channel is required")
	}
	if i.externalID == "" {
		return fmt.Errorf("external_id is required")
	}
	return nil
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
	GetUserABGroup(userID string) (string, bool)
	SetUserABGroup(userID, group string) error
	UserChannel(externalID string) (string, bool)
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
	UpdateConversationChallengeState(conversationID, state string, challengeState ConversationChallengeState) error
	ClearConversationChallengeState(conversationID, state string) error
	EndConversation(id string) error
	// ResolveUserUUID maps an external chat ID to an internal users.id UUID.
	// Returns ("", nil) if the user does not exist.
	ResolveUserUUID(externalID string) (string, error)
}

// IdentityConversationStore exposes user-keyed persistence without losing the
// provider namespace. Conversation-ID methods remain on ConversationStore.
type IdentityConversationStore interface {
	UserExistsFor(identity LearnerIdentity) bool
	GetUserNameFor(identity LearnerIdentity) (string, bool)
	SetUserNameFor(identity LearnerIdentity, name string) error
	GetUserFormFor(identity LearnerIdentity) (string, bool)
	SetUserFormFor(identity LearnerIdentity, form string) error
	GetUserPreferredLanguageFor(identity LearnerIdentity) (string, bool)
	SetUserPreferredLanguageFor(identity LearnerIdentity, lang string) error
	GetUserPreferredQuizIntensityFor(identity LearnerIdentity) (string, bool)
	SetUserPreferredQuizIntensityFor(identity LearnerIdentity, intensity string) error
	GetUserABGroupFor(identity LearnerIdentity) (string, bool)
	SetUserABGroupFor(identity LearnerIdentity, group string) error
	CreateConversationFor(identity LearnerIdentity, conv Conversation) (string, error)
	GetActiveConversationFor(identity LearnerIdentity) (*Conversation, bool)
	CreateConversationForThread(identity LearnerIdentity, threadID string, conv Conversation) (string, error)
	GetActiveConversationForThread(identity LearnerIdentity, threadID string) (*Conversation, bool)
	GetLatestActiveConversationFor(identity LearnerIdentity) (*Conversation, bool)
	ResolveUserUUIDFor(identity LearnerIdentity) (string, error)
}

// MemoryStore is an in-memory implementation of ConversationStore.
type MemoryStore struct {
	conversations map[string]*Conversation
	userUUID      map[LearnerIdentity]string
	userName      map[LearnerIdentity]string
	userForm      map[LearnerIdentity]string
	userLang      map[LearnerIdentity]string
	userQuizLevel map[LearnerIdentity]string
	userABGroup   map[LearnerIdentity]string
	mu            sync.RWMutex
}

// NewMemoryStore creates a new in-memory conversation store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(map[string]*Conversation),
		userUUID:      make(map[LearnerIdentity]string),
		userName:      make(map[LearnerIdentity]string),
		userForm:      make(map[LearnerIdentity]string),
		userLang:      make(map[LearnerIdentity]string),
		userQuizLevel: make(map[LearnerIdentity]string),
		userABGroup:   make(map[LearnerIdentity]string),
	}
}

func (s *MemoryStore) CreateConversation(conv Conversation) (string, error) {
	channel := strings.TrimSpace(conv.Channel)
	if channel == "" {
		channel = defaultChannel
	}
	identity, err := NewLearnerIdentity(channel, conv.UserID)
	if err != nil {
		return "", err
	}
	return s.CreateConversationForThread(identity, conv.ThreadID, conv)
}

func (s *MemoryStore) CreateConversationFor(identity LearnerIdentity, conv Conversation) (string, error) {
	return s.CreateConversationForThread(identity, "", conv)
}

func (s *MemoryStore) CreateConversationForThread(identity LearnerIdentity, threadID string, conv Conversation) (string, error) {
	if err := identity.validate(); err != nil {
		return "", err
	}
	if conv.UserID != "" && conv.UserID != identity.externalID {
		return "", fmt.Errorf("conversation user_id does not match learner identity")
	}
	if conv.Channel != "" && conv.Channel != identity.channel {
		return "", fmt.Errorf("conversation channel does not match learner identity")
	}
	if conv.ThreadID != "" && conv.ThreadID != threadID {
		return "", fmt.Errorf("conversation thread_id does not match requested thread")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, active := range s.conversations {
		if active.UserID == identity.externalID &&
			active.Channel == identity.channel &&
			active.ThreadID == threadID &&
			active.EndedAt == nil {
			return "", fmt.Errorf("%w: %s", ErrActiveConversationExists, active.ID)
		}
	}

	id := generateID()
	conv.ID = id
	conv.UserID = identity.externalID
	conv.Channel = identity.channel
	conv.ThreadID = threadID
	conv.StartedAt = time.Now()
	if conv.Messages == nil {
		conv.Messages = []StoredMessage{}
	}
	s.ensureUserLocked(identity)
	s.conversations[id] = &conv
	return id, nil
}

func (s *MemoryStore) UserExists(userID string) bool {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return false
	}
	return s.UserExistsFor(identity)
}

func (s *MemoryStore) UserExistsFor(identity LearnerIdentity) bool {
	if identity.validate() != nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.userUUID[identity]; ok {
		return true
	}
	for _, conv := range s.conversations {
		if conv.UserID == identity.externalID && conv.Channel == identity.channel {
			return true
		}
	}
	return false
}

func (s *MemoryStore) GetUserName(userID string) (string, bool) {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return "", false
	}
	return s.GetUserNameFor(identity)
}

func (s *MemoryStore) GetUserNameFor(identity LearnerIdentity) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	name, ok := s.userName[identity]
	return name, ok
}

func (s *MemoryStore) SetUserName(userID, name string) error {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return err
	}
	return s.SetUserNameFor(identity, name)
}

func (s *MemoryStore) SetUserNameFor(identity LearnerIdentity, name string) error {
	if err := identity.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" {
		delete(s.userName, identity)
		return nil
	}
	s.ensureUserLocked(identity)
	s.userName[identity] = name
	return nil
}

func (s *MemoryStore) GetUserForm(userID string) (string, bool) {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return "", false
	}
	return s.GetUserFormFor(identity)
}

func (s *MemoryStore) GetUserFormFor(identity LearnerIdentity) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	form, ok := s.userForm[identity]
	return form, ok
}

func (s *MemoryStore) SetUserForm(userID, form string) error {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return err
	}
	return s.SetUserFormFor(identity, form)
}

func (s *MemoryStore) SetUserFormFor(identity LearnerIdentity, form string) error {
	if err := identity.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	form = strings.TrimSpace(form)
	if form == "" {
		delete(s.userForm, identity)
		return nil
	}
	s.ensureUserLocked(identity)
	s.userForm[identity] = form
	return nil
}

func (s *MemoryStore) GetUserPreferredLanguage(userID string) (string, bool) {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return "", false
	}
	return s.GetUserPreferredLanguageFor(identity)
}

func (s *MemoryStore) GetUserPreferredLanguageFor(identity LearnerIdentity) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lang, ok := s.userLang[identity]
	return lang, ok
}

func (s *MemoryStore) SetUserPreferredLanguage(userID, lang string) error {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return err
	}
	return s.SetUserPreferredLanguageFor(identity, lang)
}

func (s *MemoryStore) SetUserPreferredLanguageFor(identity LearnerIdentity, lang string) error {
	if err := identity.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if lang == "" {
		delete(s.userLang, identity)
		return nil
	}
	s.ensureUserLocked(identity)
	s.userLang[identity] = lang
	return nil
}

func (s *MemoryStore) GetUserPreferredQuizIntensity(userID string) (string, bool) {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return "", false
	}
	return s.GetUserPreferredQuizIntensityFor(identity)
}

func (s *MemoryStore) GetUserPreferredQuizIntensityFor(identity LearnerIdentity) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	intensity, ok := s.userQuizLevel[identity]
	return intensity, ok
}

func (s *MemoryStore) SetUserPreferredQuizIntensity(userID, intensity string) error {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return err
	}
	return s.SetUserPreferredQuizIntensityFor(identity, intensity)
}

func (s *MemoryStore) SetUserPreferredQuizIntensityFor(identity LearnerIdentity, intensity string) error {
	if err := identity.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if intensity == "" {
		delete(s.userQuizLevel, identity)
		return nil
	}
	s.ensureUserLocked(identity)
	s.userQuizLevel[identity] = intensity
	return nil
}

func (s *MemoryStore) GetUserABGroup(userID string) (string, bool) {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return "", false
	}
	return s.GetUserABGroupFor(identity)
}

func (s *MemoryStore) GetUserABGroupFor(identity LearnerIdentity) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	group, ok := s.userABGroup[identity]
	return group, ok
}

func (s *MemoryStore) SetUserABGroup(userID, group string) error {
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return err
	}
	return s.SetUserABGroupFor(identity, group)
}

func (s *MemoryStore) SetUserABGroupFor(identity LearnerIdentity, group string) error {
	if err := identity.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if group == "" {
		delete(s.userABGroup, identity)
		return nil
	}
	s.ensureUserLocked(identity)
	s.userABGroup[identity] = group
	return nil
}

func (s *MemoryStore) UserChannel(externalID string) (string, bool) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return "", false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	channel := ""
	for identity := range s.userUUID {
		if identity.externalID != externalID {
			continue
		}
		if channel != "" && channel != identity.channel {
			return "", false
		}
		channel = identity.channel
	}
	for _, conv := range s.conversations {
		if conv.UserID != externalID {
			continue
		}
		if channel != "" && channel != conv.Channel {
			return "", false
		}
		channel = conv.Channel
	}
	return channel, channel != ""
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
	identity, err := NewLearnerIdentity(defaultChannel, userID)
	if err != nil {
		return nil, false
	}
	return s.GetActiveConversationFor(identity)
}

func (s *MemoryStore) GetActiveConversationFor(identity LearnerIdentity) (*Conversation, bool) {
	return s.GetActiveConversationForThread(identity, "")
}

func (s *MemoryStore) GetActiveConversationForThread(identity LearnerIdentity, threadID string) (*Conversation, bool) {
	if identity.validate() != nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conv := range s.conversations {
		if conv.UserID == identity.externalID &&
			conv.Channel == identity.channel &&
			conv.ThreadID == threadID &&
			conv.EndedAt == nil {
			return conv, true
		}
	}
	return nil, false
}

func (s *MemoryStore) GetLatestActiveConversationFor(identity LearnerIdentity) (*Conversation, bool) {
	if identity.validate() != nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *Conversation
	for _, conv := range s.conversations {
		if conv.UserID != identity.externalID ||
			conv.Channel != identity.channel ||
			conv.EndedAt != nil {
			continue
		}
		if latest == nil || conv.StartedAt.After(latest.StartedAt) {
			latest = conv
		}
	}
	return latest, latest != nil
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

func (s *MemoryStore) UpdateConversationChallengeState(conversationID, state string, challengeState ConversationChallengeState) error {
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
	stateCopy := challengeState
	conv.ChallengeState = &stateCopy
	return nil
}

func (s *MemoryStore) ClearConversationChallengeState(conversationID, state string) error {
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
	conv.ChallengeState = nil
	return nil
}

func (s *MemoryStore) ResolveUserUUID(externalID string) (string, error) {
	if strings.TrimSpace(externalID) == "" {
		return "", fmt.Errorf("external_id is required")
	}
	// Preserve the legacy in-memory store contract for current group-store callers.
	return externalID, nil
}

func (s *MemoryStore) ResolveUserUUIDFor(identity LearnerIdentity) (string, error) {
	if err := identity.validate(); err != nil {
		return "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userUUID[identity], nil
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

func (s *MemoryStore) ensureUserLocked(identity LearnerIdentity) {
	if _, ok := s.userUUID[identity]; !ok {
		s.userUUID[identity] = generateID()
	}
}

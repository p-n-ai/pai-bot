// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

var (
	_ agent.IdentityConversationStore = (*agent.MemoryStore)(nil)
	_ agent.IdentityConversationStore = (*agent.PostgresStore)(nil)
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

func TestConversationStore_UserChannel(t *testing.T) {
	store := agent.NewMemoryStore()
	if _, ok := store.UserChannel("u-channel"); ok {
		t.Fatal("UserChannel() hit before user exists, want miss")
	}

	if err := store.SetUserName("u-channel", "Channel User"); err != nil {
		t.Fatalf("SetUserName() error = %v", err)
	}
	channel, ok := store.UserChannel("u-channel")
	if !ok || channel != "telegram" {
		t.Fatalf("UserChannel() = %q, %v, want telegram, true", channel, ok)
	}
}

func TestIdentityConversationStore_IsolatesSameExternalIDAcrossChannels(t *testing.T) {
	store := agent.NewMemoryStore()
	telegram, err := agent.NewLearnerIdentity("telegram", "shared-user")
	if err != nil {
		t.Fatalf("NewLearnerIdentity(telegram) error = %v", err)
	}
	slack, err := agent.NewLearnerIdentity("slack", "shared-user")
	if err != nil {
		t.Fatalf("NewLearnerIdentity(slack) error = %v", err)
	}

	if err := store.SetUserNameFor(telegram, "Aina on Telegram"); err != nil {
		t.Fatalf("SetUserNameFor(telegram) error = %v", err)
	}
	if err := store.SetUserNameFor(slack, "Aina on Slack"); err != nil {
		t.Fatalf("SetUserNameFor(slack) error = %v", err)
	}
	if _, err := store.CreateConversationFor(telegram, agent.Conversation{State: "teaching"}); err != nil {
		t.Fatalf("CreateConversationFor(telegram) error = %v", err)
	}
	if _, err := store.CreateConversationFor(slack, agent.Conversation{State: "teaching"}); err != nil {
		t.Fatalf("CreateConversationFor(slack) error = %v", err)
	}

	if got, ok := store.GetUserNameFor(telegram); !ok || got != "Aina on Telegram" {
		t.Fatalf("GetUserNameFor(telegram) = %q, %v, want Aina on Telegram, true", got, ok)
	}
	if got, ok := store.GetUserNameFor(slack); !ok || got != "Aina on Slack" {
		t.Fatalf("GetUserNameFor(slack) = %q, %v, want Aina on Slack, true", got, ok)
	}
	telegramConversation, ok := store.GetActiveConversationFor(telegram)
	if !ok || telegramConversation.Channel != "telegram" {
		t.Fatalf("GetActiveConversationFor(telegram) = %#v, %v, want telegram conversation", telegramConversation, ok)
	}
	slackConversation, ok := store.GetActiveConversationFor(slack)
	if !ok || slackConversation.Channel != "slack" {
		t.Fatalf("GetActiveConversationFor(slack) = %#v, %v, want slack conversation", slackConversation, ok)
	}
	telegramUUID, err := store.ResolveUserUUIDFor(telegram)
	if err != nil {
		t.Fatalf("ResolveUserUUIDFor(telegram) error = %v", err)
	}
	slackUUID, err := store.ResolveUserUUIDFor(slack)
	if err != nil {
		t.Fatalf("ResolveUserUUIDFor(slack) error = %v", err)
	}
	if telegramUUID == "" || slackUUID == "" || telegramUUID == slackUUID {
		t.Fatalf("resolved UUIDs = %q, %q, want distinct non-empty values", telegramUUID, slackUUID)
	}
	if channel, ok := store.UserChannel("shared-user"); ok {
		t.Fatalf("UserChannel(shared-user) = %q, true, want ambiguous identity miss", channel)
	}
}

func TestIdentityConversationStore_IsolatesActiveConversationsByThread(t *testing.T) {
	store := agent.NewMemoryStore()
	learner, err := agent.NewLearnerIdentity("slack", "U123")
	if err != nil {
		t.Fatalf("NewLearnerIdentity() error = %v", err)
	}

	firstID, err := store.CreateConversationForThread(learner, "slack:C123:1700000000.000001", agent.Conversation{
		State: "teaching",
	})
	if err != nil {
		t.Fatalf("CreateConversationForThread(first) error = %v", err)
	}
	secondID, err := store.CreateConversationForThread(learner, "slack:C123:1700000000.000002", agent.Conversation{
		State: "teaching",
	})
	if err != nil {
		t.Fatalf("CreateConversationForThread(second) error = %v", err)
	}
	if firstID == secondID {
		t.Fatalf("conversation IDs = %q for both threads, want distinct conversations", firstID)
	}

	first, ok := store.GetActiveConversationForThread(learner, "slack:C123:1700000000.000001")
	if !ok {
		t.Fatal("GetActiveConversationForThread(first) found = false, want true")
	}
	if first.ID != firstID || first.ThreadID != "slack:C123:1700000000.000001" {
		t.Fatalf("first conversation = %#v, want ID %q and exact opaque thread ID", first, firstID)
	}
	second, ok := store.GetActiveConversationForThread(learner, "slack:C123:1700000000.000002")
	if !ok {
		t.Fatal("GetActiveConversationForThread(second) found = false, want true")
	}
	if second.ID != secondID || second.ThreadID != "slack:C123:1700000000.000002" {
		t.Fatalf("second conversation = %#v, want ID %q and exact opaque thread ID", second, secondID)
	}
}

func TestIdentityConversationStore_LatestActiveConversationPreservesSavedRoute(t *testing.T) {
	store := agent.NewMemoryStore()
	learner, err := agent.NewLearnerIdentity("slack", "U-latest")
	if err != nil {
		t.Fatal(err)
	}

	firstID, err := store.CreateConversationForThread(learner, "slack:C-old:1", agent.Conversation{
		State: "teaching",
	})
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := store.CreateConversationForThread(learner, "slack:C-new:2", agent.Conversation{
		State: "teaching",
	})
	if err != nil {
		t.Fatal(err)
	}

	latest, found := store.GetLatestActiveConversationFor(learner)
	if !found || latest.ID != secondID || latest.ThreadID != "slack:C-new:2" {
		t.Fatalf("latest conversation = %#v, %v, want second saved route", latest, found)
	}
	if err := store.EndConversation(secondID); err != nil {
		t.Fatal(err)
	}
	latest, found = store.GetLatestActiveConversationFor(learner)
	if !found || latest.ID != firstID || latest.ThreadID != "slack:C-old:1" {
		t.Fatalf("latest conversation after end = %#v, %v, want first saved route", latest, found)
	}
	if err := store.EndConversation(firstID); err != nil {
		t.Fatal(err)
	}
	if latest, found := store.GetLatestActiveConversationFor(learner); found {
		t.Fatalf("latest conversation = %#v, true, want no active route", latest)
	}
}

func TestIdentityConversationStore_EmptyThreadPreservesLegacyConversationScope(t *testing.T) {
	store := agent.NewMemoryStore()
	learner, err := agent.NewLearnerIdentity("telegram", "123")
	if err != nil {
		t.Fatalf("NewLearnerIdentity() error = %v", err)
	}

	id, err := store.CreateConversationFor(learner, agent.Conversation{State: "teaching"})
	if err != nil {
		t.Fatalf("CreateConversationFor() error = %v", err)
	}
	legacy, ok := store.GetActiveConversationFor(learner)
	if !ok {
		t.Fatal("GetActiveConversationFor() found = false, want true")
	}
	scoped, ok := store.GetActiveConversationForThread(learner, "")
	if !ok {
		t.Fatal("GetActiveConversationForThread(empty) found = false, want true")
	}
	if legacy.ID != id || scoped.ID != id || scoped.ThreadID != "" {
		t.Fatalf("legacy IDs = %q/%q, thread = %q, want %q/%q and empty thread", legacy.ID, scoped.ID, scoped.ThreadID, id, id)
	}
}

func TestIdentityConversationStore_RejectsSecondActiveConversationInSameThread(t *testing.T) {
	store := agent.NewMemoryStore()
	learner, err := agent.NewLearnerIdentity("discord", "123")
	if err != nil {
		t.Fatalf("NewLearnerIdentity() error = %v", err)
	}
	if _, err := store.CreateConversationForThread(learner, "discord:guild:channel", agent.Conversation{}); err != nil {
		t.Fatalf("CreateConversationForThread(first) error = %v", err)
	}
	if _, err := store.CreateConversationForThread(learner, "discord:guild:channel", agent.Conversation{}); !errors.Is(err, agent.ErrActiveConversationExists) {
		t.Fatalf("CreateConversationForThread(second) error = %v, want ErrActiveConversationExists", err)
	}
}

func TestNewLearnerIdentity_RequiresBothParts(t *testing.T) {
	if _, err := agent.NewLearnerIdentity("", "user"); err == nil {
		t.Fatal("NewLearnerIdentity(empty channel) error = nil, want error")
	}
	if _, err := agent.NewLearnerIdentity("slack", ""); err == nil {
		t.Fatal("NewLearnerIdentity(empty external ID) error = nil, want error")
	}

	identity, err := agent.NewLearnerIdentity(" slack ", " user ")
	if err != nil {
		t.Fatalf("NewLearnerIdentity() error = %v", err)
	}
	if identity.Channel() != "slack" || identity.ExternalID() != "user" {
		t.Fatalf("identity = %q/%q, want slack/user", identity.Channel(), identity.ExternalID())
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

func TestConversationStore_CurriculumStatePersistsReplayEvidenceAfterQuizClear(t *testing.T) {
	store := agent.NewMemoryStore()
	id, err := store.CreateConversation(agent.Conversation{
		UserID:  "u-curriculum-state",
		State:   "quiz_active",
		TopicID: "legacy-topic",
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationQuizState(id, "quiz_active", agent.ConversationQuizState{
		TopicID:   "legacy-topic",
		Intensity: "mixed",
		GeneratedQuestions: []agent.QuizQuestion{{
			ID:   "legacy-question",
			Text: "Legacy quiz question",
		}},
	}); err != nil {
		t.Fatalf("UpdateConversationQuizState() error = %v", err)
	}

	state := agent.ConversationCurriculumState{
		GoalTopicID:       "goal-topic",
		ActiveTopicID:     "prerequisite-topic",
		ActiveObjectiveID: "objective-1",
		ActiveQuestionID:  "question-1",
		RunID:             "run-1",
	}
	if err := store.SetConversationCurriculumState(id, state); err != nil {
		t.Fatalf("SetConversationCurriculumState() error = %v", err)
	}
	if err := store.UpdateConversationCurriculumAttempt(id, agent.ConversationCurriculumAttempt{
		AttemptID:     "chat:v1:telegram:delivery-1",
		LearnerAnswer: "B",
		Applied:       true,
		Correct:       false,
		Score:         0,
		Response:      "Belum tepat—cuba semak langkah pertama.",
	}); err != nil {
		t.Fatalf("UpdateConversationCurriculumAttempt() error = %v", err)
	}
	if err := store.ClearConversationQuizState(id, "teaching"); err != nil {
		t.Fatalf("ClearConversationQuizState() error = %v", err)
	}

	got, err := store.GetConversation(id)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if got.TopicID != "legacy-topic" {
		t.Fatalf("legacy TopicID = %q, want unchanged", got.TopicID)
	}
	if got.QuizState != nil {
		t.Fatalf("QuizState = %#v, want nil", got.QuizState)
	}
	if got.CurriculumState == nil {
		t.Fatal("CurriculumState = nil, want persisted state")
	}
	want := &agent.ConversationCurriculumState{
		GoalTopicID:       "goal-topic",
		ActiveTopicID:     "prerequisite-topic",
		ActiveObjectiveID: "objective-1",
		ActiveQuestionID:  "question-1",
		RunID:             "run-1",
		LastAttempt: &agent.ConversationCurriculumAttempt{
			AttemptID:     "chat:v1:telegram:delivery-1",
			LearnerAnswer: "B",
			Applied:       true,
			Correct:       false,
			Score:         0,
			Response:      "Belum tepat—cuba semak langkah pertama.",
		},
	}
	if !reflect.DeepEqual(got.CurriculumState, want) {
		t.Fatalf("CurriculumState = %#v, want %#v", got.CurriculumState, want)
	}

	if err := store.ClearConversationCurriculumState(id); err != nil {
		t.Fatalf("ClearConversationCurriculumState() error = %v", err)
	}
	got, err = store.GetConversation(id)
	if err != nil {
		t.Fatalf("GetConversation(after clear) error = %v", err)
	}
	if got.CurriculumState != nil {
		t.Fatalf("CurriculumState = %#v, want nil after clear", got.CurriculumState)
	}
	if got.TopicID != "legacy-topic" {
		t.Fatalf("legacy TopicID after clear = %q, want unchanged", got.TopicID)
	}
}

func TestConversationStore_AddMessage_NotFound(t *testing.T) {
	store := agent.NewMemoryStore()

	_, err := store.AddMessage("nonexistent", agent.StoredMessage{Role: "user", Content: "Hello"})
	if err == nil {
		t.Error("AddMessage() should error for non-existent conversation")
	}
}

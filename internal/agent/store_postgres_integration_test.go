// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package agent

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestPostgresStore_ResetProfileClearsFormAndLanguage(t *testing.T) {
	ctx := context.Background()
	pool, _ := startSchedulerPostgres(t, ctx)

	store, err := NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}

	userID := "store-reset-profile-user"
	if err := store.SetUserForm(userID, "2"); err != nil {
		t.Fatalf("SetUserForm() error = %v", err)
	}
	if err := store.SetUserPreferredLanguage(userID, "en"); err != nil {
		t.Fatalf("SetUserPreferredLanguage() error = %v", err)
	}

	if err := store.SetUserForm(userID, ""); err != nil {
		t.Fatalf("SetUserForm(clear) error = %v", err)
	}
	if err := store.SetUserPreferredLanguage(userID, ""); err != nil {
		t.Fatalf("SetUserPreferredLanguage(clear) error = %v", err)
	}

	if form, ok := store.GetUserForm(userID); ok || form != "" {
		t.Fatalf("GetUserForm() = %q, %v, want empty, false", form, ok)
	}
	if lang, ok := store.GetUserPreferredLanguage(userID); ok || lang != "" {
		t.Fatalf("GetUserPreferredLanguage() = %q, %v, want empty, false", lang, ok)
	}
}

func TestPostgresStore_IdentityMethodsIsolateSameExternalIDAcrossChannels(t *testing.T) {
	ctx := context.Background()
	pool, _ := startSchedulerPostgres(t, ctx)
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260407100000_add_groups.sql"))

	store, err := NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	telegram, err := NewLearnerIdentity("telegram", "shared-postgres-user")
	if err != nil {
		t.Fatalf("NewLearnerIdentity(telegram) error = %v", err)
	}
	slack, err := NewLearnerIdentity("slack", "shared-postgres-user")
	if err != nil {
		t.Fatalf("NewLearnerIdentity(slack) error = %v", err)
	}

	if err := store.SetUserNameFor(telegram, "Telegram Learner"); err != nil {
		t.Fatalf("SetUserNameFor(telegram) error = %v", err)
	}
	if err := store.SetUserNameFor(slack, "Slack Learner"); err != nil {
		t.Fatalf("SetUserNameFor(slack) error = %v", err)
	}
	if _, err := store.CreateConversationFor(telegram, Conversation{State: "teaching"}); err != nil {
		t.Fatalf("CreateConversationFor(telegram) error = %v", err)
	}
	if _, err := store.CreateConversationFor(slack, Conversation{State: "teaching"}); err != nil {
		t.Fatalf("CreateConversationFor(slack) error = %v", err)
	}

	if got, ok := store.GetUserNameFor(telegram); !ok || got != "Telegram Learner" {
		t.Fatalf("GetUserNameFor(telegram) = %q, %v, want Telegram Learner, true", got, ok)
	}
	if got, ok := store.GetUserNameFor(slack); !ok || got != "Slack Learner" {
		t.Fatalf("GetUserNameFor(slack) = %q, %v, want Slack Learner, true", got, ok)
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
	tracker := progress.NewPostgresTracker(pool, store.TenantID())
	telegramLearnerID, err := progress.NewLearnerID(telegramUUID)
	if err != nil {
		t.Fatal(err)
	}
	slackLearnerID, err := progress.NewLearnerID(slackUUID)
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.UpdateMasteryForLearner(telegramLearnerID, "identity-test", "shared-topic", 0.9); err != nil {
		t.Fatalf("UpdateMasteryForLearner(telegram) error = %v", err)
	}
	if err := tracker.UpdateMasteryForLearner(slackLearnerID, "identity-test", "shared-topic", 0.2); err != nil {
		t.Fatalf("UpdateMasteryForLearner(slack) error = %v", err)
	}
	telegramMastery, err := tracker.GetMasteryForLearner(telegramLearnerID, "identity-test", "shared-topic")
	if err != nil {
		t.Fatal(err)
	}
	slackMastery, err := tracker.GetMasteryForLearner(slackLearnerID, "identity-test", "shared-topic")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(telegramMastery-0.9) > 1e-6 || math.Abs(slackMastery-0.2) > 1e-6 {
		t.Fatalf("mastery = %v, %v, want provider-isolated 0.9 and 0.2", telegramMastery, slackMastery)
	}
	if channel, ok := store.UserChannel("shared-postgres-user"); ok {
		t.Fatalf("UserChannel(shared-postgres-user) = %q, true, want ambiguous identity miss", channel)
	}
}

func TestPostgresStore_IsolatesActiveConversationsByThread(t *testing.T) {
	ctx := context.Background()
	pool, _ := startSchedulerPostgres(t, ctx)

	store, err := NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	learner, err := NewLearnerIdentity("slack", "threaded-postgres-user")
	if err != nil {
		t.Fatalf("NewLearnerIdentity() error = %v", err)
	}

	firstID, err := store.CreateConversationForThread(learner, "slack:C123:1700000000.000001", Conversation{State: "teaching"})
	if err != nil {
		t.Fatalf("CreateConversationForThread(first) error = %v", err)
	}
	secondID, err := store.CreateConversationForThread(learner, "slack:C123:1700000000.000002", Conversation{State: "teaching"})
	if err != nil {
		t.Fatalf("CreateConversationForThread(second) error = %v", err)
	}
	if firstID == secondID {
		t.Fatalf("conversation IDs = %q for both threads, want distinct conversations", firstID)
	}

	first, ok := store.GetActiveConversationForThread(learner, "slack:C123:1700000000.000001")
	if !ok || first.ID != firstID || first.ThreadID != "slack:C123:1700000000.000001" {
		t.Fatalf("first conversation = %#v, %v, want ID %q and exact thread ID", first, ok, firstID)
	}
	second, ok := store.GetActiveConversationForThread(learner, "slack:C123:1700000000.000002")
	if !ok || second.ID != secondID || second.ThreadID != "slack:C123:1700000000.000002" {
		t.Fatalf("second conversation = %#v, %v, want ID %q and exact thread ID", second, ok, secondID)
	}
	latest, ok := store.GetLatestActiveConversationFor(learner)
	if !ok || latest.ID != secondID || latest.ThreadID != "slack:C123:1700000000.000002" {
		t.Fatalf("latest conversation = %#v, %v, want ID %q and exact thread ID", latest, ok, secondID)
	}
	if _, err := store.CreateConversationForThread(learner, "slack:C123:1700000000.000001", Conversation{}); !errors.Is(err, ErrActiveConversationExists) {
		t.Fatalf("CreateConversationForThread(duplicate active) error = %v, want ErrActiveConversationExists", err)
	}
}

func TestPostgresStore_CurriculumStateRoundTripsThroughMetadata(t *testing.T) {
	ctx := context.Background()
	pool, _ := startSchedulerPostgres(t, ctx)

	store, err := NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	conversationID, err := store.CreateConversation(Conversation{
		UserID:  "curriculum-state-postgres-user",
		State:   "quiz_active",
		TopicID: "legacy-topic",
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationQuizState(conversationID, "quiz_active", ConversationQuizState{
		TopicID:   "legacy-topic",
		Intensity: "mixed",
		GeneratedQuestions: []QuizQuestion{{
			ID:   "legacy-question",
			Text: "Legacy quiz question",
		}},
	}); err != nil {
		t.Fatalf("UpdateConversationQuizState() error = %v", err)
	}
	if err := store.SetConversationCurriculumState(conversationID, ConversationCurriculumState{
		GoalTopicID:       "goal-topic",
		ActiveTopicID:     "active-topic",
		ActiveObjectiveID: "objective-1",
		ActiveQuestionID:  "question-1",
		RunID:             "run-1",
	}); err != nil {
		t.Fatalf("SetConversationCurriculumState() error = %v", err)
	}
	if err := store.UpdateConversationCurriculumAttempt(conversationID, ConversationCurriculumAttempt{
		AttemptID:     "chat:v1:telegram:delivery-1",
		LearnerAnswer: "A",
		Applied:       true,
		Correct:       true,
		Score:         1,
		Response:      "Tepat. Kita teruskan.",
	}); err != nil {
		t.Fatalf("UpdateConversationCurriculumAttempt() error = %v", err)
	}
	if err := store.ClearConversationQuizState(conversationID, "teaching"); err != nil {
		t.Fatalf("ClearConversationQuizState() error = %v", err)
	}

	got, err := store.GetConversation(conversationID)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if got.QuizState != nil || got.TopicID != "legacy-topic" {
		t.Fatalf("legacy state = quiz %#v, topic %q; want cleared quiz and unchanged topic", got.QuizState, got.TopicID)
	}
	if got.CurriculumState == nil || got.CurriculumState.LastAttempt == nil {
		t.Fatalf("CurriculumState = %#v, want persisted plan and attempt", got.CurriculumState)
	}
	if got.CurriculumState.LastAttempt.AttemptID != "chat:v1:telegram:delivery-1" ||
		!got.CurriculumState.LastAttempt.Applied ||
		!got.CurriculumState.LastAttempt.Correct ||
		got.CurriculumState.LastAttempt.Response != "Tepat. Kita teruskan." {
		t.Fatalf("LastAttempt = %#v, want applied correct attempt", got.CurriculumState.LastAttempt)
	}

	var metadataHasCurriculumState bool
	if err := pool.QueryRow(ctx,
		`SELECT metadata ? 'curriculum_state'
		 FROM conversations
		 WHERE id = $1::uuid`,
		conversationID,
	).Scan(&metadataHasCurriculumState); err != nil {
		t.Fatalf("query curriculum metadata: %v", err)
	}
	if !metadataHasCurriculumState {
		t.Fatal("metadata curriculum_state key = false, want true")
	}

	if err := store.ClearConversationCurriculumState(conversationID); err != nil {
		t.Fatalf("ClearConversationCurriculumState() error = %v", err)
	}
	got, err = store.GetConversation(conversationID)
	if err != nil {
		t.Fatalf("GetConversation(after clear) error = %v", err)
	}
	if got.CurriculumState != nil {
		t.Fatalf("CurriculumState = %#v, want nil after clear", got.CurriculumState)
	}
}

func TestPostgresGroupStore_DeliveryIncludesLatestActiveThreadRoute(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260407100000_add_groups.sql"))
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260407100100_group_closed.sql"))

	store, err := NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	learner, err := NewLearnerIdentity("slack", "leaderboard-slack-user")
	if err != nil {
		t.Fatalf("NewLearnerIdentity() error = %v", err)
	}
	const threadID = "slack:C123:1700000000.000001"
	if _, err := store.CreateConversationForThread(learner, threadID, Conversation{State: "teaching"}); err != nil {
		t.Fatalf("CreateConversationForThread() error = %v", err)
	}
	userID, err := store.ResolveUserUUIDFor(learner)
	if err != nil {
		t.Fatalf("ResolveUserUUIDFor() error = %v", err)
	}

	groups := NewPostgresGroupStore(pool)
	group, err := groups.CreateGroup(tenantID, "Route Test", "study_group", "", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	if err := groups.JoinGroup(group.ID, userID, tenantID, "member"); err != nil {
		t.Fatalf("JoinGroup() error = %v", err)
	}

	members, err := groups.GetGroupMembersWithChannel(group.ID)
	if err != nil {
		t.Fatalf("GetGroupMembersWithChannel() error = %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("members = %d, want 1", len(members))
	}
	if got := members[0]; got.Channel != "slack" || got.ExternalID != "leaderboard-slack-user" || got.ThreadID != threadID {
		t.Fatalf("member route = %#v, want Slack identity and latest thread route", got)
	}
}

func TestPostgresStore_ConcurrentThreadCreationCreatesOneProviderQualifiedUser(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)

	store, err := NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	learner, err := NewLearnerIdentity("discord", "concurrent-user")
	if err != nil {
		t.Fatalf("NewLearnerIdentity() error = %v", err)
	}

	const threadCount = 8
	errs := make(chan error, threadCount)
	var group sync.WaitGroup
	for index := 0; index < threadCount; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, err := store.CreateConversationForThread(
				learner,
				fmt.Sprintf("discord:guild:channel-%d", index),
				Conversation{State: "teaching"},
			)
			errs <- err
		}(index)
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("CreateConversationForThread() error = %v", err)
		}
	}

	var userCount, conversationCount int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = 'discord'
		   AND external_id = 'concurrent-user'`,
		tenantID,
	).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM conversations AS conversation
		 JOIN users AS learner ON learner.id = conversation.user_id
		 WHERE conversation.tenant_id = $1::uuid
		   AND learner.channel = 'discord'
		   AND learner.external_id = 'concurrent-user'
		   AND conversation.ended_at IS NULL`,
		tenantID,
	).Scan(&conversationCount); err != nil {
		t.Fatalf("count conversations: %v", err)
	}
	if userCount != 1 || conversationCount != threadCount {
		t.Fatalf("created users/conversations = %d/%d, want 1/%d", userCount, conversationCount, threadCount)
	}
}

func TestConversationThreadMigration_DeduplicatesLegacyActiveConversationsBeforeUniqueIndex(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgresBeforeConversationThreads(t, ctx)

	var userID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, external_id, channel)
		 VALUES ($1::uuid, 'student', 'Duplicate Learner', 'duplicate-active-user', 'telegram')
		 RETURNING id::text`,
		tenantID,
	).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO conversations (user_id, tenant_id, state, started_at)
		 VALUES
		   ($1::uuid, $2::uuid, 'teaching', '2026-01-01T00:00:00Z'),
		   ($1::uuid, $2::uuid, 'teaching', '2026-01-02T00:00:00Z')`,
		userID,
		tenantID,
	); err != nil {
		t.Fatalf("insert duplicate active conversations: %v", err)
	}

	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260726120000_conversation_threads.sql"))

	var activeCount, endedCount int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FILTER (WHERE ended_at IS NULL),
		        COUNT(*) FILTER (WHERE ended_at IS NOT NULL)
		 FROM conversations
		 WHERE user_id = $1::uuid`,
		userID,
	).Scan(&activeCount, &endedCount); err != nil {
		t.Fatalf("count migrated conversations: %v", err)
	}
	if activeCount != 1 || endedCount != 1 {
		t.Fatalf("migration left active/ended counts = %d/%d, want 1/1", activeCount, endedCount)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO conversations (user_id, tenant_id, thread_id, state)
		 VALUES ($1::uuid, $2::uuid, '', 'teaching')`,
		userID,
		tenantID,
	); err == nil {
		t.Fatal("duplicate active legacy conversation insert error = nil, want unique-index violation")
	}
}

func TestConversationThreadMigration_ReportsDuplicateProviderQualifiedUsers(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgresBeforeConversationThreads(t, ctx)

	if _, err := pool.Exec(ctx,
		`INSERT INTO users (tenant_id, role, name, external_id, channel)
		 VALUES
		   ($1::uuid, 'student', 'First Duplicate', 'duplicate-user', 'slack'),
		   ($1::uuid, 'student', 'Second Duplicate', 'duplicate-user', 'slack')`,
		tenantID,
	); err != nil {
		t.Fatalf("insert duplicate users: %v", err)
	}

	migrationPath := filepath.Join("..", "..", "migrations", "20260726120000_conversation_threads.sql")
	migrationBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	upSQL, err := gooseUpSQL(string(migrationBytes))
	if err != nil {
		t.Fatalf("parse migration: %v", err)
	}
	if _, err := pool.Exec(ctx, upSQL); err == nil {
		t.Fatal("migration error = nil, want duplicate-user preflight error")
	} else if !strings.Contains(err.Error(), "reconcile duplicate users before retrying") {
		t.Fatalf("migration error = %v, want actionable duplicate-user preflight", err)
	}
}

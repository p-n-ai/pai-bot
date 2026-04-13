// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresChallengeStore_StartChallengeSearch_TimedOutSearchCreatesAIFallbackReadyChallenge(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	firstResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-1) error = %v", err)
	}
	if firstResult.Search == nil {
		t.Fatal("first search should not be nil")
	}

	_, err = pool.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		    SET expires_at = NOW() - INTERVAL '1 second'
		  WHERE id = $1::uuid`,
		firstResult.Search.ID,
	)
	if err != nil {
		t.Fatalf("expire searching ticket: %v", err)
	}

	resumeInput := input
	resumeInput.QuestionCount = input.QuestionCount + 2

	secondResult, err := store.StartChallengeSearch("queue-user-1", resumeInput)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-1) after timeout error = %v", err)
	}
	if secondResult.Search == nil || secondResult.Challenge == nil {
		t.Fatal("timed-out search should return both search and AI fallback challenge")
	}
	if secondResult.Search.Status != MatchmakingStatusMatched {
		t.Fatalf("search.Status = %q, want %q", secondResult.Search.Status, MatchmakingStatusMatched)
	}
	if secondResult.Search.MatchedChallengeID != secondResult.Challenge.ID {
		t.Fatalf("search.MatchedChallengeID = %q, want %q", secondResult.Search.MatchedChallengeID, secondResult.Challenge.ID)
	}
	if secondResult.Challenge.MatchSource != ChallengeMatchSourceAIFallback {
		t.Fatalf("challenge.MatchSource = %q, want %q", secondResult.Challenge.MatchSource, ChallengeMatchSourceAIFallback)
	}
	if secondResult.Challenge.OpponentKind != ChallengeOpponentKindAI {
		t.Fatalf("challenge.OpponentKind = %q, want %q", secondResult.Challenge.OpponentKind, ChallengeOpponentKindAI)
	}
	if secondResult.Challenge.State != ChallengeStateReady {
		t.Fatalf("challenge.State = %q, want %q", secondResult.Challenge.State, ChallengeStateReady)
	}
	if secondResult.Challenge.ReadyAt == nil {
		t.Fatal("challenge.ReadyAt should not be nil for AI fallback ready challenge")
	}
	if secondResult.Search.QuestionCount != input.QuestionCount {
		t.Fatalf("search.QuestionCount = %d, want %d", secondResult.Search.QuestionCount, input.QuestionCount)
	}
	if secondResult.Challenge.QuestionCount != input.QuestionCount {
		t.Fatalf("challenge.QuestionCount = %d, want %d", secondResult.Challenge.QuestionCount, input.QuestionCount)
	}

	var (
		matchSource   string
		opponentKind  string
		questionCount int
		state         string
		status        string
	)
	err = pool.QueryRow(ctx,
		`SELECT c.match_source,
		        c.opponent_kind,
		        c.question_count,
		        c.state,
		        t.status
		   FROM challenges c
		   JOIN challenge_matchmaking_tickets t ON t.matched_challenge_id = c.id
		  WHERE c.id = $1::uuid`,
		secondResult.Challenge.ID,
	).Scan(&matchSource, &opponentKind, &questionCount, &state, &status)
	if err != nil {
		t.Fatalf("query AI fallback challenge: %v", err)
	}
	if matchSource != ChallengeMatchSourceAIFallback {
		t.Fatalf("persisted match_source = %q, want %q", matchSource, ChallengeMatchSourceAIFallback)
	}
	if opponentKind != ChallengeOpponentKindAI {
		t.Fatalf("persisted opponent_kind = %q, want %q", opponentKind, ChallengeOpponentKindAI)
	}
	if questionCount != input.QuestionCount {
		t.Fatalf("persisted question_count = %d, want %d", questionCount, input.QuestionCount)
	}
	if state != ChallengeStateReady {
		t.Fatalf("persisted state = %q, want %q", state, ChallengeStateReady)
	}
	if status != MatchmakingStatusMatched {
		t.Fatalf("persisted ticket status = %q, want %q", status, MatchmakingStatusMatched)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_TimedOutSearchAIFallbackConcurrentCallsReturnSingleReadyChallenge(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	firstResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-1) error = %v", err)
	}
	if firstResult.Search == nil {
		t.Fatal("first search should not be nil")
	}

	_, err = pool.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		    SET expires_at = NOW() - INTERVAL '1 second'
		  WHERE id = $1::uuid`,
		firstResult.Search.ID,
	)
	if err != nil {
		t.Fatalf("expire searching ticket: %v", err)
	}

	type callResult struct {
		result *StartChallengeSearchResult
		err    error
	}

	start := make(chan struct{})
	results := make(chan callResult, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := store.StartChallengeSearch("queue-user-1", input)
			results <- callResult{result: result, err: err}
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var challengeID string
	for result := range results {
		if result.err != nil {
			t.Fatalf("StartChallengeSearch() concurrent error = %v", result.err)
		}
		if result.result == nil || result.result.Challenge == nil || result.result.Search == nil {
			t.Fatal("concurrent result should include search and AI fallback challenge")
		}
		if result.result.Challenge.MatchSource != ChallengeMatchSourceAIFallback {
			t.Fatalf("challenge.MatchSource = %q, want %q", result.result.Challenge.MatchSource, ChallengeMatchSourceAIFallback)
		}
		if challengeID == "" {
			challengeID = result.result.Challenge.ID
			continue
		}
		if result.result.Challenge.ID != challengeID {
			t.Fatalf("challenge.ID = %q, want %q", result.result.Challenge.ID, challengeID)
		}
	}

	assertSingleAIFallbackChallenge(t, ctx, pool, tenantID, "queue-user-1")
}

func assertSingleAIFallbackChallenge(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, externalUserID string) {
	t.Helper()

	var aiFallbackCount int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM challenges c
		   JOIN users u ON u.id = c.creator_user_id
		  WHERE c.tenant_id = $1::uuid
		    AND u.external_id = $2
		    AND c.match_source = $3`,
		tenantID,
		externalUserID,
		ChallengeMatchSourceAIFallback,
	).Scan(&aiFallbackCount)
	if err != nil {
		t.Fatalf("count AI fallback challenges: %v", err)
	}
	if aiFallbackCount != 1 {
		t.Fatalf("AI fallback challenges = %d, want 1", aiFallbackCount)
	}
}

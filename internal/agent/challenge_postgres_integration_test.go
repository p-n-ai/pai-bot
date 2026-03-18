//go:build integration
// +build integration

package agent

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func applyChallengeMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318102000_challenges.sql"))
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318102100_challenge_acceptance.sql"))
	applyMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318102200_challenge_matchmaking_question_count.sql"))
}

func TestPostgresChallengeStore_GetChallengeIgnoresCompletedInviteCodes(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	creatorUUID := seedChallengeUser(t, ctx, pool, tenantID, "challenge-completed-creator", "telegram")

	_, err := pool.Exec(ctx,
		`INSERT INTO challenges (
			tenant_id,
			creator_user_id,
			topic_id,
			topic_name,
			syllabus_id,
			match_source,
			invite_code,
			question_count,
			state,
			completed_at
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, NOW())`,
		tenantID,
		creatorUUID,
		"F1-02",
		"Linear Equations",
		"kssm-f1",
		ChallengeMatchSourceInviteCode,
		"ABC123",
		5,
		"completed",
	)
	if err != nil {
		t.Fatalf("insert completed challenge: %v", err)
	}

	store := NewPostgresChallengeStore(pool, tenantID)
	_, err = store.GetChallenge("ABC123")
	if err != ErrChallengeNotFound {
		t.Fatalf("GetChallenge() error = %v, want ErrChallengeNotFound", err)
	}
}

func TestPostgresChallengeStore_CreateInviteChallengeSupportsExplicitChannel(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "terminal-challenge-user", "terminal")

	store := NewPostgresChallengeStoreForChannel(pool, tenantID, "terminal")
	challenge, err := store.CreateInviteChallenge("terminal-challenge-user", ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}
	if challenge.CreatorID != "terminal-challenge-user" {
		t.Fatalf("CreatorID = %q, want terminal-challenge-user", challenge.CreatorID)
	}
	if challenge.State != ChallengeStateWaiting {
		t.Fatalf("State = %q, want %q", challenge.State, ChallengeStateWaiting)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_DoesNotPairAcrossChannels(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "terminal-user", "terminal")
	seedChallengeUser(t, ctx, pool, tenantID, "telegram-user", "telegram")

	terminalStore := NewPostgresChallengeStoreForChannel(pool, tenantID, "terminal")
	telegramStore := NewPostgresChallengeStoreForChannel(pool, tenantID, "telegram")
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	terminalResult, err := terminalStore.StartChallengeSearch("terminal-user", input)
	if err != nil {
		t.Fatalf("terminal StartChallengeSearch() error = %v", err)
	}
	if terminalResult.Search == nil || terminalResult.Challenge != nil {
		t.Fatalf("terminal result = %#v, want one searching ticket only", terminalResult)
	}

	telegramResult, err := telegramStore.StartChallengeSearch("telegram-user", input)
	if err != nil {
		t.Fatalf("telegram StartChallengeSearch() error = %v", err)
	}
	if telegramResult.Search == nil || telegramResult.Challenge != nil {
		t.Fatalf("telegram result = %#v, want one searching ticket only", telegramResult)
	}
	if telegramResult.Search.Status != MatchmakingStatusSearching {
		t.Fatalf("telegram search.Status = %q, want %q", telegramResult.Search.Status, MatchmakingStatusSearching)
	}

	var matchedCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM challenge_matchmaking_tickets
		  WHERE tenant_id = $1::uuid
		    AND status = 'matched'`,
		tenantID,
	).Scan(&matchedCount)
	if err != nil {
		t.Fatalf("count matched tickets: %v", err)
	}
	if matchedCount != 0 {
		t.Fatalf("matched tickets = %d, want 0 across channels", matchedCount)
	}
}

func TestPostgresChallengeStore_JoinChallenge_RejectsCrossChannelInviteJoin(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "terminal-owner", "terminal")
	seedChallengeUser(t, ctx, pool, tenantID, "telegram-joiner", "telegram")

	terminalStore := NewPostgresChallengeStoreForChannel(pool, tenantID, "terminal")
	telegramStore := NewPostgresChallengeStoreForChannel(pool, tenantID, "telegram")
	challenge, err := terminalStore.CreateInviteChallenge("terminal-owner", ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	_, err = telegramStore.JoinChallenge(challenge.Code, "telegram-joiner")
	if err != ErrChallengeNotFound {
		t.Fatalf("JoinChallenge() error = %v, want ErrChallengeNotFound for cross-channel join", err)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_ResumesSearchingTicket(t *testing.T) {
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
		t.Fatalf("first StartChallengeSearch() error = %v", err)
	}
	if firstResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil before pairing", firstResult.Challenge)
	}

	secondResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("second StartChallengeSearch() error = %v", err)
	}
	if secondResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil when resuming same ticket", secondResult.Challenge)
	}
	if firstResult.Search == nil || secondResult.Search == nil {
		t.Fatal("tickets should not be nil")
	}
	if secondResult.Search.ID != firstResult.Search.ID {
		t.Fatalf("ticket ID = %q, want %q", secondResult.Search.ID, firstResult.Search.ID)
	}
	if secondResult.Search.Status != MatchmakingStatusSearching {
		t.Fatalf("ticket.Status = %q, want %q", secondResult.Search.Status, MatchmakingStatusSearching)
	}

	var searchingCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM challenge_matchmaking_tickets cmt
		   JOIN users u ON u.id = cmt.user_id
		  WHERE cmt.tenant_id = $1::uuid
		    AND u.external_id = $2
		    AND cmt.status = 'searching'`,
		tenantID,
		"queue-user-1",
	).Scan(&searchingCount)
	if err != nil {
		t.Fatalf("count searching tickets: %v", err)
	}
	if searchingCount != 1 {
		t.Fatalf("searching tickets = %d, want 1", searchingCount)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_PairsCompatibleUsers(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")
	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-2", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	firstResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("first StartChallengeSearch() error = %v", err)
	}
	if firstResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil before pairing", firstResult.Challenge)
	}
	if firstResult.Search == nil {
		t.Fatal("first ticket should not be nil")
	}
	if firstResult.Search.Status != MatchmakingStatusSearching {
		t.Fatalf("ticket.Status = %q, want %q", firstResult.Search.Status, MatchmakingStatusSearching)
	}

	secondResult, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("second StartChallengeSearch() error = %v", err)
	}
	if secondResult.Search == nil {
		t.Fatal("second ticket should not be nil")
	}
	if secondResult.Challenge == nil {
		t.Fatal("matched challenge should not be nil after compatible second user joins")
	}
	if secondResult.Search.Status != MatchmakingStatusMatched {
		t.Fatalf("ticket.Status = %q, want %q", secondResult.Search.Status, MatchmakingStatusMatched)
	}
	if secondResult.Challenge.MatchSource != ChallengeMatchSourceQueue {
		t.Fatalf("challenge.MatchSource = %q, want %q", secondResult.Challenge.MatchSource, ChallengeMatchSourceQueue)
	}
	if secondResult.Challenge.State != ChallengeStatePendingAcceptance {
		t.Fatalf("challenge.State = %q, want %q", secondResult.Challenge.State, ChallengeStatePendingAcceptance)
	}
	if secondResult.Challenge.ReadyAt != nil {
		t.Fatalf("challenge.ReadyAt = %v, want nil before both accepts land", secondResult.Challenge.ReadyAt)
	}
	if secondResult.Challenge.CreatorID != "queue-user-1" {
		t.Fatalf("challenge.CreatorID = %q, want queue-user-1", secondResult.Challenge.CreatorID)
	}
	if secondResult.Challenge.OpponentID != "queue-user-2" {
		t.Fatalf("challenge.OpponentID = %q, want queue-user-2", secondResult.Challenge.OpponentID)
	}

	var matchedCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM challenge_matchmaking_tickets
		  WHERE tenant_id = $1::uuid
		    AND topic_id = $2
		    AND status = 'matched'`,
		tenantID,
		"F1-02",
	).Scan(&matchedCount)
	if err != nil {
		t.Fatalf("count matched tickets: %v", err)
	}
	if matchedCount != 2 {
		t.Fatalf("matched tickets = %d, want 2", matchedCount)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_ResumesMatchedChallenge(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")
	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-2", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	_, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("first StartChallengeSearch() error = %v", err)
	}
	pairResult, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("second StartChallengeSearch() error = %v", err)
	}
	if pairResult.Challenge == nil {
		t.Fatal("pair result challenge should not be nil")
	}

	resumedResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("resume StartChallengeSearch() error = %v", err)
	}
	if resumedResult.Search == nil || resumedResult.Challenge == nil {
		t.Fatal("resumed result should include ticket and challenge")
	}
	if resumedResult.Search.Status != MatchmakingStatusMatched {
		t.Fatalf("ticket.Status = %q, want %q", resumedResult.Search.Status, MatchmakingStatusMatched)
	}
	if resumedResult.Challenge.ID != pairResult.Challenge.ID {
		t.Fatalf("challenge.ID = %q, want %q", resumedResult.Challenge.ID, pairResult.Challenge.ID)
	}
	if resumedResult.Challenge.State != ChallengeStatePendingAcceptance {
		t.Fatalf("challenge.State = %q, want %q", resumedResult.Challenge.State, ChallengeStatePendingAcceptance)
	}
}

func TestPostgresChallengeStore_AcceptPendingChallenge_MovesPendingAcceptanceQueueMatchToReadyAfterBothUsersAccept(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")
	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-2", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	_, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-1) error = %v", err)
	}
	pairResult, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-2) error = %v", err)
	}
	if pairResult.Challenge == nil {
		t.Fatal("pair result challenge should not be nil")
	}

	firstAccepted, err := store.AcceptPendingChallenge("queue-user-1")
	if err != nil {
		t.Fatalf("AcceptPendingChallenge(queue-user-1) error = %v", err)
	}
	if firstAccepted.State != ChallengeStatePendingAcceptance {
		t.Fatalf("challenge.State after first accept = %q, want %q", firstAccepted.State, ChallengeStatePendingAcceptance)
	}
	if firstAccepted.CreatorAcceptedAt == nil {
		t.Fatal("CreatorAcceptedAt should not be nil after first accept")
	}
	if firstAccepted.ReadyAt != nil {
		t.Fatalf("challenge.ReadyAt after first accept = %v, want nil", firstAccepted.ReadyAt)
	}

	secondAccepted, err := store.AcceptPendingChallenge("queue-user-2")
	if err != nil {
		t.Fatalf("AcceptPendingChallenge(queue-user-2) error = %v", err)
	}
	if secondAccepted.State != ChallengeStateReady {
		t.Fatalf("challenge.State after second accept = %q, want %q", secondAccepted.State, ChallengeStateReady)
	}
	if secondAccepted.CreatorAcceptedAt == nil || secondAccepted.OpponentAcceptedAt == nil {
		t.Fatal("both acceptance timestamps should be set after second accept")
	}
	if secondAccepted.ReadyAt == nil {
		t.Fatal("ReadyAt should not be nil after both accepts")
	}

	var (
		state              string
		creatorAcceptedAt  *string
		opponentAcceptedAt *string
	)
	err = pool.QueryRow(ctx,
		`SELECT state,
		        creator_accepted_at::text,
		        opponent_accepted_at::text
		   FROM challenges
		  WHERE id = $1::uuid`,
		secondAccepted.ID,
	).Scan(&state, &creatorAcceptedAt, &opponentAcceptedAt)
	if err != nil {
		t.Fatalf("query accepted challenge: %v", err)
	}
	if state != ChallengeStateReady {
		t.Fatalf("persisted challenge state = %q, want %q", state, ChallengeStateReady)
	}
	if creatorAcceptedAt == nil || opponentAcceptedAt == nil {
		t.Fatal("persisted acceptance timestamps should not be nil")
	}
}

func TestPostgresChallengeStore_CancelChallengeSearch_CancelsSearchingTicket(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	_, err := store.StartChallengeSearch("queue-user-1", ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	})
	if err != nil {
		t.Fatalf("StartChallengeSearch() error = %v", err)
	}

	cancelled, err := store.CancelChallengeSearch("queue-user-1")
	if err != nil {
		t.Fatalf("CancelChallengeSearch() error = %v", err)
	}
	if !cancelled {
		t.Fatal("CancelChallengeSearch() = false, want true")
	}

	var status string
	err = pool.QueryRow(ctx,
		`SELECT cmt.status
		   FROM challenge_matchmaking_tickets cmt
		   JOIN users u ON u.id = cmt.user_id
		  WHERE cmt.tenant_id = $1::uuid
		    AND u.external_id = $2
		  ORDER BY cmt.created_at DESC
		  LIMIT 1`,
		tenantID,
		"queue-user-1",
	).Scan(&status)
	if err != nil {
		t.Fatalf("query cancelled ticket: %v", err)
	}
	if status != MatchmakingStatusCancelled {
		t.Fatalf("ticket status = %q, want %q", status, MatchmakingStatusCancelled)
	}
}

func TestPostgresChallengeStore_CancelOpenChallenge_CancelsAIFallbackAndUnblocksUser(t *testing.T) {
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
		t.Fatalf("StartChallengeSearch() error = %v", err)
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

	fallbackResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() after timeout error = %v", err)
	}
	if fallbackResult.Challenge == nil {
		t.Fatal("AI fallback challenge should not be nil")
	}

	cancelled, err := store.CancelOpenChallenge("queue-user-1")
	if err != nil {
		t.Fatalf("CancelOpenChallenge() error = %v", err)
	}
	if !cancelled {
		t.Fatal("CancelOpenChallenge() = false, want true")
	}

	var (
		challengeState string
		ticketStatus   string
	)
	err = pool.QueryRow(ctx,
		`SELECT c.state, t.status
		   FROM challenges c
		   JOIN challenge_matchmaking_tickets t ON t.matched_challenge_id = c.id
		  WHERE c.id = $1::uuid`,
		fallbackResult.Challenge.ID,
	).Scan(&challengeState, &ticketStatus)
	if err != nil {
		t.Fatalf("query cancelled AI fallback challenge: %v", err)
	}
	if challengeState != "cancelled" {
		t.Fatalf("challenge state = %q, want cancelled", challengeState)
	}
	if ticketStatus != MatchmakingStatusCancelled {
		t.Fatalf("ticket status = %q, want %q", ticketStatus, MatchmakingStatusCancelled)
	}

	reopenedResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() after cancel error = %v", err)
	}
	if reopenedResult.Search == nil || reopenedResult.Challenge != nil {
		t.Fatalf("reopened result = %#v, want one new searching ticket", reopenedResult)
	}
	if reopenedResult.Search.Status != MatchmakingStatusSearching {
		t.Fatalf("search.Status = %q, want %q", reopenedResult.Search.Status, MatchmakingStatusSearching)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_TimedOutSearchCreatesAIFallbackChallenge(t *testing.T) {
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
		t.Fatalf("first StartChallengeSearch() error = %v", err)
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

	secondResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("second StartChallengeSearch() error = %v", err)
	}
	if secondResult.Search == nil || secondResult.Challenge == nil {
		t.Fatal("timed-out search should return both search and AI fallback challenge")
	}
	if secondResult.Search.ID != firstResult.Search.ID {
		t.Fatalf("search ID = %q, want %q when same ticket is claimed", secondResult.Search.ID, firstResult.Search.ID)
	}
	if secondResult.Search.Status != MatchmakingStatusMatched {
		t.Fatalf("search.Status = %q, want %q", secondResult.Search.Status, MatchmakingStatusMatched)
	}
	if secondResult.Challenge.MatchSource != ChallengeMatchSourceAIFallback {
		t.Fatalf("challenge.MatchSource = %q, want %q", secondResult.Challenge.MatchSource, ChallengeMatchSourceAIFallback)
	}
	if secondResult.Challenge.State != ChallengeStateReady {
		t.Fatalf("challenge.State = %q, want %q", secondResult.Challenge.State, ChallengeStateReady)
	}

	var matchedCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM challenge_matchmaking_tickets
		  WHERE tenant_id = $1::uuid
		    AND user_id = (
		        SELECT id
		          FROM users
		         WHERE tenant_id = $1::uuid AND external_id = $2 AND channel = 'telegram'
		         LIMIT 1
		    )
		    AND status = $3`,
		tenantID,
		"queue-user-1",
		MatchmakingStatusMatched,
	).Scan(&matchedCount)
	if err != nil {
		t.Fatalf("count matched tickets: %v", err)
	}
	if matchedCount != 1 {
		t.Fatalf("matched tickets = %d, want 1", matchedCount)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_DoesNotMatchExpiredOpponent(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")
	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-2", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	firstResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("first StartChallengeSearch() error = %v", err)
	}

	_, err = pool.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		 SET expires_at = NOW() - INTERVAL '1 second'
		 WHERE id = $1::uuid`,
		firstResult.Search.ID,
	)
	if err != nil {
		t.Fatalf("expire opponent ticket: %v", err)
	}

	secondResult, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("second StartChallengeSearch() error = %v", err)
	}
	if secondResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil when only opponent ticket is expired", secondResult.Challenge)
	}
	if secondResult.Search == nil {
		t.Fatal("second search should not be nil")
	}
	if secondResult.Search.Status != MatchmakingStatusSearching {
		t.Fatalf("search.Status = %q, want %q", secondResult.Search.Status, MatchmakingStatusSearching)
	}
}

func TestPostgresChallengeStore_CreateInviteChallenge_RejectsWhenUserAlreadySearching(t *testing.T) {
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

	_, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() error = %v", err)
	}

	_, err = store.CreateInviteChallenge("queue-user-1", input)
	if err != ErrChallengeAlreadyActive {
		t.Fatalf("CreateInviteChallenge() error = %v, want ErrChallengeAlreadyActive", err)
	}
}

func TestPostgresChallengeStore_JoinChallenge_RejectsWhenUserAlreadySearching(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "challenge-owner", "telegram")
	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	challenge, err := store.CreateInviteChallenge("challenge-owner", input)
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}
	_, err = store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() error = %v", err)
	}

	_, err = store.JoinChallenge(challenge.Code, "queue-user-1")
	if err != ErrChallengeAlreadyActive {
		t.Fatalf("JoinChallenge() error = %v, want ErrChallengeAlreadyActive", err)
	}
}

func TestPostgresChallengeStore_JoinChallenge_InviteJoinStillBecomesReady(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "invite-owner", "telegram")
	seedChallengeUser(t, ctx, pool, tenantID, "invite-joiner", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	challenge, err := store.CreateInviteChallenge("invite-owner", ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	joined, err := store.JoinChallenge(challenge.Code, "invite-joiner")
	if err != nil {
		t.Fatalf("JoinChallenge() error = %v", err)
	}
	if joined.State != ChallengeStateReady {
		t.Fatalf("challenge.State = %q, want %q", joined.State, ChallengeStateReady)
	}
	if joined.ReadyAt == nil {
		t.Fatal("ReadyAt should not be nil after invite join")
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_MatchedTicketWithInactiveChallengeRepairsTicket(t *testing.T) {
	ctx := context.Background()
	pool, tenantID := startSchedulerPostgres(t, ctx)
	applyChallengeMigrations(t, ctx, pool)

	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-1", "telegram")
	seedChallengeUser(t, ctx, pool, tenantID, "queue-user-2", "telegram")

	store := NewPostgresChallengeStore(pool, tenantID)
	input := ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	_, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("first StartChallengeSearch() error = %v", err)
	}
	pairResult, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("second StartChallengeSearch() error = %v", err)
	}
	if pairResult.Challenge == nil {
		t.Fatal("pair result challenge should not be nil")
	}

	_, err = pool.Exec(ctx,
		`UPDATE challenges
		 SET state = 'completed',
		     completed_at = NOW(),
		     updated_at = NOW()
		 WHERE id = $1::uuid`,
		pairResult.Challenge.ID,
	)
	if err != nil {
		t.Fatalf("complete linked challenge: %v", err)
	}

	reopenedResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() after stale match error = %v", err)
	}
	if reopenedResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil after stale matched ticket repair", reopenedResult.Challenge)
	}
	if reopenedResult.Search == nil {
		t.Fatal("reopened search should not be nil")
	}
	if reopenedResult.Search.Status != MatchmakingStatusSearching {
		t.Fatalf("search.Status = %q, want %q", reopenedResult.Search.Status, MatchmakingStatusSearching)
	}

	var staleStatus string
	err = pool.QueryRow(ctx,
		`SELECT status
		   FROM challenge_matchmaking_tickets
		  WHERE tenant_id = $1::uuid
		    AND matched_challenge_id = $2::uuid
		  ORDER BY created_at ASC
		  LIMIT 1`,
		tenantID,
		pairResult.Challenge.ID,
	).Scan(&staleStatus)
	if err != nil {
		t.Fatalf("query stale matched ticket: %v", err)
	}
	if staleStatus != MatchmakingStatusExpired {
		t.Fatalf("stale matched ticket status = %q, want %q", staleStatus, MatchmakingStatusExpired)
	}
}

func TestPostgresChallengeStore_StartChallengeSearch_SameUserConcurrentCallsReturnSingleSearchingTicket(t *testing.T) {
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

	var searchID string
	for result := range results {
		if result.err != nil {
			t.Fatalf("StartChallengeSearch() concurrent error = %v", result.err)
		}
		if result.result == nil || result.result.Search == nil {
			t.Fatal("concurrent result search should not be nil")
		}
		if result.result.Challenge != nil {
			t.Fatalf("challenge = %#v, want nil for same-user search", result.result.Challenge)
		}
		if searchID == "" {
			searchID = result.result.Search.ID
			continue
		}
		if result.result.Search.ID != searchID {
			t.Fatalf("search ID = %q, want %q", result.result.Search.ID, searchID)
		}
	}

	var searchingCount int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM challenge_matchmaking_tickets cmt
		   JOIN users u ON u.id = cmt.user_id
		  WHERE cmt.tenant_id = $1::uuid
		    AND u.external_id = $2
		    AND cmt.status = 'searching'`,
		tenantID,
		"queue-user-1",
	).Scan(&searchingCount)
	if err != nil {
		t.Fatalf("count searching tickets: %v", err)
	}
	if searchingCount != 1 {
		t.Fatalf("searching tickets = %d, want 1", searchingCount)
	}
}

func seedChallengeUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, externalID, channel string) string {
	t.Helper()

	var userID string
	err := pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, external_id, channel)
		 VALUES ($1::uuid, 'student', $2, $3, $4)
		 RETURNING id::text`,
		tenantID,
		"Student "+externalID,
		externalID,
		channel,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seed user %s/%s: %v", externalID, channel, err)
	}
	return userID
}

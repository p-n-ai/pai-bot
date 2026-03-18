package agent

import (
	"testing"
	"time"
)

func TestMemoryChallengeStore_StartChallengeSearch_TimedOutSearchCreatesAIFallbackChallenge(t *testing.T) {
	store := NewMemoryChallengeStore()
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

	store.mu.Lock()
	store.searches["queue-user-1"].ExpiresAt = time.Now().Add(-time.Second)
	expiredSearch := store.searches["queue-user-1"]
	store.mu.Unlock()

	secondResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() after expiry error = %v", err)
	}
	if secondResult.Search == nil || secondResult.Challenge == nil || firstResult.Search == nil {
		t.Fatal("timed-out search should return both search and AI fallback challenge")
	}
	if secondResult.Search.ID != firstResult.Search.ID {
		t.Fatalf("search ID = %q, want %q when same ticket is claimed", secondResult.Search.ID, firstResult.Search.ID)
	}
	if secondResult.Search.Status != MatchmakingStatusMatched {
		t.Fatalf("search status = %q, want %q", secondResult.Search.Status, MatchmakingStatusMatched)
	}
	if secondResult.Challenge.MatchSource != ChallengeMatchSourceAIFallback {
		t.Fatalf("challenge.MatchSource = %q, want %q", secondResult.Challenge.MatchSource, ChallengeMatchSourceAIFallback)
	}
	if secondResult.Challenge.State != ChallengeStateReady {
		t.Fatalf("challenge.State = %q, want %q", secondResult.Challenge.State, ChallengeStateReady)
	}
	if secondResult.Challenge.ReadyAt == nil {
		t.Fatal("challenge.ReadyAt should not be nil for AI fallback")
	}
	if expiredSearch.Status != MatchmakingStatusMatched {
		t.Fatalf("timed-out search status = %q, want %q", expiredSearch.Status, MatchmakingStatusMatched)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_DoesNotPairWithExpiredOpponent(t *testing.T) {
	store := NewMemoryChallengeStore()
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

	store.mu.Lock()
	store.searches["queue-user-1"].ExpiresAt = time.Now().Add(-time.Second)
	store.mu.Unlock()

	result, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-2) error = %v", err)
	}
	if result.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil when only opponent is expired", result.Challenge)
	}
	if result.Search == nil {
		t.Fatal("search should not be nil")
	}
	if result.Search.Status != MatchmakingStatusSearching {
		t.Fatalf("search.Status = %q, want %q", result.Search.Status, MatchmakingStatusSearching)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_MatchedTicketWithoutChallengeReopensSearch(t *testing.T) {
	store := NewMemoryChallengeStore()
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
	pairResult, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-2) error = %v", err)
	}
	if pairResult.Challenge == nil {
		t.Fatal("pair result challenge should not be nil")
	}

	store.mu.Lock()
	delete(store.challengesByID, pairResult.Challenge.ID)
	staleSearch := store.searches["queue-user-1"]
	store.mu.Unlock()

	reopenedResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-1) after stale match error = %v", err)
	}
	if reopenedResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil when stale matched ticket reopens search", reopenedResult.Challenge)
	}
	if reopenedResult.Search == nil || firstResult.Search == nil {
		t.Fatal("searches should not be nil")
	}
	if reopenedResult.Search.ID == firstResult.Search.ID {
		t.Fatalf("search ID = %q, want a new search after stale matched ticket cleanup", reopenedResult.Search.ID)
	}
	if staleSearch.Status != MatchmakingStatusExpired {
		t.Fatalf("stale matched search status = %q, want %q", staleSearch.Status, MatchmakingStatusExpired)
	}
}

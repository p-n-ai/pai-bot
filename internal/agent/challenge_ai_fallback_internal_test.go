package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestMemoryChallengeStore_StartChallengeSearch_TimedOutSearchCreatesAIFallbackReadyChallenge(t *testing.T) {
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
	if firstResult.Search == nil {
		t.Fatal("first search should not be nil")
	}

	store.mu.Lock()
	store.searches["queue-user-1"].ExpiresAt = time.Now().Add(-time.Second)
	store.mu.Unlock()

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
	if secondResult.Challenge.TopicID != input.TopicID {
		t.Fatalf("challenge.TopicID = %q, want %q", secondResult.Challenge.TopicID, input.TopicID)
	}
	if secondResult.Challenge.QuestionCount != input.QuestionCount {
		t.Fatalf("challenge.QuestionCount = %d, want %d", secondResult.Challenge.QuestionCount, input.QuestionCount)
	}
	if secondResult.Search.QuestionCount != input.QuestionCount {
		t.Fatalf("search.QuestionCount = %d, want %d", secondResult.Search.QuestionCount, input.QuestionCount)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_TimedOutSearchAIFallbackIsIdempotent(t *testing.T) {
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

	firstFallback, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("first fallback StartChallengeSearch() error = %v", err)
	}
	secondFallback, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("second fallback StartChallengeSearch() error = %v", err)
	}
	if firstFallback.Challenge == nil || secondFallback.Challenge == nil {
		t.Fatal("fallback challenge should not be nil")
	}
	if secondFallback.Challenge.ID != firstFallback.Challenge.ID {
		t.Fatalf("challenge.ID = %q, want %q", secondFallback.Challenge.ID, firstFallback.Challenge.ID)
	}

	store.mu.RLock()
	var aiFallbackCount int
	for _, challenge := range store.challengesByID {
		if challenge.MatchSource == ChallengeMatchSourceAIFallback {
			aiFallbackCount++
		}
	}
	store.mu.RUnlock()
	if aiFallbackCount != 1 {
		t.Fatalf("AI fallback challenges = %d, want 1", aiFallbackCount)
	}

	_, err = store.CreateInviteChallenge("queue-user-1", input)
	if err != ErrChallengeAlreadyActive {
		t.Fatalf("CreateInviteChallenge() error = %v, want ErrChallengeAlreadyActive", err)
	}
}

func TestEngine_ChallengeCommand_ResumesTimedOutSearchAsAIReadyChallenge(t *testing.T) {
	store := NewMemoryStore()
	challenges := NewMemoryChallengeStore()
	convID, err := store.CreateConversation(Conversation{UserID: "queue-ai-user"})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
		t.Fatalf("UpdateConversationTopicID() error = %v", err)
	}

	engine := NewEngine(EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createChallengeFallbackCurriculumLoader(t),
	})

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) error = %v", err)
	}

	challenges.mu.Lock()
	challenges.searches["queue-ai-user"].ExpiresAt = time.Now().Add(-time.Second)
	challenges.mu.Unlock()

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) after timeout error = %v", err)
	}
	if !strings.Contains(resp, "No human opponent found in time.") {
		t.Fatalf("response = %q, want timeout-to-AI copy", resp)
	}
	if !strings.Contains(resp, "Opponent: AI") {
		t.Fatalf("response = %q, want AI opponent label", resp)
	}
	if !strings.Contains(resp, "State: ready") {
		t.Fatalf("response = %q, want ready state", resp)
	}
	if strings.Contains(resp, "/challenge accept") {
		t.Fatalf("response = %q, should not show accept controls for AI fallback", resp)
	}
}

func TestEngine_ChallengeAcceptCommand_RejectsWhenOnlyAIFallbackChallengeExists(t *testing.T) {
	store := NewMemoryStore()
	challenges := NewMemoryChallengeStore()
	convID, err := store.CreateConversation(Conversation{UserID: "queue-ai-user"})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
		t.Fatalf("UpdateConversationTopicID() error = %v", err)
	}

	engine := NewEngine(EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createChallengeFallbackCurriculumLoader(t),
	})

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) error = %v", err)
	}

	challenges.mu.Lock()
	challenges.searches["queue-ai-user"].ExpiresAt = time.Now().Add(-time.Second)
	challenges.mu.Unlock()

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) after timeout error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge accept",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge accept) error = %v", err)
	}
	if !strings.Contains(resp, "queue match waiting for acceptance") {
		t.Fatalf("response = %q, want no-acceptance guidance", resp)
	}
}

func TestEngine_ChallengeCancelCommand_CancelsReadyAIFallbackChallenge(t *testing.T) {
	store := NewMemoryStore()
	challenges := NewMemoryChallengeStore()
	convID, err := store.CreateConversation(Conversation{UserID: "queue-ai-user"})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
		t.Fatalf("UpdateConversationTopicID() error = %v", err)
	}

	engine := NewEngine(EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createChallengeFallbackCurriculumLoader(t),
	})

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) error = %v", err)
	}

	challenges.mu.Lock()
	challenges.searches["queue-ai-user"].ExpiresAt = time.Now().Add(-time.Second)
	challenges.mu.Unlock()

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) after timeout error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge cancel",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge cancel) error = %v", err)
	}
	if !strings.Contains(resp, "Challenge cancelled.") {
		t.Fatalf("response = %q, want open challenge cancellation confirmation", resp)
	}

	resp, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-ai-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) after cancel error = %v", err)
	}
	if !strings.Contains(resp, "Searching for an opponent.") {
		t.Fatalf("response = %q, want search reopened after cancelling AI fallback", resp)
	}
}

func createChallengeFallbackCurriculumLoader(t *testing.T) *curriculum.Loader {
	t.Helper()

	dir := t.TempDir()
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	yamlPath := filepath.Join(topicsDir, "01-linear-equations.yaml")
	yamlData := `id: F1-02
name: Linear Equations
subject_id: math
syllabus_id: kssm-f1
difficulty: beginner
learning_objectives:
  - id: LO1
    text: Solve linear equations in one variable
    bloom: apply
`
	if err := os.WriteFile(yamlPath, []byte(yamlData), 0o644); err != nil {
		t.Fatalf("WriteFile(yaml) error = %v", err)
	}

	notesPath := filepath.Join(topicsDir, "01-linear-equations.teaching.md")
	if err := os.WriteFile(notesPath, []byte("# Linear Equations"), 0o644); err != nil {
		t.Fatalf("WriteFile(notes) error = %v", err)
	}

	assessmentPath := filepath.Join(topicsDir, "01-linear-equations.assessments.yaml")
	assessment := `topic_id: F1-02
provenance: human
questions:
  - id: Q1
    text: "Solve x + 3 = 7."
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "4"
      working: "Subtract 3 from both sides."
    marks: 1
`
	if err := os.WriteFile(assessmentPath, []byte(assessment), 0o644); err != nil {
		t.Fatalf("WriteFile(assessment) error = %v", err)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

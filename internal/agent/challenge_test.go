package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestMemoryChallengeStore_AIFallbackCompletesAndAwardsWinner(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	queued, err := store.CreatePublicQueue("creator", agent.ChallengeInput{
		TopicID:    "F1-02",
		TopicName:  "Linear Equations",
		SubjectID:  "math",
		SyllabusID: "kssm-f1",
		Metadata: agent.ChallengeMetadata{
			RequestedTopicID:   "F1-02",
			RequestedTopicName: "Linear Equations",
		},
	})
	if err != nil {
		t.Fatalf("CreatePublicQueue() error = %v", err)
	}

	active, err := store.ActivateAIFallback(queued.Code, agent.ChallengeInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SubjectID:     "math",
		SyllabusID:    "kssm-f1",
		Questions:     sampleChallengeQuestions(),
		QuestionCount: 2,
		Metadata: agent.ChallengeMetadata{
			AIProfile: &agent.ChallengeAIProfile{Label: "Adaptive AI rival", PlannedCorrect: 1},
		},
	})
	if err != nil {
		t.Fatalf("ActivateAIFallback() error = %v", err)
	}
	if active.OpponentType != "ai" {
		t.Fatalf("opponent type = %q, want ai", active.OpponentType)
	}

	completion, err := store.CompleteChallenge(active.Code, "creator", 2)
	if err != nil {
		t.Fatalf("CompleteChallenge() error = %v", err)
	}
	if !completion.AwardFinishXP {
		t.Fatal("finish xp not awarded")
	}
	if !completion.AwardWinnerXP {
		t.Fatal("winner xp not awarded")
	}
	if completion.Challenge.State != "completed" {
		t.Fatalf("state = %q, want completed", completion.Challenge.State)
	}
	if completion.WinnerUserID != "creator" {
		t.Fatalf("winner = %q, want creator", completion.WinnerUserID)
	}
}

func TestEngine_ChallengeCommand_EmptyState(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "challenge-empty",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("/challenge error = %v", err)
	}
	if !contains(resp, "You don't have an active challenge.") {
		t.Fatalf("response = %q, want empty state", resp)
	}
}

func TestEngine_PublicChallengeMatchesNearbyTopic(t *testing.T) {
	store := agent.NewMemoryStore()
	challenges := agent.NewMemoryChallengeStore()
	tracker := progress.NewMemoryTracker()
	loader := createChallengeMatchLoader(t)
	if err := store.SetUserForm("user-a", "1"); err != nil {
		t.Fatalf("SetUserForm(user-a) error = %v", err)
	}
	if err := store.SetUserForm("user-b", "1"); err != nil {
		t.Fatalf("SetUserForm(user-b) error = %v", err)
	}
	_ = tracker.UpdateMastery("user-a", "kssm-f1", "F1-02", 0.8)
	_ = tracker.UpdateMastery("user-a", "kssm-f1", "F1-03", 0.2)
	_ = tracker.UpdateMastery("user-b", "kssm-f1", "F1-02", 0.7)
	_ = tracker.UpdateMastery("user-b", "kssm-f1", "F1-03", 0.1)

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(ai.NewMockProvider("unused")),
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: loader,
		Tracker:          tracker,
	})

	waitingResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-a",
		Text:    "/challenge linear equations",
	})
	if err != nil {
		t.Fatalf("queue create error = %v", err)
	}
	if !contains(waitingResp, "Looking for a challenge opponent.") {
		t.Fatalf("response = %q, want queue status", waitingResp)
	}

	matchResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-b",
		Text:    "/challenge fractions",
	})
	if err != nil {
		t.Fatalf("match error = %v", err)
	}
	if !contains(matchResp, "Challenge matched.") {
		t.Fatalf("response = %q, want matched status", matchResp)
	}
	if !contains(matchResp, "Topic: Fractions") {
		t.Fatalf("response = %q, want nearby-topic selection", matchResp)
	}
}

func TestEngine_PrivateChallengeCreateJoinAndStart(t *testing.T) {
	mockAI := ai.NewMockProvider("unused")
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	createResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "private-creator",
		Text:    "/challenge private linear equations",
	})
	if err != nil {
		t.Fatalf("create private error = %v", err)
	}
	if !contains(createResp, "Code:") {
		t.Fatalf("response = %q, want private code", createResp)
	}
	code := extractChallengeCode(t, createResp)

	joinResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "private-opponent",
		Text:    "/challenge " + code,
	})
	if err != nil {
		t.Fatalf("join private error = %v", err)
	}
	if !contains(joinResp, "Challenge matched.") {
		t.Fatalf("response = %q, want matched state", joinResp)
	}

	creatorReadyResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "private-creator",
		Text:    "/challenge start",
	})
	if err != nil {
		t.Fatalf("creator start error = %v", err)
	}
	if !contains(creatorReadyResp, "You: ready") || !contains(creatorReadyResp, "Opponent: not ready") {
		t.Fatalf("response = %q, want readiness status", creatorReadyResp)
	}

	opponentStartResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "private-opponent",
		Text:    "/challenge start",
	})
	if err != nil {
		t.Fatalf("opponent start error = %v", err)
	}
	if !contains(opponentStartResp, "Question 1/3") {
		t.Fatalf("response = %q, want first challenge question", opponentStartResp)
	}
}

func TestEngine_ChallengeAIFallbackUsesChallengeOnlyXP(t *testing.T) {
	mockAI := ai.NewMockProvider("unused")
	store := agent.NewMemoryStore()
	tracker := progress.NewMemoryTracker()
	xpTracker := progress.NewMemoryXPTracker()
	now := time.Now()

	if err := store.SetUserForm("ai-user", "1"); err != nil {
		t.Fatalf("SetUserForm() error = %v", err)
	}
	_ = tracker.UpdateMastery("ai-user", "kssm-f1", "F1-02", 0.1)

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         mockRouter(mockAI),
		Store:            store,
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
		Tracker:          tracker,
		XP:               xpTracker,
		Now: func() time.Time {
			return now
		},
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "ai-user",
		Text:    "/challenge linear equations",
	})
	if err != nil {
		t.Fatalf("queue error = %v", err)
	}

	now = now.Add(31 * time.Second)
	startResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "ai-user",
		Text:    "/challenge start",
	})
	if err != nil {
		t.Fatalf("start error = %v", err)
	}
	if !contains(startResp, "Question 1/3") {
		t.Fatalf("response = %q, want challenge question", startResp)
	}

	nextResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "ai-user",
		Text:    "wrong",
	})
	if err != nil {
		t.Fatalf("wrong answer error = %v", err)
	}
	if !contains(nextResp, "Question 2/3") {
		t.Fatalf("response = %q, want advance after wrong answer", nextResp)
	}
	if contains(nextResp, "Try the same question again.") {
		t.Fatalf("response = %q, challenge should not retry same question", nextResp)
	}

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "ai-user",
		Text:    "4+3=7",
	})
	if err != nil {
		t.Fatalf("second answer error = %v", err)
	}
	finalResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "ai-user",
		Text:    "19",
	})
	if err != nil {
		t.Fatalf("final answer error = %v", err)
	}
	if !contains(finalResp, "Challenge result.") || !contains(finalResp, "Result: you win.") {
		t.Fatalf("response = %q, want final win", finalResp)
	}

	totalXP, err := xpTracker.GetTotal("ai-user")
	if err != nil {
		t.Fatalf("GetTotal() error = %v", err)
	}
	if totalXP != 45 {
		t.Fatalf("total xp = %d, want 45 (finish + win only)", totalXP)
	}
}

func sampleChallengeQuestions() []agent.QuizQuestion {
	return []agent.QuizQuestion{
		{ID: "Q1", Text: "Solve x + 3 = 7", AnswerType: "exact", Answer: "4"},
		{ID: "Q2", Text: "Solve 2x = 10", AnswerType: "exact", Answer: "5"},
	}
}

func extractChallengeCode(t *testing.T, response string) string {
	t.Helper()
	match := regexp.MustCompile(`Code:\s*([A-Z0-9]{6})`).FindStringSubmatch(response)
	if len(match) != 2 {
		t.Fatalf("response = %q, missing challenge code", response)
	}
	return match[1]
}

func createChallengeMatchLoader(t *testing.T) *curriculum.Loader {
	t.Helper()

	dir := t.TempDir()
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(topicsDir, name), []byte(data), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	write("01-linear-equations.yaml", `id: F1-02
name: Linear Equations
subject_id: math
syllabus_id: kssm-f1
difficulty: beginner
learning_objectives:
  - id: LO1
    text: Solve linear equations
    bloom: apply
`)
	write("01-linear-equations.assessments.yaml", `topic_id: F1-02
provenance: human
questions:
  - id: Q1
    text: "Solve x + 3 = 7"
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "4"
      working: "Subtract 3."
    marks: 1
`)

	write("02-fractions.yaml", `id: F1-03
name: Fractions
subject_id: math
syllabus_id: kssm-f1
difficulty: beginner
learning_objectives:
  - id: LO1
    text: Simplify fractions
    bloom: apply
`)
	write("02-fractions.assessments.yaml", `topic_id: F1-03
provenance: human
questions:
  - id: Q1
    text: "Simplify 2/4"
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "1/2"
      working: "Divide top and bottom by 2."
    marks: 1
`)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

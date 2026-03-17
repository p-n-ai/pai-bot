package agent_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

var challengeCodePattern = regexp.MustCompile(`Code:\s*([A-Z2-9]{6})`)

func TestGenerateChallengeCode(t *testing.T) {
	code := agent.GenerateChallengeCode()
	if len(code) != 6 {
		t.Fatalf("len(code) = %d, want 6", len(code))
	}
	if !regexp.MustCompile(`^[A-Z2-9]{6}$`).MatchString(code) {
		t.Fatalf("code = %q, want uppercase challenge alphabet", code)
	}
}

func TestMemoryChallengeStore_CreateInviteChallenge(t *testing.T) {
	store := agent.NewMemoryChallengeStore()

	ch, err := store.CreateInviteChallenge("user1", agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}
	if ch.Code == "" {
		t.Fatal("challenge code should not be empty")
	}
	if ch.CreatorID != "user1" {
		t.Fatalf("CreatorID = %q, want user1", ch.CreatorID)
	}
	if ch.State != agent.ChallengeStateWaiting {
		t.Fatalf("State = %q, want waiting", ch.State)
	}
	if ch.MatchSource != agent.ChallengeMatchSourceInviteCode {
		t.Fatalf("MatchSource = %q, want invite_code", ch.MatchSource)
	}
}

func TestMemoryChallengeStore_JoinChallenge(t *testing.T) {
	store := agent.NewMemoryChallengeStore()

	ch, err := store.CreateInviteChallenge("user1", agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	joined, err := store.JoinChallenge(ch.Code, "user2")
	if err != nil {
		t.Fatalf("JoinChallenge() error = %v", err)
	}
	if joined.OpponentID != "user2" {
		t.Fatalf("OpponentID = %q, want user2", joined.OpponentID)
	}
	if joined.State != agent.ChallengeStateReady {
		t.Fatalf("State = %q, want ready", joined.State)
	}
}

func TestMemoryChallengeStore_JoinChallengeRejectsSelfJoin(t *testing.T) {
	store := agent.NewMemoryChallengeStore()

	ch, err := store.CreateInviteChallenge("user1", agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	_, err = store.JoinChallenge(ch.Code, "user1")
	if err != agent.ErrChallengeSelfJoin {
		t.Fatalf("JoinChallenge() error = %v, want ErrChallengeSelfJoin", err)
	}
}

func TestEngine_ChallengeInviteCommand_CreatesChallenge(t *testing.T) {
	store := agent.NewMemoryStore()
	challenges := agent.NewMemoryChallengeStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel:   "telegram",
		UserID:    "challenge-creator",
		FirstName: "Aina",
		Text:      "/challenge invite linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Challenge created.") {
		t.Fatalf("response = %q, want create confirmation", resp)
	}
	if !contains(resp, "Linear Equations") {
		t.Fatalf("response = %q, want topic name", resp)
	}

	code := extractChallengeCode(t, resp)
	ch, err := challenges.GetChallenge(code)
	if err != nil {
		t.Fatalf("GetChallenge() error = %v", err)
	}
	if ch.CreatorID != "challenge-creator" {
		t.Fatalf("CreatorID = %q, want challenge-creator", ch.CreatorID)
	}
}

func TestEngine_ChallengeJoinCommand_JoinsReadyChallenge(t *testing.T) {
	store := agent.NewMemoryStore()
	challenges := agent.NewMemoryChallengeStore()
	if err := store.SetUserName("challenge-owner", "Aina"); err != nil {
		t.Fatalf("SetUserName() error = %v", err)
	}
	ch, err := challenges.CreateInviteChallenge("challenge-owner", agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "challenge-joiner",
		Text:    "/challenge " + ch.Code,
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "Joined challenge") {
		t.Fatalf("response = %q, want join confirmation", resp)
	}
	if !contains(resp, "Creator: Aina") {
		t.Fatalf("response = %q, want creator display name", resp)
	}

	joined, err := challenges.GetChallenge(ch.Code)
	if err != nil {
		t.Fatalf("GetChallenge() error = %v", err)
	}
	if joined.OpponentID != "challenge-joiner" {
		t.Fatalf("OpponentID = %q, want challenge-joiner", joined.OpponentID)
	}
	if joined.State != agent.ChallengeStateReady {
		t.Fatalf("State = %q, want ready", joined.State)
	}
}

func TestEngine_ChallengeCommand_BlocksDuringActiveQuiz(t *testing.T) {
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "challenge-quiz-user",
		Text:    "quiz me on linear equations",
	})
	if err != nil {
		t.Fatalf("quiz start error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "challenge-quiz-user",
		Text:    "/challenge invite linear equations",
	})
	if err != nil {
		t.Fatalf("/challenge error = %v", err)
	}
	if !contains(resp, "Finish or cancel the quiz first") {
		t.Fatalf("response = %q, want quiz-first guidance", resp)
	}
}

func extractChallengeCode(t *testing.T, response string) string {
	t.Helper()
	matches := challengeCodePattern.FindStringSubmatch(response)
	if len(matches) != 2 {
		t.Fatalf("response = %q, want challenge code", response)
	}
	return matches[1]
}

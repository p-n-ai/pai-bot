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

func TestMemoryChallengeStore_CreateInviteChallenge_RejectsWhenUserAlreadySearching(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
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
	if err != agent.ErrChallengeAlreadyActive {
		t.Fatalf("CreateInviteChallenge() error = %v, want ErrChallengeAlreadyActive", err)
	}
}

func TestMemoryChallengeStore_JoinChallenge_RejectsWhenUserAlreadySearching(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
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
	if err != agent.ErrChallengeAlreadyActive {
		t.Fatalf("JoinChallenge() error = %v, want ErrChallengeAlreadyActive", err)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_RejectsWhenUserHasLiveInviteChallenge(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	_, err := store.CreateInviteChallenge("queue-user-1", input)
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	_, err = store.StartChallengeSearch("queue-user-1", input)
	if err != agent.ErrChallengeAlreadyActive {
		t.Fatalf("StartChallengeSearch() error = %v, want ErrChallengeAlreadyActive", err)
	}
}

func TestMemoryChallengeStore_CancelOpenChallenge_CancelsWaitingInvite(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	challenge, err := store.CreateInviteChallenge("queue-user-1", input)
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	cancelled, err := store.CancelOpenChallenge("queue-user-1")
	if err != nil {
		t.Fatalf("CancelOpenChallenge() error = %v", err)
	}
	if !cancelled {
		t.Fatal("CancelOpenChallenge() = false, want true")
	}

	_, err = store.GetChallenge(challenge.Code)
	if err != agent.ErrChallengeNotFound {
		t.Fatalf("GetChallenge() error = %v, want ErrChallengeNotFound after cancel", err)
	}

	reopened, err := store.CreateInviteChallenge("queue-user-1", input)
	if err != nil {
		t.Fatalf("CreateInviteChallenge() after cancel error = %v", err)
	}
	if reopened.Code == challenge.Code {
		t.Fatalf("challenge code = %q, want a new invite after cancel", reopened.Code)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_PairsIntoPendingAcceptance(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	_, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-1) error = %v", err)
	}

	result, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch(queue-user-2) error = %v", err)
	}
	if result.Challenge == nil {
		t.Fatal("challenge should not be nil after pairing")
	}
	if result.Challenge.State != agent.ChallengeStatePendingAcceptance {
		t.Fatalf("challenge.State = %q, want %q", result.Challenge.State, agent.ChallengeStatePendingAcceptance)
	}
	if result.Challenge.ReadyAt != nil {
		t.Fatalf("challenge.ReadyAt = %v, want nil before both sides accept", result.Challenge.ReadyAt)
	}
	if result.Challenge.JoinDeadlineAt == nil {
		t.Fatal("challenge.JoinDeadlineAt should not be nil while awaiting acceptance")
	}
}

func TestMemoryChallengeStore_AcceptPendingChallenge_TransitionsToReadyAfterBothAccept(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
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
		t.Fatal("challenge should not be nil after pairing")
	}

	firstAccepted, err := store.AcceptPendingChallenge("queue-user-1")
	if err != nil {
		t.Fatalf("AcceptPendingChallenge(queue-user-1) error = %v", err)
	}
	if firstAccepted.State != agent.ChallengeStatePendingAcceptance {
		t.Fatalf("challenge.State after first accept = %q, want %q", firstAccepted.State, agent.ChallengeStatePendingAcceptance)
	}
	if firstAccepted.ReadyAt != nil {
		t.Fatalf("challenge.ReadyAt after first accept = %v, want nil", firstAccepted.ReadyAt)
	}
	if firstAccepted.CreatorAcceptedAt == nil {
		t.Fatal("CreatorAcceptedAt should not be nil after creator accepts")
	}
	if firstAccepted.OpponentAcceptedAt != nil {
		t.Fatalf("OpponentAcceptedAt = %v, want nil before second accept", firstAccepted.OpponentAcceptedAt)
	}

	secondAccepted, err := store.AcceptPendingChallenge("queue-user-2")
	if err != nil {
		t.Fatalf("AcceptPendingChallenge(queue-user-2) error = %v", err)
	}
	if secondAccepted.State != agent.ChallengeStateReady {
		t.Fatalf("challenge.State after second accept = %q, want %q", secondAccepted.State, agent.ChallengeStateReady)
	}
	if secondAccepted.ReadyAt == nil {
		t.Fatal("ReadyAt should not be nil after both sides accept")
	}
	if secondAccepted.CreatorAcceptedAt == nil || secondAccepted.OpponentAcceptedAt == nil {
		t.Fatal("both accepted timestamps should be present after second accept")
	}
	if secondAccepted.JoinDeadlineAt != nil {
		t.Fatalf("challenge.JoinDeadlineAt = %v, want nil once ready", secondAccepted.JoinDeadlineAt)
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

func TestEngine_ChallengeCommand_ShowsAcceptPromptWhenQueueMatchNeedsAcceptance(t *testing.T) {
	store := agent.NewMemoryStore()
	challenges := agent.NewMemoryChallengeStore()
	if err := store.SetUserName("queue-user-2", "Bala"); err != nil {
		t.Fatalf("SetUserName() error = %v", err)
	}
	for _, userID := range []string{"queue-user-1", "queue-user-2"} {
		convID, err := store.CreateConversation(agent.Conversation{UserID: userID})
		if err != nil {
			t.Fatalf("CreateConversation(%s) error = %v", userID, err)
		}
		if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
			t.Fatalf("UpdateConversationTopicID(%s) error = %v", userID, err)
		}
	}
	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-user-1",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(queue-user-1) error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-user-2",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(queue-user-2) error = %v", err)
	}
	if !contains(resp, "Opponent found.") {
		t.Fatalf("response = %q, want opponent found message", resp)
	}
	if !contains(resp, "State: pending_acceptance") {
		t.Fatalf("response = %q, want pending acceptance state", resp)
	}
	if !contains(resp, "Use: /challenge accept") {
		t.Fatalf("response = %q, want accept instruction", resp)
	}
}

func TestEngine_ChallengeAcceptCommand_TransitionsQueueMatchToReady(t *testing.T) {
	store := agent.NewMemoryStore()
	challenges := agent.NewMemoryChallengeStore()
	if err := store.SetUserName("queue-user-1", "Aina"); err != nil {
		t.Fatalf("SetUserName() error = %v", err)
	}
	if err := store.SetUserName("queue-user-2", "Bala"); err != nil {
		t.Fatalf("SetUserName() error = %v", err)
	}
	for _, userID := range []string{"queue-user-1", "queue-user-2"} {
		convID, err := store.CreateConversation(agent.Conversation{UserID: userID})
		if err != nil {
			t.Fatalf("CreateConversation(%s) error = %v", userID, err)
		}
		if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
			t.Fatalf("UpdateConversationTopicID(%s) error = %v", userID, err)
		}
	}
	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	for _, userID := range []string{"queue-user-1", "queue-user-2"} {
		_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
			Channel: "telegram",
			UserID:  userID,
			Text:    "/challenge",
		})
		if err != nil {
			t.Fatalf("ProcessMessage(%s /challenge) error = %v", userID, err)
		}
	}

	firstResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-user-1",
		Text:    "/challenge accept",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(queue-user-1 /challenge accept) error = %v", err)
	}
	if !contains(firstResp, "Accepted.") {
		t.Fatalf("response = %q, want accepted confirmation", firstResp)
	}
	if !contains(firstResp, "State: pending_acceptance") {
		t.Fatalf("response = %q, want pending acceptance state", firstResp)
	}

	secondResp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-user-2",
		Text:    "/challenge accept",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(queue-user-2 /challenge accept) error = %v", err)
	}
	if !contains(secondResp, "State: ready") {
		t.Fatalf("response = %q, want ready state", secondResp)
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

func TestMemoryChallengeStore_StartChallengeSearch_CreatesSearchingTicket(t *testing.T) {
	store := agent.NewMemoryChallengeStore()

	result, err := store.StartChallengeSearch("queue-user-1", agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	})
	if err != nil {
		t.Fatalf("StartChallengeSearch() error = %v", err)
	}
	if result.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil before pairing", result.Challenge)
	}
	if result.Search == nil {
		t.Fatal("ticket should not be nil")
	}
	if result.Search.Status != agent.MatchmakingStatusSearching {
		t.Fatalf("ticket.Status = %q, want %q", result.Search.Status, agent.MatchmakingStatusSearching)
	}
	if result.Search.UserID != "queue-user-1" {
		t.Fatalf("ticket.UserID = %q, want queue-user-1", result.Search.UserID)
	}
	if result.Search.TopicID != "F1-02" {
		t.Fatalf("ticket.TopicID = %q, want F1-02", result.Search.TopicID)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_ResumesExistingSearchingTicket(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
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
		t.Fatalf("challenge = %#v, want nil when resuming same search", secondResult.Challenge)
	}
	if firstResult.Search == nil || secondResult.Search == nil {
		t.Fatal("tickets should not be nil")
	}
	if firstResult.Search.ID == "" {
		t.Fatal("first ticket ID should not be empty")
	}
	if secondResult.Search.ID != firstResult.Search.ID {
		t.Fatalf("resumed ticket ID = %q, want %q", secondResult.Search.ID, firstResult.Search.ID)
	}
	if secondResult.Search.Status != agent.MatchmakingStatusSearching {
		t.Fatalf("ticket.Status = %q, want %q", secondResult.Search.Status, agent.MatchmakingStatusSearching)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_PairsCompatibleUsers(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
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
	if firstResult.Search.Status != agent.MatchmakingStatusSearching {
		t.Fatalf("ticket.Status = %q, want %q", firstResult.Search.Status, agent.MatchmakingStatusSearching)
	}

	secondResult, err := store.StartChallengeSearch("queue-user-2", input)
	if err != nil {
		t.Fatalf("second StartChallengeSearch() error = %v", err)
	}
	if secondResult.Search == nil {
		t.Fatal("second ticket should not be nil")
	}
	if secondResult.Challenge == nil {
		t.Fatal("matched challenge should not be nil after compatible second user joins the queue")
	}
	if secondResult.Search.Status != agent.MatchmakingStatusMatched {
		t.Fatalf("ticket.Status = %q, want %q", secondResult.Search.Status, agent.MatchmakingStatusMatched)
	}
	if secondResult.Challenge.MatchSource != agent.ChallengeMatchSourceQueue {
		t.Fatalf("challenge.MatchSource = %q, want %q", secondResult.Challenge.MatchSource, agent.ChallengeMatchSourceQueue)
	}
	if secondResult.Challenge.State != agent.ChallengeStatePendingAcceptance {
		t.Fatalf("challenge.State = %q, want %q", secondResult.Challenge.State, agent.ChallengeStatePendingAcceptance)
	}
	if secondResult.Challenge.ReadyAt != nil {
		t.Fatalf("challenge.ReadyAt = %v, want nil before acceptance completes", secondResult.Challenge.ReadyAt)
	}
	if secondResult.Challenge.CreatorID != "queue-user-1" {
		t.Fatalf("challenge.CreatorID = %q, want queue-user-1", secondResult.Challenge.CreatorID)
	}
	if secondResult.Challenge.OpponentID != "queue-user-2" {
		t.Fatalf("challenge.OpponentID = %q, want queue-user-2", secondResult.Challenge.OpponentID)
	}
}

func TestMemoryChallengeStore_StartChallengeSearch_ResumesMatchedChallenge(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
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
	if resumedResult.Search.Status != agent.MatchmakingStatusMatched {
		t.Fatalf("ticket.Status = %q, want %q", resumedResult.Search.Status, agent.MatchmakingStatusMatched)
	}
	if resumedResult.Challenge.ID != pairResult.Challenge.ID {
		t.Fatalf("challenge.ID = %q, want %q", resumedResult.Challenge.ID, pairResult.Challenge.ID)
	}
	if resumedResult.Challenge.State != agent.ChallengeStatePendingAcceptance {
		t.Fatalf("challenge.State = %q, want %q", resumedResult.Challenge.State, agent.ChallengeStatePendingAcceptance)
	}
}

func TestMemoryChallengeStore_CancelChallengeSearch_StopsActiveSearch(t *testing.T) {
	store := agent.NewMemoryChallengeStore()
	input := agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	}

	firstResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() error = %v", err)
	}
	if firstResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil before pairing", firstResult.Challenge)
	}

	cancelled, err := store.CancelChallengeSearch("queue-user-1")
	if err != nil {
		t.Fatalf("CancelChallengeSearch() error = %v", err)
	}
	if !cancelled {
		t.Fatal("CancelChallengeSearch() = false, want true")
	}

	secondResult, err := store.StartChallengeSearch("queue-user-1", input)
	if err != nil {
		t.Fatalf("StartChallengeSearch() after cancel error = %v", err)
	}
	if secondResult.Challenge != nil {
		t.Fatalf("challenge = %#v, want nil after re-opening search", secondResult.Challenge)
	}
	if secondResult.Search == nil || firstResult.Search == nil {
		t.Fatal("tickets should not be nil")
	}
	if secondResult.Search.Status != agent.MatchmakingStatusSearching {
		t.Fatalf("ticket.Status = %q, want %q", secondResult.Search.Status, agent.MatchmakingStatusSearching)
	}
	if secondResult.Search.ID == firstResult.Search.ID {
		t.Fatalf("ticket ID after cancel = %q, want a new active ticket", secondResult.Search.ID)
	}
}

func TestEngine_ChallengeCommand_StartsMatchmakingFromConversationTopic(t *testing.T) {
	store := agent.NewMemoryStore()
	convID, err := store.CreateConversation(agent.Conversation{UserID: "queue-engine-user"})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
		t.Fatalf("UpdateConversationTopicID() error = %v", err)
	}

	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-engine-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if !contains(resp, "searching") {
		t.Fatalf("response = %q, want searching state", resp)
	}
	if !contains(resp, "/challenge cancel") {
		t.Fatalf("response = %q, want cancel control", resp)
	}
}

func TestEngine_ChallengeCommand_CancelCancelsSearchingTicket(t *testing.T) {
	store := agent.NewMemoryStore()
	convID, err := store.CreateConversation(agent.Conversation{UserID: "queue-cancel-user"})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
		t.Fatalf("UpdateConversationTopicID() error = %v", err)
	}

	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-cancel-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-cancel-user",
		Text:    "/challenge cancel",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge cancel) error = %v", err)
	}
	if !contains(resp, "cancel") {
		t.Fatalf("response = %q, want cancellation confirmation", resp)
	}
}

func TestEngine_ChallengeCommand_CancelCancelsInviteChallenge(t *testing.T) {
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "invite-cancel-user",
		Text:    "/challenge invite linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge invite) error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "invite-cancel-user",
		Text:    "/challenge cancel",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge cancel) error = %v", err)
	}
	if !contains(resp, "Challenge cancelled.") {
		t.Fatalf("response = %q, want open challenge cancellation confirmation", resp)
	}

	resp, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "invite-cancel-user",
		Text:    "/challenge invite linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(second /challenge invite) error = %v", err)
	}
	if !contains(resp, "Challenge created.") {
		t.Fatalf("response = %q, want a fresh invite after cancel", resp)
	}
}

func TestEngine_ChallengeInviteCommand_RejectsWhenUserAlreadySearching(t *testing.T) {
	store := agent.NewMemoryStore()
	convID, err := store.CreateConversation(agent.Conversation{UserID: "queue-invite-user"})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
		t.Fatalf("UpdateConversationTopicID() error = %v", err)
	}

	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       agent.NewMemoryChallengeStore(),
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-invite-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-invite-user",
		Text:    "/challenge invite linear equations",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge invite) error = %v", err)
	}
	if !contains(resp, "already have a live challenge") {
		t.Fatalf("response = %q, want exclusivity guidance", resp)
	}
}

func TestEngine_ChallengeJoinCommand_RejectsWhenUserAlreadySearching(t *testing.T) {
	store := agent.NewMemoryStore()
	challenges := agent.NewMemoryChallengeStore()
	convID, err := store.CreateConversation(agent.Conversation{UserID: "queue-join-user"})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if err := store.UpdateConversationTopicID(convID, "F1-02"); err != nil {
		t.Fatalf("UpdateConversationTopicID() error = %v", err)
	}
	challenge, err := challenges.CreateInviteChallenge("challenge-owner", agent.ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 5,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}

	engine := agent.NewEngine(agent.EngineConfig{
		Store:            store,
		Challenges:       challenges,
		CurriculumLoader: createTestCurriculumLoader(t),
	})

	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-join-user",
		Text:    "/challenge",
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge) error = %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "queue-join-user",
		Text:    "/challenge " + challenge.Code,
	})
	if err != nil {
		t.Fatalf("ProcessMessage(/challenge <code>) error = %v", err)
	}
	if !contains(resp, "already have a live challenge") {
		t.Fatalf("response = %q, want exclusivity guidance", resp)
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

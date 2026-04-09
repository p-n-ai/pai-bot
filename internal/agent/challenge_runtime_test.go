package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func createChallengeRuntimeCurriculumLoader(t *testing.T) *curriculum.Loader {
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

	questions := ""
	for i := 1; i <= 5; i++ {
		questions += fmt.Sprintf(`  - id: Q%d
    text: "Solve x + %d = %d."
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "%d"
      working: "Subtract %d from both sides."
    marks: 1
`, i, i, i+3, 3, i)
	}

	assessmentPath := filepath.Join(topicsDir, "01-linear-equations.assessments.yaml")
	assessment := fmt.Sprintf(`topic_id: F1-02
provenance: human
questions:
%s`, questions)
	if err := os.WriteFile(assessmentPath, []byte(assessment), 0o644); err != nil {
		t.Fatalf("WriteFile(assessment) error = %v", err)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

func testChallengeEngine(t *testing.T) (*Engine, *MemoryStore, *MemoryChallengeStore, *progress.MemoryXPTracker) {
	t.Helper()
	store := NewMemoryStore()
	challengeStore := NewMemoryChallengeStore()
	xpTracker := progress.NewMemoryXPTracker()
	loader := createChallengeRuntimeCurriculumLoader(t)

	e := NewEngine(EngineConfig{
		Store:            store,
		Challenges:       challengeStore,
		XP:               xpTracker,
		CurriculumLoader: loader,
		EventLogger:      NopEventLogger{},
	})
	return e, store, challengeStore, xpTracker
}

func setupReadyChallenge(t *testing.T, challengeStore *MemoryChallengeStore, userID string) *Challenge {
	t.Helper()
	ch, err := challengeStore.CreateInviteChallenge(userID, ChallengeCreateInput{
		TopicID:       "F1-02",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-f1",
		QuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("CreateInviteChallenge() error = %v", err)
	}
	ch, err = challengeStore.JoinChallenge(ch.Code, "opponent1")
	if err != nil {
		t.Fatalf("JoinChallenge() error = %v", err)
	}
	if ch.State != ChallengeStateReady {
		t.Fatalf("challenge state = %q, want ready", ch.State)
	}
	return ch
}

func TestChallengeOwnsConversation(t *testing.T) {
	tests := []struct {
		name string
		conv *Conversation
		want bool
	}{
		{name: "nil conversation", conv: nil, want: false},
		{name: "teaching state", conv: &Conversation{State: conversationStateTeaching}, want: false},
		{name: "quiz active state", conv: &Conversation{State: conversationStateQuizActive}, want: false},
		{name: "challenge active state", conv: &Conversation{State: conversationStateChallengeActive}, want: true},
		{name: "challenge review state", conv: &Conversation{State: conversationStateChallengeReview}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := challengeOwnsConversation(tt.conv)
			if got != tt.want {
				t.Errorf("challengeOwnsConversation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartChallengeFromReadyState(t *testing.T) {
	e, store, challengeStore, _ := testChallengeEngine(t)
	userID := "user1"
	ch := setupReadyChallenge(t, challengeStore, userID)

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateTeaching})
	conv, _ := store.GetConversation(convID)

	msg := chat.InboundMessage{UserID: userID, Text: "ready", Channel: "test"}
	response := e.startChallengePlay(context.Background(), msg, conv, ch)

	if response == "" {
		t.Fatal("expected non-empty response from startChallengePlay")
	}
	if !strings.Contains(response, "Challenge:") {
		t.Errorf("response should contain 'Challenge:', got: %s", response)
	}
	if !strings.Contains(response, "Question 1/") {
		t.Errorf("response should contain 'Question 1/', got: %s", response)
	}
}

func TestChallengeAnswerCorrect(t *testing.T) {
	e, store, challengeStore, _ := testChallengeEngine(t)
	userID := "user1"
	ch := setupReadyChallenge(t, challengeStore, userID)
	_, _ = challengeStore.StartChallenge(ch.ID)

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
		{ID: "q2", Text: "Solve x + 2 = 7.", AnswerType: "exact", Answer: "5"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeActive})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeActive, ConversationChallengeState{
		ChallengeID:  ch.ID,
		Phase:        challengePhasePlaying,
		Questions:    questions,
		CurrentIndex: 0,
		CorrectCount: 0,
		Answers:      []ChallengeAnswerRecord{},
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "3", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected challenge turn to be handled")
	}
	// "Betul" (ms) or "Correct" (en) depending on default locale
	if !strings.Contains(response, "Betul") && !strings.Contains(response, "Correct") {
		t.Errorf("response should contain correct feedback, got: %s", response)
	}
	if !strings.Contains(response, "Question 2/") {
		t.Errorf("response should contain 'Question 2/', got: %s", response)
	}
}

func TestChallengeAnswerIncorrect_StillAdvances(t *testing.T) {
	e, store, challengeStore, _ := testChallengeEngine(t)
	userID := "user1"
	ch := setupReadyChallenge(t, challengeStore, userID)
	_, _ = challengeStore.StartChallenge(ch.ID)

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
		{ID: "q2", Text: "Solve x + 2 = 7.", AnswerType: "exact", Answer: "5"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeActive})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeActive, ConversationChallengeState{
		ChallengeID:  ch.ID,
		Phase:        challengePhasePlaying,
		Questions:    questions,
		CurrentIndex: 0,
		CorrectCount: 0,
		Answers:      []ChallengeAnswerRecord{},
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "wrong answer", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected challenge turn to be handled")
	}
	// Key difference from quiz: wrong answer should still advance
	if !strings.Contains(response, "Question 2/") {
		t.Errorf("response should advance to 'Question 2/' even on wrong answer, got: %s", response)
	}
}

func TestChallengeCompletion_WithMissed_OffersReview(t *testing.T) {
	e, store, challengeStore, _ := testChallengeEngine(t)
	userID := "user1"
	ch := setupReadyChallenge(t, challengeStore, userID)
	_, _ = challengeStore.StartChallenge(ch.ID)

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeActive})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeActive, ConversationChallengeState{
		ChallengeID:  ch.ID,
		Phase:        challengePhasePlaying,
		Questions:    questions,
		CurrentIndex: 0,
		CorrectCount: 0,
		Answers:      []ChallengeAnswerRecord{},
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "wrong", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected challenge turn to be handled")
	}
	if !strings.Contains(response, "0/1") {
		t.Errorf("response should contain '0/1', got: %s", response)
	}
	if !strings.Contains(strings.ToLower(response), "review") && !strings.Contains(strings.ToLower(response), "ulang") {
		t.Errorf("response should offer review, got: %s", response)
	}
}

func TestChallengeCompletion_PerfectScore_NoReview(t *testing.T) {
	e, store, challengeStore, _ := testChallengeEngine(t)
	userID := "user1"
	ch := setupReadyChallenge(t, challengeStore, userID)
	_, _ = challengeStore.StartChallenge(ch.ID)

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeActive})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeActive, ConversationChallengeState{
		ChallengeID:  ch.ID,
		Phase:        challengePhasePlaying,
		Questions:    questions,
		CurrentIndex: 0,
		CorrectCount: 0,
		Answers:      []ChallengeAnswerRecord{},
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "3", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected challenge turn to be handled")
	}
	if !strings.Contains(response, "1/1") {
		t.Errorf("response should contain '1/1', got: %s", response)
	}
	// Perfect score should not offer review (no "review" or "ulang" in the response after the score line)
	if strings.Contains(strings.ToLower(response), "review") || strings.Contains(strings.ToLower(response), "ulang kaji") {
		t.Errorf("perfect score should NOT offer review, got: %s", response)
	}

	// Should return to teaching state
	conv, _ = store.GetConversation(convID)
	if conv.ChallengeState != nil {
		t.Error("challenge state should be cleared after perfect score")
	}
}

func TestChallengeReview_AcceptReview(t *testing.T) {
	e, store, _, _ := testChallengeEngine(t)
	userID := "user1"

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
		{ID: "q2", Text: "Solve x + 2 = 7.", AnswerType: "exact", Answer: "5"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeActive})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeActive, ConversationChallengeState{
		ChallengeID:   "ch1",
		Phase:         challengePhaseReviewOffered,
		Questions:     questions,
		CurrentIndex:  2,
		CorrectCount:  1,
		Answers:       []ChallengeAnswerRecord{{QuestionIndex: 0, UserAnswer: "wrong", Correct: false}, {QuestionIndex: 1, UserAnswer: "5", Correct: true}},
		MissedIndices: []int{0},
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "review", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected review accept to be handled")
	}
	if !strings.Contains(response, "Review Question 1/") {
		t.Errorf("response should contain 'Review Question 1/', got: %s", response)
	}
}

func TestChallengeReview_SkipReview(t *testing.T) {
	e, store, _, _ := testChallengeEngine(t)
	userID := "user1"

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeActive})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeActive, ConversationChallengeState{
		ChallengeID:   "ch1",
		Phase:         challengePhaseReviewOffered,
		Questions:     []QuizQuestion{{ID: "q1", Text: "Q1", AnswerType: "exact", Answer: "3"}},
		CurrentIndex:  1,
		CorrectCount:  0,
		MissedIndices: []int{0},
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "no thanks", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected skip to be handled")
	}
	conv, _ = store.GetConversation(convID)
	if conv.ChallengeState != nil {
		t.Error("challenge state should be cleared after skipping review")
	}
	if conv.State != conversationStateTeaching {
		t.Errorf("state = %q, want teaching", conv.State)
	}
	if response == "" {
		t.Error("expected non-empty response for skip")
	}
}

func TestChallengeReview_CorrectAnswer_Advances(t *testing.T) {
	e, store, _, _ := testChallengeEngine(t)
	userID := "user1"

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
		{ID: "q2", Text: "Solve x + 2 = 7.", AnswerType: "exact", Answer: "5"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeReview})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeReview, ConversationChallengeState{
		ChallengeID:   "ch1",
		Phase:         challengePhaseReviewing,
		Questions:     questions,
		CurrentIndex:  2,
		CorrectCount:  0,
		MissedIndices: []int{0, 1},
		ReviewIndex:   0,
		ReviewCorrect: 0,
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "3", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected review turn to be handled")
	}
	if !strings.Contains(response, "Betul") && !strings.Contains(response, "Correct") {
		t.Errorf("response should contain correct feedback, got: %s", response)
	}
	if !strings.Contains(response, "Review Question 2/") {
		t.Errorf("response should contain 'Review Question 2/', got: %s", response)
	}
}

func TestChallengeReview_Completion_AwardsXP(t *testing.T) {
	e, store, _, xpTracker := testChallengeEngine(t)
	userID := "user1"

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeReview})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeReview, ConversationChallengeState{
		ChallengeID:   "ch1",
		Phase:         challengePhaseReviewing,
		Questions:     questions,
		CurrentIndex:  1,
		CorrectCount:  0,
		MissedIndices: []int{0},
		ReviewIndex:   0,
		ReviewCorrect: 0,
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "3", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected review completion to be handled")
	}
	if !strings.Contains(response, "+50 XP") {
		t.Errorf("response should contain '+50 XP', got: %s", response)
	}

	total, err := xpTracker.GetTotal(userID)
	if err != nil {
		t.Fatalf("GetTotal() error = %v", err)
	}
	if total != progress.XPReviewCompleted {
		t.Errorf("total XP = %d, want %d", total, progress.XPReviewCompleted)
	}

	conv, _ = store.GetConversation(convID)
	if conv.ChallengeState != nil {
		t.Error("challenge state should be cleared after review completion")
	}
}

func TestChallengeReview_WrongAnswer_AllowsRetry(t *testing.T) {
	e, store, _, _ := testChallengeEngine(t)
	userID := "user1"

	questions := []QuizQuestion{
		{ID: "q1", Text: "Solve x + 1 = 4.", AnswerType: "exact", Answer: "3"},
	}

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeReview})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeReview, ConversationChallengeState{
		ChallengeID:   "ch1",
		Phase:         challengePhaseReviewing,
		Questions:     questions,
		CurrentIndex:  1,
		CorrectCount:  0,
		MissedIndices: []int{0},
		ReviewIndex:   0,
		ReviewCorrect: 0,
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "wrong", Channel: "test"}
	response, handled := e.maybeHandleChallengeTurn(context.Background(), msg, conv)

	if !handled {
		t.Fatal("expected review retry to be handled")
	}
	if strings.Contains(response, "Review Question 2/") {
		t.Errorf("wrong review answer should NOT advance, got: %s", response)
	}
	// "Cuba lagi" (ms), "Try again" (en), or "再试" (zh)
	lower := strings.ToLower(response)
	if !strings.Contains(lower, "try again") && !strings.Contains(lower, "cuba lagi") && !strings.Contains(lower, "再试") {
		t.Errorf("response should encourage retry, got: %s", response)
	}
}

func TestQuizBlockedDuringChallenge(t *testing.T) {
	e, store, _, _ := testChallengeEngine(t)
	userID := "user1"

	convID, _ := store.CreateConversation(Conversation{UserID: userID, State: conversationStateChallengeActive})
	_ = store.UpdateConversationChallengeState(convID, conversationStateChallengeActive, ConversationChallengeState{
		ChallengeID: "ch1",
		Phase:       challengePhasePlaying,
		Questions:   []QuizQuestion{{ID: "q1", Text: "Q1", AnswerType: "exact", Answer: "3"}},
	})

	conv, _ := store.GetConversation(convID)
	msg := chat.InboundMessage{UserID: userID, Text: "quiz", Channel: "test"}
	response, handled := e.maybeHandleQuizTurn(context.Background(), msg, conv)

	if handled {
		t.Errorf("quiz should NOT be handled during challenge, but got response: %s", response)
	}
}

func TestRenderChallengeQuestion(t *testing.T) {
	q := QuizQuestion{ID: "q1", Text: "Solve x + 2 = 5", AnswerType: "exact", Answer: "3"}
	result := renderChallengeQuestion("Linear Equations", 0, 3, q)

	if !strings.Contains(result, "Challenge:") {
		t.Errorf("should contain 'Challenge:', got: %s", result)
	}
	if !strings.Contains(result, "Linear Equations") {
		t.Errorf("should contain topic name, got: %s", result)
	}
	if !strings.Contains(result, "Question 1/3") {
		t.Errorf("should contain 'Question 1/3', got: %s", result)
	}
}

func TestRenderChallengeResult(t *testing.T) {
	result := renderChallengeResultLocalized("en", 3, 5)
	if !strings.Contains(result, "3/5") {
		t.Errorf("should contain '3/5', got: %s", result)
	}
	if !strings.Contains(result, "60%") {
		t.Errorf("should contain '60%%', got: %s", result)
	}
}

func TestRenderChallengeReviewOffer(t *testing.T) {
	result := renderChallengeReviewOfferLocalized("en", 2)
	if !strings.Contains(result, "2") {
		t.Errorf("should contain missed count, got: %s", result)
	}
	if !strings.Contains(strings.ToLower(result), "review") {
		t.Errorf("should mention review, got: %s", result)
	}
}

func TestRenderChallengeReviewComplete(t *testing.T) {
	result := renderChallengeReviewCompleteLocalized("en", 2, 3)
	if !strings.Contains(result, "+50 XP") {
		t.Errorf("should contain '+50 XP', got: %s", result)
	}
}

func TestMemoryChallengeStore_StartChallenge(t *testing.T) {
	s := NewMemoryChallengeStore()
	ch, _ := s.CreateInviteChallenge("user1", ChallengeCreateInput{
		TopicID: "F1-02", TopicName: "Linear Equations", SyllabusID: "kssm-f1", QuestionCount: 3,
	})
	ch, _ = s.JoinChallenge(ch.Code, "user2")

	started, err := s.StartChallenge(ch.ID)
	if err != nil {
		t.Fatalf("StartChallenge() error = %v", err)
	}
	if started.State != ChallengeStateActive {
		t.Errorf("state = %q, want active", started.State)
	}

	_, err = s.StartChallenge(ch.ID)
	if err == nil {
		t.Error("expected error when starting non-ready challenge")
	}
}

func TestMemoryChallengeStore_CompleteChallenge(t *testing.T) {
	s := NewMemoryChallengeStore()
	ch, _ := s.CreateInviteChallenge("user1", ChallengeCreateInput{
		TopicID: "F1-02", TopicName: "Linear Equations", SyllabusID: "kssm-f1", QuestionCount: 3,
	})
	ch, _ = s.JoinChallenge(ch.Code, "user2")
	ch, _ = s.StartChallenge(ch.ID)

	completed, err := s.CompleteChallenge(ch.ID)
	if err != nil {
		t.Fatalf("CompleteChallenge() error = %v", err)
	}
	if completed.State != ChallengeStateCompleted {
		t.Errorf("state = %q, want completed", completed.State)
	}
}

func TestMemoryChallengeStore_GetActiveChallengeForUser(t *testing.T) {
	s := NewMemoryChallengeStore()
	ch, _ := s.CreateInviteChallenge("user1", ChallengeCreateInput{
		TopicID: "F1-02", TopicName: "Linear Equations", SyllabusID: "kssm-f1", QuestionCount: 3,
	})
	ch, _ = s.JoinChallenge(ch.Code, "user2")

	found, err := s.GetActiveChallengeForUser("user1")
	if err != nil {
		t.Fatalf("GetActiveChallengeForUser() error = %v", err)
	}
	if found.ID != ch.ID {
		t.Errorf("found challenge ID = %q, want %q", found.ID, ch.ID)
	}
}

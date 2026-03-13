package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestEngine_GoalCommand_EmptyState(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		Goals: agent.NewMemoryGoalStore(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-empty-user",
		Text:    "/goal",
	})
	if err != nil {
		t.Fatalf("/goal error = %v", err)
	}
	if !contains(resp, "You don't have any active goals yet.") {
		t.Fatalf("response = %q, want empty goal state", resp)
	}
}

func TestEngine_GoalCommand_AddsSpecificGoalImmediately(t *testing.T) {
	goalStore := agent.NewMemoryGoalStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Goals:           goalStore,
		ContextResolver: keywordGoalResolver(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-user",
		Text:    "/goal help me master linear equations",
	})
	if err != nil {
		t.Fatalf("/goal error = %v", err)
	}
	if !contains(resp, "Goal saved.") {
		t.Fatalf("response = %q, want goal confirmation", resp)
	}
	if !contains(resp, "0% / 75%") {
		t.Fatalf("response = %q, want default target line", resp)
	}

	goals, err := goalStore.ListActiveGoals("goal-user")
	if err != nil {
		t.Fatalf("ListActiveGoals() error = %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("active goals = %d, want 1", len(goals))
	}
}

func TestEngine_GoalCommand_AddsSpecificGoalImmediatelyForBilingualTopicName(t *testing.T) {
	goalStore := agent.NewMemoryGoalStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Goals: goalStore,
		ContextResolver: &goalKeywordResolver{
			topics: map[string]*curriculum.Topic{
				"linear equations": {ID: "algebra-linear-eq", Name: "Persamaan Linear (Linear Equations)", SyllabusID: "kssm-form1"},
			},
		},
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-bilingual-user",
		Text:    "/goal help me master linear equations",
	})
	if err != nil {
		t.Fatalf("/goal error = %v", err)
	}
	if !contains(resp, "Goal saved.") {
		t.Fatalf("response = %q, want goal confirmation", resp)
	}
	if contains(resp, "Reply yes to save it") {
		t.Fatalf("response = %q, should not require confirmation", resp)
	}
}

func TestEngine_GoalCommand_UsesAIParseWhenAvailable(t *testing.T) {
	goalStore := agent.NewMemoryGoalStore()
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond},
		BreakerFailureThreshold: 3,
		BreakerCooldown:         10 * time.Millisecond,
	})
	router.Register("openai", ai.NewMockProvider(`{"goal_summary":"Reach 80% mastery in Linear Equations","target_mastery":0.8,"needs_confirmation":false}`))
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:        router,
		Goals:           goalStore,
		ContextResolver: keywordGoalResolver(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-ai-user",
		Text:    "/goal help me master linear equations",
	})
	if err != nil {
		t.Fatalf("/goal error = %v", err)
	}
	if !contains(resp, "0% / 80%") {
		t.Fatalf("response = %q, want AI target line", resp)
	}
}

func TestEngine_GoalCommand_ListsMultipleGoalsNewestFirst(t *testing.T) {
	goalStore := agent.NewMemoryGoalStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Goals:           goalStore,
		ContextResolver: keywordGoalResolver(),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-list-user",
		Text:    "/goal help me master linear equations",
	})
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-list-user",
		Text:    "/goal help me master fractions",
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-list-user",
		Text:    "/goal",
	})
	if err != nil {
		t.Fatalf("/goal list error = %v", err)
	}
	if !contains(resp, "Reach 75% mastery in Fractions") || !contains(resp, "Reach 75% mastery in Linear Equations") {
		t.Fatalf("response = %q, want both goals listed", resp)
	}
	if idxFractions, idxLinear := indexOf(resp, "Fractions"), indexOf(resp, "Linear Equations"); idxFractions == -1 || idxLinear == -1 || idxFractions > idxLinear {
		t.Fatalf("response = %q, want newest goal listed first", resp)
	}
}

func TestEngine_GoalCommand_ClearArchivesAllGoals(t *testing.T) {
	goalStore := agent.NewMemoryGoalStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Goals:           goalStore,
		ContextResolver: keywordGoalResolver(),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-clear-user",
		Text:    "/goal help me master linear equations",
	})
	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-clear-user",
		Text:    "/goal help me master fractions",
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-clear-user",
		Text:    "/goal clear",
	})
	if err != nil {
		t.Fatalf("/goal clear error = %v", err)
	}
	if !contains(resp, "All active goals cleared.") {
		t.Fatalf("response = %q, want clear confirmation", resp)
	}

	goals, err := goalStore.ListActiveGoals("goal-clear-user")
	if err != nil {
		t.Fatalf("ListActiveGoals() error = %v", err)
	}
	if len(goals) != 0 {
		t.Fatalf("active goals = %d, want 0", len(goals))
	}
}

func TestEngine_GoalCommand_VagueGoalSuggestsPendingGoal(t *testing.T) {
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Store:           store,
		Goals:           agent.NewMemoryGoalStore(),
		ContextResolver: keywordGoalResolver(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-vague-user",
		Text:    "/goal I want to get better at algebra",
	})
	if err != nil {
		t.Fatalf("/goal vague error = %v", err)
	}
	if !contains(resp, "Reply yes to save it") {
		t.Fatalf("response = %q, want suggestion confirmation prompt", resp)
	}

	conv, found := store.GetActiveConversation("goal-vague-user")
	if !found || conv.PendingGoal == nil {
		t.Fatalf("pending goal draft missing: found=%v conv=%#v", found, conv)
	}
}

func TestEngine_GoalCommand_VagueGoalStillNeedsConfirmationWhenAIOverreaches(t *testing.T) {
	store := agent.NewMemoryStore()
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond},
		BreakerFailureThreshold: 3,
		BreakerCooldown:         10 * time.Millisecond,
	})
	router.Register("openai", ai.NewMockProvider(`{"goal_summary":"Master algebra quickly","target_mastery":0.85,"needs_confirmation":false}`))
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:        router,
		Store:           store,
		Goals:           agent.NewMemoryGoalStore(),
		ContextResolver: keywordGoalResolver(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-vague-ai-user",
		Text:    "/goal I want to get better at algebra",
	})
	if err != nil {
		t.Fatalf("/goal vague error = %v", err)
	}
	if !contains(resp, "Reply yes to save it") {
		t.Fatalf("response = %q, want suggestion confirmation prompt", resp)
	}
	if contains(resp, "Goal saved.") {
		t.Fatalf("response = %q, should not create goal immediately", resp)
	}
}

func TestEngine_GoalCommand_ConfirmationCreatesPendingGoal(t *testing.T) {
	store := agent.NewMemoryStore()
	goalStore := agent.NewMemoryGoalStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Store:           store,
		Goals:           goalStore,
		ContextResolver: keywordGoalResolver(),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-confirm-user",
		Text:    "/goal I want to get better at algebra",
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-confirm-user",
		Text:    "yes",
	})
	if err != nil {
		t.Fatalf("goal confirmation error = %v", err)
	}
	if !contains(resp, "Goal saved.") {
		t.Fatalf("response = %q, want saved confirmation", resp)
	}

	goals, err := goalStore.ListActiveGoals("goal-confirm-user")
	if err != nil {
		t.Fatalf("ListActiveGoals() error = %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("active goals = %d, want 1", len(goals))
	}

	conv, found := store.GetActiveConversation("goal-confirm-user")
	if !found || conv.PendingGoal != nil {
		t.Fatalf("pending goal draft should be cleared: found=%v conv=%#v", found, conv)
	}
}

func TestEngine_GoalCommand_RewriteAfterSuggestionReparses(t *testing.T) {
	store := agent.NewMemoryStore()
	goalStore := agent.NewMemoryGoalStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Store:           store,
		Goals:           goalStore,
		ContextResolver: keywordGoalResolver(),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-rewrite-user",
		Text:    "/goal I want to get better at algebra",
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-rewrite-user",
		Text:    "help me master linear equations to 80%",
	})
	if err != nil {
		t.Fatalf("goal rewrite error = %v", err)
	}
	if !contains(resp, "Goal saved.") || !contains(resp, "0% / 80%") {
		t.Fatalf("response = %q, want rewritten goal creation", resp)
	}

	goals, err := goalStore.ListActiveGoals("goal-rewrite-user")
	if err != nil {
		t.Fatalf("ListActiveGoals() error = %v", err)
	}
	if len(goals) != 1 || goals[0].TargetMastery != 0.8 {
		t.Fatalf("active goals = %#v, want one 80%% goal", goals)
	}
}

func TestEngine_GoalCommand_UnresolvedTopicReturnsExamples(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		Goals: agent.NewMemoryGoalStore(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-unresolved-user",
		Text:    "/goal help me pass exams",
	})
	if err != nil {
		t.Fatalf("/goal unresolved error = %v", err)
	}
	if !contains(resp, "I couldn't map that to a topic yet.") {
		t.Fatalf("response = %q, want unresolved-goal guidance", resp)
	}
}

func TestMemoryGoalStore_SyncGoalProgress_CompletesMatchingGoals(t *testing.T) {
	store := agent.NewMemoryGoalStore()
	_, _ = store.AddGoal("user-1", agent.GoalInput{
		Summary:       "Reach 60% mastery in Linear Equations",
		TopicID:       "algebra-linear-eq",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-form1",
		TargetMastery: 0.6,
	})
	_, _ = store.AddGoal("user-1", agent.GoalInput{
		Summary:       "Reach 80% mastery in Linear Equations",
		TopicID:       "algebra-linear-eq",
		TopicName:     "Linear Equations",
		SyllabusID:    "kssm-form1",
		TargetMastery: 0.8,
	})

	updated, err := store.SyncGoalProgress("user-1", "kssm-form1", "algebra-linear-eq", 0.7)
	if err != nil {
		t.Fatalf("SyncGoalProgress() error = %v", err)
	}
	if len(updated) != 2 {
		t.Fatalf("updated goals = %d, want 2", len(updated))
	}

	active, err := store.ListActiveGoals("user-1")
	if err != nil {
		t.Fatalf("ListActiveGoals() error = %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("active goals = %d, want 1", len(active))
	}
}

func TestEngine_ProgressCommand_IncludesActiveGoals(t *testing.T) {
	progressTracker := progress.NewMemoryTracker()
	engine := agent.NewEngine(agent.EngineConfig{
		Tracker:         progressTracker,
		Goals:           agent.NewMemoryGoalStore(),
		ContextResolver: keywordGoalResolver(),
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-progress-user",
		Text:    "/goal help me master linear equations",
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "goal-progress-user",
		Text:    "/progress",
	})
	if err != nil {
		t.Fatalf("/progress error = %v", err)
	}
	if !contains(resp, "🎯 Active Goals") || !contains(resp, "Reach 75% mastery in Linear Equations") {
		t.Fatalf("response = %q, want goal section", resp)
	}
}

func keywordGoalResolver() *goalKeywordResolver {
	return &goalKeywordResolver{
		topics: map[string]*curriculum.Topic{
			"linear equations": {ID: "algebra-linear-eq", Name: "Linear Equations", SyllabusID: "kssm-form1"},
			"fractions":        {ID: "fractions", Name: "Fractions", SyllabusID: "kssm-form1"},
			"algebra":          {ID: "algebra-linear-eq", Name: "Linear Equations", SyllabusID: "kssm-form1"},
		},
	}
}

type goalKeywordResolver struct {
	topics map[string]*curriculum.Topic
}

func (r *goalKeywordResolver) Resolve(text string) (*curriculum.Topic, string) {
	for keyword, topic := range r.topics {
		if contains(lower(text), keyword) {
			return topic, ""
		}
	}
	return nil, ""
}

func lower(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			out[i] = c + ('a' - 'A')
		} else {
			out[i] = c
		}
	}
	return string(out)
}

func indexOf(s, needle string) int {
	for i := 0; i+len(needle) <= len(s); i++ {
		if s[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

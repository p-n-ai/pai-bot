package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

const defaultGoalTargetMastery = 0.75

var goalPercentPattern = regexp.MustCompile(`(?i)(\d{1,3})\s*%`)

var goalConfirmationReplies = map[string]struct{}{
	"yes":         {},
	"y":           {},
	"ok":          {},
	"okay":        {},
	"confirm":     {},
	"sounds good": {},
	"yes please":  {},
}

var goalCancelReplies = map[string]struct{}{
	"cancel": {},
	"stop":   {},
	"no":     {},
}

// Goal tracks a student's mastery target for a topic.
type Goal struct {
	ID             string
	UserID         string
	Summary        string
	TopicID        string
	TopicName      string
	SyllabusID     string
	TargetMastery  float64
	CurrentMastery float64
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

// GoalInput captures new goal creation input.
type GoalInput struct {
	Summary        string
	TopicID        string
	TopicName      string
	SyllabusID     string
	TargetMastery  float64
	CurrentMastery float64
}

// GoalStore persists goals separately from conversation state.
type GoalStore interface {
	AddGoal(userID string, input GoalInput) (*Goal, error)
	ListActiveGoals(userID string) ([]*Goal, error)
	ClearActiveGoals(userID string) error
	SyncGoalProgress(userID, syllabusID, topicID string, mastery float64) ([]*Goal, error)
}

// MemoryGoalStore is an in-memory GoalStore.
type MemoryGoalStore struct {
	mu    sync.RWMutex
	goals map[string][]*Goal
}

func NewMemoryGoalStore() *MemoryGoalStore {
	return &MemoryGoalStore{goals: make(map[string][]*Goal)}
}

func (s *MemoryGoalStore) AddGoal(userID string, input GoalInput) (*Goal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	goal := newGoalRecord(userID, input, now)
	goal.ID = generateID()
	s.goals[userID] = append(s.goals[userID], goal)
	return cloneGoal(goal), nil
}

func (s *MemoryGoalStore) ListActiveGoals(userID string) ([]*Goal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	goals := activeGoalsDescending(s.goals[userID])
	return cloneGoalSlice(goals), nil
}

func (s *MemoryGoalStore) ClearActiveGoals(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, goal := range s.goals[userID] {
		if goal != nil && goal.Status == "active" {
			goal.Status = "archived"
			goal.UpdatedAt = now
		}
	}
	return nil
}

func (s *MemoryGoalStore) SyncGoalProgress(userID, syllabusID, topicID string, mastery float64) ([]*Goal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var updated []*Goal
	now := time.Now()
	for _, goal := range s.goals[userID] {
		if goal == nil || goal.Status != "active" {
			continue
		}
		if goal.TopicID != topicID || goal.SyllabusID != syllabusID {
			continue
		}
		goal.CurrentMastery = clampGoalMastery(mastery)
		goal.UpdatedAt = now
		markGoalCompletedIfReached(goal, now)
		updated = append(updated, cloneGoal(goal))
	}
	return updated, nil
}

// PostgresGoalStore persists goals in PostgreSQL.
type PostgresGoalStore struct {
	pool     *pgxpool.Pool
	tenantID string
	channel  string
}

func NewPostgresGoalStore(pool *pgxpool.Pool, tenantID string) *PostgresGoalStore {
	return &PostgresGoalStore{
		pool:     pool,
		tenantID: tenantID,
		channel:  defaultChannel,
	}
}

func (s *PostgresGoalStore) AddGoal(externalID string, input GoalInput) (*Goal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	goal := newGoalRecord(externalID, input, time.Time{})
	goal.CompletedAt = nil

	var completedAt *time.Time
	err := s.pool.QueryRow(ctx,
		`INSERT INTO goals (user_id, tenant_id, summary, topic_id, topic_name, syllabus_id, target_mastery, current_mastery, status, completed_at)
		 VALUES (
		   (
		     SELECT id FROM users
		     WHERE tenant_id = $1::uuid AND channel = $2 AND external_id = $3
		     ORDER BY created_at ASC
		     LIMIT 1
		   ),
		   $1::uuid, $4, $5, $6, $7, $8, $9, $10,
		   CASE WHEN $10 = 'completed' THEN NOW() ELSE NULL END
		 )
		 RETURNING id::text, created_at, updated_at, completed_at`,
		s.tenantID,
		s.channel,
		externalID,
		goal.Summary,
		goal.TopicID,
		goal.TopicName,
		goal.SyllabusID,
		goal.TargetMastery,
		goal.CurrentMastery,
		goal.Status,
	).Scan(&goal.ID, &goal.CreatedAt, &goal.UpdatedAt, &completedAt)
	if err != nil {
		return nil, fmt.Errorf("insert goal: %w", err)
	}
	goal.CompletedAt = completedAt
	return goal, nil
}

func (s *PostgresGoalStore) ListActiveGoals(externalID string) ([]*Goal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT g.id::text, u.external_id, g.summary, g.topic_id, g.topic_name, g.syllabus_id, g.target_mastery, g.current_mastery, g.status, g.created_at, g.updated_at, g.completed_at
		 FROM goals g
		 JOIN users u ON u.id = g.user_id
		 WHERE g.tenant_id = $1::uuid
		   AND u.channel = $2
		   AND u.external_id = $3
		   AND g.status = 'active'
		 ORDER BY g.created_at DESC`,
		s.tenantID,
		s.channel,
		externalID,
	)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	defer rows.Close()

	var goals []*Goal
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		goals = append(goals, goal)
	}
	return goals, rows.Err()
}

func (s *PostgresGoalStore) ClearActiveGoals(externalID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`UPDATE goals
		 SET status = 'archived',
		     updated_at = NOW()
		 WHERE tenant_id = $1::uuid
		   AND status = 'active'
		   AND user_id = (
		     SELECT id FROM users
		     WHERE tenant_id = $1::uuid AND channel = $2 AND external_id = $3
		     ORDER BY created_at ASC
		     LIMIT 1
		   )`,
		s.tenantID,
		s.channel,
		externalID,
	)
	if err != nil {
		return fmt.Errorf("clear goals: %w", err)
	}
	return nil
}

func (s *PostgresGoalStore) SyncGoalProgress(externalID, syllabusID, topicID string, mastery float64) ([]*Goal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`UPDATE goals g
		 SET current_mastery = $4,
		     updated_at = NOW(),
		     status = CASE WHEN $4 >= g.target_mastery THEN 'completed' ELSE g.status END,
		     completed_at = CASE WHEN $4 >= g.target_mastery AND g.completed_at IS NULL THEN NOW() ELSE g.completed_at END
		 FROM users u
		 WHERE u.id = g.user_id
		   AND g.tenant_id = $1::uuid
		   AND u.channel = $2
		   AND u.external_id = $3
		   AND g.syllabus_id = $5
		   AND g.topic_id = $6
		   AND g.status = 'active'
		 RETURNING g.id::text, u.external_id, g.summary, g.topic_id, g.topic_name, g.syllabus_id, g.target_mastery, g.current_mastery, g.status, g.created_at, g.updated_at, g.completed_at`,
		s.tenantID,
		s.channel,
		externalID,
		clampGoalMastery(mastery),
		syllabusID,
		topicID,
	)
	if err != nil {
		return nil, fmt.Errorf("sync goal progress: %w", err)
	}
	defer rows.Close()

	var goals []*Goal
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan synced goal: %w", err)
		}
		goals = append(goals, goal)
	}
	return goals, rows.Err()
}

type goalParseResult struct {
	GoalSummary       string  `json:"goal_summary"`
	TargetMastery     float64 `json:"target_mastery"`
	NeedsConfirmation bool    `json:"needs_confirmation"`
}

func (e *Engine) handleGoalCommand(ctx context.Context, msg chat.InboundMessage, args []string) (string, error) {
	if e.goals == nil {
		return "Goal tracking is not enabled.", nil
	}

	conv, err := e.getOrCreateConversation(msg.UserID)
	if err != nil {
		slog.Error("failed to init user for /goal", "user_id", msg.UserID, "error", err)
		return "I hit a technical issue while setting up your goal.", nil
	}

	if len(args) == 0 {
		return e.describeGoals(conv)
	}

	raw := strings.TrimSpace(strings.Join(args, " "))
	switch strings.ToLower(raw) {
	case "clear":
		if err := e.goals.ClearActiveGoals(msg.UserID); err != nil {
			slog.Error("failed to clear goals", "user_id", msg.UserID, "error", err)
			return "I hit a technical issue while clearing your goals.", nil
		}
		if err := e.store.ClearConversationPendingGoal(conv.ID); err != nil {
			slog.Warn("failed to clear pending goal draft", "conversation_id", conv.ID, "error", err)
		}
		return "All active goals cleared.", nil
	case "cancel":
		if err := e.store.ClearConversationPendingGoal(conv.ID); err != nil {
			slog.Warn("failed to clear pending goal draft", "conversation_id", conv.ID, "error", err)
		}
		return "Okay, I dropped that goal suggestion.", nil
	}

	return e.applyGoalText(ctx, msg, conv, raw)
}

func (e *Engine) maybeHandlePendingGoal(ctx context.Context, msg chat.InboundMessage, conv *Conversation) (string, bool) {
	if conv == nil || conv.PendingGoal == nil {
		return "", false
	}

	trimmed := strings.TrimSpace(msg.Text)
	if trimmed == "" {
		return "", false
	}

	normalized := normalizeGoalReply(trimmed)
	if isGoalConfirmation(normalized) {
		resp, err := e.createGoal(msg.UserID, conv, *conv.PendingGoal)
		if err != nil {
			slog.Error("failed to confirm pending goal", "user_id", msg.UserID, "error", err)
			return "I hit a technical issue while saving your goal.", true
		}
		return resp, true
	}
	if isGoalCancel(normalized) {
		if err := e.store.ClearConversationPendingGoal(conv.ID); err != nil {
			slog.Warn("failed to clear pending goal draft", "conversation_id", conv.ID, "error", err)
		}
		return "Okay, I dropped that goal suggestion.", true
	}

	resp, err := e.applyGoalText(ctx, msg, conv, trimmed)
	if err != nil {
		slog.Error("failed to reparse pending goal", "user_id", msg.UserID, "error", err)
		return "I hit a technical issue while updating your goal.", true
	}
	return resp, true
}

func (e *Engine) applyGoalText(ctx context.Context, msg chat.InboundMessage, conv *Conversation, raw string) (string, error) {
	topic, _ := e.contextResolver.Resolve(raw)
	if topic == nil {
		_ = e.store.ClearConversationPendingGoal(conv.ID)
		return unresolvedGoalMessage(), nil
	}

	parsed := e.parseGoalRequest(ctx, raw, topic)
	draft := PendingGoalDraft{
		Summary:       parsed.goalSummary(topic),
		TopicID:       topic.ID,
		TopicName:     topic.Name,
		SyllabusID:    parsed.syllabusID(topic),
		TargetMastery: parsed.targetMastery(),
	}

	if parsed.NeedsConfirmation {
		if err := e.store.SetConversationPendingGoal(conv.ID, draft); err != nil {
			return "", err
		}
		return formatPendingGoalSuggestion(draft), nil
	}

	return e.createGoal(msg.UserID, conv, draft)
}

func (e *Engine) createGoal(userID string, conv *Conversation, draft PendingGoalDraft) (string, error) {
	currentMastery := 0.0
	if e.tracker != nil {
		score, err := e.tracker.GetMastery(userID, draft.SyllabusID, draft.TopicID)
		if err == nil {
			currentMastery = score
		}
	}

	goal, err := e.goals.AddGoal(userID, GoalInput{
		Summary:        draft.Summary,
		TopicID:        draft.TopicID,
		TopicName:      draft.TopicName,
		SyllabusID:     draft.SyllabusID,
		TargetMastery:  draft.TargetMastery,
		CurrentMastery: currentMastery,
	})
	if err != nil {
		return "", err
	}
	if err := e.store.ClearConversationPendingGoal(conv.ID); err != nil {
		slog.Warn("failed to clear pending goal draft", "conversation_id", conv.ID, "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         userID,
		EventType:      "goal_created",
		Data: map[string]any{
			"goal_id":         goal.ID,
			"topic_id":        goal.TopicID,
			"target_mastery":  goal.TargetMastery,
			"current_mastery": goal.CurrentMastery,
			"status":          goal.Status,
		},
	})
	return formatGoalSetMessage(goal), nil
}

func (e *Engine) describeGoals(conv *Conversation) (string, error) {
	if conv == nil {
		return goalEmptyStateMessage(), nil
	}

	goals, err := e.goals.ListActiveGoals(conv.UserID)
	if err != nil {
		slog.Error("failed to list active goals", "user_id", conv.UserID, "error", err)
		return "I hit a technical issue while loading your goals.", nil
	}

	var sections []string
	if conv.PendingGoal != nil {
		sections = append(sections, formatPendingGoalSuggestion(*conv.PendingGoal))
	}
	if len(goals) == 0 {
		if len(sections) > 0 {
			sections = append(sections, goalEmptyStateMessage())
			return strings.Join(sections, "\n\n"), nil
		}
		return goalEmptyStateMessage(), nil
	}

	sections = append(sections, formatGoalList(goals, 0, "🎯 Active Goals"))
	return strings.Join(sections, "\n\n"), nil
}

func (e *Engine) parseGoalRequest(ctx context.Context, raw string, topic *curriculum.Topic) goalParseResult {
	fallback := fallbackGoalParse(raw, topic)
	if e.aiRouter == nil {
		return fallback
	}

	var out goalParseResult
	_, err := e.aiRouter.CompleteJSON(ctx, ai.CompletionRequest{
		Task: ai.TaskAnalysis,
		Messages: []ai.Message{
			{Role: "system", Content: "Turn a student's study-goal request into a topic mastery goal. Return JSON only. Use target_mastery as a decimal between 0.55 and 1.0. If the topic is precise enough to create immediately, set needs_confirmation to false. If the request is broad or vague, set needs_confirmation to true so the bot suggests one concrete goal first. Keep goal_summary short and student-facing."},
			{Role: "user", Content: fmt.Sprintf("Resolved topic: %s\nTopic ID: %s\nStudent request: %s", topic.Name, topic.ID, raw)},
		},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name: "goal_parse",
			JSONSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"goal_summary":{"type":"string"},
					"target_mastery":{"type":"number","minimum":0.55,"maximum":1.0},
					"needs_confirmation":{"type":"boolean"}
				},
				"required":["goal_summary","target_mastery","needs_confirmation"],
				"additionalProperties":false
			}`),
			Strict: true,
		},
		MaxTokens: 120,
	}, &out)
	if err != nil {
		return fallback
	}
	if strings.TrimSpace(out.GoalSummary) == "" {
		out.GoalSummary = fallback.GoalSummary
	}
	out.TargetMastery = normalizeGoalTarget(out.TargetMastery)
	out.NeedsConfirmation = out.NeedsConfirmation || fallback.NeedsConfirmation
	return out
}

func (e *Engine) appendGoalToProgressReport(userID, report string) string {
	if e.goals == nil {
		return report
	}
	goals, err := e.goals.ListActiveGoals(userID)
	if err != nil || len(goals) == 0 {
		return report
	}
	return strings.TrimSpace(report) + "\n\n" + formatGoalList(goals, 5, "🎯 Active Goals")
}

func (e *Engine) syncGoalProgress(userID, syllabusID, topicID string) {
	if e.goals == nil || e.tracker == nil {
		return
	}
	mastery, err := e.tracker.GetMastery(userID, syllabusID, topicID)
	if err != nil {
		slog.Warn("failed to read mastery for goal sync", "user_id", userID, "topic_id", topicID, "error", err)
		return
	}
	goals, err := e.goals.SyncGoalProgress(userID, syllabusID, topicID, mastery)
	if err != nil {
		slog.Warn("failed to sync goal progress", "user_id", userID, "topic_id", topicID, "error", err)
		return
	}
	for _, goal := range goals {
		if goal != nil && goal.Status == "completed" {
			e.logEventAsync(Event{
				UserID:    userID,
				EventType: "goal_completed",
				Data: map[string]any{
					"goal_id":         goal.ID,
					"topic_id":        goal.TopicID,
					"target_mastery":  goal.TargetMastery,
					"current_mastery": goal.CurrentMastery,
				},
			})
		}
	}
}

func activeGoalsDescending(goals []*Goal) []*Goal {
	var active []*Goal
	for _, goal := range goals {
		if goal != nil && goal.Status == "active" {
			active = append(active, goal)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].CreatedAt.After(active[j].CreatedAt)
	})
	return active
}

func cloneGoal(goal *Goal) *Goal {
	if goal == nil {
		return nil
	}
	cp := *goal
	if goal.CompletedAt != nil {
		t := *goal.CompletedAt
		cp.CompletedAt = &t
	}
	return &cp
}

func cloneGoalSlice(goals []*Goal) []*Goal {
	cloned := make([]*Goal, 0, len(goals))
	for _, goal := range goals {
		cloned = append(cloned, cloneGoal(goal))
	}
	return cloned
}

type goalRowScanner interface {
	Scan(...any) error
}

func newGoalRecord(userID string, input GoalInput, now time.Time) *Goal {
	goal := &Goal{
		UserID:         userID,
		Summary:        strings.TrimSpace(input.Summary),
		TopicID:        strings.TrimSpace(input.TopicID),
		TopicName:      strings.TrimSpace(input.TopicName),
		SyllabusID:     strings.TrimSpace(input.SyllabusID),
		TargetMastery:  normalizeGoalTarget(input.TargetMastery),
		CurrentMastery: clampGoalMastery(input.CurrentMastery),
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	markGoalCompletedIfReached(goal, now)
	return goal
}

func markGoalCompletedIfReached(goal *Goal, now time.Time) {
	if goal == nil || goal.CurrentMastery < goal.TargetMastery {
		return
	}
	goal.Status = "completed"
	goal.CompletedAt = &now
}

func scanGoal(scanner goalRowScanner) (*Goal, error) {
	goal := &Goal{}
	var completedAt *time.Time
	if err := scanner.Scan(
		&goal.ID,
		&goal.UserID,
		&goal.Summary,
		&goal.TopicID,
		&goal.TopicName,
		&goal.SyllabusID,
		&goal.TargetMastery,
		&goal.CurrentMastery,
		&goal.Status,
		&goal.CreatedAt,
		&goal.UpdatedAt,
		&completedAt,
	); err != nil {
		return nil, err
	}
	goal.CompletedAt = completedAt
	return goal, nil
}

func fallbackGoalParse(raw string, topic *curriculum.Topic) goalParseResult {
	target := defaultGoalTargetMastery
	if match := goalPercentPattern.FindStringSubmatch(raw); len(match) == 2 {
		target = normalizeGoalTarget(parseGoalPercent(match[1]))
	}
	explicitTopic := topicExplicitlyMentioned(raw, topic)
	return goalParseResult{
		GoalSummary:       fmt.Sprintf("Reach %d%% mastery in %s", goalPercentRounded(target), goalTopicName(topic)),
		TargetMastery:     target,
		NeedsConfirmation: !explicitTopic,
	}
}

func normalizeGoalTarget(v float64) float64 {
	if v > 1 {
		v = v / 100
	}
	if v <= 0 {
		v = defaultGoalTargetMastery
	}
	if v < 0.55 {
		v = 0.55
	}
	if v > 1 {
		v = 1
	}
	return v
}

func clampGoalMastery(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func parseGoalPercent(raw string) float64 {
	var pct float64
	_, _ = fmt.Sscanf(strings.TrimSpace(raw), "%f", &pct)
	return pct
}

func goalPercentRounded(v float64) int {
	if v > 1 {
		v = v / 100
	}
	return int(math.Round(clampGoalMastery(v) * 100))
}

func goalTopicName(topic *curriculum.Topic) string {
	if topic == nil || strings.TrimSpace(topic.Name) == "" {
		return "this topic"
	}
	return topic.Name
}

func topicExplicitlyMentioned(raw string, topic *curriculum.Topic) bool {
	if topic == nil {
		return false
	}
	text := normalizedWordSet(raw)
	if _, ok := text[strings.ToLower(strings.TrimSpace(topic.ID))]; ok {
		return true
	}
	words := keywordWords(topic.Name)
	if len(words) == 0 {
		return false
	}
	matched := 0
	for _, word := range words {
		if _, ok := text[word]; ok {
			matched++
		}
	}
	if len(words) == 1 {
		return matched == 1
	}
	return matched >= 2
}

func normalizedWordSet(text string) map[string]struct{} {
	text = strings.ToLower(text)
	replacer := strings.NewReplacer(",", " ", ".", " ", "!", " ", "?", " ", "-", " ", "_", " ", "/", " ", "(", " ", ")", " ", ":", " ")
	text = replacer.Replace(text)
	words := strings.Fields(text)
	set := make(map[string]struct{}, len(words))
	for _, word := range words {
		set[word] = struct{}{}
	}
	return set
}

func keywordWords(text string) []string {
	stop := map[string]struct{}{
		"the": {}, "and": {}, "of": {}, "in": {}, "a": {}, "an": {},
	}
	var words []string
	for word := range normalizedWordSet(text) {
		if len(word) <= 2 {
			continue
		}
		if _, skip := stop[word]; skip {
			continue
		}
		words = append(words, word)
	}
	sort.Strings(words)
	return words
}

func normalizeGoalReply(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.Trim(text, "!.? ")
	return text
}

func isGoalConfirmation(text string) bool {
	_, ok := goalConfirmationReplies[text]
	return ok
}

func isGoalCancel(text string) bool {
	_, ok := goalCancelReplies[text]
	return ok
}

func (r goalParseResult) targetMastery() float64 {
	return normalizeGoalTarget(r.TargetMastery)
}

func (r goalParseResult) goalSummary(topic *curriculum.Topic) string {
	if strings.TrimSpace(r.GoalSummary) != "" {
		return strings.TrimSpace(r.GoalSummary)
	}
	return fmt.Sprintf("Reach %d%% mastery in %s", goalPercentRounded(r.TargetMastery), goalTopicName(topic))
}

func (r goalParseResult) syllabusID(topic *curriculum.Topic) string {
	if topic != nil && strings.TrimSpace(topic.SyllabusID) != "" {
		return topic.SyllabusID
	}
	return "default"
}

func unresolvedGoalMessage() string {
	return "I couldn't map that to a topic yet.\n\nTry something like:\n- /goal help me master linear equations\n- /goal I want to reach 80% in algebra\n- /goal help me get better at fractions"
}

func goalEmptyStateMessage() string {
	return "You don't have any active goals yet.\n\nTry something like:\n- /goal help me master linear equations\n- /goal I want to reach 80% in algebra\n- /goal help me get better at fractions\n\nTell me what you want to improve, and I'll turn it into a concrete study goal."
}

func formatPendingGoalSuggestion(goal PendingGoalDraft) string {
	return fmt.Sprintf(
		"I can turn that into this goal:\n\n%s\nTopic: %s\nTarget: %d%%\n\nReply yes to save it, or rewrite the goal.",
		goal.Summary,
		goal.TopicName,
		goalPercentRounded(goal.TargetMastery),
	)
}

func formatGoalSetMessage(goal *Goal) string {
	if goal == nil {
		return "Goal saved."
	}
	prefix := "Goal saved."
	if goal.Status == "completed" {
		prefix = "Goal saved. You already hit it."
	}
	return prefix + "\n\n" + formatSingleGoal(goal)
}

func formatSingleGoal(goal *Goal) string {
	return fmt.Sprintf(
		"🎯 Goal\n%s\nTopic: %s\nProgress: %d%% / %d%%",
		goal.Summary,
		goal.TopicName,
		goalPercentRounded(goal.CurrentMastery),
		goalPercentRounded(goal.TargetMastery),
	)
}

func formatGoalList(goals []*Goal, limit int, title string) string {
	if len(goals) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(goals) {
		limit = len(goals)
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	for i, goal := range goals[:limit] {
		fmt.Fprintf(&b, "%d. %s (%d%% / %d%%)\n", i+1, goal.Summary, goalPercentRounded(goal.CurrentMastery), goalPercentRounded(goal.TargetMastery))
	}
	if len(goals) > limit {
		fmt.Fprintf(&b, "+%d more", len(goals)-limit)
	}
	return strings.TrimSpace(b.String())
}

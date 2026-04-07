package agent

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/i18n"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

// Quiet hours in Malaysia (MYT = UTC+8).
const (
	QuietHoursStart = 21 // 9 PM MYT
	QuietHoursEnd   = 7  // 7 AM MYT
	MaxNudgesPerDay = 3
)

// SchedulerConfig holds scheduler settings.
type SchedulerConfig struct {
	CheckInterval               time.Duration
	MaxNudgesPerDay             int
	AIPersonalizedNudgesEnabled bool
}

// DefaultSchedulerConfig returns production defaults.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		CheckInterval:               5 * time.Minute,
		MaxNudgesPerDay:             MaxNudgesPerDay,
		AIPersonalizedNudgesEnabled: true,
	}
}

type nudgeLanguageStore interface {
	GetUserPreferredLanguage(userID string) (string, bool)
	GetUserABGroup(userID string) (string, bool)
}

var nudgeSentenceBreakRE = regexp.MustCompile(`([.!?。！？])\s+`)

// IsQuietHours returns true if the given time falls within quiet hours (21:00-07:00 MYT).
func IsQuietHours(t time.Time) bool {
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		loc = time.FixedZone("MYT", 8*60*60)
	}
	hour := t.In(loc).Hour()
	return hour >= QuietHoursStart || hour < QuietHoursEnd
}

// CanNudge returns true if a nudge can be sent at the given time with the given daily count.
func CanNudge(t time.Time, nudgesSentToday int) bool {
	if IsQuietHours(t) {
		return false
	}
	return nudgesSentToday < MaxNudgesPerDay
}

// NudgeTracker tracks how many nudges have been sent per user per day.
type NudgeTracker interface {
	// NudgeCountToday returns how many nudges were sent to the user today.
	NudgeCountToday(userID string) (int, error)
	// RecordNudge records that a nudge was sent to the user.
	RecordNudge(userID, nudgeType, topicID string) error
}

// MemoryNudgeTracker is an in-memory implementation for testing.
type MemoryNudgeTracker struct {
	nudges map[string][]time.Time // userID → list of nudge times
}

// NewMemoryNudgeTracker creates a new in-memory nudge tracker.
func NewMemoryNudgeTracker() *MemoryNudgeTracker {
	return &MemoryNudgeTracker{
		nudges: make(map[string][]time.Time),
	}
}

func (t *MemoryNudgeTracker) NudgeCountToday(userID string) (int, error) {
	times, ok := t.nudges[userID]
	if !ok {
		return 0, nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	count := 0
	for _, ts := range times {
		if ts.UTC().Truncate(24*time.Hour).Equal(today) {
			count++
		}
	}
	return count, nil
}

func (t *MemoryNudgeTracker) RecordNudge(userID, nudgeType, topicID string) error {
	t.nudges[userID] = append(t.nudges[userID], time.Now())
	return nil
}

// Scheduler runs periodic checks for due reviews and sends nudges.
type Scheduler struct {
	config   SchedulerConfig
	tracker  progress.Tracker
	streaks  progress.StreakTracker
	xp       progress.XPTracker
	goals    GoalStore
	nudges   NudgeTracker
	groups   GroupStore
	tenantID string
	gateway  *chat.Gateway
	aiRouter *ai.Router
	store    nudgeLanguageStore
	logger   *slog.Logger
}

// NewScheduler creates a new proactive scheduler.
func NewScheduler(
	cfg SchedulerConfig,
	tracker progress.Tracker,
	streaks progress.StreakTracker,
	xp progress.XPTracker,
	goals GoalStore,
	nudges NudgeTracker,
	gateway *chat.Gateway,
	aiRouter *ai.Router,
	store nudgeLanguageStore,
) *Scheduler {
	return &Scheduler{
		config:   cfg,
		tracker:  tracker,
		streaks:  streaks,
		xp:       xp,
		goals:    goals,
		nudges:   nudges,
		gateway:  gateway,
		aiRouter: aiRouter,
		store:    store,
		logger:   slog.Default(),
	}
}

// SetGroupStore enables the weekly leaderboard recap for the scheduler.
func (s *Scheduler) SetGroupStore(groups GroupStore, tenantID string) {
	s.groups = groups
	s.tenantID = tenantID
}

// Start begins the scheduler loop. Blocks until context is cancelled.
func (s *Scheduler) Start(ctx context.Context, userIDs []string) {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	// Start daily summary on a precise timer (22:00 MYT), not a polling tick.
	go s.runDailySummaryTimer(ctx, userIDs)

	// Start weekly leaderboard recap on Monday 8:00 AM MYT.
	if s.groups != nil {
		go s.runWeeklyLeaderboardTimer(ctx)
	}

	s.logger.Info("scheduler started", "interval", s.config.CheckInterval)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.checkAndNudge(ctx, userIDs)
		}
	}
}

// runDailySummaryTimer fires at exactly 22:00 MYT each day.
func (s *Scheduler) runDailySummaryTimer(ctx context.Context, userIDs []string) {
	for {
		delay := timeUntilNext(dailySummaryHour, 0)
		s.logger.Info("daily summary scheduled", "fires_in", delay.Round(time.Second))

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case now := <-timer.C:
			s.SendDailySummaries(ctx, userIDs, now)
		}
	}
}

// timeUntilNext returns the duration until the next occurrence of hour:minute in MYT.
func timeUntilNext(hour, minute int) time.Duration {
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		loc = time.FixedZone("MYT", 8*60*60)
	}
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (s *Scheduler) checkAndNudge(ctx context.Context, userIDs []string) {
	now := time.Now()

	if IsQuietHours(now) {
		return
	}

	for _, userID := range userIDs {
		if err := s.checkUser(ctx, userID, now); err != nil {
			s.logger.Error("scheduler check failed",
				"user_id", userID, "error", err)
		}
	}
}

// SendDailySummaries sends a daily progress summary to each user with activity.
func (s *Scheduler) SendDailySummaries(ctx context.Context, userIDs []string, now time.Time) {
	for _, userID := range userIDs {
		summary := ComputeDailySummary(userID, s.tracker, s.streaks, s.xp)
		locale := s.userLocale(userID)
		msg := FormatDailySummary(summary, locale)
		if msg == "" {
			continue
		}
		out := chat.OutboundMessage{
			Channel:   "telegram",
			UserID:    userID,
			Text:      msg,
			ParseMode: "Markdown",
		}
		if err := s.gateway.Send(ctx, out); err != nil {
			s.logger.Error("failed to send daily summary", "user_id", userID, "error", err)
			continue
		}
		s.logger.Info("daily summary sent", "user_id", userID)
	}
}

const weeklyLeaderboardHour = 8 // 8:00 AM MYT

// runWeeklyLeaderboardTimer fires every Monday at 8:00 AM MYT.
func (s *Scheduler) runWeeklyLeaderboardTimer(ctx context.Context) {
	for {
		delay := timeUntilNextWeekday(time.Monday, weeklyLeaderboardHour, 0)
		s.logger.Info("weekly leaderboard scheduled", "fires_in", delay.Round(time.Second))

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.sendWeeklyLeaderboards(ctx)
		}
	}
}

// timeUntilNextWeekday returns the duration until the next given weekday at hour:minute MYT.
func timeUntilNextWeekday(day time.Weekday, hour, minute int) time.Duration {
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		loc = time.FixedZone("MYT", 8*60*60)
	}
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)

	// Advance to the target weekday.
	daysUntil := (int(day) - int(now.Weekday()) + 7) % 7
	if daysUntil == 0 && !next.After(now) {
		daysUntil = 7
	}
	next = next.AddDate(0, 0, daysUntil)
	return next.Sub(now)
}

func (s *Scheduler) sendWeeklyLeaderboards(ctx context.Context) {
	if s.groups == nil || s.tenantID == "" {
		return
	}

	allGroups, err := s.groups.ListGroups(s.tenantID, "")
	if err != nil {
		s.logger.Error("failed to list groups for leaderboard", "error", err)
		return
	}

	for _, g := range allGroups {
		if g.MemberCount < 2 {
			continue
		}

		entries, err := s.groups.GetWeeklyLeaderboard(g.ID, 10)
		if err != nil {
			s.logger.Error("failed to get leaderboard", "group_id", g.ID, "error", err)
			continue
		}
		if len(entries) == 0 {
			continue
		}

		msg := formatLeaderboard(g.Name, entries, i18n.DefaultLocale)

		recipients, err := s.groups.GetGroupMembersWithChannel(g.ID)
		if err != nil {
			s.logger.Error("failed to get group members", "group_id", g.ID, "error", err)
			continue
		}

		for _, r := range recipients {
			out := chat.OutboundMessage{
				Channel:   r.Channel,
				UserID:    r.ExternalID,
				Text:      msg,
				ParseMode: "Markdown",
			}
			if err := s.gateway.Send(ctx, out); err != nil {
				s.logger.Error("failed to send leaderboard", "user", r.ExternalID, "group", g.Name, "error", err)
			}
		}
		s.logger.Info("weekly leaderboard sent", "group", g.Name, "recipients", len(recipients))
	}
}

func (s *Scheduler) checkUser(ctx context.Context, userID string, now time.Time) error {
	count, err := s.nudges.NudgeCountToday(userID)
	if err != nil {
		return fmt.Errorf("get nudge count: %w", err)
	}
	if !CanNudge(now, count) {
		return nil
	}

	// Skip nudges for AB group B.
	if s.store != nil {
		if group, ok := s.store.GetUserABGroup(userID); ok && group == ABGroupB {
			return nil
		}
	}

	// Check for due reviews.
	dueItems, err := s.tracker.GetDueReviews(userID)
	if err != nil {
		return fmt.Errorf("get due reviews: %w", err)
	}
	if len(dueItems) == 0 {
		return nil
	}

	// Pick the most overdue topic.
	item := dueItems[0]
	for _, di := range dueItems[1:] {
		if di.NextReviewAt.Before(item.NextReviewAt) {
			item = di
		}
	}

	// Build nudge message.
	msg := s.buildNudgeMessage(ctx, userID, item, now)

	// Send via chat gateway (default to telegram channel).
	out := chat.OutboundMessage{
		Channel: "telegram",
		UserID:  userID,
		Text:    msg,
	}
	if err := s.gateway.Send(ctx, out); err != nil {
		return fmt.Errorf("send nudge: %w", err)
	}

	// Record the nudge.
	if err := s.nudges.RecordNudge(userID, "review_due", item.TopicID); err != nil {
		s.logger.Error("failed to record nudge", "user_id", userID, "error", err)
	}

	s.logger.Info("nudge sent",
		"user_id", userID,
		"topic_id", item.TopicID,
		"due_since", now.Sub(item.NextReviewAt).Round(time.Minute),
	)

	return nil
}

// CheckUserForNudge triggers a single due-review nudge check for the user at the given time.
func (s *Scheduler) CheckUserForNudge(ctx context.Context, userID string, now time.Time) error {
	return s.checkUser(ctx, userID, now)
}

func (s *Scheduler) buildNudgeMessage(ctx context.Context, userID string, item progress.ProgressItem, now time.Time) string {
	locale := s.userLocale(userID)
	if s.config.AIPersonalizedNudgesEnabled && s.aiRouter != nil && s.aiRouter.HasProvider() {
		if msg, ok := s.generateAINudge(ctx, userID, item, now, locale); ok {
			return msg
		}
	}
	return buildDefaultNudgeMessage(item, now, locale)
}

func (s *Scheduler) generateAINudge(ctx context.Context, userID string, item progress.ProgressItem, now time.Time, locale string) (string, bool) {
	streakDays := 0
	if s.streaks != nil {
		streak, err := s.streaks.GetStreak(userID)
		if err == nil {
			streakDays = streak.CurrentStreak
		}
	}

	totalXP := 0
	if s.xp != nil {
		xp, err := s.xp.GetTotal(userID)
		if err == nil {
			totalXP = xp
		}
	}

	activeGoal := ""
	if s.goals != nil {
		goals, err := s.goals.ListActiveGoals(userID)
		if err == nil && len(goals) > 0 && goals[0] != nil {
			activeGoal = goals[0].Summary
		}
	}

	struggleArea := ""
	if s.tracker != nil {
		allProgress, err := s.tracker.GetAllProgress(userID)
		if err == nil {
			struggleArea = weakestTopicID(allProgress)
		}
	}

	overdueHours := int(now.Sub(item.NextReviewAt).Round(time.Hour).Hours())
	if overdueHours < 0 {
		overdueHours = 0
	}

	resp, err := s.aiRouter.Complete(ctx, ai.CompletionRequest{
		Task:        ai.TaskNudge,
		MaxTokens:   60,
		Temperature: 0.7,
		Messages: []ai.Message{
			{
				Role: "system",
				Content: "You write personalized proactive study nudges for a Malaysian math learning bot. " +
					"Write naturally in the student's preferred language and use the student context to decide what to emphasize. " +
					"Keep it short: 1 to 3 short sentences, ideally under 45 words. " +
					"Use line breaks when there are multiple sentences so the message is easy to scan. " +
					"Include a clear invitation to continue learning now. " +
					"Return plain text only.",
			},
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Write one personalized nudge message for this student.\nPreferred language: %s\nTopic ID: %s\nMastery score: %d%%\nHours overdue: %d\nCurrent streak: %d days\nTotal XP: %d\nActive goal: %s\nStruggle area: %s\nAudience: secondary school student in Malaysia learning math via chat.",
					locale,
					item.TopicID,
					int(item.MasteryScore*100),
					overdueHours,
					streakDays,
					totalXP,
					emptyIfBlank(activeGoal, "none"),
					emptyIfBlank(struggleArea, "none"),
				),
			},
		},
	})
	if err != nil {
		s.logger.Warn("ai nudge generation failed", "user_id", userID, "error", err)
		return "", false
	}

	msg := strings.TrimSpace(resp.Content)
	if msg == "" {
		return "", false
	}
	return formatAINudgeMessage(msg), true
}

func formatAINudgeMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}

	if strings.Contains(msg, "\n") {
		lines := strings.Split(msg, "\n")
		for i, line := range lines {
			lines[i] = strings.Join(strings.Fields(line), " ")
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}

	msg = strings.Join(strings.Fields(msg), " ")
	return strings.TrimSpace(nudgeSentenceBreakRE.ReplaceAllString(msg, "$1\n"))
}

func (s *Scheduler) userLocale(userID string) string {
	if s.store != nil {
		if lang, ok := s.store.GetUserPreferredLanguage(userID); ok && lang != "" {
			return lang
		}
	}
	return i18n.DefaultLocale
}

func buildDefaultNudgeMessage(item progress.ProgressItem, now time.Time, locale string) string {
	overdue := now.Sub(item.NextReviewAt)

	var urgency string
	var topicLabel string
	var masteryLabel string
	var cta string

	switch locale {
	case "en":
		topicLabel = "Topic"
		masteryLabel = "Mastery"
		cta = "Reply with anything to start learning now."
		switch {
		case overdue > 7*24*time.Hour:
			urgency = "It's been a while since you reviewed this topic."
		case overdue > 3*24*time.Hour:
			urgency = "Time for a quick review."
		default:
			urgency = "Let's do a quick revision."
		}
	case "zh":
		topicLabel = "主题"
		masteryLabel = "掌握度"
		cta = "直接回复任何内容，就可以马上开始学习。"
		switch {
		case overdue > 7*24*time.Hour:
			urgency = "这个主题已经很久没复习了。"
		case overdue > 3*24*time.Hour:
			urgency = "现在很适合做个小复习。"
		default:
			urgency = "我们来快速复习一下吧。"
		}
	default:
		topicLabel = "Topik"
		masteryLabel = "Penguasaan"
		cta = "Taip apa sahaja untuk mula belajar!"
		switch {
		case overdue > 7*24*time.Hour:
			urgency = "Dah lama tak ulangkaji topik ini!"
		case overdue > 3*24*time.Hour:
			urgency = "Masa untuk ulangkaji!"
		default:
			urgency = "Jom ulangkaji!"
		}
	}

	pct := int(item.MasteryScore * 100)
	return fmt.Sprintf(
		"%s\n\n%s: %s\n%s: %d%%\n\n%s",
		urgency, topicLabel, item.TopicID, masteryLabel, pct, cta,
	)
}

func weakestTopicID(items []progress.ProgressItem) string {
	if len(items) == 0 {
		return ""
	}

	weakest := items[0]
	for _, item := range items[1:] {
		if item.MasteryScore < weakest.MasteryScore {
			weakest = item
		}
	}
	return weakest.TopicID
}

func emptyIfBlank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

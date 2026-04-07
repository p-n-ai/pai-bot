package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

const (
	weeklyParentReportHour    = 20
	weeklyParentReportWeekday = time.Sunday
)

type WeeklyParentWeeklyStats struct {
	DaysActive        int
	MessagesExchanged int
	QuizzesCompleted  int
	NeedsReviewCount  int
}

type WeeklyParentReportSummary struct {
	ParentExternalID   string
	ParentChannel      string
	ParentName         string
	ChildName          string
	ChildForm          string
	CurrentStreak      int
	TotalXP            int
	NeedsReviewCount   int
	WeakestTopicID     string
	EncouragementTitle string
	EncouragementText  string
	WeeklyStats        WeeklyParentWeeklyStats
}

type WeeklyParentReportSource interface {
	ListWeeklyParentReportSummaries(ctx context.Context) ([]WeeklyParentReportSummary, error)
}

func (s *Scheduler) SetWeeklyParentReportSource(source WeeklyParentReportSource) {
	s.parentReports = source
}

func (s *Scheduler) runWeeklyParentReportTimer(ctx context.Context) {
	for {
		delay := timeUntilNextWeekday(weeklyParentReportWeekday, weeklyParentReportHour, 0)
		s.logger.Info("weekly parent reports scheduled", "fires_in", delay.Round(time.Second))

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case now := <-timer.C:
			s.SendWeeklyParentReports(ctx, now)
		}
	}
}

func timeUntilNextWeekday(weekday time.Weekday, hour, minute int) time.Duration {
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		loc = time.FixedZone("MYT", 8*60*60)
	}
	now := time.Now().In(loc)
	daysAhead := (int(weekday) - int(now.Weekday()) + 7) % 7
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc).AddDate(0, 0, daysAhead)
	if !next.After(now) {
		next = next.AddDate(0, 0, 7)
	}
	return next.Sub(now)
}

func (s *Scheduler) SendWeeklyParentReports(ctx context.Context, now time.Time) {
	if s.parentReports == nil {
		return
	}

	summaries, err := s.parentReports.ListWeeklyParentReportSummaries(ctx)
	if err != nil {
		s.logger.Error("failed to load weekly parent reports", "error", err)
		return
	}

	for _, summary := range summaries {
		if strings.TrimSpace(summary.ParentExternalID) == "" || summary.ParentChannel != "telegram" {
			continue
		}

		msg := s.buildWeeklyParentReport(ctx, summary, now)
		if strings.TrimSpace(msg) == "" {
			continue
		}

		out := chat.OutboundMessage{
			Channel: "telegram",
			UserID:  summary.ParentExternalID,
			Text:    msg,
		}
		if err := s.gateway.Send(ctx, out); err != nil {
			s.logger.Error("failed to send weekly parent report", "parent_id", summary.ParentExternalID, "error", err)
			continue
		}
		s.logger.Info("weekly parent report sent", "parent_id", summary.ParentExternalID, "child_name", summary.ChildName)
	}
}

func (s *Scheduler) buildWeeklyParentReport(ctx context.Context, summary WeeklyParentReportSummary, now time.Time) string {
	if s.aiRouter != nil && s.aiRouter.HasProvider() {
		if msg, ok := s.generateAIWeeklyParentReport(ctx, summary, now); ok {
			return msg
		}
	}
	return buildFallbackWeeklyParentReport(summary, now)
}

func (s *Scheduler) generateAIWeeklyParentReport(ctx context.Context, summary WeeklyParentReportSummary, now time.Time) (string, bool) {
	resp, err := s.aiRouter.Complete(ctx, ai.CompletionRequest{
		Task:        ai.TaskAnalysis,
		MaxTokens:   220,
		Temperature: 0.5,
		Messages: []ai.Message{
			{
				Role: "system",
				Content: "You write weekly parent progress reports for a Malaysian math tutoring bot. " +
					"Write exactly 3 short paragraphs in plain text. No bullets, no markdown headings, no emojis. " +
					"Paragraph 1: weekly momentum and study activity. " +
					"Paragraph 2: mastery and what needs attention. " +
					"Paragraph 3: one specific, supportive action the parent can take this week. " +
					"Keep the tone warm, concrete, and concise.",
			},
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Parent: %s\nChild: %s\nForm: %s\nReport week ending: %s\nDays active: %d\nMessages exchanged: %d\nQuizzes completed: %d\nNeeds review count: %d\nCurrent streak: %d days\nTotal XP: %d\nWeakest topic: %s\nSuggested encouragement headline: %s\nSuggested encouragement text: %s",
					emptyIfBlank(summary.ParentName, "Parent"),
					summary.ChildName,
					emptyIfBlank(summary.ChildForm, "Unknown form"),
					now.UTC().Format("2006-01-02"),
					summary.WeeklyStats.DaysActive,
					summary.WeeklyStats.MessagesExchanged,
					summary.WeeklyStats.QuizzesCompleted,
					summary.WeeklyStats.NeedsReviewCount,
					summary.CurrentStreak,
					summary.TotalXP,
					emptyIfBlank(humanizeReportTopicID(summary.WeakestTopicID), "No single weak topic identified"),
					summary.EncouragementTitle,
					summary.EncouragementText,
				),
			},
		},
	})
	if err != nil {
		s.logger.Warn("ai weekly parent report generation failed", "parent_id", summary.ParentExternalID, "error", err)
		return "", false
	}

	msg := strings.TrimSpace(resp.Content)
	if msg == "" {
		return "", false
	}
	return msg, true
}

func buildFallbackWeeklyParentReport(summary WeeklyParentReportSummary, now time.Time) string {
	topic := humanizeReportTopicID(summary.WeakestTopicID)
	if topic == "" {
		topic = "current review topics"
	}

	paragraph1 := fmt.Sprintf(
		"%s studied on %d days this week, exchanged %d messages, completed %d quizzes, and built a %d-day streak with %d total XP.",
		summary.ChildName,
		summary.WeeklyStats.DaysActive,
		summary.WeeklyStats.MessagesExchanged,
		summary.WeeklyStats.QuizzesCompleted,
		summary.CurrentStreak,
		summary.TotalXP,
	)

	paragraph2 := fmt.Sprintf(
		"The main area to watch this week is %s, with %d topic%s currently needing review before the next study cycle.",
		topic,
		summary.WeeklyStats.NeedsReviewCount,
		map[bool]string{true: "", false: "s"}[summary.WeeklyStats.NeedsReviewCount == 1],
	)

	paragraph3 := strings.TrimSpace(summary.EncouragementText)
	if paragraph3 == "" {
		paragraph3 = fmt.Sprintf(
			"For the week ending %s, keep encouragement specific and ask for one short follow-up practice session at home.",
			now.UTC().Format("2006-01-02"),
		)
	}

	return strings.Join([]string{paragraph1, paragraph2, paragraph3}, "\n\n")
}

func humanizeReportTopicID(topicID string) string {
	parts := strings.Split(strings.TrimSpace(topicID), "-")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, strings.ToUpper(part[:1])+part[1:])
	}
	return strings.Join(out, " ")
}

package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

type stubWeeklyParentReportSource struct {
	summaries []WeeklyParentReportSummary
	err       error
}

func (s stubWeeklyParentReportSource) ListWeeklyParentReportSummaries(context.Context) ([]WeeklyParentReportSummary, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.summaries, nil
}

func TestScheduler_SendWeeklyParentReports_UsesAIWhenAvailable(t *testing.T) {
	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("telegram", mockCh)

	mockAI := ai.NewMockProvider("Paragraph one.\n\nParagraph two.\n\nParagraph three.")
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{}})
	router.Register("mock", mockAI)

	scheduler := NewScheduler(
		SchedulerConfig{
			CheckInterval:               time.Minute,
			MaxNudgesPerDay:             MaxNudgesPerDay,
			AIPersonalizedNudgesEnabled: true,
		},
		nil, nil, nil, nil, nil, gw, router, nil,
	)
	scheduler.SetWeeklyParentReportSource(stubWeeklyParentReportSource{
		summaries: []WeeklyParentReportSummary{
			{
				ParentExternalID:   "parent-1",
				ParentChannel:      "telegram",
				ParentName:         "Farah Parent",
				ChildName:          "Alya Sofea",
				ChildForm:          "Form 1",
				CurrentStreak:      4,
				TotalXP:            120,
				NeedsReviewCount:   2,
				WeakestTopicID:     "linear-equations",
				EncouragementTitle: "Alya is building confidence.",
				EncouragementText:  "Keep support specific and short this week.",
				WeeklyStats: WeeklyParentWeeklyStats{
					DaysActive:        4,
					MessagesExchanged: 18,
					QuizzesCompleted:  3,
					NeedsReviewCount:  2,
				},
			},
		},
	})

	scheduler.SendWeeklyParentReports(context.Background(), time.Date(2026, 4, 5, 20, 0, 0, 0, time.UTC))

	if len(mockCh.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(mockCh.SentMessages))
	}
	if got := mockCh.SentMessages[0].UserID; got != "parent-1" {
		t.Fatalf("user_id = %q, want parent-1", got)
	}
	if got := mockCh.SentMessages[0].Channel; got != "telegram" {
		t.Fatalf("channel = %q, want telegram", got)
	}
	if got := mockCh.SentMessages[0].Text; got != "Paragraph one.\n\nParagraph two.\n\nParagraph three." {
		t.Fatalf("text = %q, want AI report", got)
	}
	if mockAI.LastRequest == nil {
		t.Fatal("expected AI request to be captured")
	}
	if mockAI.LastRequest.Task != ai.TaskAnalysis {
		t.Fatalf("task = %v, want %v", mockAI.LastRequest.Task, ai.TaskAnalysis)
	}
	if !strings.Contains(mockAI.LastRequest.Messages[1].Content, "Child: Alya Sofea") {
		t.Fatalf("AI prompt = %q, want child context", mockAI.LastRequest.Messages[1].Content)
	}
}

func TestScheduler_SendWeeklyParentReports_FallsBackWhenAIUnavailable(t *testing.T) {
	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("telegram", mockCh)

	mockAI := ai.NewMockProvider("")
	mockAI.Err = context.DeadlineExceeded
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{}})
	router.Register("mock", mockAI)

	scheduler := NewScheduler(
		SchedulerConfig{
			CheckInterval:               time.Minute,
			MaxNudgesPerDay:             MaxNudgesPerDay,
			AIPersonalizedNudgesEnabled: true,
		},
		nil, nil, nil, nil, nil, gw, router, nil,
	)
	scheduler.SetWeeklyParentReportSource(stubWeeklyParentReportSource{
		summaries: []WeeklyParentReportSummary{
			{
				ParentExternalID:   "parent-1",
				ParentChannel:      "telegram",
				ParentName:         "Farah Parent",
				ChildName:          "Alya Sofea",
				ChildForm:          "Form 1",
				CurrentStreak:      5,
				TotalXP:            140,
				NeedsReviewCount:   1,
				WeakestTopicID:     "linear-equations",
				EncouragementTitle: "Alya is showing strong consistency.",
				EncouragementText:  "Celebrate the streak and ask for one short practice round.",
				WeeklyStats: WeeklyParentWeeklyStats{
					DaysActive:        5,
					MessagesExchanged: 21,
					QuizzesCompleted:  4,
					NeedsReviewCount:  1,
				},
			},
		},
	})

	scheduler.SendWeeklyParentReports(context.Background(), time.Date(2026, 4, 5, 20, 0, 0, 0, time.UTC))

	if len(mockCh.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(mockCh.SentMessages))
	}
	got := mockCh.SentMessages[0].Text
	if strings.Count(got, "\n\n") != 2 {
		t.Fatalf("fallback report = %q, want 3 paragraphs", got)
	}
	if !strings.Contains(got, "Alya Sofea studied on 5 days this week") {
		t.Fatalf("fallback report = %q, want weekly activity paragraph", got)
	}
	if !strings.Contains(got, "Linear Equations") {
		t.Fatalf("fallback report = %q, want weakest topic", got)
	}
	if !strings.Contains(got, "Celebrate the streak") {
		t.Fatalf("fallback report = %q, want encouragement guidance", got)
	}
}

func TestScheduler_SendWeeklyParentReports_SkipsNonTelegramRecipients(t *testing.T) {
	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("telegram", mockCh)

	scheduler := NewScheduler(
		SchedulerConfig{CheckInterval: time.Minute, MaxNudgesPerDay: MaxNudgesPerDay},
		nil, nil, nil, nil, nil, gw, nil, nil,
	)
	scheduler.SetWeeklyParentReportSource(stubWeeklyParentReportSource{
		summaries: []WeeklyParentReportSummary{
			{ParentExternalID: "parent-1", ParentChannel: "whatsapp", ChildName: "Alya Sofea"},
			{ParentExternalID: "", ParentChannel: "telegram", ChildName: "Hakim"},
		},
	})

	scheduler.SendWeeklyParentReports(context.Background(), time.Now())

	if len(mockCh.SentMessages) != 0 {
		t.Fatalf("sent messages = %d, want 0", len(mockCh.SentMessages))
	}
}

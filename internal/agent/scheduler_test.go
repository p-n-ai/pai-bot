// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

type dueReviewTracker struct {
	item progress.ProgressItem
}

func (t dueReviewTracker) UpdateMastery(string, string, string, float64) error {
	return nil
}

func (t dueReviewTracker) GetMastery(string, string, string) (float64, error) {
	return t.item.MasteryScore, nil
}

func (t dueReviewTracker) GetAllProgress(string) ([]progress.ProgressItem, error) {
	return []progress.ProgressItem{t.item}, nil
}

func (t dueReviewTracker) GetDueReviews(string) ([]progress.ProgressItem, error) {
	return []progress.ProgressItem{t.item}, nil
}

type learnerDueReviewTracker struct {
	due map[string][]progress.ProgressItem
}

func (t learnerDueReviewTracker) UpdateMastery(string, string, string, float64) error {
	return nil
}

func (t learnerDueReviewTracker) GetMastery(string, string, string) (float64, error) {
	return 0, nil
}

func (t learnerDueReviewTracker) GetAllProgress(string) ([]progress.ProgressItem, error) {
	return nil, nil
}

func (t learnerDueReviewTracker) GetDueReviews(string) ([]progress.ProgressItem, error) {
	return nil, nil
}

func (t learnerDueReviewTracker) UpdateMasteryForLearner(progress.LearnerID, string, string, float64) error {
	return nil
}

func (t learnerDueReviewTracker) GetMasteryForLearner(progress.LearnerID, string, string) (float64, error) {
	return 0, nil
}

func (t learnerDueReviewTracker) GetAllProgressForLearner(learnerID progress.LearnerID) ([]progress.ProgressItem, error) {
	return t.due[learnerID.String()], nil
}

func (t learnerDueReviewTracker) GetDueReviewsForLearner(learnerID progress.LearnerID) ([]progress.ProgressItem, error) {
	return t.due[learnerID.String()], nil
}

func TestIsQuietHours(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")

	tests := []struct {
		name  string
		hour  int
		quiet bool
	}{
		{"midnight", 0, true},
		{"1am", 1, true},
		{"5am", 5, true},
		{"6am", 6, true},
		{"7am-boundary", 7, false},
		{"8am", 8, false},
		{"noon", 12, false},
		{"3pm", 15, false},
		{"8pm", 20, false},
		{"9pm-boundary", 21, true},
		{"10pm", 22, true},
		{"11pm", 23, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 3, 9, tt.hour, 0, 0, 0, loc)
			got := agent.IsQuietHours(now)
			if got != tt.quiet {
				t.Errorf("IsQuietHours(%d:00 MYT) = %v, want %v", tt.hour, got, tt.quiet)
			}
		})
	}
}

func TestIsQuietHours_UTC(t *testing.T) {
	// 13:00 UTC = 21:00 MYT → quiet
	utcQuiet := time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC)
	if !agent.IsQuietHours(utcQuiet) {
		t.Error("13:00 UTC (21:00 MYT) should be quiet hours")
	}

	// 02:00 UTC = 10:00 MYT → not quiet
	utcActive := time.Date(2026, 3, 9, 2, 0, 0, 0, time.UTC)
	if agent.IsQuietHours(utcActive) {
		t.Error("02:00 UTC (10:00 MYT) should not be quiet hours")
	}
}

func TestCanNudge(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")

	tests := []struct {
		name       string
		hour       int
		nudgesSent int
		canNudge   bool
	}{
		{"active-hours-no-nudges", 10, 0, true},
		{"active-hours-2-nudges", 14, 2, true},
		{"active-hours-at-limit", 14, 3, false},
		{"quiet-hours", 22, 0, false},
		{"quiet-hours-with-budget", 1, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 3, 9, tt.hour, 0, 0, 0, loc)
			got := agent.CanNudge(now, tt.nudgesSent)
			if got != tt.canNudge {
				t.Errorf("CanNudge(hour=%d, sent=%d) = %v, want %v",
					tt.hour, tt.nudgesSent, got, tt.canNudge)
			}
		})
	}
}

func TestSchedulerConfig_Defaults(t *testing.T) {
	cfg := agent.DefaultSchedulerConfig()

	if cfg.CheckInterval != 5*time.Minute {
		t.Errorf("CheckInterval = %v, want 5m", cfg.CheckInterval)
	}
	if cfg.MaxNudgesPerDay != 3 {
		t.Errorf("MaxNudgesPerDay = %d, want 3", cfg.MaxNudgesPerDay)
	}
}

func TestScheduler_DailySummaryTick(t *testing.T) {
	tracker := progress.NewMemoryTracker()
	streaks := progress.NewMemoryStreakTracker()
	xpTracker := progress.NewMemoryXPTracker()
	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("telegram", mockCh)

	_ = tracker.UpdateMastery("user1", "default", "F1-01", 0.7)
	_ = xpTracker.Award("user1", progress.XPSourceSession, 50, nil)
	_ = streaks.RecordActivity("user1", time.Now())

	scheduler := agent.NewScheduler(
		agent.SchedulerConfig{CheckInterval: 1 * time.Second, MaxNudgesPerDay: 3},
		tracker, streaks, xpTracker, nil,
		agent.NewMemoryNudgeTracker(), gw, nil, nil,
	)

	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")
	summaryTime := time.Date(2026, 3, 18, 22, 1, 0, 0, loc)

	scheduler.SendDailySummaries(context.Background(), []string{"user1"}, summaryTime)

	if len(mockCh.SentMessages) == 0 {
		t.Fatal("expected daily summary message to be sent")
	}
	msg := mockCh.SentMessages[0].Text
	if !strings.Contains(msg, "Ringkasan Kemajuan") && !strings.Contains(msg, "Progress Snapshot") {
		t.Errorf("expected summary content, got: %s", msg)
	}
}

func TestScheduler_DailySummarySkipsInactiveUser(t *testing.T) {
	tracker := progress.NewMemoryTracker()
	streaks := progress.NewMemoryStreakTracker()
	xpTracker := progress.NewMemoryXPTracker()
	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("telegram", mockCh)

	scheduler := agent.NewScheduler(
		agent.SchedulerConfig{CheckInterval: 1 * time.Second, MaxNudgesPerDay: 3},
		tracker, streaks, xpTracker, nil,
		agent.NewMemoryNudgeTracker(), gw, nil, nil,
	)

	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")
	summaryTime := time.Date(2026, 3, 18, 22, 1, 0, 0, loc)

	scheduler.SendDailySummaries(context.Background(), []string{"inactive-user"}, summaryTime)

	if len(mockCh.SentMessages) != 0 {
		t.Errorf("expected no message for inactive user, got %d", len(mockCh.SentMessages))
	}
}

func TestScheduler_DailySummaryUsesExplicitChatDestination(t *testing.T) {
	tracker := progress.NewMemoryTracker()
	streaks := progress.NewMemoryStreakTracker()
	xpTracker := progress.NewMemoryXPTracker()
	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("slack", mockCh)

	_ = tracker.UpdateMastery("U123", "default", "F1-01", 0.7)
	_ = xpTracker.Award("U123", progress.XPSourceSession, 50, nil)
	_ = streaks.RecordActivity("U123", time.Now())

	scheduler := agent.NewScheduler(
		agent.SchedulerConfig{CheckInterval: time.Second, MaxNudgesPerDay: 3},
		tracker, streaks, xpTracker, nil,
		agent.NewMemoryNudgeTracker(), gw, nil, nil,
	)

	scheduler.SendDailySummariesTo(
		context.Background(),
		[]agent.ScheduledRecipient{{
			Channel:  "slack",
			UserID:   "U123",
			ThreadID: "slack:C456:1712345678.000100",
		}},
		time.Date(2026, 3, 18, 22, 1, 0, 0, time.UTC),
	)

	if len(mockCh.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(mockCh.SentMessages))
	}
	if got := mockCh.SentMessages[0]; got.Channel != "slack" || got.UserID != "U123" || got.ThreadID != "slack:C456:1712345678.000100" {
		t.Fatalf("destination = %#v, want explicit Slack recipient and thread", got)
	}
}

func TestScheduler_NudgeUsesExplicitChatDestination(t *testing.T) {
	now := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("discord", mockCh)

	scheduler := agent.NewScheduler(
		agent.SchedulerConfig{CheckInterval: time.Second, MaxNudgesPerDay: 3},
		dueReviewTracker{item: progress.ProgressItem{
			UserID:       "discord-user",
			TopicID:      "linear-equations",
			MasteryScore: 0.6,
			NextReviewAt: now.Add(-48 * time.Hour),
		}},
		nil, nil, nil, agent.NewMemoryNudgeTracker(), gw, nil, nil,
	)

	err := scheduler.CheckRecipientForNudge(context.Background(), agent.ScheduledRecipient{
		Channel:  "discord",
		UserID:   "discord-user",
		ThreadID: "discord:guild-1:channel-2",
	}, now)
	if err != nil {
		t.Fatalf("CheckRecipientForNudge() error = %v", err)
	}

	if len(mockCh.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(mockCh.SentMessages))
	}
	if got := mockCh.SentMessages[0]; got.Channel != "discord" || got.UserID != "discord-user" || got.ThreadID != "discord:guild-1:channel-2" {
		t.Fatalf("destination = %#v, want explicit Discord recipient and thread", got)
	}
}

func TestScheduler_NudgeDataUsesProviderQualifiedLearner(t *testing.T) {
	now := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	store := agent.NewMemoryStore()
	slackIdentity, err := agent.NewLearnerIdentity("slack", "shared-user")
	if err != nil {
		t.Fatal(err)
	}
	discordIdentity, err := agent.NewLearnerIdentity("discord", "shared-user")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetUserNameFor(slackIdentity, "Slack learner"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetUserNameFor(discordIdentity, "Discord learner"); err != nil {
		t.Fatal(err)
	}
	slackUserID, err := store.ResolveUserUUIDFor(slackIdentity)
	if err != nil {
		t.Fatal(err)
	}
	discordUserID, err := store.ResolveUserUUIDFor(discordIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if slackUserID == discordUserID {
		t.Fatal("provider-qualified learners resolved to the same internal user")
	}

	mockCh := &chat.MockChannel{}
	gw := chat.NewGateway()
	gw.Register("slack", mockCh)
	gw.Register("discord", mockCh)
	scheduler := agent.NewScheduler(
		agent.SchedulerConfig{CheckInterval: time.Second, MaxNudgesPerDay: 3},
		learnerDueReviewTracker{due: map[string][]progress.ProgressItem{
			discordUserID: {{
				UserID:       discordUserID,
				TopicID:      "discord-only-topic",
				MasteryScore: 0.5,
				NextReviewAt: now.Add(-time.Hour),
			}},
		}},
		nil,
		nil,
		nil,
		agent.NewMemoryNudgeTracker(),
		gw,
		nil,
		store,
	)

	if err := scheduler.CheckRecipientForNudge(context.Background(), agent.ScheduledRecipient{
		Channel:  "slack",
		UserID:   "shared-user",
		ThreadID: "slack:C1:T1",
	}, now); err != nil {
		t.Fatalf("Slack check error = %v", err)
	}
	if err := scheduler.CheckRecipientForNudge(context.Background(), agent.ScheduledRecipient{
		Channel:  "discord",
		UserID:   "shared-user",
		ThreadID: "discord:G1:C1",
	}, now); err != nil {
		t.Fatalf("Discord check error = %v", err)
	}

	if len(mockCh.SentMessages) != 1 {
		t.Fatalf("sent messages = %d, want only Discord learner nudge", len(mockCh.SentMessages))
	}
	if got := mockCh.SentMessages[0]; got.Channel != "discord" || !strings.Contains(got.Text, "discord-only-topic") {
		t.Fatalf("message = %#v, want Discord learner's due topic", got)
	}
}

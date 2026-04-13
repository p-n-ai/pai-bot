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

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

const dailySummaryHour = 22

// DailySummary holds a cumulative progress snapshot for a user.
type DailySummary struct {
	UserID         string
	TopicsStudied  int
	MasteredTopics int
	TotalXP        int
	CurrentStreak  int
	BestTopic      string
	BestMastery    float64
}

// ComputeDailySummary builds a cumulative progress snapshot for the given user.
func ComputeDailySummary(userID string, tracker progress.Tracker, streaks progress.StreakTracker, xp progress.XPTracker) DailySummary {
	summary := DailySummary{UserID: userID}
	if tracker != nil {
		items, err := tracker.GetAllProgress(userID)
		if err == nil {
			summary.TopicsStudied = len(items)
			for _, item := range items {
				if progress.IsMastered(item.MasteryScore) {
					summary.MasteredTopics++
				}
				if item.MasteryScore > summary.BestMastery {
					summary.BestMastery = item.MasteryScore
					summary.BestTopic = item.TopicID
				}
			}
		}
	}
	if streaks != nil {
		s, err := streaks.GetStreak(userID)
		if err == nil {
			summary.CurrentStreak = s.CurrentStreak
		}
	}
	if xp != nil {
		total, err := xp.GetTotal(userID)
		if err == nil {
			summary.TotalXP = total
		}
	}
	return summary
}

// FormatDailySummary returns a formatted message for the given locale.
// Returns an empty string if the user had no activity (TopicsStudied == 0).
func FormatDailySummary(summary DailySummary, locale string) string {
	if summary.TopicsStudied == 0 {
		return ""
	}
	var sb strings.Builder
	switch locale {
	case "en":
		sb.WriteString("📋 *Progress Snapshot*\n\n")
		fmt.Fprintf(&sb, "📚 Topics studied: %d\n", summary.TopicsStudied)
		if summary.MasteredTopics > 0 {
			fmt.Fprintf(&sb, "✅ Topics mastered: %d\n", summary.MasteredTopics)
		}
		fmt.Fprintf(&sb, "⭐ Total XP: %d\n", summary.TotalXP)
		if summary.CurrentStreak > 0 {
			fmt.Fprintf(&sb, "🔥 Streak: %d days\n", summary.CurrentStreak)
		}
		if summary.BestTopic != "" {
			fmt.Fprintf(&sb, "\n💪 Best topic: %s (%d%%)\n", summary.BestTopic, int(summary.BestMastery*100))
		}
		sb.WriteString("\nGood night! See you tomorrow! 🌙")
	case "zh":
		sb.WriteString("📋 *学习进度*\n\n")
		fmt.Fprintf(&sb, "📚 学习主题: %d\n", summary.TopicsStudied)
		if summary.MasteredTopics > 0 {
			fmt.Fprintf(&sb, "✅ 已掌握: %d\n", summary.MasteredTopics)
		}
		fmt.Fprintf(&sb, "⭐ 总 XP: %d\n", summary.TotalXP)
		if summary.CurrentStreak > 0 {
			fmt.Fprintf(&sb, "🔥 连续学习: %d 天\n", summary.CurrentStreak)
		}
		if summary.BestTopic != "" {
			fmt.Fprintf(&sb, "\n💪 最佳主题: %s (%d%%)\n", summary.BestTopic, int(summary.BestMastery*100))
		}
		sb.WriteString("\n晚安！明天见！🌙")
	default:
		sb.WriteString("📋 *Ringkasan Kemajuan*\n\n")
		fmt.Fprintf(&sb, "📚 Topik dipelajari: %d\n", summary.TopicsStudied)
		if summary.MasteredTopics > 0 {
			fmt.Fprintf(&sb, "✅ Topik dikuasai: %d\n", summary.MasteredTopics)
		}
		fmt.Fprintf(&sb, "⭐ Jumlah XP: %d\n", summary.TotalXP)
		if summary.CurrentStreak > 0 {
			fmt.Fprintf(&sb, "🔥 Streak: %d hari\n", summary.CurrentStreak)
		}
		if summary.BestTopic != "" {
			fmt.Fprintf(&sb, "\n💪 Topik terbaik: %s (%d%%)\n", summary.BestTopic, int(summary.BestMastery*100))
		}
		sb.WriteString("\nSelamat malam! Jumpa esok! 🌙")
	}
	return sb.String()
}

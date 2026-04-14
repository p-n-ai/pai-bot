// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"fmt"
	"strings"
)

// FormatProgressBar creates a Unicode progress bar.
func FormatProgressBar(score float64, width int) string {
	filled := int(score * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// FormatProgressReport creates a text report of all progress items.
func FormatProgressReport(items []ProgressItem, totalXP int, streak int) string {
	var sb strings.Builder

	sb.WriteString("📊 Your Progress\n\n")

	if streak > 0 {
		fmt.Fprintf(&sb, "🔥 Streak: %d days\n", streak)
	}
	fmt.Fprintf(&sb, "⭐ XP: %d\n\n", totalXP)

	for _, item := range items {
		bar := FormatProgressBar(item.MasteryScore, 10)
		pct := int(item.MasteryScore * 100)
		status := "📖"
		if IsMastered(item.MasteryScore) {
			status = "✅"
		}
		fmt.Fprintf(&sb, "%s %s %s %d%%\n", status, item.TopicID, bar, pct)
	}

	if len(items) == 0 {
		sb.WriteString("Belum ada kemajuan lagi. Mari mula belajar! 🚀\n")
	}

	return sb.String()
}

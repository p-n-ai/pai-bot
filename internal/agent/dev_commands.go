package agent

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

// handleDevBoost sets the current topic's mastery to a target value (default 0.85).
// Usage: /dev-boost [score]  e.g. /dev-boost 0.9
func (e *Engine) handleDevBoost(msg chat.InboundMessage, args []string) (string, error) {
	if e.tracker == nil {
		return "[DEV] No tracker configured.", nil
	}

	conv, _ := e.store.GetActiveConversation(msg.UserID)
	if conv == nil || conv.TopicID == "" {
		return "[DEV] No active topic. Use /learn <topic> first.", nil
	}

	target := 0.85
	if len(args) > 0 {
		if v, err := strconv.ParseFloat(args[0], 64); err == nil && v > 0 && v <= 1.0 {
			target = v
		}
	}

	// Look up topic for syllabus ID.
	syllabusID := "default"
	topicName := conv.TopicID
	if e.curriculumLoader != nil {
		if t, ok := e.curriculumLoader.GetTopic(conv.TopicID); ok {
			if t.SyllabusID != "" {
				syllabusID = t.SyllabusID
			}
			topicName = t.Name
		}
	}

	// Set mastery directly via the postgres/memory tracker.
	if setter, ok := e.tracker.(interface {
		SetMastery(userID, syllabusID, topicID string, score float64) error
	}); ok {
		if err := setter.SetMastery(msg.UserID, syllabusID, conv.TopicID, target); err != nil {
			slog.Error("dev-boost: failed to set mastery", "error", err)
			return "[DEV] Failed to set mastery.", nil
		}
	} else {
		// Fallback: use UpdateMastery with target as delta (will be EMA'd, not exact).
		if err := e.tracker.UpdateMastery(msg.UserID, syllabusID, conv.TopicID, target); err != nil {
			slog.Error("dev-boost: failed to update mastery", "error", err)
			return "[DEV] Failed to update mastery.", nil
		}
	}

	// Trigger unlock check.
	if e.curriculumLoader != nil {
		if t, ok := e.curriculumLoader.GetTopic(conv.TopicID); ok {
			e.checkTopicUnlocks(msg.UserID, syllabusID, &t)
		}
	}

	slog.Info("dev-boost: mastery set", "user_id", msg.UserID, "topic_id", conv.TopicID, "target", target)
	return fmt.Sprintf("[DEV] Mastery for %s set to %.0f%%.", topicName, target*100), nil
}

// handleDevSummary triggers the daily summary for the current user.
func (e *Engine) handleDevSummary(msg chat.InboundMessage) (string, error) {
	locale := "ms"
	if lang, ok := e.store.GetUserPreferredLanguage(msg.UserID); ok && lang != "" {
		locale = lang
	}
	summary := ComputeDailySummary(msg.UserID, e.tracker, e.streaks, e.xp)
	result := FormatDailySummary(summary, locale)
	if result == "" {
		return "[DEV] No activity to summarize.", nil
	}
	return result, nil
}

// handleDevAB manually sets the user's AB test group.
// Usage: /dev-ab A  or  /dev-ab B
func (e *Engine) handleDevAB(msg chat.InboundMessage, args []string) (string, error) {
	if len(args) == 0 {
		current := e.userABGroup(msg.UserID)
		return fmt.Sprintf("[DEV] Current AB group: %s. Usage: /dev-ab A or /dev-ab B", current), nil
	}
	group := strings.ToUpper(strings.TrimSpace(args[0]))
	if group != ABGroupA && group != ABGroupB {
		return "[DEV] Invalid group. Use: /dev-ab A or /dev-ab B", nil
	}
	if err := e.store.SetUserABGroup(msg.UserID, group); err != nil {
		slog.Error("dev-ab: failed to set AB group", "user_id", msg.UserID, "error", err)
		return "[DEV] Failed to set AB group.", nil
	}
	slog.Info("dev-ab: AB group set", "user_id", msg.UserID, "group", group)
	return fmt.Sprintf("[DEV] AB group set to %s.", group), nil
}

// handleDevReset fully resets a user's state: conversation, profile, mastery, XP, streaks, goals.
// Only available when DevMode is enabled (LEARN_DEV_MODE=true).
func (e *Engine) handleDevReset(msg chat.InboundMessage) (string, error) {
	userID := msg.UserID

	// End active conversation.
	e.endActiveConversation(userID)

	// Reset profile (form, language, quiz intensity).
	e.resetLearnerProfile(userID)

	// Reset mastery/progress.
	if e.tracker != nil {
		if resetter, ok := e.tracker.(interface{ ResetAll(userID string) error }); ok {
			if err := resetter.ResetAll(userID); err != nil {
				slog.Error("dev-reset: failed to reset mastery", "user_id", userID, "error", err)
			}
		}
	}

	// Reset streaks.
	if e.streaks != nil {
		if resetter, ok := e.streaks.(interface{ ResetAll(userID string) error }); ok {
			if err := resetter.ResetAll(userID); err != nil {
				slog.Error("dev-reset: failed to reset streaks", "user_id", userID, "error", err)
			}
		}
	}

	// Reset XP.
	if e.xp != nil {
		if resetter, ok := e.xp.(interface{ ResetAll(userID string) error }); ok {
			if err := resetter.ResetAll(userID); err != nil {
				slog.Error("dev-reset: failed to reset XP", "user_id", userID, "error", err)
			}
		}
	}

	// Clear goals.
	if e.goals != nil {
		if err := e.goals.ClearActiveGoals(userID); err != nil {
			slog.Error("dev-reset: failed to clear goals", "user_id", userID, "error", err)
		}
	}

	// Clear pending unlocks.
	if e.unlocks != nil {
		e.unlocks.drain(userID)
	}

	slog.Info("dev-reset: user fully reset", "user_id", userID)
	return "[DEV] Full reset complete. Mastery, XP, streaks, goals, and profile cleared.", nil
}

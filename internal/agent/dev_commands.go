package agent

import (
	"log/slog"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

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

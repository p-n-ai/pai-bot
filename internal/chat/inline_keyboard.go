// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"regexp"
	"strings"
)

var reviewActionPattern = regexp.MustCompile(`\[\[PAI_REVIEW(?::([A-Za-z0-9-]+))?\]\]`)

type TelegramInlineKeyboardContext struct {
	QuizIntensityPending bool
	QuizActive           bool
	QuizPaused           bool
	ChallengeActive      bool
	ChallengeReview      bool
}

// BuildTelegramInlineKeyboard returns inline keyboard rows inferred from the
// outgoing message text. Returns nil when no inline keyboard is needed.
func BuildTelegramInlineKeyboard(text string) [][]InlineButton {
	return BuildTelegramInlineKeyboardWithContext(text, TelegramInlineKeyboardContext{})
}

// AppendFocusedPageButton keeps channel-specific focused-page rendering in the Telegram adapter layer.
func AppendFocusedPageButton(rows [][]InlineButton, pageURL string) [][]InlineButton {
	if strings.TrimSpace(pageURL) == "" {
		return rows
	}
	return append(rows, []InlineButton{{Text: "Open focused page", URL: pageURL}})
}

// BuildTelegramInlineKeyboardWithContext returns inline keyboard rows inferred
// from the outgoing message text plus explicit runtime state when available.
func BuildTelegramInlineKeyboardWithContext(text string, ctx TelegramInlineKeyboardContext) [][]InlineButton {
	lower := strings.ToLower(text)

	hasLangPrompt :=
		strings.Contains(lower, "bahasa pilihan anda") ||
			strings.Contains(lower, "language preference") ||
			strings.Contains(lower, "english") && strings.Contains(lower, "bahasa melayu")
	if hasLangPrompt {
		return [][]InlineButton{
			{
				{Text: "English", CallbackData: "lang:en"},
				{Text: "BM", CallbackData: "lang:ms"},
				{Text: "中文", CallbackData: "lang:zh"},
			},
		}
	}

	hasQuizIntensityPrompt :=
		strings.Contains(lower, "what intensity do you want for this quiz?") &&
			strings.Contains(lower, "reply with: easy, medium, hard, or mixed.")
	if ctx.QuizIntensityPending || hasQuizIntensityPrompt {
		return [][]InlineButton{
			{
				{Text: "Easy", CallbackData: "quiz:intensity:easy"},
				{Text: "Medium", CallbackData: "quiz:intensity:medium"},
			},
			{
				{Text: "Hard", CallbackData: "quiz:intensity:hard"},
				{Text: "Mixed", CallbackData: "quiz:intensity:mixed"},
			},
		}
	}

	hasQuizQuestionPrompt :=
		!ctx.ChallengeActive && !ctx.ChallengeReview &&
			strings.Contains(text, "Question ") &&
			(strings.Contains(lower, "reply with your answer.") || strings.Contains(lower, "reply with a short explanation."))
	hasQuizRetryPrompt := strings.Contains(lower, "try the same question again.")
	if ctx.QuizActive || hasQuizQuestionPrompt || hasQuizRetryPrompt {
		return [][]InlineButton{
			{
				{Text: "Hint", CallbackData: "hint"},
				{Text: "Repeat", CallbackData: "repeat"},
				{Text: "Stop", CallbackData: "stop quiz"},
			},
		}
	}

	hasQuizPausedPrompt := strings.Contains(lower, "i paused the quiz") && strings.Contains(lower, "continue quiz")
	if ctx.QuizPaused || hasQuizPausedPrompt {
		return [][]InlineButton{
			{
				{Text: "Continue", CallbackData: "continue quiz"},
				{Text: "Stop", CallbackData: "stop quiz"},
			},
		}
	}

	// Challenge: searching for opponent → Cancel button
	if strings.Contains(lower, "searching for an opponent") {
		return [][]InlineButton{
			{
				{Text: "Cancel", CallbackData: "challenge:cancel"},
			},
		}
	}

	// Challenge: pending acceptance → Accept / Decline buttons
	if strings.Contains(lower, "state: pending_acceptance") && strings.Contains(lower, "/challenge accept") {
		return [][]InlineButton{
			{
				{Text: "Accept", CallbackData: "challenge:accept"},
				{Text: "Decline", CallbackData: "challenge:cancel"},
			},
		}
	}

	// Challenge: review offer → Review / Skip buttons
	hasChallengeReviewOffer :=
		(strings.Contains(lower, "review") || strings.Contains(lower, "ulang kaji") || strings.Contains(lower, "复习")) &&
			(strings.Contains(lower, "missed") || strings.Contains(lower, "salah") || strings.Contains(lower, "答错"))
	if hasChallengeReviewOffer {
		return [][]InlineButton{
			{
				{Text: "Review", CallbackData: "challenge:review"},
				{Text: "Skip", CallbackData: "challenge:skip"},
			},
		}
	}

	return nil
}

// StripReviewActionCodes removes review control tokens from outgoing text.
func StripReviewActionCodes(text string) string {
	return strings.TrimSpace(reviewActionPattern.ReplaceAllString(text, ""))
}

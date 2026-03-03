package chat

import (
	"strconv"
	"regexp"
	"strings"
)

var reviewActionPattern = regexp.MustCompile(`\[\[PAI_REVIEW(?::([A-Za-z0-9-]+))?\]\]`)

// BuildTelegramInlineKeyboard returns inline keyboard rows inferred from the
// outgoing message text. Returns nil when no inline keyboard is needed.
func BuildTelegramInlineKeyboard(text string) [][]InlineButton {
	lower := strings.ToLower(text)
	hasLegacyPrompt := strings.Contains(lower, "nilai penerangan saya (1-5)")
	hasGenericPrompt := strings.Contains(lower, "rating 1-5")

	reviewMatch := reviewActionPattern.FindStringSubmatch(text)
	hasReviewCode := len(reviewMatch) > 0
	reviewMessageID := ""
	if len(reviewMatch) > 1 {
		reviewMessageID = strings.TrimSpace(reviewMatch[1])
	}
	if hasReviewCode || hasLegacyPrompt || hasGenericPrompt {
		callbackData := func(score int) string {
			if hasReviewCode && reviewMessageID != "" {
				return "rating:" + reviewMessageID + ":" + strconv.Itoa(score)
			}
			return strconv.Itoa(score)
		}
		return [][]InlineButton{
			{
				{Text: "1⭐", CallbackData: callbackData(1)},
				{Text: "2⭐", CallbackData: callbackData(2)},
				{Text: "3⭐", CallbackData: callbackData(3)},
				{Text: "4⭐", CallbackData: callbackData(4)},
				{Text: "5⭐", CallbackData: callbackData(5)},
			},
		}
	}

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

	return nil
}

// StripReviewActionCodes removes review control tokens from outgoing text.
func StripReviewActionCodes(text string) string {
	return strings.TrimSpace(reviewActionPattern.ReplaceAllString(text, ""))
}

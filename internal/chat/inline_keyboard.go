package chat

import "strings"

// BuildTelegramInlineKeyboard returns inline keyboard rows inferred from the
// outgoing message text. Returns nil when no inline keyboard is needed.
func BuildTelegramInlineKeyboard(text string) [][]InlineButton {
	lower := strings.ToLower(text)
	hasLegacyPrompt := strings.Contains(lower, "nilai penerangan saya (1-5)")
	hasGenericPrompt := strings.Contains(lower, "rating 1-5")
	hasReviewCode := strings.Contains(text, "[[PAI_REVIEW]]")
	if hasLegacyPrompt || hasGenericPrompt || hasReviewCode {
		return [][]InlineButton{
			{
				{Text: "1⭐", CallbackData: "1"},
				{Text: "2⭐", CallbackData: "2"},
				{Text: "3⭐", CallbackData: "3"},
				{Text: "4⭐", CallbackData: "4"},
				{Text: "5⭐", CallbackData: "5"},
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

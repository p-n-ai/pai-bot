package chat

import "strings"

// BuildTelegramReplyKeyboard returns Telegram reply keyboard rows inferred from
// the outgoing message text. Returns nil when no keyboard is needed.
func BuildTelegramReplyKeyboard(text string) [][]string {
	lower := strings.ToLower(text)

	if strings.Contains(lower, "nilai penerangan saya (1-5)") {
		return [][]string{
			{"1", "2", "3", "4", "5"},
		}
	}

	if strings.Contains(lower, "tingkatan berapa anda sekarang?") ||
		strings.Contains(lower, "balas dengan: 1, 2, atau 3") {
		return [][]string{
			{"1", "2", "3"},
		}
	}

	return nil
}


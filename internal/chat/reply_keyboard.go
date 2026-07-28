// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import "strings"

// BuildTelegramReplyKeyboard returns Telegram reply keyboard rows inferred from
// the outgoing message text. Returns nil when no keyboard is needed.
func BuildTelegramReplyKeyboard(text string) [][]string {
	normalized := strings.ToLower(text)
	if containsAny(normalized,
		"choose your preferred session language",
		"choose your language",
		"bahasa pilihan anda",
		"请选择本次学习语言",
		"请选择你的语言",
		"couldn't determine your preferred language",
		"belum pasti bahasa pilihan anda",
		"还不能确定你的语言偏好",
	) {
		return [][]string{{"English", "Bahasa Melayu", "中文"}}
	}
	if containsAny(normalized,
		"what form are you in now",
		"which form are you in now",
		"tingkatan berapa anda sekarang",
		"你现在是几年级",
		"couldn't determine your form",
		"belum pasti tingkatan anda",
		"还不能确定你的年级",
	) {
		return [][]string{{"1", "2", "3"}}
	}
	return nil
}

func containsAny(text string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

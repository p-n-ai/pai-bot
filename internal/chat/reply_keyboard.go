// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import "strings"

// BuildTelegramReplyKeyboard returns Telegram reply keyboard rows inferred from
// the outgoing message text. Returns nil when no keyboard is needed.
func BuildTelegramReplyKeyboard(text string) [][]string {
	_ = strings.ToLower(text)
	return nil
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import "strings"

// RenderTurn projects semantic tutor text and an optional artifact into channel-owned formatting.
func RenderTurn(in InboundMessage, text, focusedPageURL string, telegramContext TelegramInlineKeyboardContext) (OutboundMessage, bool) {
	out := OutboundMessage{Channel: in.Channel, UserID: in.UserID, Text: StripReviewActionCodes(text)}
	if in.Channel == "telegram" {
		out.Text = ConvertLaTeXToUnicode(text)
		out.Text = NormalizeTelegramMarkdown(out.Text)
		out.ParseMode = "Markdown"
		out.ReplyKeyboard = BuildTelegramReplyKeyboard(text)
		out.InlineKeyboard = BuildTelegramInlineKeyboardWithContext(text, telegramContext)
		out.InlineKeyboard = AppendFocusedPageButton(out.InlineKeyboard, focusedPageURL)
		out.Text = StripReviewActionCodes(out.Text)
	}
	return out, strings.TrimSpace(out.Text) != ""
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestBuildTelegramReplyKeyboard_RatingPrompt(t *testing.T) {
	got := chat.BuildTelegramReplyKeyboard("Nilai penerangan saya (1-5): balas dengan 1, 2, 3, 4, atau 5.")
	if got != nil {
		t.Fatalf("BuildTelegramReplyKeyboard() = %#v, want nil for rating prompt", got)
	}
}

func TestBuildTelegramReplyKeyboard_OnboardingPrompt(t *testing.T) {
	got := chat.BuildTelegramReplyKeyboard("Tingkatan berapa anda sekarang?\nBalas dengan: 1, 2, atau 3.")
	if got != nil {
		t.Fatalf("BuildTelegramReplyKeyboard() = %#v, want nil for onboarding prompt", got)
	}
}

func TestBuildTelegramReplyKeyboard_NoPrompt(t *testing.T) {
	got := chat.BuildTelegramReplyKeyboard("Terangkan persamaan linear.")
	if got != nil {
		t.Fatalf("BuildTelegramReplyKeyboard() = %#v, want nil", got)
	}
}

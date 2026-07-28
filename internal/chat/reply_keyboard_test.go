// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat_test

import (
	"reflect"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestBuildTelegramReplyKeyboard_RatingPrompt(t *testing.T) {
	got := chat.BuildTelegramReplyKeyboard("Nilai penerangan saya (1-5): balas dengan 1, 2, 3, 4, atau 5.")
	if got != nil {
		t.Fatalf("BuildTelegramReplyKeyboard() = %#v, want nil for rating prompt", got)
	}
}

func TestBuildTelegramReplyKeyboard_OnboardingLanguagePrompts(t *testing.T) {
	for _, prompt := range []string{
		"Choose your preferred session language:\n- English\n- Bahasa Melayu\n- 中文",
		"Choose your language:\n- English\n- Bahasa Melayu\n- 中文",
		"Bahasa pilihan anda untuk sesi ini?\n- English\n- Bahasa Melayu\n- 中文",
		"请选择本次学习语言：\n- English\n- Bahasa Melayu\n- 中文",
		"请选择你的语言：\n- English\n- Bahasa Melayu\n- 中文",
		"I couldn't determine your preferred language. Please reply with: English, Bahasa Melayu, or 中文.",
	} {
		got := chat.BuildTelegramReplyKeyboard(prompt)
		want := [][]string{{"English", "Bahasa Melayu", "中文"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("BuildTelegramReplyKeyboard(%q) = %#v, want %#v", prompt, got, want)
		}
	}
}

func TestBuildTelegramReplyKeyboard_OnboardingFormPrompts(t *testing.T) {
	for _, prompt := range []string{
		"What form are you in now?\nReply with: 1, 2, or 3.",
		"Tingkatan berapa anda sekarang?\nBalas dengan: 1, 2, atau 3.",
		"你现在是几年级？\n请回复：1、2 或 3。",
		"I couldn't determine your form. Reply with 1, 2, or 3.",
	} {
		got := chat.BuildTelegramReplyKeyboard(prompt)
		want := [][]string{{"1", "2", "3"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("BuildTelegramReplyKeyboard(%q) = %#v, want %#v", prompt, got, want)
		}
	}
}

func TestBuildTelegramReplyKeyboard_NoPrompt(t *testing.T) {
	got := chat.BuildTelegramReplyKeyboard("Terangkan persamaan linear.")
	if got != nil {
		t.Fatalf("BuildTelegramReplyKeyboard() = %#v, want nil", got)
	}
}

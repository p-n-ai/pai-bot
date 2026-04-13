// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat_test

import (
	"reflect"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestBuildTelegramInlineKeyboard_RatingPrompt(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboard("Nilai penerangan saya (1-5): balas dengan 1, 2, 3, 4, atau 5.")
	want := [][]chat.InlineButton{
		{
			{Text: "1⭐", CallbackData: "1"},
			{Text: "2⭐", CallbackData: "2"},
			{Text: "3⭐", CallbackData: "3"},
			{Text: "4⭐", CallbackData: "4"},
			{Text: "5⭐", CallbackData: "5"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTelegramInlineKeyboard() = %#v, want %#v", got, want)
	}
}

func TestBuildTelegramInlineKeyboard_NoPrompt(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboard("Tingkatan berapa anda sekarang?")
	if got != nil {
		t.Fatalf("BuildTelegramInlineKeyboard() = %#v, want nil", got)
	}
}

func TestBuildTelegramInlineKeyboard_ReviewTokenWithMessageID(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboard("Thanks!\n\n[[PAI_REVIEW:msg-123]]")
	want := [][]chat.InlineButton{
		{
			{Text: "1⭐", CallbackData: "rating:msg-123:1"},
			{Text: "2⭐", CallbackData: "rating:msg-123:2"},
			{Text: "3⭐", CallbackData: "rating:msg-123:3"},
			{Text: "4⭐", CallbackData: "rating:msg-123:4"},
			{Text: "5⭐", CallbackData: "rating:msg-123:5"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTelegramInlineKeyboard() = %#v, want %#v", got, want)
	}
}

func TestBuildTelegramInlineKeyboard_QuizIntensityPrompt(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboard("Aina (Form 2), what intensity do you want for this quiz?\nReply with: easy, medium, hard, or mixed.")
	want := [][]chat.InlineButton{
		{
			{Text: "Easy", CallbackData: "quiz:intensity:easy"},
			{Text: "Medium", CallbackData: "quiz:intensity:medium"},
		},
		{
			{Text: "Hard", CallbackData: "quiz:intensity:hard"},
			{Text: "Mixed", CallbackData: "quiz:intensity:mixed"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTelegramInlineKeyboard() = %#v, want %#v", got, want)
	}
}

func TestBuildTelegramInlineKeyboard_QuizQuestionPrompt(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboard("Quiz mode: Linear Equations\nQuestion 1/3\nSolve 2x + 1 = 9\nReply with your answer.")
	want := [][]chat.InlineButton{
		{
			{Text: "Hint", CallbackData: "hint"},
			{Text: "Repeat", CallbackData: "repeat"},
			{Text: "Stop", CallbackData: "stop quiz"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTelegramInlineKeyboard() = %#v, want %#v", got, want)
	}
}

func TestBuildTelegramInlineKeyboardWithContext_QuizRetryPrompt(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboardWithContext(
		"Not quite.\nHint: Isolate x first.\nTry the same question again.",
		chat.TelegramInlineKeyboardContext{QuizActive: true},
	)
	want := [][]chat.InlineButton{
		{
			{Text: "Hint", CallbackData: "hint"},
			{Text: "Repeat", CallbackData: "repeat"},
			{Text: "Stop", CallbackData: "stop quiz"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTelegramInlineKeyboardWithContext() = %#v, want %#v", got, want)
	}
}

func TestBuildTelegramInlineKeyboard_QuizPausedPrompt(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboard("Okay, I paused the quiz. We can talk first. Say continue quiz when you want to resume.")
	want := [][]chat.InlineButton{
		{
			{Text: "Continue", CallbackData: "continue quiz"},
			{Text: "Stop", CallbackData: "stop quiz"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTelegramInlineKeyboard() = %#v, want %#v", got, want)
	}
}

func TestBuildTelegramInlineKeyboardWithContext_PausedQuizDetourReply(t *testing.T) {
	got := chat.BuildTelegramInlineKeyboardWithContext(
		"The weather looks clear today. Want to continue your quiz after this?",
		chat.TelegramInlineKeyboardContext{QuizPaused: true},
	)
	want := [][]chat.InlineButton{
		{
			{Text: "Continue", CallbackData: "continue quiz"},
			{Text: "Stop", CallbackData: "stop quiz"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTelegramInlineKeyboardWithContext() = %#v, want %#v", got, want)
	}
}

func TestStripReviewActionCodes(t *testing.T) {
	got := chat.StripReviewActionCodes("Nice explanation\n\n[[PAI_REVIEW:abc-123]]")
	if got != "Nice explanation" {
		t.Fatalf("StripReviewActionCodes() = %q, want %q", got, "Nice explanation")
	}
}

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

func TestStripReviewActionCodes(t *testing.T) {
	got := chat.StripReviewActionCodes("Nice explanation\n\n[[PAI_REVIEW:abc-123]]")
	if got != "Nice explanation" {
		t.Fatalf("StripReviewActionCodes() = %q, want %q", got, "Nice explanation")
	}
}

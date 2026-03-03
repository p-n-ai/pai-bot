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

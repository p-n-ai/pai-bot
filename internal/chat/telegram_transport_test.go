// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTelegramChannel_SendMessage_QuizInlineKeyboardPayload(t *testing.T) {
	type requestCapture struct {
		path   string
		values url.Values
	}

	var captures []requestCapture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		captures = append(captures, requestCapture{
			path:   r.URL.Path,
			values: values,
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":101}}`))
	}))
	defer server.Close()

	ch, err := NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	ch.baseURL = server.URL

	msg := OutboundMessage{
		Channel:        "telegram",
		UserID:         "123456",
		Text:           "Quiz mode: Linear Equations\nQuestion 1/3\nSolve 2x + 1 = 9\nReply with your answer.",
		InlineKeyboard: BuildTelegramInlineKeyboard("Quiz mode: Linear Equations\nQuestion 1/3\nSolve 2x + 1 = 9\nReply with your answer."),
	}

	if err := ch.SendMessage(context.Background(), msg.UserID, msg); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if len(captures) != 1 {
		t.Fatalf("captures = %d, want 1", len(captures))
	}
	if captures[0].path != "/sendMessage" {
		t.Fatalf("path = %q, want /sendMessage", captures[0].path)
	}

	replyMarkup := captures[0].values.Get("reply_markup")
	if replyMarkup == "" {
		t.Fatal("expected reply_markup payload")
	}

	var payload struct {
		InlineKeyboard [][]struct {
			Text         string `json:"text"`
			CallbackData string `json:"callback_data"`
		} `json:"inline_keyboard"`
	}
	if err := json.Unmarshal([]byte(replyMarkup), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.InlineKeyboard) != 1 || len(payload.InlineKeyboard[0]) != 3 {
		t.Fatalf("inline keyboard = %#v, want 1 row with 3 buttons", payload.InlineKeyboard)
	}

	got := payload.InlineKeyboard[0]
	want := []struct {
		text string
		data string
	}{
		{text: "Hint", data: "hint"},
		{text: "Repeat", data: "repeat"},
		{text: "Stop", data: "stop quiz"},
	}
	for i, button := range got {
		if button.Text != want[i].text || button.CallbackData != want[i].data {
			t.Fatalf("button[%d] = %#v, want text=%q callback_data=%q", i, button, want[i].text, want[i].data)
		}
	}
}

func TestTelegramChannel_DoesNotBuildRatingUI(t *testing.T) {
	for _, text := range []string{
		"Nilai penerangan saya (1-5)",
		"Please give a rating 1-5",
		"Thanks!\n\n[[PAI_REVIEW:msg-123]]",
	} {
		if got := BuildTelegramInlineKeyboard(text); got != nil {
			t.Fatalf("BuildTelegramInlineKeyboard(%q) = %#v, want nil", text, got)
		}
	}
}

func TestTelegramChannel_AnswerCallbackQuery_SendsCallbackAck(t *testing.T) {
	var capturedPath string
	var capturedValues url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		capturedPath = r.URL.Path
		capturedValues = values
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	ch, err := NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	ch.baseURL = server.URL

	if err := ch.answerCallbackQuery(context.Background(), "cb-123"); err != nil {
		t.Fatalf("answerCallbackQuery() error = %v", err)
	}
	if capturedPath != "/answerCallbackQuery" {
		t.Fatalf("path = %q, want /answerCallbackQuery", capturedPath)
	}
	if got := strings.TrimSpace(capturedValues.Get("callback_query_id")); got != "cb-123" {
		t.Fatalf("callback_query_id = %q, want cb-123", got)
	}
}

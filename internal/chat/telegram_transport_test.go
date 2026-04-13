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
	"strconv"
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
		Channel: "telegram",
		UserID:  "123456",
		Text:    "Quiz mode: Linear Equations\nQuestion 1/3\nSolve 2x + 1 = 9\nReply with your answer.",
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

func TestTelegramChannel_SendMessage_RatingInlineKeyboardPayload(t *testing.T) {
	var replyMarkup string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		replyMarkup = values.Get("reply_markup")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":202}}`))
	}))
	defer server.Close()

	ch, err := NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	ch.baseURL = server.URL

	text := "Thanks!\n\n[[PAI_REVIEW:msg-123]]"
	msg := OutboundMessage{
		Channel:        "telegram",
		UserID:         "123456",
		Text:           StripReviewActionCodes(text),
		InlineKeyboard: BuildTelegramInlineKeyboard(text),
	}
	if err := ch.SendMessage(context.Background(), msg.UserID, msg); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
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
	if len(payload.InlineKeyboard) != 1 || len(payload.InlineKeyboard[0]) != 5 {
		t.Fatalf("inline keyboard = %#v, want 1 row with 5 rating buttons", payload.InlineKeyboard)
	}
	if payload.InlineKeyboard[0][3].CallbackData != "rating:msg-123:4" {
		t.Fatalf("rating callback = %q, want rating:msg-123:4", payload.InlineKeyboard[0][3].CallbackData)
	}
}

func TestTelegramChannel_MarkSelectedRatingInlineKeyboard_SendsEditRequest(t *testing.T) {
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

	if err := ch.markSelectedRatingInlineKeyboard(context.Background(), "123456", 88, 4, "rating:msg-123:4"); err != nil {
		t.Fatalf("markSelectedRatingInlineKeyboard() error = %v", err)
	}
	if capturedPath != "/editMessageReplyMarkup" {
		t.Fatalf("path = %q, want /editMessageReplyMarkup", capturedPath)
	}
	if capturedValues.Get("chat_id") != "123456" {
		t.Fatalf("chat_id = %q, want 123456", capturedValues.Get("chat_id"))
	}
	if capturedValues.Get("message_id") != strconv.Itoa(88) {
		t.Fatalf("message_id = %q, want 88", capturedValues.Get("message_id"))
	}

	var payload struct {
		InlineKeyboard [][]struct {
			Text         string `json:"text"`
			CallbackData string `json:"callback_data"`
		} `json:"inline_keyboard"`
	}
	if err := json.Unmarshal([]byte(capturedValues.Get("reply_markup")), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.InlineKeyboard) != 1 || len(payload.InlineKeyboard[0]) != 1 {
		t.Fatalf("inline keyboard = %#v, want single selected-star button", payload.InlineKeyboard)
	}
	if payload.InlineKeyboard[0][0].Text != "4⭐" {
		t.Fatalf("button text = %q, want 4⭐", payload.InlineKeyboard[0][0].Text)
	}
	if payload.InlineKeyboard[0][0].CallbackData != "rating:msg-123:4" {
		t.Fatalf("callback_data = %q, want rating:msg-123:4", payload.InlineKeyboard[0][0].CallbackData)
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

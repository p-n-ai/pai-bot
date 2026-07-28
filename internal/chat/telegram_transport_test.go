// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

type telegramRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f telegramRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestTelegramChannelTransportErrorDoesNotExposeToken(t *testing.T) {
	channel, err := NewTelegramChannel("secret-test-token")
	if err != nil {
		t.Fatal(err)
	}
	channel.client = &http.Client{
		Transport: telegramRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
			return nil, errors.New(request.URL.String())
		}),
	}

	err = channel.SendTyping(t.Context(), "123")

	if err == nil || !strings.Contains(err.Error(), "sending typing indicator request failed") {
		t.Fatalf("SendTyping() error = %v, want redacted operation", err)
	}
	if strings.Contains(err.Error(), "secret-test-token") {
		t.Fatalf("SendTyping() error exposed bot token: %v", err)
	}
}

func TestTelegramChannelSendMessageHonorsCanceledContext(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel, err := NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	channel.baseURL = server.URL

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err = channel.SendMessage(ctx, "123", OutboundMessage{Text: "hello"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendMessage() error = %v, want context.Canceled", err)
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Fatalf("SendMessage() error exposed bot token: %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("requests = %d, want 0 after caller cancellation", got)
	}
}

func TestTelegramChannelSendMessageRetryHonorsCanceledContext(t *testing.T) {
	var requests atomic.Int32
	ctx, cancel := context.WithCancel(t.Context())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			cancel()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel, err := NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	channel.baseURL = server.URL

	err = channel.SendMessage(ctx, "123", OutboundMessage{
		Text:      "hello",
		ParseMode: "Markdown",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendMessage() error = %v, want context.Canceled", err)
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Fatalf("SendMessage() retry error exposed bot token: %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want no retry after caller cancellation", got)
	}
}

func TestTelegramChannelGetUpdatesCancellationDoesNotExposeToken(t *testing.T) {
	channel, err := NewTelegramChannel("secret-test-token")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = channel.getUpdates(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("getUpdates() error = %v, want context.Canceled", err)
	}
	if strings.Contains(err.Error(), "secret-test-token") {
		t.Fatalf("getUpdates() error exposed bot token: %v", err)
	}
}

func TestTelegramChannelSendMessageUsesTopicRoute(t *testing.T) {
	var values url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		values, err = url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":101}}`))
	}))
	defer server.Close()

	channel, err := NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	channel.baseURL = server.URL

	if err := channel.SendMessage(t.Context(), "telegram:-100123:42", OutboundMessage{Text: "topic reply"}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if got := values.Get("chat_id"); got != "-100123" {
		t.Fatalf("chat_id = %q, want -100123", got)
	}
	if got := values.Get("message_thread_id"); got != "42" {
		t.Fatalf("message_thread_id = %q, want 42", got)
	}
}

func TestTelegramChannelSendTypingUsesTopicRoute(t *testing.T) {
	var values url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		values, err = url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	channel, err := NewTelegramChannel("test-token")
	if err != nil {
		t.Fatalf("NewTelegramChannel() error = %v", err)
	}
	channel.baseURL = server.URL

	if err := channel.SendTyping(t.Context(), "telegram:-100123:42"); err != nil {
		t.Fatalf("SendTyping() error = %v", err)
	}
	if got := values.Get("chat_id"); got != "-100123" {
		t.Fatalf("chat_id = %q, want -100123", got)
	}
	if got := values.Get("message_thread_id"); got != "42" {
		t.Fatalf("message_thread_id = %q, want 42", got)
	}
}

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

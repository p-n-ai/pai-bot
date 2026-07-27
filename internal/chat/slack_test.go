// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestSlackChannelWebhookNormalizesSignedMessage(t *testing.T) {
	now := time.Unix(1_725_000_000, 0)
	channel, err := NewSlackChannel("xoxb-test", "signing-secret")
	if err != nil {
		t.Fatalf("NewSlackChannel() error = %v", err)
	}
	channel.now = func() time.Time { return now }

	body := []byte(`{
		"type":"event_callback",
		"event_id":"Ev123",
		"event":{
			"type":"message",
			"user":"U123",
			"channel":"C123",
			"text":"hello from Slack",
			"ts":"1725000000.000100"
		}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/webhook/slack", bytes.NewReader(body))
	signSlackTestRequest(request, body, "signing-secret", now)
	recorder := httptest.NewRecorder()

	var got InboundMessage
	channel.WebhookHandler(func(message InboundMessage) {
		got = message
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got.Channel != "slack" {
		t.Fatalf("Channel = %q, want slack", got.Channel)
	}
	if got.UserID != "U123" {
		t.Fatalf("UserID = %q, want Slack learner U123", got.UserID)
	}
	if got.ThreadID != "slack:C123:1725000000.000100" {
		t.Fatalf("ThreadID = %q, want Slack thread route", got.ThreadID)
	}
	if got.MessageID != "1725000000.000100" {
		t.Fatalf("MessageID = %q, want Slack message timestamp", got.MessageID)
	}
	if got.DeliveryID != "Ev123" {
		t.Fatalf("DeliveryID = %q, want event envelope ID", got.DeliveryID)
	}
	if got.Text != "hello from Slack" {
		t.Fatalf("Text = %q, want normalized Slack text", got.Text)
	}
}

func TestSlackChannelWebhookPreservesThreadRoute(t *testing.T) {
	now := time.Unix(1_725_000_000, 0)
	channel, err := NewSlackChannel("xoxb-test", "signing-secret")
	if err != nil {
		t.Fatalf("NewSlackChannel() error = %v", err)
	}
	channel.now = func() time.Time { return now }

	body := []byte(`{
		"type":"event_callback",
		"event_id":"Ev124",
		"event":{
			"type":"message",
			"user":"U123",
			"channel":"C123",
			"text":"thread reply",
			"ts":"1725000001.000100",
			"thread_ts":"1724999999.000100"
		}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/webhook/slack", bytes.NewReader(body))
	signSlackTestRequest(request, body, "signing-secret", now)
	recorder := httptest.NewRecorder()

	var got InboundMessage
	channel.WebhookHandler(func(message InboundMessage) { got = message }).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got.ThreadID != "slack:C123:1724999999.000100" {
		t.Fatalf("ThreadID = %q, want parent Slack thread route", got.ThreadID)
	}
}

func TestSlackChannelWebhookAnswersSignedURLVerification(t *testing.T) {
	now := time.Unix(1_725_000_000, 0)
	channel, err := NewSlackChannel("xoxb-test", "signing-secret")
	if err != nil {
		t.Fatalf("NewSlackChannel() error = %v", err)
	}
	channel.now = func() time.Time { return now }

	body := []byte(`{"type":"url_verification","challenge":"challenge-value"}`)
	request := httptest.NewRequest(http.MethodPost, "/webhook/slack", bytes.NewReader(body))
	signSlackTestRequest(request, body, "signing-secret", now)
	recorder := httptest.NewRecorder()
	channel.WebhookHandler(func(InboundMessage) {
		t.Fatal("URL verification must not dispatch an inbound message")
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "challenge-value" {
		t.Fatalf("body = %q, want Slack challenge", got)
	}
}

func TestSlackChannelWebhookRejectsInvalidSignature(t *testing.T) {
	channel, err := NewSlackChannel("xoxb-test", "signing-secret")
	if err != nil {
		t.Fatalf("NewSlackChannel() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/webhook/slack", bytes.NewReader([]byte(`{"type":"event_callback"}`)))
	request.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	request.Header.Set("X-Slack-Signature", "v0=bad")
	recorder := httptest.NewRecorder()

	channel.WebhookHandler(func(InboundMessage) {
		t.Fatal("invalid request must not dispatch an inbound message")
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestSlackChannelWebhookRejectsStaleSignature(t *testing.T) {
	now := time.Unix(1_725_000_000, 0)
	channel, err := NewSlackChannel("xoxb-test", "signing-secret")
	if err != nil {
		t.Fatalf("NewSlackChannel() error = %v", err)
	}
	channel.now = func() time.Time { return now }
	body := []byte(`{"type":"event_callback"}`)
	request := httptest.NewRequest(http.MethodPost, "/webhook/slack", bytes.NewReader(body))
	signSlackTestRequest(request, body, "signing-secret", now.Add(-6*time.Minute))
	recorder := httptest.NewRecorder()

	channel.WebhookHandler(func(InboundMessage) {
		t.Fatal("stale request must not dispatch")
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestSlackChannelWebhookIgnoresBotMessage(t *testing.T) {
	now := time.Unix(1_725_000_000, 0)
	channel, err := NewSlackChannel("xoxb-test", "signing-secret")
	if err != nil {
		t.Fatalf("NewSlackChannel() error = %v", err)
	}
	channel.now = func() time.Time { return now }
	body := []byte(`{
		"type":"event_callback",
		"event_id":"EvBot",
		"event":{"type":"message","bot_id":"B123","channel":"C123","text":"bot output","ts":"1725000000.1"}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/webhook/slack", bytes.NewReader(body))
	signSlackTestRequest(request, body, "signing-secret", now)
	recorder := httptest.NewRecorder()

	channel.WebhookHandler(func(InboundMessage) {
		t.Fatal("bot message must not dispatch")
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestSlackChannelSendMessageUsesWebAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat.postMessage" {
			t.Fatalf("path = %q, want /chat.postMessage", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer xoxb-test" {
			t.Fatalf("Authorization = %q, want bearer bot token", got)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["channel"] != "C123" || payload["thread_ts"] != "1725000000.000100" || payload["text"] != "Tutor reply" {
			t.Fatalf("payload = %#v, want Slack channel, thread, and text", payload)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ok":true,"channel":"C123","ts":"1725000001.000100"}`))
	}))
	defer server.Close()

	channel, err := NewSlackChannel("xoxb-test", "signing-secret")
	if err != nil {
		t.Fatalf("NewSlackChannel() error = %v", err)
	}
	channel.baseURL = server.URL

	if err := channel.SendMessage(t.Context(), "slack:C123:1725000000.000100", OutboundMessage{Text: "Tutor reply"}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
}

func signSlackTestRequest(request *http.Request, body []byte, secret string, now time.Time) {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("v0:" + timestamp + ":"))
	_, _ = mac.Write(body)
	request.Header.Set("X-Slack-Request-Timestamp", timestamp)
	request.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(mac.Sum(nil)))
}

var _ Channel = (*SlackChannel)(nil)
var _ WebhookChannel = (*SlackChannel)(nil)

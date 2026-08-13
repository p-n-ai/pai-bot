// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscordChannelWebhookRespondsToSignedPing(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 key: %v", err)
	}
	channel, err := NewDiscordChannel(DiscordConfig{
		BotToken:      "discord-bot-token",
		PublicKey:     hex.EncodeToString(publicKey),
		ApplicationID: "app-123",
	})
	if err != nil {
		t.Fatalf("NewDiscordChannel() error = %v", err)
	}

	body := []byte(`{"type":1}`)
	timestamp := "1725000000"
	request := httptest.NewRequest(http.MethodPost, "/webhook/discord", bytes.NewReader(body))
	request.Header.Set("X-Signature-Timestamp", timestamp)
	request.Header.Set("X-Signature-Ed25519", hex.EncodeToString(ed25519.Sign(privateKey, append([]byte(timestamp), body...))))
	recorder := httptest.NewRecorder()

	channel.WebhookHandler(func(context.Context, InboundMessage) error {
		t.Fatal("Discord ping must not dispatch a tutor message")
		return nil
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Type int `json:"type"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Type != 1 {
		t.Fatalf("response type = %d, want Discord PONG type 1", response.Type)
	}
}

func TestDiscordChannelWebhookRejectsBotTokenForwarding(t *testing.T) {
	channel, err := NewDiscordChannel(DiscordConfig{
		BotToken:      "discord-bot-token",
		PublicKey:     hex.EncodeToString(make([]byte, ed25519.PublicKeySize)),
		ApplicationID: "app-123",
	})
	if err != nil {
		t.Fatalf("NewDiscordChannel() error = %v", err)
	}

	body := []byte(`{
		"type":"GATEWAY_MESSAGE_CREATE",
		"timestamp":1725000000,
		"data":{
			"id":"msg123",
			"channel_id":"channel456",
			"guild_id":"guild789",
			"content":"Hello from Discord",
			"author":{"id":"user123","username":"testuser","global_name":"Test User","bot":false}
		}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/webhook/discord", bytes.NewReader(body))
	request.Header.Set("X-Discord-Gateway-Token", "discord-bot-token")
	recorder := httptest.NewRecorder()

	channel.WebhookHandler(func(context.Context, InboundMessage) error {
		t.Fatal("bot-token forwarding must not dispatch")
		return nil
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestDiscordChannelWebhookRejectsInvalidInteractionSignature(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 key: %v", err)
	}
	channel, err := NewDiscordChannel(DiscordConfig{
		BotToken:      "discord-bot-token",
		PublicKey:     hex.EncodeToString(publicKey),
		ApplicationID: "app-123",
	})
	if err != nil {
		t.Fatalf("NewDiscordChannel() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/webhook/discord", bytes.NewReader([]byte(`{"type":1}`)))
	request.Header.Set("X-Signature-Timestamp", "1725000000")
	request.Header.Set("X-Signature-Ed25519", hex.EncodeToString(make([]byte, ed25519.SignatureSize)))
	recorder := httptest.NewRecorder()

	channel.WebhookHandler(func(context.Context, InboundMessage) error {
		t.Fatal("invalid interaction must not dispatch")
		return nil
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestDiscordChannelSendMessageUsesThreadAndLimit(t *testing.T) {
	var gotPath string
	var gotContent string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		gotContent = payload.Content
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel, err := NewDiscordChannel(DiscordConfig{
		BotToken:      "discord-bot-token",
		PublicKey:     hex.EncodeToString(make([]byte, ed25519.PublicKeySize)),
		ApplicationID: "app-123",
	})
	if err != nil {
		t.Fatalf("NewDiscordChannel() error = %v", err)
	}
	channel.baseURL = server.URL

	if err := channel.SendMessage(t.Context(), "discord:guild789:channel456:thread999", OutboundMessage{
		Text: strings.Repeat("a", 2500),
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if gotPath != "/channels/thread999/messages" {
		t.Fatalf("path = %q, want Discord thread message endpoint", gotPath)
	}
	if len(gotContent) > 2000 || !strings.HasSuffix(gotContent, "...") {
		t.Fatalf("content length/suffix = %d/%q, want <=2000 ending in ellipsis", len(gotContent), gotContent[len(gotContent)-3:])
	}
}

var _ Channel = (*DiscordChannel)(nil)
var _ WebhookChannel = (*DiscordChannel)(nil)

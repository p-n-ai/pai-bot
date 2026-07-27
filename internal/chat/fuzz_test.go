// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const maxChatFuzzInput = 64 << 10

func FuzzSlackWebhook(f *testing.F) {
	for _, seed := range [][]byte{
		{},
		[]byte(`{`),
		[]byte(`{"type":"url_verification","challenge":"seed"}`),
		[]byte(`{"type":"event_callback","event_id":"Ev1","event":{"type":"message","user":"U1","channel":"C1","text":"hello","ts":"1.0"}}`),
		[]byte(`{"type":"event_callback","event":{"type":"message","bot_id":"B1","channel":"C1"}}`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > maxChatFuzzInput {
			return
		}
		now := time.Unix(1_725_000_000, 0)
		channel, err := NewSlackChannel("xoxb-fuzz", "fuzz-signing-secret")
		if err != nil {
			t.Fatal(err)
		}
		channel.now = func() time.Time { return now }

		request := httptest.NewRequest(http.MethodPost, "/webhook/slack", bytes.NewReader(body))
		signSlackTestRequest(request, body, "fuzz-signing-secret", now)
		recorder := httptest.NewRecorder()
		channel.WebhookHandler(func(message InboundMessage) {
			if message.Channel != "slack" ||
				message.UserID == "" ||
				!strings.HasPrefix(message.ThreadID, "slack:") {
				t.Fatalf("invalid normalized Slack message: %#v", message)
			}
		}).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK && recorder.Code != http.StatusBadRequest {
			t.Fatalf("signed Slack webhook status = %d", recorder.Code)
		}
	})
}

func FuzzDiscordInteractionWebhook(f *testing.F) {
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	for _, body := range [][]byte{
		{},
		[]byte(`{`),
		[]byte(`{"type":1}`),
		[]byte(`{"type":2,"data":{"name":"seed"}}`),
	} {
		f.Add(body)
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > maxChatFuzzInput {
			return
		}
		channel, err := NewDiscordChannel(DiscordConfig{
			BotToken:      "discord-fuzz-token",
			PublicKey:     hex.EncodeToString(publicKey),
			ApplicationID: "fuzz-app",
		})
		if err != nil {
			t.Fatal(err)
		}

		const timestamp = "1725000000"
		signed := make([]byte, 0, len(timestamp)+len(body))
		signed = append(signed, timestamp...)
		signed = append(signed, body...)
		request := httptest.NewRequest(http.MethodPost, "/webhook/discord", bytes.NewReader(body))
		request.Header.Set("X-Signature-Timestamp", timestamp)
		request.Header.Set("X-Signature-Ed25519", hex.EncodeToString(ed25519.Sign(privateKey, signed)))
		recorder := httptest.NewRecorder()
		channel.WebhookHandler(func(message InboundMessage) {
			t.Fatalf("interaction webhook dispatched an inbound message: %#v", message)
		}).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK && recorder.Code != http.StatusBadRequest {
			t.Fatalf("signed Discord interaction status = %d", recorder.Code)
		}
	})
}

func FuzzTeamsWebhook(f *testing.F) {
	for _, seed := range [][]byte{
		{},
		[]byte(`{`),
		[]byte(`{"type":"typing"}`),
		[]byte(`{"type":"message","id":"m1","text":"hello","from":{"id":"u1"},"conversation":{"id":"c1"},"serviceUrl":"https://smba.trafficmanager.net/teams/"}`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > maxChatFuzzInput {
			return
		}
		channel, err := NewTeamsChannel(TeamsConfig{
			TokenValidator: teamsTokenValidatorFunc(func(context.Context, string, TeamsAuthenticationContext) error {
				return nil
			}),
			TokenProvider: teamsTokenProviderFunc(func(context.Context) (string, error) {
				return "connector-token", nil
			}),
		})
		if err != nil {
			t.Fatal(err)
		}

		request := httptest.NewRequest(http.MethodPost, "/webhook/teams", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer fuzz-token")
		recorder := httptest.NewRecorder()
		channel.WebhookHandler(func(message InboundMessage) {
			if message.Channel != "teams" ||
				message.UserID == "" ||
				!strings.HasPrefix(message.ThreadID, "teams:") {
				t.Fatalf("invalid normalized Teams message: %#v", message)
			}
		}).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK && recorder.Code != http.StatusBadRequest {
			t.Fatalf("authenticated Teams webhook status = %d", recorder.Code)
		}
	})
}

func FuzzTeamsContinuation(f *testing.F) {
	valid := "teams:" +
		base64.RawURLEncoding.EncodeToString([]byte("conversation-1")) +
		":" +
		base64.RawURLEncoding.EncodeToString([]byte("https://smba.trafficmanager.net/teams/"))
	for _, seed := range []string{
		"",
		"teams:",
		"teams:not-base64:not-base64",
		valid,
		"teams:" + base64.RawURLEncoding.EncodeToString([]byte("conversation-1")) + ":" +
			base64.RawURLEncoding.EncodeToString([]byte("https://127.0.0.1/")),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, route string) {
		if len(route) > maxTeamsThreadIDSize+1 {
			return
		}
		continuation, err := parseTeamsContinuation(route)
		if err != nil {
			return
		}
		if continuation.conversationID == "" ||
			continuation.serviceURL == nil ||
			continuation.serviceURL.Scheme != "https" ||
			!trustedTeamsConnectorHost(continuation.serviceURL.Hostname()) {
			t.Fatalf("unsafe accepted Teams continuation: %#v", continuation)
		}
		activityURL := teamsActivityURL(continuation)
		if !strings.HasPrefix(activityURL, continuation.serviceURL.String()) {
			t.Fatalf("activity URL escaped service URL: %q from %q", activityURL, continuation.serviceURL)
		}
	})
}

func FuzzTeamsJWTParser(f *testing.F) {
	for _, seed := range []string{
		"",
		".",
		"..",
		"not-base64.not-base64.not-base64",
		base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"seed"}`)) + "." +
			base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"issuer","aud":"app","exp":1}`)) + "." +
			base64.RawURLEncoding.EncodeToString([]byte("signature")),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, token string) {
		if len(token) > maxChatFuzzInput {
			return
		}
		_, _, signed, _, err := parseTeamsJWT(token)
		if err != nil {
			return
		}
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			t.Fatalf("parser accepted token with %d segments", len(parts))
		}
		if string(signed) != parts[0]+"."+parts[1] {
			t.Fatalf("signed bytes = %q, want original header and claims", signed)
		}
	})
}

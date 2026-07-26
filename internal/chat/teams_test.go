// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type teamsTokenValidatorFunc func(context.Context, string, string) error

func (f teamsTokenValidatorFunc) Validate(ctx context.Context, token, serviceURL string) error {
	return f(ctx, token, serviceURL)
}

func TestTeamsChannelWebhookNormalizesAuthenticatedMessageActivity(t *testing.T) {
	channel, err := NewTeamsChannel(TeamsConfig{
		TokenValidator: teamsTokenValidatorFunc(func(_ context.Context, token, serviceURL string) error {
			if token != "valid-token" {
				return errors.New("invalid token")
			}
			if serviceURL != "https://smba.trafficmanager.net/teams/" {
				return errors.New("service URL mismatch")
			}
			return nil
		}),
	})
	if err != nil {
		t.Fatalf("NewTeamsChannel() error = %v", err)
	}

	body := []byte(`{
		"type":"message",
		"id":"msg-100",
		"text":"  Hello from Teams  ",
		"from":{"id":"user-1","name":"Alice","role":"user"},
		"conversation":{"id":"19:abc@thread.tacv2"},
		"serviceUrl":"https://smba.trafficmanager.net/teams/",
		"timestamp":"2024-01-01T00:00:00Z"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/webhook/teams", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-token")
	recorder := httptest.NewRecorder()

	var got InboundMessage
	channel.WebhookHandler(func(message InboundMessage) { got = message }).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got.Channel != "teams" || got.UserID != "user-1" {
		t.Fatalf("identity = channel:%q user:%q, want teams user-1", got.Channel, got.UserID)
	}
	const wantThread = "teams:MTk6YWJjQHRocmVhZC50YWN2Mg:aHR0cHM6Ly9zbWJhLnRyYWZmaWNtYW5hZ2VyLm5ldC90ZWFtcy8"
	if got.ThreadID != wantThread {
		t.Fatalf("ThreadID = %q, want encoded Teams continuation", got.ThreadID)
	}
	if got.MessageID != "msg-100" || got.DeliveryID != "msg-100" || got.Text != "Hello from Teams" {
		t.Fatalf(
			"message = id:%q delivery:%q text:%q, want normalized Teams message",
			got.MessageID,
			got.DeliveryID,
			got.Text,
		)
	}
	if got.Username != "Alice" {
		t.Fatalf("Username = %q, want Alice", got.Username)
	}
}

func TestTeamsChannelWebhookRejectsUnauthenticatedActivity(t *testing.T) {
	channel, err := NewTeamsChannel(TeamsConfig{
		TokenValidator: teamsTokenValidatorFunc(func(context.Context, string, string) error {
			return errors.New("invalid token")
		}),
	})
	if err != nil {
		t.Fatalf("NewTeamsChannel() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/webhook/teams", bytes.NewReader([]byte(`{"type":"message"}`)))
	recorder := httptest.NewRecorder()

	channel.WebhookHandler(func(InboundMessage) {
		t.Fatal("unauthenticated activity must not dispatch")
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

var _ Channel = (*TeamsChannel)(nil)
var _ WebhookChannel = (*TeamsChannel)(nil)

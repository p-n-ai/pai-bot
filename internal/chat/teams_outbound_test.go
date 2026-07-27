// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type teamsTokenProviderFunc func(context.Context) (string, error)

func (f teamsTokenProviderFunc) Token(ctx context.Context) (string, error) {
	return f(ctx)
}

type teamsRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f teamsRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestTeamsChannelSendMessagePostsTextActivityToContinuation(t *testing.T) {
	requests := make(chan *http.Request, 1)
	channel := newTeamsOutboundTestChannel(t, teamsTokenProviderFunc(func(context.Context) (string, error) {
		return "connector-token", nil
	}), teamsRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests <- request
		return teamsHTTPResponse(http.StatusCreated, `{"id":"activity-1"}`), nil
	}))

	threadID := teamsTestThreadID("19:abc/part@thread.tacv2", "https://smba.trafficmanager.net/teams/")
	if err := channel.SendMessage(context.Background(), threadID, OutboundMessage{Text: "Hello from PaiBot"}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	request := <-requests
	if request.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST", request.Method)
	}
	const wantURL = "https://smba.trafficmanager.net/teams/v3/conversations/19:abc%2Fpart@thread.tacv2/activities"
	if request.URL.String() != wantURL {
		t.Fatalf("URL = %q, want %q", request.URL, wantURL)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer connector-token" {
		t.Fatalf("Authorization = %q, want bearer connector token", got)
	}
	if got := request.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var activity struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(request.Body).Decode(&activity); err != nil {
		t.Fatalf("decode activity: %v", err)
	}
	if activity.Type != "message" || activity.Text != "Hello from PaiBot" {
		t.Fatalf("activity = %#v, want message text activity", activity)
	}
}

func TestTeamsChannelSendTypingPostsTypingActivityToContinuation(t *testing.T) {
	requests := make(chan *http.Request, 1)
	channel := newTeamsOutboundTestChannel(t, teamsTokenProviderFunc(func(context.Context) (string, error) {
		return "connector-token", nil
	}), teamsRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests <- request
		return teamsHTTPResponse(http.StatusOK, `{}`), nil
	}))

	threadID := teamsTestThreadID("conversation-1", "https://smba.infra.gcc.teams.microsoft.com/teams")
	if err := channel.SendTyping(context.Background(), threadID); err != nil {
		t.Fatalf("SendTyping() error = %v", err)
	}

	request := <-requests
	const wantURL = "https://smba.infra.gcc.teams.microsoft.com/teams/v3/conversations/conversation-1/activities"
	if request.URL.String() != wantURL {
		t.Fatalf("URL = %q, want %q", request.URL, wantURL)
	}
	var activity struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(request.Body).Decode(&activity); err != nil {
		t.Fatalf("decode activity: %v", err)
	}
	if activity.Type != "typing" {
		t.Fatalf("activity type = %q, want typing", activity.Type)
	}
}

func TestTeamsChannelOutboundRejectsUntrustedContinuationServiceURL(t *testing.T) {
	channel := newTeamsOutboundTestChannel(t, teamsTokenProviderFunc(func(context.Context) (string, error) {
		t.Fatal("token provider must not be called for an untrusted service URL")
		return "", nil
	}), teamsRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("HTTP transport must not be called for an untrusted service URL")
		return nil, nil
	}))

	for _, serviceURL := range []string{
		"http://smba.trafficmanager.net/teams/",
		"https://smba.trafficmanager.net.evil.example/teams/",
		"https://127.0.0.1/teams/",
		"https://user@smba.trafficmanager.net/teams/",
		"https://smba.trafficmanager.net:8443/teams/",
	} {
		t.Run(serviceURL, func(t *testing.T) {
			threadID := teamsTestThreadID("conversation-1", serviceURL)
			if err := channel.SendTyping(context.Background(), threadID); err == nil {
				t.Fatal("SendTyping() error = nil, want rejected continuation")
			}
		})
	}
}

func TestTeamsChannelOutboundRequiresTokenProvider(t *testing.T) {
	_, err := NewTeamsChannel(TeamsConfig{
		TokenValidator: teamsTokenValidatorFunc(func(context.Context, string, TeamsAuthenticationContext) error { return nil }),
	})
	if err == nil || !strings.Contains(err.Error(), "token provider") {
		t.Fatalf("NewTeamsChannel() error = %v, want token provider error", err)
	}
}

func TestTeamsChannelOutboundReportsConnectorFailure(t *testing.T) {
	channel := newTeamsOutboundTestChannel(t, teamsTokenProviderFunc(func(context.Context) (string, error) {
		return "connector-token", nil
	}), teamsRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		return teamsHTTPResponse(http.StatusUnauthorized, `{"error":"invalid token"}`), nil
	}))

	err := channel.SendMessage(
		context.Background(),
		teamsTestThreadID("conversation-1", "https://smba.trafficmanager.net/teams/"),
		OutboundMessage{Text: "hello"},
	)
	if err == nil || !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("SendMessage() error = %v, want Connector status", err)
	}
}

func TestTeamsChannelOutboundReportsTokenProviderFailure(t *testing.T) {
	tokenErr := errors.New("token unavailable")
	channel := newTeamsOutboundTestChannel(t, teamsTokenProviderFunc(func(context.Context) (string, error) {
		return "", tokenErr
	}), teamsRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("HTTP transport must not be called after token failure")
		return nil, nil
	}))

	err := channel.SendTyping(
		context.Background(),
		teamsTestThreadID("conversation-1", "https://smba.trafficmanager.net/teams/"),
	)
	if !errors.Is(err, tokenErr) {
		t.Fatalf("SendTyping() error = %v, want wrapped token provider error", err)
	}
}

func newTeamsOutboundTestChannel(
	t *testing.T,
	tokenProvider TeamsTokenProvider,
	transport http.RoundTripper,
) *TeamsChannel {
	t.Helper()
	channel, err := NewTeamsChannel(TeamsConfig{
		TokenValidator: teamsTokenValidatorFunc(func(context.Context, string, TeamsAuthenticationContext) error { return nil }),
		TokenProvider:  tokenProvider,
	})
	if err != nil {
		t.Fatalf("NewTeamsChannel() error = %v", err)
	}
	channel.httpClient.Transport = transport
	return channel
}

func teamsTestThreadID(conversationID, serviceURL string) string {
	encode := base64.RawURLEncoding.EncodeToString
	return "teams:" + encode([]byte(conversationID)) + ":" + encode([]byte(serviceURL))
}

func teamsHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

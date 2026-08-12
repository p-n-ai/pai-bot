// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxTeamsWebhookBody  = 1 << 20
	maxTeamsThreadIDSize = 16 << 10
)

var errTeamsChannelNotEndorsed = errors.New("teams signing key is not endorsed for activity channel")

// TeamsTokenValidator authenticates Bot Framework bearer tokens.
type TeamsTokenValidator interface {
	Validate(context.Context, string, TeamsAuthenticationContext) error
}

// TeamsAuthenticationContext binds a Bot Framework token to the activity route
// and channel that the signing key is authorized to represent.
type TeamsAuthenticationContext struct {
	ServiceURL string
	ChannelID  string
}

// TeamsTokenProvider supplies a Bot Connector bearer token for outbound calls.
type TeamsTokenProvider interface {
	Token(context.Context) (string, error)
}

// TeamsConfig contains the credentials and authentication boundary for Teams.
type TeamsConfig struct {
	TokenValidator TeamsTokenValidator
	TokenProvider  TeamsTokenProvider
}

// TeamsChannel implements Microsoft Teams Bot Framework webhook ingress.
type TeamsChannel struct {
	tokenValidator TeamsTokenValidator
	tokenProvider  TeamsTokenProvider
	httpClient     *http.Client
}

// NewTeamsChannel creates a Microsoft Teams channel adapter.
func NewTeamsChannel(config TeamsConfig) (*TeamsChannel, error) {
	if config.TokenValidator == nil {
		return nil, fmt.Errorf("teams token validator is required")
	}
	if config.TokenProvider == nil {
		return nil, fmt.Errorf("teams token provider is required")
	}
	return &TeamsChannel{
		tokenValidator: config.TokenValidator,
		tokenProvider:  config.TokenProvider,
		httpClient:     newTeamsHTTPClient(),
	}, nil
}

// SendMessage posts a text activity to a Teams continuation route.
func (t *TeamsChannel) SendMessage(ctx context.Context, destination string, message OutboundMessage) error {
	activity := teamsOutboundActivity{
		Type: "message",
		Text: message.Text,
	}
	if strings.EqualFold(message.ParseMode, "Markdown") {
		activity.TextFormat = "markdown"
	}
	return t.sendActivity(ctx, destination, activity)
}

// SendTyping posts a short-lived typing activity to a Teams continuation route.
func (t *TeamsChannel) SendTyping(ctx context.Context, destination string) error {
	return t.sendActivity(ctx, destination, teamsOutboundActivity{Type: "typing"})
}

// Start is a no-op because Teams activities arrive through the webhook.
func (t *TeamsChannel) Start(context.Context, func(InboundMessage)) error { return nil }

// Stop is a no-op for webhook-mode Teams channels.
func (t *TeamsChannel) Stop() error { return nil }

func (t *TeamsChannel) sendActivity(
	ctx context.Context,
	destination string,
	activity teamsOutboundActivity,
) error {
	continuation, err := parseTeamsContinuation(destination)
	if err != nil {
		return err
	}
	token, err := t.tokenProvider.Token(ctx)
	if err != nil {
		return fmt.Errorf("get Teams Connector token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("teams token provider returned an empty token")
	}

	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("encode Teams activity: %w", err)
	}
	endpoint := teamsActivityURL(continuation)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Teams Connector request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := t.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("post Teams activity: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("teams connector returned status %d", response.StatusCode)
	}
	return nil
}

type teamsOutboundActivity struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	TextFormat string `json:"textFormat,omitempty"`
}

type teamsContinuation struct {
	conversationID string
	serviceURL     *url.URL
}

func teamsActivityURL(continuation teamsContinuation) string {
	endpoint := *continuation.serviceURL
	basePath := strings.TrimRight(endpoint.Path, "/")
	escapedBasePath := strings.TrimRight(endpoint.EscapedPath(), "/")
	endpoint.Path = basePath + "/v3/conversations/" + continuation.conversationID + "/activities"
	endpoint.RawPath = escapedBasePath +
		"/v3/conversations/" +
		url.PathEscape(continuation.conversationID) +
		"/activities"
	return endpoint.String()
}

func parseTeamsContinuation(threadID string) (teamsContinuation, error) {
	if len(threadID) == 0 || len(threadID) > maxTeamsThreadIDSize {
		return teamsContinuation{}, fmt.Errorf("invalid Teams continuation")
	}
	parts := strings.Split(threadID, ":")
	if len(parts) != 3 || parts[0] != "teams" {
		return teamsContinuation{}, fmt.Errorf("invalid Teams continuation")
	}

	decode := base64.RawURLEncoding.DecodeString
	conversationBytes, err := decode(parts[1])
	if err != nil || len(conversationBytes) == 0 {
		return teamsContinuation{}, fmt.Errorf("invalid Teams continuation conversation")
	}
	serviceURLBytes, err := decode(parts[2])
	if err != nil {
		return teamsContinuation{}, fmt.Errorf("invalid Teams continuation service URL")
	}
	serviceURL, err := parseTeamsServiceURL(string(serviceURLBytes))
	if err != nil {
		return teamsContinuation{}, err
	}
	return teamsContinuation{
		conversationID: string(conversationBytes),
		serviceURL:     serviceURL,
	}, nil
}

func parseTeamsServiceURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil ||
		parsed.Scheme != "https" ||
		parsed.User != nil ||
		parsed.Port() != "" ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		!trustedTeamsConnectorHost(parsed.Hostname()) {
		return nil, fmt.Errorf("untrusted Teams service URL")
	}
	if strings.Contains(parsed.EscapedPath(), "%") ||
		hasTeamsPathTraversal(parsed.Path) {
		return nil, fmt.Errorf("untrusted Teams service URL")
	}
	return parsed, nil
}

func trustedTeamsConnectorHost(host string) bool {
	switch strings.ToLower(host) {
	case "smba.trafficmanager.net",
		"smba.infra.gcc.teams.microsoft.com",
		"smba.infra.gov.teams.microsoft.us",
		"smba.infra.dod.teams.microsoft.us":
		return true
	default:
		return false
	}
}

func hasTeamsPathTraversal(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func newTeamsHTTPClient() *http.Client {
	client := *http.DefaultClient
	client.Timeout = 30 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

// WebhookHandler authenticates and normalizes Bot Framework message activities.
func (t *TeamsChannel) WebhookHandler(handler InboundWebhookHandler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token, ok := teamsBearerToken(request.Header.Get("Authorization"))
		if !ok {
			http.Error(response, "unauthorized", http.StatusUnauthorized)
			return
		}

		var activity teamsActivity
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, maxTeamsWebhookBody))
		if err := decoder.Decode(&activity); err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		if err := ensureJSONEnd(decoder); err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		if err := t.tokenValidator.Validate(request.Context(), token, TeamsAuthenticationContext{
			ServiceURL: activity.ServiceURL,
			ChannelID:  activity.ChannelID,
		}); err != nil {
			if errors.Is(err, errTeamsChannelNotEndorsed) {
				http.Error(response, "forbidden", http.StatusForbidden)
			} else {
				http.Error(response, "unauthorized", http.StatusUnauthorized)
			}
			return
		}
		if activity.Type != "message" {
			response.WriteHeader(http.StatusOK)
			return
		}

		message, ok := activity.inboundMessage()
		if !ok {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		if handler != nil {
			if err := handler(request.Context(), message); err != nil {
				http.Error(response, "temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		response.WriteHeader(http.StatusOK)
	})
}

func teamsBearerToken(authorization string) (string, bool) {
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return fmt.Errorf("multiple JSON values")
	}
	return err
}

type teamsActivity struct {
	Type         string            `json:"type"`
	ID           string            `json:"id"`
	Text         string            `json:"text"`
	From         teamsActivityFrom `json:"from"`
	Conversation teamsConversation `json:"conversation"`
	ChannelID    string            `json:"channelId"`
	ServiceURL   string            `json:"serviceUrl"`
}

type teamsActivityFrom struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type teamsConversation struct {
	ID string `json:"id"`
}

func (a teamsActivity) inboundMessage() (InboundMessage, bool) {
	messageID := strings.TrimSpace(a.ID)
	text := strings.TrimSpace(a.Text)
	authorID := strings.TrimSpace(a.From.ID)
	conversationID := strings.TrimSpace(a.Conversation.ID)
	serviceURL := strings.TrimSpace(a.ServiceURL)
	channelID := strings.TrimSpace(a.ChannelID)
	if messageID == "" || text == "" || authorID == "" || conversationID == "" || channelID == "" || !validTeamsServiceURL(serviceURL) {
		return InboundMessage{}, false
	}

	encode := base64.RawURLEncoding.EncodeToString
	return InboundMessage{
		Channel:    "teams",
		UserID:     authorID,
		ExternalID: authorID,
		ThreadID:   "teams:" + encode([]byte(conversationID)) + ":" + encode([]byte(serviceURL)),
		MessageID:  messageID,
		DeliveryID: messageID,
		Text:       text,
		Username:   strings.TrimSpace(a.From.Name),
	}, true
}

func validTeamsServiceURL(raw string) bool {
	_, err := parseTeamsServiceURL(raw)
	return err == nil
}

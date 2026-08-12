// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSlackBaseURL  = "https://slack.com/api"
	slackSignatureMaxAge = 5 * time.Minute
	maxSlackWebhookBody  = 1 << 20
)

// SlackChannel implements Slack Events API ingress and Web API delivery.
type SlackChannel struct {
	botToken      string
	signingSecret string
	baseURL       string
	client        *http.Client
	now           func() time.Time
}

// NewSlackChannel creates a Slack channel adapter.
func NewSlackChannel(botToken, signingSecret string) (*SlackChannel, error) {
	if strings.TrimSpace(botToken) == "" {
		return nil, fmt.Errorf("slack bot token is required")
	}
	if strings.TrimSpace(signingSecret) == "" {
		return nil, fmt.Errorf("slack signing secret is required")
	}
	return &SlackChannel{
		botToken:      botToken,
		signingSecret: signingSecret,
		baseURL:       defaultSlackBaseURL,
		client:        &http.Client{Timeout: 30 * time.Second},
		now:           time.Now,
	}, nil
}

// SendMessage posts a text message to a Slack conversation.
func (s *SlackChannel) SendMessage(ctx context.Context, destinationID string, msg OutboundMessage) error {
	conversationID, threadTimestamp := decodeSlackThreadID(destinationID)
	body := map[string]string{
		"channel": conversationID,
		"text":    msg.Text,
	}
	if threadTimestamp != "" {
		body["thread_ts"] = threadTimestamp
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal Slack message: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat.postMessage", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create Slack message request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+s.botToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Slack message: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxSlackWebhookBody)).Decode(&result); err != nil {
		return fmt.Errorf("decode Slack message response: %w", err)
	}
	if response.StatusCode >= http.StatusBadRequest || !result.OK {
		return fmt.Errorf("slack chat.postMessage failed: %s", result.Error)
	}
	return nil
}

// SendTyping is a no-op because Slack does not expose a typing Web API method.
func (s *SlackChannel) SendTyping(context.Context, string) error { return nil }

// Start is a no-op because Slack messages arrive through the Events API webhook.
func (s *SlackChannel) Start(context.Context, func(InboundMessage)) error { return nil }

// Stop is a no-op for webhook-mode Slack channels.
func (s *SlackChannel) Stop() error { return nil }

// WebhookHandler verifies and normalizes Slack Events API requests.
func (s *SlackChannel) WebhookHandler(handler InboundWebhookHandler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, maxSlackWebhookBody))
		if err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		if !s.validSignature(request.Header, body) {
			http.Error(response, "invalid signature", http.StatusUnauthorized)
			return
		}

		var envelope slackEventEnvelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		if envelope.Type == "url_verification" {
			response.Header().Set("Content-Type", "text/plain; charset=utf-8")
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write([]byte(envelope.Challenge))
			return
		}

		if envelope.Type != "event_callback" ||
			envelope.Event.Type != "message" ||
			envelope.Event.Subtype != "" ||
			envelope.Event.BotID != "" ||
			envelope.Event.User == "" ||
			envelope.Event.Channel == "" {
			response.WriteHeader(http.StatusOK)
			return
		}
		message := InboundMessage{
			Channel:    "slack",
			UserID:     envelope.Event.User,
			ExternalID: envelope.Event.User,
			ThreadID:   slackThreadID(envelope.Event.Channel, envelope.Event.ThreadTS, envelope.Event.TS),
			MessageID:  envelope.Event.TS,
			DeliveryID: envelope.EventID,
			Text:       envelope.Event.Text,
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

func (s *SlackChannel) validSignature(headers http.Header, body []byte) bool {
	timestampText := headers.Get("X-Slack-Request-Timestamp")
	signatureText := headers.Get("X-Slack-Signature")
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil || s.now().Sub(time.Unix(timestamp, 0)).Abs() > slackSignatureMaxAge {
		return false
	}
	if !strings.HasPrefix(signatureText, "v0=") {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signatureText, "v0="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(s.signingSecret))
	_, _ = mac.Write([]byte("v0:" + timestampText + ":"))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

type slackEventEnvelope struct {
	Type      string     `json:"type"`
	EventID   string     `json:"event_id"`
	Challenge string     `json:"challenge"`
	Event     slackEvent `json:"event"`
}

type slackEvent struct {
	Type     string `json:"type"`
	Subtype  string `json:"subtype"`
	User     string `json:"user"`
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	TS       string `json:"ts"`
	ThreadTS string `json:"thread_ts"`
	BotID    string `json:"bot_id"`
}

func slackThreadID(channelID, threadTimestamp, messageTimestamp string) string {
	if threadTimestamp == "" {
		threadTimestamp = messageTimestamp
	}
	return "slack:" + channelID + ":" + threadTimestamp
}

func decodeSlackThreadID(value string) (string, string) {
	parts := strings.Split(value, ":")
	if len(parts) == 3 && parts[0] == "slack" && parts[1] != "" {
		return parts[1], parts[2]
	}
	return value, ""
}

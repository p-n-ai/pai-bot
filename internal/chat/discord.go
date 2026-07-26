// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultDiscordBaseURL = "https://discord.com/api/v10"

// DiscordConfig contains the credentials required by the Discord adapter.
type DiscordConfig struct {
	BotToken      string
	PublicKey     string
	ApplicationID string
}

// DiscordChannel implements Discord interaction ingress, direct Gateway event
// ingress, and REST API delivery.
type DiscordChannel struct {
	botToken   string
	publicKey  ed25519.PublicKey
	baseURL    string
	gatewayURL string
	client     *http.Client

	lifecycleMu             sync.Mutex
	runtime                 *discordGatewayRuntime
	runtimeErr              error
	terminalRuntimeErr      error
	gatewayHandshakeTimeout time.Duration
	gatewayReconnectWait    func(context.Context, int) error
}

// NewDiscordChannel creates a Discord channel adapter.
func NewDiscordChannel(config DiscordConfig) (*DiscordChannel, error) {
	botToken := strings.TrimSpace(config.BotToken)
	if botToken == "" {
		return nil, fmt.Errorf("discord bot token is required")
	}
	if strings.TrimSpace(config.ApplicationID) == "" {
		return nil, fmt.Errorf("discord application ID is required")
	}
	publicKey, err := hex.DecodeString(strings.TrimSpace(config.PublicKey))
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("discord public key must be a 32-byte hexadecimal Ed25519 key")
	}

	return &DiscordChannel{
		botToken:                botToken,
		publicKey:               ed25519.PublicKey(publicKey),
		baseURL:                 defaultDiscordBaseURL,
		gatewayURL:              defaultDiscordGateway,
		client:                  &http.Client{Timeout: 30 * time.Second},
		gatewayHandshakeTimeout: defaultDiscordGatewayHandshakeTimeout,
		gatewayReconnectWait:    waitDiscordGatewayReconnect,
	}, nil
}

// SendMessage posts a text message to a Discord channel or thread.
func (d *DiscordChannel) SendMessage(ctx context.Context, destination string, message OutboundMessage) error {
	channelID, err := discordChannelID(destination)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(struct {
		Content string `json:"content"`
	}{Content: truncateDiscordContent(message.Text)})
	if err != nil {
		return fmt.Errorf("marshal Discord message: %w", err)
	}
	return d.doREST(ctx, http.MethodPost, "/channels/"+url.PathEscape(channelID)+"/messages", payload)
}

// SendTyping starts Discord's short-lived typing indicator.
func (d *DiscordChannel) SendTyping(ctx context.Context, destination string) error {
	channelID, err := discordChannelID(destination)
	if err != nil {
		return err
	}
	return d.doREST(ctx, http.MethodPost, "/channels/"+url.PathEscape(channelID)+"/typing", nil)
}

func (d *DiscordChannel) doREST(ctx context.Context, method, path string, body []byte) error {
	request, err := http.NewRequestWithContext(ctx, method, d.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Discord API request: %w", err)
	}
	request.Header.Set("Authorization", "Bot "+d.botToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("call Discord API: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		if text := strings.TrimSpace(string(detail)); text != "" {
			return fmt.Errorf("discord API returned %s: %s", response.Status, text)
		}
		return fmt.Errorf("discord API returned %s", response.Status)
	}
	return nil
}

func truncateDiscordContent(content string) string {
	const maxRunes = 2000
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes-3]) + "..."
}

func discordChannelID(destination string) (string, error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return "", fmt.Errorf("discord destination is required")
	}
	if !strings.HasPrefix(destination, "discord:") {
		if strings.Contains(destination, ":") {
			return "", fmt.Errorf("invalid Discord destination %q", destination)
		}
		return destination, nil
	}
	parts := strings.Split(destination, ":")
	if len(parts) != 3 && len(parts) != 4 {
		return "", fmt.Errorf("invalid Discord thread ID %q", destination)
	}
	for _, part := range parts[1:] {
		if part == "" {
			return "", fmt.Errorf("invalid Discord thread ID %q", destination)
		}
	}
	return parts[len(parts)-1], nil
}

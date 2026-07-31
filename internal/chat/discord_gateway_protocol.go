// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
)

func (d *DiscordChannel) dispatchGatewayMessage(message discordGatewayData, handler func(InboundMessage)) {
	if message.ID == "" ||
		message.ChannelID == "" ||
		message.Author.ID == "" ||
		message.Author.Bot {
		return
	}

	guildID := message.GuildID
	if guildID == "" {
		guildID = "@me"
	}
	firstName := message.Author.GlobalName
	if firstName == "" {
		firstName = message.Author.Username
	}
	handler(InboundMessage{
		Channel:    "discord",
		UserID:     message.Author.ID,
		ExternalID: message.Author.ID,
		ThreadID:   "discord:" + guildID + ":" + message.ChannelID,
		MessageID:  message.ID,
		DeliveryID: message.ID,
		Text:       message.Content,
		Username:   message.Author.Username,
		FirstName:  firstName,
	})
}

func readDiscordGatewayPayload(ctx context.Context, connection *websocket.Conn) (discordGatewayPayload, error) {
	_, data, err := connection.Read(ctx)
	if err != nil {
		return discordGatewayPayload{}, err
	}
	var payload discordGatewayPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return discordGatewayPayload{}, fmt.Errorf("decode Discord Gateway payload: %w", err)
	}
	return payload, nil
}

func writeDiscordGatewayPayload(ctx context.Context, connection *websocket.Conn, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Discord Gateway payload: %w", err)
	}
	if err := connection.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("write Discord Gateway payload: %w", err)
	}
	return nil
}

func sendDiscordHeartbeat(ctx context.Context, connection *websocket.Conn, sequence *int64) error {
	return writeDiscordGatewayPayload(ctx, connection, struct {
		Op   int    `json:"op"`
		Data *int64 `json:"d"`
	}{
		Op:   1,
		Data: sequence,
	})
}

type discordGatewayPayload struct {
	Op       int             `json:"op"`
	Data     json.RawMessage `json:"d"`
	Sequence *int64          `json:"s"`
	Type     string          `json:"t"`
}

type discordGatewayIdentify struct {
	Token      string                   `json:"token"`
	Intents    int                      `json:"intents"`
	Properties discordGatewayProperties `json:"properties"`
}

type discordGatewayResume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

type discordGatewayProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type discordGatewayReady struct {
	SessionID        string `json:"session_id"`
	ResumeGatewayURL string `json:"resume_gateway_url"`
}

type discordGatewayRead struct {
	payload discordGatewayPayload
	err     error
}

type discordGatewayData struct {
	ID        string        `json:"id"`
	ChannelID string        `json:"channel_id"`
	GuildID   string        `json:"guild_id"`
	Content   string        `json:"content"`
	Author    discordAuthor `json:"author"`
}

type discordAuthor struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Bot        bool   `json:"bot"`
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestDiscordChannelGatewayReceivesMessagesAndMaintainsSession(t *testing.T) {
	identify := make(chan discordGatewayPayload, 1)
	heartbeat := make(chan discordGatewayPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(response, request, nil)
		if err != nil {
			t.Errorf("accept Gateway connection: %v", err)
			return
		}
		defer func() { _ = connection.CloseNow() }()

		if err := writeDiscordTestPayload(request.Context(), connection, map[string]any{
			"op": 10,
			"d":  map[string]any{"heartbeat_interval": 10},
		}); err != nil {
			t.Errorf("write HELLO: %v", err)
			return
		}

		var gotIdentify discordGatewayPayload
		if err := readDiscordTestPayload(request.Context(), connection, &gotIdentify); err != nil {
			t.Errorf("read IDENTIFY: %v", err)
			return
		}
		identify <- gotIdentify

		for _, event := range []map[string]any{
			{
				"op": 0, "s": 1, "t": "MESSAGE_CREATE",
				"d": map[string]any{
					"id": "bot-message", "channel_id": "channel456", "guild_id": "guild789",
					"content": "ignore me",
					"author":  map[string]any{"id": "bot123", "username": "paibot", "bot": true},
				},
			},
			{
				"op": 0, "s": 2, "t": "MESSAGE_CREATE",
				"d": map[string]any{
					"id": "msg123", "channel_id": "channel456", "guild_id": "guild789",
					"content": "Hello from Discord",
					"author": map[string]any{
						"id": "user123", "username": "testuser", "global_name": "Test User", "bot": false,
					},
				},
			},
		} {
			if err := writeDiscordTestPayload(request.Context(), connection, event); err != nil {
				t.Errorf("write MESSAGE_CREATE: %v", err)
				return
			}
		}

		for {
			var payload discordGatewayPayload
			if err := readDiscordTestPayload(request.Context(), connection, &payload); err != nil {
				return
			}
			if payload.Op == 1 {
				select {
				case heartbeat <- payload:
				default:
				}
				if err := writeDiscordTestPayload(request.Context(), connection, map[string]any{"op": 11}); err != nil {
					return
				}
			}
		}
	}))
	defer server.Close()

	channel := newDiscordGatewayTestChannel(t, server.URL)
	messages := make(chan InboundMessage, 2)
	if err := channel.Start(t.Context(), func(message InboundMessage) {
		messages <- message
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = channel.Stop() })

	select {
	case payload := <-identify:
		if payload.Op != 2 {
			t.Fatalf("IDENTIFY opcode = %d, want 2", payload.Op)
		}
		var data struct {
			Token      string `json:"token"`
			Intents    int    `json:"intents"`
			Properties struct {
				OS      string `json:"os"`
				Browser string `json:"browser"`
				Device  string `json:"device"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(payload.Data, &data); err != nil {
			t.Fatalf("decode IDENTIFY data: %v", err)
		}
		if data.Token != "discord-bot-token" {
			t.Fatalf("IDENTIFY token = %q, want configured bot token", data.Token)
		}
		const wantedIntents = (1 << 9) | (1 << 12) | (1 << 15)
		if data.Intents != wantedIntents {
			t.Fatalf("IDENTIFY intents = %d, want %d", data.Intents, wantedIntents)
		}
		if data.Properties.OS == "" || data.Properties.Browser != "pai-bot" || data.Properties.Device != "pai-bot" {
			t.Fatalf("IDENTIFY properties = %+v, want OS and pai-bot client identity", data.Properties)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for IDENTIFY")
	}

	select {
	case got := <-messages:
		if got.Channel != "discord" || got.UserID != "user123" {
			t.Fatalf("identity = channel:%q user:%q, want discord user123", got.Channel, got.UserID)
		}
		if got.ThreadID != "discord:guild789:channel456" {
			t.Fatalf("ThreadID = %q, want Discord channel route", got.ThreadID)
		}
		if got.MessageID != "msg123" || got.Text != "Hello from Discord" {
			t.Fatalf("message = id:%q text:%q, want normalized Discord message", got.MessageID, got.Text)
		}
		if got.Username != "testuser" || got.FirstName != "Test User" {
			t.Fatalf("author = username:%q name:%q, want Discord author", got.Username, got.FirstName)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for MESSAGE_CREATE")
	}
	select {
	case extra := <-messages:
		t.Fatalf("bot message was dispatched: %+v", extra)
	case <-time.After(30 * time.Millisecond):
	}

	select {
	case payload := <-heartbeat:
		if string(payload.Data) != "2" {
			t.Fatalf("heartbeat sequence = %s, want last dispatch sequence 2", payload.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for heartbeat")
	}
}

func TestDiscordChannelGatewayStopsOnContextCancellation(t *testing.T) {
	connected := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(response, request, nil)
		if err != nil {
			t.Errorf("accept Gateway connection: %v", err)
			return
		}
		defer func() { _ = connection.CloseNow() }()
		if err := writeDiscordTestPayload(request.Context(), connection, map[string]any{
			"op": 10,
			"d":  map[string]any{"heartbeat_interval": 1000},
		}); err != nil {
			t.Errorf("write HELLO: %v", err)
			return
		}
		var identify discordGatewayPayload
		if err := readDiscordTestPayload(request.Context(), connection, &identify); err != nil {
			t.Errorf("read IDENTIFY: %v", err)
			return
		}
		close(connected)
		var payload discordGatewayPayload
		_ = readDiscordTestPayload(request.Context(), connection, &payload)
	}))
	defer server.Close()

	channel := newDiscordGatewayTestChannel(t, server.URL)
	ctx, cancel := context.WithCancel(t.Context())
	if err := channel.Start(ctx, func(InboundMessage) {}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-connected
	cancel()

	stopped := make(chan error, 1)
	go func() { stopped <- channel.Stop() }()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("Stop() after cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop() did not await Gateway shutdown")
	}
	if err := channel.Stop(); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
}

func TestDiscordChannelGatewayStartTimesOutWaitingForHello(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(response, request, nil)
		if err != nil {
			return
		}
		defer func() { _ = connection.CloseNow() }()
		<-request.Context().Done()
	}))
	defer server.Close()

	channel := newDiscordGatewayTestChannel(t, server.URL)
	channel.gatewayHandshakeTimeout = 250 * time.Millisecond

	startedAt := time.Now()
	err := channel.Start(t.Context(), func(InboundMessage) {})
	if err == nil || !strings.Contains(err.Error(), "HELLO") {
		t.Fatalf("Start() error = %v, want bounded HELLO timeout", err)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("Start() elapsed = %v, want bounded startup", elapsed)
	}
	if err := channel.Stop(); err != nil {
		t.Fatalf("Stop() after failed Start error = %v", err)
	}
}

func TestDiscordChannelGatewayReconnectsWithResumeAfterOpcodeSeven(t *testing.T) {
	resume := make(chan discordGatewayPayload, 1)
	message := make(chan InboundMessage, 1)
	var connections atomic.Int32

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(response, request, nil)
		if err != nil {
			t.Errorf("accept Gateway connection: %v", err)
			return
		}
		defer func() { _ = connection.CloseNow() }()
		number := connections.Add(1)
		if err := writeDiscordTestPayload(request.Context(), connection, map[string]any{
			"op": 10, "d": map[string]any{"heartbeat_interval": 1000},
		}); err != nil {
			return
		}
		var authenticate discordGatewayPayload
		if err := readDiscordTestPayload(request.Context(), connection, &authenticate); err != nil {
			return
		}
		if number == 1 {
			if authenticate.Op != 2 {
				t.Errorf("first authentication opcode = %d, want IDENTIFY", authenticate.Op)
				return
			}
			if err := writeDiscordTestPayload(request.Context(), connection, map[string]any{
				"op": 0, "s": 7, "t": "READY",
				"d": map[string]any{"session_id": "session-123", "resume_gateway_url": "ws" + strings.TrimPrefix(server.URL, "http")},
			}); err != nil {
				return
			}
			_ = writeDiscordTestPayload(request.Context(), connection, map[string]any{"op": 7})
			return
		}
		resume <- authenticate
		_ = writeDiscordTestPayload(request.Context(), connection, map[string]any{
			"op": 0, "s": 8, "t": "MESSAGE_CREATE",
			"d": map[string]any{
				"id": "message-after-resume", "channel_id": "channel456", "guild_id": "guild789",
				"content": "resumed",
				"author":  map[string]any{"id": "user123", "username": "testuser"},
			},
		})
		var payload discordGatewayPayload
		_ = readDiscordTestPayload(request.Context(), connection, &payload)
	}))
	defer server.Close()

	channel := newDiscordGatewayTestChannel(t, server.URL)
	channel.gatewayReconnectWait = noDiscordGatewayReconnectWait
	if err := channel.Start(t.Context(), func(got InboundMessage) { message <- got }); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = channel.Stop() })

	select {
	case payload := <-resume:
		if payload.Op != 6 {
			t.Fatalf("reconnect authentication opcode = %d, want RESUME", payload.Op)
		}
		var data struct {
			Token     string `json:"token"`
			SessionID string `json:"session_id"`
			Sequence  int64  `json:"seq"`
		}
		if err := json.Unmarshal(payload.Data, &data); err != nil {
			t.Fatalf("decode RESUME: %v", err)
		}
		if data.Token != "discord-bot-token" || data.SessionID != "session-123" || data.Sequence != 7 {
			t.Fatalf("RESUME data = %+v, want configured token, session-123, sequence 7", data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for RESUME")
	}
	select {
	case got := <-message:
		if got.MessageID != "message-after-resume" || got.Text != "resumed" {
			t.Fatalf("message after resume = %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message after RESUME")
	}
}

func TestDiscordChannelGatewayInvalidSessionReidentifiesWhenNotResumable(t *testing.T) {
	secondAuthentication := make(chan discordGatewayPayload, 1)
	var connections atomic.Int32

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(response, request, nil)
		if err != nil {
			return
		}
		defer func() { _ = connection.CloseNow() }()
		number := connections.Add(1)
		_ = writeDiscordTestPayload(request.Context(), connection, map[string]any{
			"op": 10, "d": map[string]any{"heartbeat_interval": 1000},
		})
		var authenticate discordGatewayPayload
		if err := readDiscordTestPayload(request.Context(), connection, &authenticate); err != nil {
			return
		}
		if number == 1 {
			_ = writeDiscordTestPayload(request.Context(), connection, map[string]any{
				"op": 0, "s": 11, "t": "READY",
				"d": map[string]any{"session_id": "discard-me", "resume_gateway_url": "ws" + strings.TrimPrefix(server.URL, "http")},
			})
			_ = writeDiscordTestPayload(request.Context(), connection, map[string]any{"op": 9, "d": false})
			return
		}
		secondAuthentication <- authenticate
		var payload discordGatewayPayload
		_ = readDiscordTestPayload(request.Context(), connection, &payload)
	}))
	defer server.Close()

	channel := newDiscordGatewayTestChannel(t, server.URL)
	channel.gatewayReconnectWait = noDiscordGatewayReconnectWait
	if err := channel.Start(t.Context(), func(InboundMessage) {}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = channel.Stop() })

	select {
	case payload := <-secondAuthentication:
		if payload.Op != 2 {
			t.Fatalf("authentication after non-resumable session = opcode %d, want IDENTIFY", payload.Op)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for IDENTIFY after invalid session")
	}
}

func TestDiscordChannelGatewayReconnectsAfterDisconnectAndUsesBackoff(t *testing.T) {
	reconnected := make(chan struct{}, 1)
	backoffAttempts := make(chan int, 1)
	var connections atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(response, request, nil)
		if err != nil {
			return
		}
		defer func() { _ = connection.CloseNow() }()
		number := connections.Add(1)
		_ = writeDiscordTestPayload(request.Context(), connection, map[string]any{
			"op": 10, "d": map[string]any{"heartbeat_interval": 1000},
		})
		var authenticate discordGatewayPayload
		if err := readDiscordTestPayload(request.Context(), connection, &authenticate); err != nil {
			return
		}
		if number == 1 {
			_ = connection.Close(websocket.StatusServiceRestart, "restart")
			return
		}
		reconnected <- struct{}{}
		var payload discordGatewayPayload
		_ = readDiscordTestPayload(request.Context(), connection, &payload)
	}))
	defer server.Close()

	channel := newDiscordGatewayTestChannel(t, server.URL)
	channel.gatewayReconnectWait = func(ctx context.Context, attempt int) error {
		backoffAttempts <- attempt
		return nil
	}
	if err := channel.Start(t.Context(), func(InboundMessage) {}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = channel.Stop() })

	select {
	case attempt := <-backoffAttempts:
		if attempt != 1 {
			t.Fatalf("first reconnect attempt = %d, want 1", attempt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reconnect backoff")
	}
	if err := channel.RuntimeError(); websocket.CloseStatus(err) != websocket.StatusServiceRestart {
		t.Fatalf("RuntimeError() = %v, want recoverable service-restart disconnect", err)
	}
	select {
	case <-reconnected:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reconnect after disconnect")
	}
}

func TestDiscordChannelGatewayReportsTerminalRuntimeError(t *testing.T) {
	identified := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(response, request, nil)
		if err != nil {
			return
		}
		defer func() { _ = connection.CloseNow() }()
		_ = writeDiscordTestPayload(request.Context(), connection, map[string]any{
			"op": 10, "d": map[string]any{"heartbeat_interval": 1000},
		})
		var authenticate discordGatewayPayload
		if err := readDiscordTestPayload(request.Context(), connection, &authenticate); err != nil {
			return
		}
		close(identified)
		_ = connection.Close(websocket.StatusCode(4004), "authentication failed")
	}))
	defer server.Close()

	channel := newDiscordGatewayTestChannel(t, server.URL)
	if err := channel.Start(t.Context(), func(InboundMessage) {}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-identified

	deadline := time.Now().Add(time.Second)
	for channel.RuntimeError() == nil && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	err := channel.RuntimeError()
	if err == nil || websocket.CloseStatus(err) != websocket.StatusCode(4004) {
		t.Fatalf("RuntimeError() = %v, want Discord authentication close error", err)
	}
	if stopErr := channel.Stop(); !errors.Is(stopErr, err) {
		t.Fatalf("Stop() error = %v, want terminal runtime error %v", stopErr, err)
	}
	if stopErr := channel.Stop(); !errors.Is(stopErr, err) {
		t.Fatalf("second Stop() error = %v, want same terminal runtime error", stopErr)
	}
}

func noDiscordGatewayReconnectWait(context.Context, int) error {
	return nil
}

func newDiscordGatewayTestChannel(t *testing.T, serverURL string) *DiscordChannel {
	t.Helper()
	channel, err := NewDiscordChannel(DiscordConfig{
		BotToken:      "discord-bot-token",
		PublicKey:     hex.EncodeToString(make([]byte, ed25519.PublicKeySize)),
		ApplicationID: "app-123",
	})
	if err != nil {
		t.Fatalf("NewDiscordChannel() error = %v", err)
	}
	channel.gatewayURL = "ws" + strings.TrimPrefix(serverURL, "http")
	return channel
}

func writeDiscordTestPayload(ctx context.Context, connection *websocket.Conn, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return connection.Write(ctx, websocket.MessageText, data)
}

func readDiscordTestPayload(ctx context.Context, connection *websocket.Conn, payload *discordGatewayPayload) error {
	_, data, err := connection.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, payload)
}

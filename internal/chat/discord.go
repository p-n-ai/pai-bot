// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	defaultDiscordBaseURL                 = "https://discord.com/api/v10"
	defaultDiscordGateway                 = "wss://gateway.discord.gg/"
	maxDiscordWebhookBody                 = 1 << 20
	defaultDiscordGatewayHandshakeTimeout = 10 * time.Second

	discordGatewayVersion = "10"
	discordGatewayIntents = (1 << 9) | // GUILD_MESSAGES
		(1 << 12) | // DIRECT_MESSAGES
		(1 << 15) // MESSAGE_CONTENT
)

// DiscordConfig contains the credentials required by the Discord adapter.
type DiscordConfig struct {
	BotToken      string
	PublicKey     string
	ApplicationID string
}

// DiscordChannel implements Discord interaction ingress, direct Gateway event
// ingress, and REST API delivery.
type DiscordChannel struct {
	botToken      string
	publicKey     ed25519.PublicKey
	applicationID string
	baseURL       string
	gatewayURL    string
	client        *http.Client

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
	applicationID := strings.TrimSpace(config.ApplicationID)
	if applicationID == "" {
		return nil, fmt.Errorf("discord application ID is required")
	}
	publicKey, err := hex.DecodeString(strings.TrimSpace(config.PublicKey))
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("discord public key must be a 32-byte hexadecimal Ed25519 key")
	}

	return &DiscordChannel{
		botToken:                botToken,
		publicKey:               ed25519.PublicKey(publicKey),
		applicationID:           applicationID,
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

// Start connects to Discord's Gateway for direct message ingress.
func (d *DiscordChannel) Start(ctx context.Context, handler func(InboundMessage)) error {
	d.lifecycleMu.Lock()
	if d.runtime != nil {
		d.lifecycleMu.Unlock()
		return nil
	}
	d.runtimeErr = nil
	d.terminalRuntimeErr = nil
	runCtx, cancel := context.WithCancel(ctx)
	gatewayRuntime := &discordGatewayRuntime{
		cancel: cancel,
		done:   make(chan struct{}),
	}
	d.runtime = gatewayRuntime
	d.lifecycleMu.Unlock()

	session := &discordGatewaySession{}
	connection, heartbeatInterval, err := d.connectGateway(runCtx, d.gatewayURL, session)
	if err != nil {
		cancel()
		d.finishGateway(gatewayRuntime, err)
		return err
	}
	gatewayRuntime.setConnection(connection)
	go d.runGateway(runCtx, gatewayRuntime, connection, heartbeatInterval, session, handler)
	return nil
}

// Stop disconnects the direct Gateway runtime and waits for its work to exit.
func (d *DiscordChannel) Stop() error {
	d.lifecycleMu.Lock()
	gatewayRuntime := d.runtime
	if gatewayRuntime == nil {
		err := d.terminalRuntimeErr
		d.lifecycleMu.Unlock()
		return err
	}
	gatewayRuntime.cancel()
	gatewayRuntime.closeConnection()
	d.lifecycleMu.Unlock()

	<-gatewayRuntime.done
	d.lifecycleMu.Lock()
	defer d.lifecycleMu.Unlock()
	return d.terminalRuntimeErr
}

// RuntimeError reports the latest direct Gateway failure, including failures
// that the adapter recovered from by reconnecting.
func (d *DiscordChannel) RuntimeError() error {
	d.lifecycleMu.Lock()
	defer d.lifecycleMu.Unlock()
	return d.runtimeErr
}

func (d *DiscordChannel) connectGateway(
	ctx context.Context,
	rawGatewayURL string,
	session *discordGatewaySession,
) (*websocket.Conn, time.Duration, error) {
	handshakeTimeout := d.gatewayHandshakeTimeout
	if handshakeTimeout <= 0 {
		handshakeTimeout = defaultDiscordGatewayHandshakeTimeout
	}
	handshakeCtx, cancelHandshake := context.WithTimeout(ctx, handshakeTimeout)
	defer cancelHandshake()

	gatewayURL, err := url.Parse(rawGatewayURL)
	if err != nil {
		return nil, 0, fmt.Errorf("parse Discord Gateway URL: %w", err)
	}
	query := gatewayURL.Query()
	query.Set("v", discordGatewayVersion)
	query.Set("encoding", "json")
	gatewayURL.RawQuery = query.Encode()

	connection, _, err := websocket.Dial(handshakeCtx, gatewayURL.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("connect Discord Gateway: %w", err)
	}
	closeOnError := func(err error) (*websocket.Conn, time.Duration, error) {
		connection.CloseNow()
		return nil, 0, err
	}

	payload, err := readDiscordGatewayPayload(handshakeCtx, connection)
	if err != nil {
		return closeOnError(fmt.Errorf("read Discord Gateway HELLO: %w", err))
	}
	if payload.Op != 10 {
		return closeOnError(fmt.Errorf("Discord Gateway sent opcode %d before HELLO", payload.Op))
	}
	var hello struct {
		HeartbeatInterval int64 `json:"heartbeat_interval"`
	}
	if err := json.Unmarshal(payload.Data, &hello); err != nil {
		return closeOnError(fmt.Errorf("decode Discord Gateway HELLO: %w", err))
	}
	if hello.HeartbeatInterval <= 0 {
		return closeOnError(fmt.Errorf("Discord Gateway sent invalid heartbeat interval %d", hello.HeartbeatInterval))
	}
	if session.resumable() {
		if err := writeDiscordGatewayPayload(handshakeCtx, connection, struct {
			Op   int                  `json:"op"`
			Data discordGatewayResume `json:"d"`
		}{
			Op: 6,
			Data: discordGatewayResume{
				Token:     d.botToken,
				SessionID: session.id,
				Sequence:  *session.sequence,
			},
		}); err != nil {
			return closeOnError(fmt.Errorf("resume Discord Gateway session: %w", err))
		}
	} else {
		if err := writeDiscordGatewayPayload(handshakeCtx, connection, struct {
			Op   int                    `json:"op"`
			Data discordGatewayIdentify `json:"d"`
		}{
			Op: 2,
			Data: discordGatewayIdentify{
				Token:   d.botToken,
				Intents: discordGatewayIntents,
				Properties: discordGatewayProperties{
					OS:      runtime.GOOS,
					Browser: "pai-bot",
					Device:  "pai-bot",
				},
			},
		}); err != nil {
			return closeOnError(fmt.Errorf("identify with Discord Gateway: %w", err))
		}
	}
	return connection, time.Duration(hello.HeartbeatInterval) * time.Millisecond, nil
}

func (d *DiscordChannel) runGateway(
	ctx context.Context,
	gatewayRuntime *discordGatewayRuntime,
	connection *websocket.Conn,
	heartbeatInterval time.Duration,
	session *discordGatewaySession,
	handler func(InboundMessage),
) {
	currentConnection := connection
	currentHeartbeatInterval := heartbeatInterval
	for {
		directive, err := d.consumeGateway(
			ctx,
			currentConnection,
			currentHeartbeatInterval,
			session,
			handler,
		)
		currentConnection.CloseNow()
		gatewayRuntime.setConnection(nil)

		if ctx.Err() != nil {
			d.finishGateway(gatewayRuntime, ctx.Err())
			return
		}
		if directive.resetSession {
			session.reset()
		}
		if err != nil {
			d.recordGatewayRuntimeError(err)
			if terminalDiscordGatewayError(err) {
				d.finishGateway(gatewayRuntime, err)
				return
			}
		}

		reconnectWait := d.gatewayReconnectWait
		if reconnectWait == nil {
			reconnectWait = waitDiscordGatewayReconnect
		}
		if err := reconnectWait(ctx, 1); err != nil {
			d.finishGateway(gatewayRuntime, err)
			return
		}

		attempt := 1
		for {
			gatewayURL := d.gatewayURL
			if session.resumable() && session.resumeURL != "" {
				gatewayURL = session.resumeURL
			}
			nextConnection, nextHeartbeatInterval, connectErr := d.connectGateway(ctx, gatewayURL, session)
			if connectErr == nil {
				currentConnection = nextConnection
				currentHeartbeatInterval = nextHeartbeatInterval
				gatewayRuntime.setConnection(nextConnection)
				break
			}
			if ctx.Err() != nil {
				d.finishGateway(gatewayRuntime, ctx.Err())
				return
			}
			d.recordGatewayRuntimeError(connectErr)
			if terminalDiscordGatewayError(connectErr) {
				d.finishGateway(gatewayRuntime, connectErr)
				return
			}
			attempt++
			if err := reconnectWait(ctx, attempt); err != nil {
				d.finishGateway(gatewayRuntime, err)
				return
			}
		}
	}
}

func (d *DiscordChannel) consumeGateway(
	ctx context.Context,
	connection *websocket.Conn,
	heartbeatInterval time.Duration,
	session *discordGatewaySession,
	handler func(InboundMessage),
) (discordGatewayDirective, error) {
	readCtx, cancelRead := context.WithCancel(ctx)
	defer cancelRead()

	reads := make(chan discordGatewayRead, 1)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			payload, err := readDiscordGatewayPayload(readCtx, connection)
			select {
			case reads <- discordGatewayRead{payload: payload, err: err}:
			case <-readCtx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	defer func() {
		cancelRead()
		connection.CloseNow()
		<-readerDone
	}()

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()
	heartbeatAcknowledged := true

	for {
		select {
		case <-ctx.Done():
			return discordGatewayDirective{}, ctx.Err()
		case <-heartbeat.C:
			if !heartbeatAcknowledged {
				return discordGatewayDirective{}, fmt.Errorf("Discord Gateway did not acknowledge heartbeat")
			}
			if err := sendDiscordHeartbeat(ctx, connection, session.sequence); err != nil {
				return discordGatewayDirective{}, err
			}
			heartbeatAcknowledged = false
		case read := <-reads:
			if read.err != nil {
				return discordGatewayDirective{}, read.err
			}
			if read.payload.Sequence != nil {
				session.setSequence(*read.payload.Sequence)
			}
			switch read.payload.Op {
			case 0:
				switch read.payload.Type {
				case "READY":
					var ready discordGatewayReady
					if err := json.Unmarshal(read.payload.Data, &ready); err != nil {
						return discordGatewayDirective{}, fmt.Errorf("decode Discord Gateway READY: %w", err)
					}
					if ready.SessionID == "" || ready.ResumeGatewayURL == "" {
						return discordGatewayDirective{}, fmt.Errorf("Discord Gateway READY omitted session information")
					}
					session.id = ready.SessionID
					session.resumeURL = ready.ResumeGatewayURL
				case "MESSAGE_CREATE":
					var message discordGatewayData
					if err := json.Unmarshal(read.payload.Data, &message); err != nil {
						continue
					}
					d.dispatchGatewayMessage(message, handler)
				}
			case 1:
				if err := sendDiscordHeartbeat(ctx, connection, session.sequence); err != nil {
					return discordGatewayDirective{}, err
				}
				heartbeatAcknowledged = false
			case 7:
				return discordGatewayDirective{}, nil
			case 9:
				var resumable bool
				if err := json.Unmarshal(read.payload.Data, &resumable); err != nil {
					return discordGatewayDirective{}, fmt.Errorf("decode Discord Gateway INVALID_SESSION: %w", err)
				}
				return discordGatewayDirective{
					resetSession: !resumable,
				}, nil
			case 11:
				heartbeatAcknowledged = true
			}
		}
	}
}

func (d *DiscordChannel) finishGateway(gatewayRuntime *discordGatewayRuntime, err error) {
	d.lifecycleMu.Lock()
	gatewayRuntime.err = err
	if err != nil &&
		!errors.Is(err, context.Canceled) &&
		websocket.CloseStatus(err) != websocket.StatusNormalClosure {
		d.runtimeErr = err
		if terminalDiscordGatewayError(err) {
			d.terminalRuntimeErr = err
			slog.Error("Discord Gateway stopped after a terminal failure", "error", err)
		}
	}
	if d.runtime == gatewayRuntime {
		d.runtime = nil
	}
	close(gatewayRuntime.done)
	d.lifecycleMu.Unlock()
}

func (d *DiscordChannel) recordGatewayRuntimeError(err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	d.lifecycleMu.Lock()
	d.runtimeErr = err
	d.lifecycleMu.Unlock()
	slog.Warn("Discord Gateway connection failed; reconnecting", "error", err)
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

// WebhookHandler verifies and normalizes Discord interaction and Gateway events.
func (d *DiscordChannel) WebhookHandler(handler func(InboundMessage)) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, maxDiscordWebhookBody))
		if err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}

		if !d.validInteractionSignature(request.Header, body) {
			http.Error(response, "invalid signature", http.StatusUnauthorized)
			return
		}
		d.handleInteraction(response, body)
	})
}

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
		ExternalID: message.ID,
		ThreadID:   "discord:" + guildID + ":" + message.ChannelID,
		MessageID:  message.ID,
		DeliveryID: message.ID,
		Text:       message.Content,
		Username:   message.Author.Username,
		FirstName:  firstName,
	})
}

func (d *DiscordChannel) handleInteraction(response http.ResponseWriter, body []byte) {
	var interaction struct {
		Type int `json:"type"`
	}
	if err := json.Unmarshal(body, &interaction); err != nil {
		http.Error(response, "bad request", http.StatusBadRequest)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	if interaction.Type == 1 {
		_ = json.NewEncoder(response).Encode(struct {
			Type int `json:"type"`
		}{Type: 1})
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (d *DiscordChannel) validInteractionSignature(headers http.Header, body []byte) bool {
	timestamp := headers.Get("X-Signature-Timestamp")
	signature, err := hex.DecodeString(headers.Get("X-Signature-Ed25519"))
	if timestamp == "" || err != nil || len(signature) != ed25519.SignatureSize {
		return false
	}
	signed := make([]byte, 0, len(timestamp)+len(body))
	signed = append(signed, timestamp...)
	signed = append(signed, body...)
	return ed25519.Verify(d.publicKey, signed, signature)
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
			return fmt.Errorf("Discord API returned %s: %s", response.Status, text)
		}
		return fmt.Errorf("Discord API returned %s", response.Status)
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
		return "", fmt.Errorf("Discord destination is required")
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

type discordGatewayRuntime struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	connection *websocket.Conn
	done       chan struct{}
	err        error
}

func (r *discordGatewayRuntime) setConnection(connection *websocket.Conn) {
	r.mu.Lock()
	r.connection = connection
	r.mu.Unlock()
}

func (r *discordGatewayRuntime) closeConnection() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.connection != nil {
		r.connection.CloseNow()
	}
}

type discordGatewaySession struct {
	id        string
	resumeURL string
	sequence  *int64
}

func (s *discordGatewaySession) resumable() bool {
	return s.id != "" && s.sequence != nil
}

func (s *discordGatewaySession) setSequence(sequence int64) {
	s.sequence = new(int64)
	*s.sequence = sequence
}

func (s *discordGatewaySession) reset() {
	s.id = ""
	s.resumeURL = ""
	s.sequence = nil
}

type discordGatewayDirective struct {
	resetSession bool
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

func terminalDiscordGatewayError(err error) bool {
	switch websocket.CloseStatus(err) {
	case websocket.StatusCode(4004),
		websocket.StatusCode(4010),
		websocket.StatusCode(4011),
		websocket.StatusCode(4012),
		websocket.StatusCode(4013),
		websocket.StatusCode(4014):
		return true
	default:
		return false
	}
}

func waitDiscordGatewayReconnect(ctx context.Context, attempt int) error {
	if attempt < 1 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 5 {
		shift = 5
	}
	delay := time.Second * time.Duration(1<<shift)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

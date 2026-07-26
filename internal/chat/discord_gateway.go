// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	defaultDiscordGateway                 = "wss://gateway.discord.gg/"
	defaultDiscordGatewayHandshakeTimeout = 10 * time.Second
	discordGatewayVersion                 = "10"
	discordGatewayIntents                 = (1 << 9) | // GUILD_MESSAGES
		(1 << 12) | // DIRECT_MESSAGES
		(1 << 15) // MESSAGE_CONTENT
)

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
		_ = connection.CloseNow()
		return nil, 0, err
	}

	payload, err := readDiscordGatewayPayload(handshakeCtx, connection)
	if err != nil {
		return closeOnError(fmt.Errorf("read Discord Gateway HELLO: %w", err))
	}
	if payload.Op != 10 {
		return closeOnError(fmt.Errorf("discord gateway sent opcode %d before HELLO", payload.Op))
	}
	var hello struct {
		HeartbeatInterval int64 `json:"heartbeat_interval"`
	}
	if err := json.Unmarshal(payload.Data, &hello); err != nil {
		return closeOnError(fmt.Errorf("decode Discord Gateway HELLO: %w", err))
	}
	if hello.HeartbeatInterval <= 0 {
		return closeOnError(fmt.Errorf("discord gateway sent invalid heartbeat interval %d", hello.HeartbeatInterval))
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
		_ = currentConnection.CloseNow()
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
		_ = connection.CloseNow()
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
				return discordGatewayDirective{}, fmt.Errorf("discord gateway did not acknowledge heartbeat")
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
						return discordGatewayDirective{}, fmt.Errorf("discord gateway READY omitted session information")
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
		_ = r.connection.CloseNow()
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

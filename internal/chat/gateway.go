// Package chat provides a unified interface for messaging channels (Telegram, WhatsApp, WebSocket).
package chat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// InboundMessage is a message received from any channel.
type InboundMessage struct {
	Channel      string
	UserID       string
	ExternalID   string
	Text         string
	Caption      string
	HasImage     bool
	ImageFileID  string
	ImageDataURL string
	ReplyToText  string // text of the message being replied to (if any)
	Username     string
	FirstName    string
	LastName     string
	Language     string
}

// OutboundMessage is a message to send via any channel.
type OutboundMessage struct {
	Channel   string
	UserID    string
	Text      string
	ParseMode string // "Markdown", "HTML", or ""
}

// Channel is the interface each messaging platform must implement.
type Channel interface {
	SendMessage(ctx context.Context, userID string, msg OutboundMessage) error
	SendTyping(ctx context.Context, userID string) error
	Start(ctx context.Context, handler func(InboundMessage)) error
	Stop() error
}

// Gateway routes messages to/from registered channels.
type Gateway struct {
	channels map[string]Channel
	mu       sync.RWMutex
}

// NewGateway creates a new chat gateway.
func NewGateway() *Gateway {
	return &Gateway{
		channels: make(map[string]Channel),
	}
}

// Register adds a channel to the gateway.
func (g *Gateway) Register(name string, ch Channel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.channels[name] = ch
	slog.Info("chat channel registered", "channel", name)
}

// HasChannel returns true if the named channel is registered.
func (g *Gateway) HasChannel(name string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.channels[name]
	return ok
}

// Send dispatches a message to the appropriate channel.
func (g *Gateway) Send(ctx context.Context, msg OutboundMessage) error {
	g.mu.RLock()
	ch, ok := g.channels[msg.Channel]
	g.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown channel: %s", msg.Channel)
	}

	return ch.SendMessage(ctx, msg.UserID, msg)
}

// SendTyping sends a typing indicator to the user on the given channel.
func (g *Gateway) SendTyping(ctx context.Context, channel, userID string) error {
	g.mu.RLock()
	ch, ok := g.channels[channel]
	g.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown channel: %s", channel)
	}

	return ch.SendTyping(ctx, userID)
}

// StartAll starts all registered channels with the given message handler.
func (g *Gateway) StartAll(ctx context.Context, handler func(InboundMessage)) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for name, ch := range g.channels {
		slog.Info("starting channel", "channel", name)
		if err := ch.Start(ctx, handler); err != nil {
			return fmt.Errorf("starting channel %s: %w", name, err)
		}
	}
	return nil
}

// MockChannel is a test double for Channel.
type MockChannel struct {
	SentMessages []OutboundMessage
}

func (m *MockChannel) SendMessage(_ context.Context, _ string, msg OutboundMessage) error {
	m.SentMessages = append(m.SentMessages, msg)
	return nil
}

func (m *MockChannel) SendTyping(_ context.Context, _ string) error {
	return nil
}

func (m *MockChannel) Start(_ context.Context, _ func(InboundMessage)) error {
	return nil
}

func (m *MockChannel) Stop() error {
	return nil
}

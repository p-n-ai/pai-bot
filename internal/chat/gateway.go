// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package chat provides channel-neutral messaging contracts and adapters.
package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
)

// InboundMessage is a message received from any channel.
type InboundMessage struct {
	Channel      string
	UserID       string
	ExternalID   string
	ThreadID     string
	MessageID    string
	DeliveryID   string
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
	// CallbackQueryID is populated for Telegram inline-button callbacks.
	CallbackQueryID string
	// CallbackMessageID is the Telegram message ID that contains the clicked inline button.
	CallbackMessageID int
}

// DestinationID returns the adapter-owned reply destination for this message.
func (m InboundMessage) DestinationID() string {
	if m.ThreadID != "" {
		return m.ThreadID
	}
	return m.UserID
}

type InlineButton struct {
	Text         string
	CallbackData string
	URL          string
}

// OutboundMessage is a message to send via any channel.
type OutboundMessage struct {
	Channel        string
	UserID         string
	ThreadID       string
	Text           string
	FocusedPageURL string
	ParseMode      string // "Markdown", "HTML", or ""
	// ReplyKeyboard is Telegram-style keyboard rows. Other channels may ignore it.
	ReplyKeyboard [][]string
	// InlineKeyboard is Telegram inline keyboard rows. Other channels may ignore it.
	InlineKeyboard [][]InlineButton
}

// Channel is the interface each messaging platform must implement.
type Channel interface {
	SendMessage(ctx context.Context, destination string, msg OutboundMessage) error
	SendTyping(ctx context.Context, destination string) error
	Start(ctx context.Context, handler func(InboundMessage)) error
	Stop() error
}

// WebhookChannel receives inbound messages through an HTTP webhook.
type WebhookChannel interface {
	Channel
	WebhookHandler(handler func(InboundMessage)) http.Handler
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
	g.channels[name] = ch
	g.mu.Unlock()

	slog.Info("chat channel registered", "channel", name)
}

// HasChannel returns true if the named channel is registered.
func (g *Gateway) HasChannel(name string) bool {
	_, ok := g.Channel(name)
	return ok
}

// Channel returns the registered channel with the given name.
func (g *Gateway) Channel(name string) (Channel, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	channel, ok := g.channels[name]
	return channel, ok
}

// ChannelNames returns the names of all registered channels.
func (g *Gateway) ChannelNames() []string {
	g.mu.RLock()
	names := make([]string, 0, len(g.channels))
	for name := range g.channels {
		names = append(names, name)
	}
	g.mu.RUnlock()

	sort.Strings(names)
	return names
}

// Send dispatches a message to the appropriate channel.
func (g *Gateway) Send(ctx context.Context, msg OutboundMessage) error {
	ch, ok := g.Channel(msg.Channel)
	if !ok {
		return fmt.Errorf("unknown channel: %s", msg.Channel)
	}

	destination := msg.ThreadID
	if destination == "" {
		destination = msg.UserID
	}
	return ch.SendMessage(ctx, destination, msg)
}

// SendTyping sends a typing indicator to a channel-owned destination.
func (g *Gateway) SendTyping(ctx context.Context, channel, destination string) error {
	ch, ok := g.Channel(channel)
	if !ok {
		return fmt.Errorf("unknown channel: %s", channel)
	}

	return ch.SendTyping(ctx, destination)
}

// Webhook returns the inbound HTTP handler for a registered webhook channel.
func (g *Gateway) Webhook(name string, handler func(InboundMessage)) (http.Handler, error) {
	channel, ok := g.Channel(name)
	if !ok {
		return nil, fmt.Errorf("unknown channel: %s", name)
	}
	webhook, ok := channel.(WebhookChannel)
	if !ok {
		return nil, fmt.Errorf("channel %s does not support webhooks", name)
	}
	return webhook.WebhookHandler(handler), nil
}

// Webhooks returns every registered HTTP ingress adapter by its channel name.
func (g *Gateway) Webhooks(handler func(InboundMessage)) map[string]http.Handler {
	webhooks := make(map[string]http.Handler)
	for _, entry := range g.channelEntries() {
		channel, ok := entry.channel.(WebhookChannel)
		if !ok {
			continue
		}
		webhooks[entry.name] = channel.WebhookHandler(handler)
	}
	return webhooks
}

// StartAll starts all registered channels with the given message handler.
func (g *Gateway) StartAll(ctx context.Context, handler func(InboundMessage)) error {
	var started []channelEntry
	for _, entry := range g.channelEntries() {
		name, ch := entry.name, entry.channel
		slog.Info("starting channel", "channel", name)
		if err := ch.Start(ctx, handler); err != nil {
			errs := []error{fmt.Errorf("starting channel %s: %w", name, err)}
			for i := len(started) - 1; i >= 0; i-- {
				if stopErr := started[i].channel.Stop(); stopErr != nil {
					errs = append(errs, fmt.Errorf("rolling back channel %s: %w", started[i].name, stopErr))
				}
			}
			return errors.Join(errs...)
		}
		started = append(started, entry)
	}
	return nil
}

// StopAll stops every registered channel. It returns any shutdown errors after
// giving every channel a chance to release its resources.
func (g *Gateway) StopAll() error {
	var errs []error
	for _, entry := range g.channelEntries() {
		if err := entry.channel.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stopping channel %s: %w", entry.name, err))
		}
	}
	return errors.Join(errs...)
}

type channelEntry struct {
	name    string
	channel Channel
}

func (g *Gateway) channelEntries() []channelEntry {
	g.mu.RLock()
	entries := make([]channelEntry, 0, len(g.channels))
	for name, channel := range g.channels {
		entries = append(entries, channelEntry{name: name, channel: channel})
	}
	g.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})
	return entries
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

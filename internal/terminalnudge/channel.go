// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalnudge

import (
	"context"
	"sync"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

// CaptureChannel stores outbound messages for later inspection.
type CaptureChannel struct {
	mu       sync.Mutex
	messages []chat.OutboundMessage
}

func (c *CaptureChannel) SendMessage(_ context.Context, _ string, msg chat.OutboundMessage) error {
	c.mu.Lock()
	c.messages = append(c.messages, msg)
	c.mu.Unlock()
	return nil
}

func (c *CaptureChannel) SendTyping(_ context.Context, _ string) error {
	return nil
}

func (c *CaptureChannel) Start(_ context.Context, _ func(chat.InboundMessage)) error {
	return nil
}

func (c *CaptureChannel) Stop() error {
	return nil
}

func (c *CaptureChannel) Messages() []chat.OutboundMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]chat.OutboundMessage(nil), c.messages...)
}

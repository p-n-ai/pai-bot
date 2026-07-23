// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestFocusedPageChannelPolicy(t *testing.T) {
	tests := []struct {
		name    string
		devMode bool
		channel string
		want    bool
	}{
		{name: "telegram production", channel: "telegram", want: true},
		{name: "terminal websocket in dev", devMode: true, channel: "websocket", want: true},
		{name: "embed websocket in production", channel: "websocket", want: false},
		{name: "whatsapp", devMode: true, channel: "whatsapp", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := focusedPageChannelEnabled(tt.devMode, chat.InboundMessage{Channel: tt.channel}); got != tt.want {
				t.Fatalf("focusedPageChannelEnabled(%t, %q) = %t, want %t", tt.devMode, tt.channel, got, tt.want)
			}
		})
	}
}

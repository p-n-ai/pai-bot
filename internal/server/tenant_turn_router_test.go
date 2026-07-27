// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

type turnProcessorStub struct {
	messages []chat.InboundMessage
}

func (s *turnProcessorStub) ProcessAndDeliver(_ context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
	s.messages = append(s.messages, msg)
	return agent.TurnResult{}, nil
}

func TestTenantTurnRouterRoutesAuthenticatedEmbedByTenant(t *testing.T) {
	defaultProcessor := &turnProcessorStub{}
	tenantProcessors := map[string]*turnProcessorStub{}
	router, err := NewTenantTurnRouter(defaultProcessor, func(tenantID string) (TurnProcessor, error) {
		processor := &turnProcessorStub{}
		tenantProcessors[tenantID] = processor
		return processor, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	message := chat.InboundMessage{
		Channel:         "embed",
		UserID:          "internal-user-b",
		TenantID:        "tenant-b",
		InternalUserID:  "internal-user-b",
		IdentityChannel: "telegram",
		ExternalID:      "telegram-user-b",
		Text:            "hello",
	}
	if _, err := router.ProcessAndDeliver(t.Context(), message); err != nil {
		t.Fatal(err)
	}
	if len(defaultProcessor.messages) != 0 {
		t.Fatal("tenant message reached default processor")
	}
	if got := tenantProcessors["tenant-b"]; got == nil || len(got.messages) != 1 {
		t.Fatalf("tenant-b processor messages = %#v", got)
	}
}

func TestTenantTurnRouterRejectsTenantWithoutInternalIdentity(t *testing.T) {
	router, err := NewTenantTurnRouter(&turnProcessorStub{}, func(string) (TurnProcessor, error) {
		return &turnProcessorStub{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.ProcessAndDeliver(t.Context(), chat.InboundMessage{
		Channel:  "embed",
		TenantID: "tenant-b",
		UserID:   "untrusted-external-id",
	}); err == nil {
		t.Fatal("expected incomplete authenticated identity to be rejected")
	}
}

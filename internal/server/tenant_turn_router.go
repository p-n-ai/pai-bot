// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

type TurnProcessor interface {
	ProcessTurn(ctx context.Context, msg chat.InboundMessage) (agent.TurnResult, error)
	DeliverTurn(ctx context.Context, msg chat.InboundMessage, result agent.TurnResult) error
}

type TenantTurnRouter struct {
	defaultProcessor TurnProcessor
	newForTenant     func(string) (TurnProcessor, error)

	mu         sync.Mutex
	byTenantID map[string]TurnProcessor
}

func NewTenantTurnRouter(defaultProcessor TurnProcessor, newForTenant func(string) (TurnProcessor, error)) (*TenantTurnRouter, error) {
	if defaultProcessor == nil {
		return nil, errors.New("default turn processor is required")
	}
	if newForTenant == nil {
		return nil, errors.New("tenant turn processor factory is required")
	}
	return &TenantTurnRouter{
		defaultProcessor: defaultProcessor,
		newForTenant:     newForTenant,
		byTenantID:       make(map[string]TurnProcessor),
	}, nil
}

func (r *TenantTurnRouter) ProcessTurn(ctx context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
	processor, err := r.processorFor(msg)
	if err != nil {
		return agent.TurnResult{}, err
	}
	return processor.ProcessTurn(ctx, msg)
}

func (r *TenantTurnRouter) DeliverTurn(ctx context.Context, msg chat.InboundMessage, result agent.TurnResult) error {
	processor, err := r.processorFor(msg)
	if err != nil {
		return err
	}
	return processor.DeliverTurn(ctx, msg, result)
}

func (r *TenantTurnRouter) processorFor(msg chat.InboundMessage) (TurnProcessor, error) {
	tenantID := strings.TrimSpace(msg.TenantID)
	if tenantID == "" {
		return r.defaultProcessor, nil
	}
	if strings.TrimSpace(msg.InternalUserID) == "" ||
		strings.TrimSpace(msg.IdentityChannel) == "" ||
		strings.TrimSpace(msg.ExternalID) == "" {
		return nil, errors.New("authenticated tenant message is missing internal user identity")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if processor := r.byTenantID[tenantID]; processor != nil {
		return processor, nil
	}
	processor, err := r.newForTenant(tenantID)
	if err != nil {
		return nil, err
	}
	if processor == nil {
		return nil, errors.New("tenant turn processor factory returned nil")
	}
	r.byTenantID[tenantID] = processor
	return processor, nil
}

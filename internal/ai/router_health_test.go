// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

type healthProvider struct {
	mu    sync.Mutex
	calls int
	err   error
	resp  ai.CompletionResponse
}

func (p *healthProvider) Complete(context.Context, ai.CompletionRequest) (ai.CompletionResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	return p.resp, p.err
}

func (*healthProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (*healthProvider) Models() []ai.ModelInfo { return nil }

func (p *healthProvider) HealthCheck(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	return p.err
}

func (p *healthProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func TestRouterHealthCheckUsesFallbackOrder(t *testing.T) {
	router := ai.NewRouter()
	preferred := &healthProvider{err: errors.New("preferred unavailable")}
	fallback := &healthProvider{}
	unreached := &healthProvider{}
	router.ReplaceProviders([]ai.ProviderRegistration{
		{Name: "preferred", Provider: preferred},
		{Name: "fallback", Provider: fallback},
		{Name: "unreached", Provider: unreached},
	})

	if err := router.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}
	if preferred.callCount() != 1 || fallback.callCount() != 1 || unreached.callCount() != 0 {
		t.Fatalf(
			"health calls preferred=%d fallback=%d unreached=%d, want 1,1,0",
			preferred.callCount(),
			fallback.callCount(),
			unreached.callCount(),
		)
	}
}

func TestRouterHealthCheckFailsWithoutHealthyProvider(t *testing.T) {
	tests := []struct {
		name   string
		router *ai.Router
	}{
		{name: "no providers", router: ai.NewRouter()},
		{name: "all fail", router: func() *ai.Router {
			router := ai.NewRouter()
			router.Register("failed", &healthProvider{err: errors.New("unavailable")})
			return router
		}()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.router.HealthCheck(context.Background()); err == nil {
				t.Fatal("HealthCheck() error = nil, want failure")
			}
		})
	}
}

func TestRouterHealthCheckHonorsCancellation(t *testing.T) {
	router := ai.NewRouter()
	router.Register("provider", &healthProvider{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := router.HealthCheck(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("HealthCheck() error = %v, want context canceled", err)
	}
}

func TestProbePrimaryCompletionDoesNotRetryOrFallback(t *testing.T) {
	router := ai.NewRouter()
	primary := &healthProvider{err: errors.New("primary unavailable")}
	fallback := &healthProvider{resp: ai.CompletionResponse{Content: "fallback"}}
	router.ReplaceProviders([]ai.ProviderRegistration{
		{Name: "primary", Provider: primary},
		{Name: "fallback", Provider: fallback},
	})

	if _, err := router.ProbePrimaryCompletion(context.Background(), ai.CompletionRequest{}); err == nil {
		t.Fatal("ProbePrimaryCompletion() error = nil, want primary failure")
	}
	if primary.callCount() != 1 {
		t.Fatalf("primary calls = %d, want 1", primary.callCount())
	}
	if fallback.callCount() != 0 {
		t.Fatalf("fallback calls = %d, want 0", fallback.callCount())
	}
}

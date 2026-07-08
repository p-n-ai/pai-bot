// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestRouter_SingleProvider(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider("Hello!")
	router.Register("openai", mock)

	resp, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello!")
	}
}

func TestRouter_Fallback(t *testing.T) {
	router := newTestRouter()

	failing := &ai.MockProvider{Err: errors.New("rate limited")}
	fallback := ai.NewMockProvider("Fallback response")

	router.Register("openai", failing)
	router.Register("ollama", fallback)

	resp, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "Fallback response" {
		t.Errorf("Content = %q, want %q", resp.Content, "Fallback response")
	}
}

func TestRouter_AllProvidersFail(t *testing.T) {
	router := newTestRouter()

	router.Register("openai", &ai.MockProvider{Err: errors.New("fail 1")})
	router.Register("ollama", &ai.MockProvider{Err: errors.New("fail 2")})

	_, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error when all providers fail")
	}
}

func TestRouter_NoProviders(t *testing.T) {
	router := newTestRouter()

	_, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error with no providers")
	}
}

func TestRouter_HasProvider(t *testing.T) {
	router := newTestRouter()
	if router.HasProvider() {
		t.Error("HasProvider() should be false with no providers")
	}

	router.Register("mock", ai.NewMockProvider("ok"))
	if !router.HasProvider() {
		t.Error("HasProvider() should be true after Register")
	}
}

func TestRouter_FallbackOrder(t *testing.T) {
	router := newTestRouter()

	// First registered should be tried first.
	first := ai.NewMockProvider("first")
	second := ai.NewMockProvider("second")

	router.Register("first", first)
	router.Register("second", second)

	resp, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "first" {
		t.Errorf("Content = %q, want %q (first registered should be tried first)", resp.Content, "first")
	}
}

func TestRouter_UsesConfiguredDefaultModelForProvider(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider("Hello!")
	router.ReplaceProviders([]ai.ProviderRegistration{
		{Name: "openai", Provider: mock, DefaultModel: "gpt-4.1-mini"},
	})

	_, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if mock.LastRequest == nil {
		t.Fatal("expected provider to capture request")
	}
	if mock.LastRequest.Model != "gpt-4.1-mini" {
		t.Fatalf("model = %q, want gpt-4.1-mini", mock.LastRequest.Model)
	}
}

func TestRouter_TraceFuncCapturesProviderRequest(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider("Hello!")
	router.ReplaceProviders([]ai.ProviderRegistration{
		{Name: "openai", Provider: mock, DefaultModel: "gpt-4.1-mini"},
	})

	var traces []ai.CompletionTrace
	router.SetTraceFunc(func(trace ai.CompletionTrace) {
		traces = append(traces, trace)
	})

	_, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages:  []ai.Message{{Role: "user", Content: "hi"}},
		Task:      ai.TaskTeaching,
		MaxTokens: 128,
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("trace count = %d, want 1", len(traces))
	}
	trace := traces[0]
	if trace.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", trace.Provider)
	}
	if trace.Request.Model != "gpt-4.1-mini" {
		t.Fatalf("trace request model = %q, want gpt-4.1-mini", trace.Request.Model)
	}
	if trace.Request.Task != ai.TaskTeaching {
		t.Fatalf("trace request task = %v, want teaching", trace.Request.Task)
	}
	if len(trace.Request.Messages) != 1 || trace.Request.Messages[0].Content != "hi" {
		t.Fatalf("trace request messages = %#v", trace.Request.Messages)
	}
	if trace.Response == nil || trace.Response.Content != "Hello!" {
		t.Fatalf("trace response = %#v", trace.Response)
	}
}

func TestRouter_RetryThenSuccess(t *testing.T) {
	router := newTestRouter()
	flaky := &countingProvider{failuresBeforeSuccess: 2, response: "ok"}
	router.Register("flaky", flaky)

	resp, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("Content = %q, want %q", resp.Content, "ok")
	}
	if flaky.calls != 3 {
		t.Fatalf("calls = %d, want 3", flaky.calls)
	}
}

func TestRouter_CircuitBreakerOpensAndFallsBack(t *testing.T) {
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{1 * time.Millisecond},
		BreakerFailureThreshold: 1, // open immediately after one failed request
		BreakerCooldown:         50 * time.Millisecond,
	})

	alwaysFail := &countingProvider{failuresBeforeSuccess: 99}
	fallback := &countingProvider{response: "fallback"}
	router.Register("primary", alwaysFail)
	router.Register("secondary", fallback)

	// First request fails primary and succeeds fallback.
	resp1, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("first Complete() error = %v", err)
	}
	if resp1.Content != "fallback" {
		t.Fatalf("first response = %q, want fallback", resp1.Content)
	}
	firstCalls := alwaysFail.calls
	if firstCalls == 0 {
		t.Fatal("expected primary to be attempted at least once")
	}

	// Second request should skip primary while circuit is open.
	resp2, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "again"}},
	})
	if err != nil {
		t.Fatalf("second Complete() error = %v", err)
	}
	if resp2.Content != "fallback" {
		t.Fatalf("second response = %q, want fallback", resp2.Content)
	}
	if alwaysFail.calls != firstCalls {
		t.Fatalf("primary should be skipped while circuit open; calls changed %d -> %d", firstCalls, alwaysFail.calls)
	}
}

func newTestRouter() *ai.Router {
	return ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond},
		BreakerFailureThreshold: 3,
		BreakerCooldown:         20 * time.Millisecond,
	})
}

type countingProvider struct {
	failuresBeforeSuccess int
	calls                 int
	response              string
}

func (p *countingProvider) Complete(_ context.Context, _ ai.CompletionRequest) (ai.CompletionResponse, error) {
	p.calls++
	if p.calls <= p.failuresBeforeSuccess {
		return ai.CompletionResponse{}, fmt.Errorf("temporary provider failure")
	}
	return ai.CompletionResponse{
		Content:      p.response,
		Model:        "counting",
		InputTokens:  1,
		OutputTokens: 1,
	}, nil
}

func (p *countingProvider) StreamComplete(_ context.Context, _ ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (p *countingProvider) Models() []ai.ModelInfo {
	return nil
}

func (p *countingProvider) HealthCheck(_ context.Context) error {
	return nil
}

func TestRouter_ReRegisterReplacesInPlace(t *testing.T) {
	router := newTestRouter()
	router.Register("openai", ai.NewMockProvider("first"))
	router.Register("ollama", ai.NewMockProvider("other"))
	router.Register("openai", ai.NewMockProvider("second"))

	order := router.ProviderOrder()
	if len(order) != 2 || order[0] != "openai" || order[1] != "ollama" {
		t.Fatalf("ProviderOrder() = %v, want [openai ollama]", order)
	}

	resp, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "second" {
		t.Errorf("Content = %q, want %q (re-registered provider must replace the old one)", resp.Content, "second")
	}
}

func TestRouter_ReplaceProvidersInvalidatesInFlightBreakerUpdates(t *testing.T) {
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{1 * time.Millisecond},
		BreakerFailureThreshold: 1, // any counted failure opens the circuit
		BreakerCooldown:         time.Minute,
	})

	stale := &blockingFailProvider{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	router.Register("openai", stale)

	done := make(chan error, 1)
	go func() {
		_, err := router.Complete(context.Background(), ai.CompletionRequest{
			Messages: []ai.Message{{Role: "user", Content: "hi"}},
		})
		done <- err
	}()

	<-stale.started
	router.ReplaceProviders([]ai.ProviderRegistration{
		{Name: "openai", Provider: ai.NewMockProvider("fresh")},
	})
	close(stale.release)
	if err := <-done; err == nil {
		t.Fatal("stale in-flight request should fail")
	}

	// The stale failure must not open the fresh provider's circuit.
	resp, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Complete() after swap error = %v", err)
	}
	if resp.Content != "fresh" {
		t.Fatalf("Content = %q, want %q", resp.Content, "fresh")
	}
}

type blockingFailProvider struct {
	started chan struct{}
	release chan struct{}
}

func (p *blockingFailProvider) Complete(_ context.Context, _ ai.CompletionRequest) (ai.CompletionResponse, error) {
	select {
	case p.started <- struct{}{}:
	default:
	}
	<-p.release
	return ai.CompletionResponse{}, errors.New("stale provider failure")
}

func (p *blockingFailProvider) StreamComplete(_ context.Context, _ ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (p *blockingFailProvider) Models() []ai.ModelInfo {
	return nil
}

func (p *blockingFailProvider) HealthCheck(_ context.Context) error {
	return nil
}

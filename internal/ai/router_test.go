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

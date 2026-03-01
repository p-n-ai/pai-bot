package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestRouter_SingleProvider(t *testing.T) {
	router := ai.NewRouter()
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
	router := ai.NewRouter()

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
	router := ai.NewRouter()

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
	router := ai.NewRouter()

	_, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error with no providers")
	}
}

func TestRouter_HasProvider(t *testing.T) {
	router := ai.NewRouter()
	if router.HasProvider() {
		t.Error("HasProvider() should be false with no providers")
	}

	router.Register("mock", ai.NewMockProvider("ok"))
	if !router.HasProvider() {
		t.Error("HasProvider() should be true after Register")
	}
}

func TestRouter_FallbackOrder(t *testing.T) {
	router := ai.NewRouter()

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

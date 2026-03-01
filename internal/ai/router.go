package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Router selects the best provider based on task type and availability.
type Router struct {
	providers map[string]Provider
	fallback  []string // ordered fallback chain
	mu        sync.RWMutex
}

// NewRouter creates a new AI router.
func NewRouter() *Router {
	return &Router{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the router.
func (r *Router) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
	r.fallback = append(r.fallback, name)
}

// Complete routes a request to the best available provider.
func (r *Router) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try each provider in fallback order.
	for _, name := range r.fallback {
		provider := r.providers[name]

		resp, err := provider.Complete(ctx, req)
		if err != nil {
			slog.Warn("AI provider failed, trying next",
				"provider", name,
				"error", err,
			)
			continue
		}

		slog.Debug("AI request completed",
			"provider", name,
			"model", resp.Model,
			"input_tokens", resp.InputTokens,
			"output_tokens", resp.OutputTokens,
		)
		return resp, nil
	}

	return CompletionResponse{}, fmt.Errorf("all AI providers failed")
}

// HasProvider returns true if at least one provider is registered.
func (r *Router) HasProvider() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers) > 0
}

package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Router selects the best provider based on task type and availability.
type Router struct {
	providers               map[string]Provider
	capabilitiesByProvider  map[string]ProviderCapabilities
	fallback                []string // ordered fallback chain
	retryBackoff            []time.Duration
	breakerFailureThreshold int
	breakerCooldown         time.Duration
	breakerStateByProvider  map[string]breakerState
	mu                      sync.RWMutex
}

type breakerState struct {
	consecutiveFailures int
	openUntil           time.Time
}

// RouterConfig defines retry and circuit-breaker behavior.
type RouterConfig struct {
	RetryBackoff            []time.Duration
	BreakerFailureThreshold int
	BreakerCooldown         time.Duration
}

// NewRouter creates a new AI router.
func NewRouter() *Router {
	return NewRouterWithConfig(RouterConfig{})
}

// NewRouterWithConfig creates a new AI router with custom retry/circuit-breaker settings.
func NewRouterWithConfig(cfg RouterConfig) *Router {
	retryBackoff := cfg.RetryBackoff
	if len(retryBackoff) == 0 {
		retryBackoff = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	}
	breakerThreshold := cfg.BreakerFailureThreshold
	if breakerThreshold <= 0 {
		breakerThreshold = 3
	}
	breakerCooldown := cfg.BreakerCooldown
	if breakerCooldown <= 0 {
		breakerCooldown = 30 * time.Second
	}
	return &Router{
		providers:               make(map[string]Provider),
		capabilitiesByProvider:  make(map[string]ProviderCapabilities),
		retryBackoff:            retryBackoff,
		breakerFailureThreshold: breakerThreshold,
		breakerCooldown:         breakerCooldown,
		breakerStateByProvider:  make(map[string]breakerState),
	}
}

// Register adds a provider to the router.
func (r *Router) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
	r.capabilitiesByProvider[name] = detectProviderCapabilities(provider)
	r.fallback = append(r.fallback, name)
	if _, ok := r.breakerStateByProvider[name]; !ok {
		r.breakerStateByProvider[name] = breakerState{}
	}
}

// Complete routes a request to the best available provider.
func (r *Router) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	providers, order := r.snapshotProviders()
	if len(order) == 0 {
		return CompletionResponse{}, fmt.Errorf("all AI providers failed (no providers registered)")
	}

	var failures []string
	for _, name := range order {
		provider := providers[name]
		if provider == nil {
			continue
		}
		if r.isCircuitOpen(name) {
			failures = append(failures, fmt.Sprintf("%s: circuit open", name))
			continue
		}

		resp, err := r.completeWithRetry(ctx, provider, req)
		if err != nil {
			r.markFailure(name)
			slog.Warn("AI provider failed, trying next",
				"provider", name,
				"error", err,
			)
			failures = append(failures, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		r.markSuccess(name)
		slog.Debug("AI request completed",
			"provider", name,
			"model", resp.Model,
			"input_tokens", resp.InputTokens,
			"output_tokens", resp.OutputTokens,
		)
		return resp, nil
	}

	return CompletionResponse{}, fmt.Errorf("all AI providers failed: %s", strings.Join(failures, "; "))
}

// HasProvider returns true if at least one provider is registered.
func (r *Router) HasProvider() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers) > 0
}

func (r *Router) snapshotProviders() (map[string]Provider, []string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make(map[string]Provider, len(r.providers))
	for name, provider := range r.providers {
		providers[name] = provider
	}
	order := append([]string(nil), r.fallback...)
	return providers, order
}

func (r *Router) snapshotProvidersWithCapabilities() (map[string]Provider, map[string]ProviderCapabilities, []string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make(map[string]Provider, len(r.providers))
	for name, provider := range r.providers {
		providers[name] = provider
	}

	caps := make(map[string]ProviderCapabilities, len(r.capabilitiesByProvider))
	for name, capability := range r.capabilitiesByProvider {
		caps[name] = capability
	}

	order := append([]string(nil), r.fallback...)
	return providers, caps, order
}

func detectProviderCapabilities(provider Provider) ProviderCapabilities {
	_, structured := provider.(StructuredProvider)
	return ProviderCapabilities{
		StructuredOutput: structured,
	}
}

func (r *Router) completeWithRetry(ctx context.Context, provider Provider, req CompletionRequest) (CompletionResponse, error) {
	var lastErr error
	attempts := 1 + len(r.retryBackoff)

	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := provider.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt == attempts {
			break
		}

		delay := r.retryBackoff[attempt-1]
		select {
		case <-ctx.Done():
			return CompletionResponse{}, ctx.Err()
		case <-time.After(delay):
		}
	}

	return CompletionResponse{}, lastErr
}

func (r *Router) completeStructured(ctx context.Context, req InvocationRequest) (CompletionResponse, error) {
	providers, capabilities, order := r.snapshotProvidersWithCapabilities()
	if len(order) == 0 {
		return CompletionResponse{}, &StructuredRouteError{}
	}

	selectedProviderName := ""
	var selectedProvider Provider
	selectedCapabilities := ProviderCapabilities{}
	for _, name := range order {
		provider := providers[name]
		if provider == nil {
			continue
		}
		selectedProviderName = name
		selectedProvider = provider
		selectedCapabilities = capabilities[name]
		break
	}
	if selectedProvider == nil {
		return CompletionResponse{}, &StructuredRouteError{}
	}

	targetProviderName := selectedProviderName
	targetProvider := selectedProvider
	targetCapabilities := selectedCapabilities

	if !selectedCapabilities.StructuredOutput {
		openaiProvider, openaiExists := providers["openai"]
		openaiCapabilities := capabilities["openai"]
		if !openaiExists || !openaiCapabilities.StructuredOutput || openaiProvider == nil {
			return CompletionResponse{}, &StructuredRouteError{
				SelectedProvider:         selectedProviderName,
				OpenAIFallbackConfigured: openaiExists,
				OpenAIFallbackStructured: openaiCapabilities.StructuredOutput,
			}
		}

		targetProviderName = "openai"
		targetProvider = openaiProvider
		targetCapabilities = openaiCapabilities
	}

	structuredProvider, ok := targetProvider.(StructuredProvider)
	if !ok || !targetCapabilities.StructuredOutput {
		return CompletionResponse{}, &StructuredRouteError{
			SelectedProvider:         selectedProviderName,
			OpenAIFallbackConfigured: providers["openai"] != nil,
			OpenAIFallbackStructured: capabilities["openai"].StructuredOutput,
		}
	}

	if r.isCircuitOpen(targetProviderName) {
		return CompletionResponse{}, fmt.Errorf("%s: circuit open", targetProviderName)
	}

	resp, err := r.completeStructuredWithRetry(ctx, structuredProvider, req)
	if err != nil {
		r.markFailure(targetProviderName)
		return CompletionResponse{}, err
	}

	r.markSuccess(targetProviderName)
	return resp, nil
}

func (r *Router) completeStructuredWithRetry(ctx context.Context, provider StructuredProvider, req InvocationRequest) (CompletionResponse, error) {
	var lastErr error
	attempts := 1 + len(r.retryBackoff)

	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := provider.CompleteStructured(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt == attempts {
			break
		}

		delay := r.retryBackoff[attempt-1]
		select {
		case <-ctx.Done():
			return CompletionResponse{}, ctx.Err()
		case <-time.After(delay):
		}
	}

	return CompletionResponse{}, lastErr
}

func (r *Router) isCircuitOpen(providerName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state := r.breakerStateByProvider[providerName]
	return time.Now().Before(state.openUntil)
}

func (r *Router) markFailure(providerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.breakerStateByProvider[providerName]
	state.consecutiveFailures++
	if state.consecutiveFailures >= r.breakerFailureThreshold {
		state.openUntil = time.Now().Add(r.breakerCooldown)
		state.consecutiveFailures = 0
		slog.Warn("AI provider circuit opened",
			"provider", providerName,
			"cooldown_seconds", int(r.breakerCooldown.Seconds()),
		)
	}
	r.breakerStateByProvider[providerName] = state
}

func (r *Router) markSuccess(providerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.breakerStateByProvider[providerName]
	state.consecutiveFailures = 0
	state.openUntil = time.Time{}
	r.breakerStateByProvider[providerName] = state
}

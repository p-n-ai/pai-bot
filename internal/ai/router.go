package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Router selects the best provider based on task type and availability.
type Router struct {
	providers               map[string]Provider
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

// CompleteJSON requests structured JSON output and unmarshals it into out.
// If no model is specified, it prefers a cheap default per provider.
func (r *Router) CompleteJSON(ctx context.Context, req CompletionRequest, out any) (CompletionResponse, error) {
	if err := validateCompleteJSONRequest(req, out); err != nil {
		return CompletionResponse{}, err
	}

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

		providerReq := structuredProviderRequest(name, req)
		resp, err := r.completeWithRetry(ctx, provider, providerReq)
		if err == nil {
			raw, payloadErr := completeJSONPayload(resp)
			if payloadErr == nil {
				payloadErr = unmarshalStructuredOutput(raw, out)
			}
			if payloadErr == nil {
				r.markSuccess(name)
				resp.StructuredOutput = raw
				slog.Debug("AI structured request completed",
					"provider", name,
					"model", resp.Model,
					"input_tokens", resp.InputTokens,
					"output_tokens", resp.OutputTokens,
				)
				return resp, nil
			}
			err = payloadErr
		}

		r.markFailure(name)
		slog.Warn("AI provider failed structured request, trying next",
			"provider", name,
			"error", err,
		)
		failures = append(failures, fmt.Sprintf("%s: %v", name, err))
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

func validateCompleteJSONRequest(req CompletionRequest, out any) error {
	if req.StructuredOutput == nil {
		return fmt.Errorf("structured output spec is required")
	}
	if req.StructuredOutput.Name == "" {
		return fmt.Errorf("structured output name is required")
	}
	if len(req.StructuredOutput.JSONSchema) == 0 {
		return fmt.Errorf("structured output JSON schema is required")
	}

	target := reflect.ValueOf(out)
	if !target.IsValid() || target.Kind() != reflect.Ptr || target.IsNil() {
		return fmt.Errorf("output target must be a non-nil pointer")
	}

	return nil
}

func structuredProviderRequest(providerName string, req CompletionRequest) CompletionRequest {
	if req.Model != "" {
		return req
	}

	req.Model = defaultStructuredModelForProvider(providerName)
	return req
}

func defaultStructuredModelForProvider(providerName string) string {
	switch providerName {
	case "openai":
		return "gpt-4o-mini"
	case "anthropic":
		return "claude-haiku-4-5-20251001"
	case "deepseek":
		return "deepseek-chat"
	case "google":
		return "gemini-2.5-flash"
	case "ollama":
		return "llama3:8b"
	case "openrouter":
		return "qwen/qwen-2.5-72b-instruct"
	default:
		return ""
	}
}

func completeJSONPayload(resp CompletionResponse) (json.RawMessage, error) {
	if len(resp.StructuredOutput) > 0 {
		raw := json.RawMessage(bytesTrimSpace(resp.StructuredOutput))
		if !json.Valid(raw) {
			return nil, fmt.Errorf("provider returned invalid structured JSON")
		}
		return raw, nil
	}

	raw := json.RawMessage(bytesTrimSpace([]byte(resp.Content)))
	if !json.Valid(raw) {
		return nil, fmt.Errorf("provider returned invalid JSON")
	}
	return raw, nil
}

func unmarshalStructuredOutput(raw json.RawMessage, out any) error {
	target := reflect.ValueOf(out)
	temp := reflect.New(target.Elem().Type())
	if err := json.Unmarshal(raw, temp.Interface()); err != nil {
		return fmt.Errorf("unmarshal structured output: %w", err)
	}
	target.Elem().Set(temp.Elem())
	return nil
}

func bytesTrimSpace(raw []byte) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

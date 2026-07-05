// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/xeipuuv/gojsonschema"
)

// Router selects the best provider based on task type and availability.
type Router struct {
	providers               map[string]Provider
	fallback                []string // ordered fallback chain
	defaultModels           map[string]string
	retryBackoff            []time.Duration
	breakerFailureThreshold int
	breakerCooldown         time.Duration
	breakerStateByProvider  map[string]breakerState
	structuredBreakerState  map[string]breakerState
	traceFunc               func(CompletionTrace)
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
		defaultModels:           make(map[string]string),
		retryBackoff:            retryBackoff,
		breakerFailureThreshold: breakerThreshold,
		breakerCooldown:         breakerCooldown,
		breakerStateByProvider:  make(map[string]breakerState),
		structuredBreakerState:  make(map[string]breakerState),
	}
}

// Register adds a provider to the router. Re-registering an existing name
// replaces the provider in place so the fallback order stays stable and
// duplicate-free (required for live re-apply of runtime settings).
func (r *Router) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[name]; !exists {
		r.fallback = append(r.fallback, name)
	}
	r.providers[name] = provider
	if _, ok := r.breakerStateByProvider[name]; !ok {
		r.breakerStateByProvider[name] = breakerState{}
	}
	if _, ok := r.structuredBreakerState[name]; !ok {
		r.structuredBreakerState[name] = breakerState{}
	}
}

// ProviderRegistration pairs a provider with its optional default model; slice order is the fallback order.
type ProviderRegistration struct {
	Name         string
	Provider     Provider
	DefaultModel string
}

// ReplaceProviders atomically swaps the full provider set; absent providers unregister and breaker state resets.
func (r *Router) ReplaceProviders(regs []ProviderRegistration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]Provider, len(regs))
	r.fallback = nil
	r.defaultModels = make(map[string]string, len(regs))
	r.breakerStateByProvider = make(map[string]breakerState, len(regs))
	r.structuredBreakerState = make(map[string]breakerState, len(regs))
	for _, reg := range regs {
		name := strings.TrimSpace(reg.Name)
		if name == "" || reg.Provider == nil {
			continue
		}
		if _, dup := r.providers[name]; dup {
			continue
		}
		r.providers[name] = reg.Provider
		r.fallback = append(r.fallback, name)
		if model := strings.TrimSpace(reg.DefaultModel); model != "" {
			r.defaultModels[name] = model
		}
		r.breakerStateByProvider[name] = breakerState{}
		r.structuredBreakerState[name] = breakerState{}
	}
}

// SetProviderOrder reorders the fallback chain to match order, skipping names
// that are not registered. Registered providers missing from order keep their
// current relative position at the end.
func (r *Router) SetProviderOrder(order []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := make([]string, 0, len(r.fallback))
	seen := make(map[string]struct{}, len(r.fallback))
	for _, name := range order {
		if _, registered := r.providers[name]; !registered {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		next = append(next, name)
	}
	for _, name := range r.fallback {
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		next = append(next, name)
	}
	r.fallback = next
}

// ProviderOrder returns the current fallback order.
func (r *Router) ProviderOrder() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.fallback...)
}

// SetDefaultModel sets the provider-specific default model used when a request
// does not specify one explicitly.
func (r *Router) SetDefaultModel(providerName, model string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	providerName = strings.TrimSpace(providerName)
	model = strings.TrimSpace(model)
	if providerName == "" {
		return
	}
	if model == "" {
		delete(r.defaultModels, providerName)
		return
	}
	r.defaultModels[providerName] = model
}

// SetTraceFunc registers an opt-in observer for local debugging of provider
// calls. Production callers should leave this unset because requests can
// contain raw conversation and prompt content.
func (r *Router) SetTraceFunc(traceFunc func(CompletionTrace)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.traceFunc = traceFunc
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

		providerReq := req
		if providerReq.Model == "" {
			providerReq.Model = r.defaultModelForProvider(name)
		}
		startedAt := time.Now()
		resp, err := r.completeWithRetry(ctx, provider, providerReq)
		r.emitTrace(CompletionTrace{
			Provider:    name,
			Request:     providerReq,
			Response:    completionResponsePtr(resp, err),
			Error:       completionErrorString(err),
			StartedAt:   startedAt,
			CompletedAt: time.Now(),
		})
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
		providerReq, ok := r.structuredProviderRequest(name, req)
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: structured output unsupported", name))
			continue
		}
		if r.isCircuitOpen(name) {
			failures = append(failures, fmt.Sprintf("%s: circuit open", name))
			continue
		}
		if r.isStructuredCircuitOpen(name) {
			failures = append(failures, fmt.Sprintf("%s: structured circuit open", name))
			continue
		}

		startedAt := time.Now()
		resp, err := r.completeWithRetry(ctx, provider, providerReq)
		trace := CompletionTrace{
			Provider:    name,
			Request:     providerReq,
			Response:    completionResponsePtr(resp, err),
			Error:       completionErrorString(err),
			StartedAt:   startedAt,
			CompletedAt: time.Now(),
		}
		if err != nil {
			r.emitTrace(trace)
			r.markFailure(name)
			slog.Warn("AI provider failed structured request, trying next",
				"provider", name,
				"error", err,
			)
			failures = append(failures, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		raw, payloadErr := completeJSONPayload(resp)
		if payloadErr == nil {
			payloadErr = validateStructuredJSONPayload(raw, providerReq.StructuredOutput)
		}
		if payloadErr == nil {
			payloadErr = unmarshalStructuredOutput(raw, out)
		}
		if payloadErr != nil {
			trace.Error = payloadErr.Error()
			r.emitTrace(trace)
			r.markStructuredFailure(name)
			slog.Warn("AI provider returned invalid structured payload, trying next",
				"provider", name,
				"error", payloadErr,
			)
			failures = append(failures, fmt.Sprintf("%s: %v", name, payloadErr))
			continue
		}

		r.markSuccess(name)
		r.markStructuredSuccess(name)
		resp.StructuredOutput = raw
		trace.Response = &resp
		r.emitTrace(trace)
		slog.Debug("AI structured request completed",
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

func (r *Router) emitTrace(trace CompletionTrace) {
	r.mu.RLock()
	traceFunc := r.traceFunc
	r.mu.RUnlock()
	if traceFunc != nil {
		traceFunc(trace)
	}
}

func completionResponsePtr(resp CompletionResponse, err error) *CompletionResponse {
	if err != nil {
		return nil
	}
	return &resp
}

func completionErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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

func (r *Router) isStructuredCircuitOpen(providerName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state := r.structuredBreakerState[providerName]
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

func (r *Router) markStructuredFailure(providerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.structuredBreakerState[providerName]
	state.consecutiveFailures++
	if state.consecutiveFailures >= r.breakerFailureThreshold {
		state.openUntil = time.Now().Add(r.breakerCooldown)
		state.consecutiveFailures = 0
		slog.Warn("AI provider structured-output circuit opened",
			"provider", providerName,
			"cooldown_seconds", int(r.breakerCooldown.Seconds()),
		)
	}
	r.structuredBreakerState[providerName] = state
}

func (r *Router) markStructuredSuccess(providerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.structuredBreakerState[providerName]
	state.consecutiveFailures = 0
	state.openUntil = time.Time{}
	r.structuredBreakerState[providerName] = state
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
	if _, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(req.StructuredOutput.JSONSchema)); err != nil {
		return fmt.Errorf("invalid structured output JSON schema: %w", err)
	}

	target := reflect.ValueOf(out)
	if !target.IsValid() || target.Kind() != reflect.Ptr || target.IsNil() {
		return fmt.Errorf("output target must be a non-nil pointer")
	}

	return nil
}

func (r *Router) structuredProviderRequest(providerName string, req CompletionRequest) (CompletionRequest, bool) {
	capabilities, ok := structuredProviderCapabilities(providerName)
	if !ok || !capabilities.StructuredOutput {
		return CompletionRequest{}, false
	}
	if requestNeedsSystemPrompt(req) && !capabilities.SystemMessages {
		return CompletionRequest{}, false
	}
	if requestNeedsImageInputs(req) && !capabilities.ImageInputs {
		return CompletionRequest{}, false
	}
	if req.Model != "" {
		return req, true
	}

	req.Model = r.structuredDefaultModelForProvider(providerName)
	return req, true
}

type providerStructuredCapabilities struct {
	StructuredOutput bool
	SystemMessages   bool
	ImageInputs      bool
}

func structuredProviderCapabilities(providerName string) (providerStructuredCapabilities, bool) {
	switch providerName {
	case "openai", "deepseek", "openrouter", "google", "anthropic":
		return providerStructuredCapabilities{
			StructuredOutput: true,
			SystemMessages:   true,
			ImageInputs:      true,
		}, true
	default:
		return providerStructuredCapabilities{}, false
	}
}

func requestNeedsSystemPrompt(req CompletionRequest) bool {
	for _, message := range req.Messages {
		if message.Role == "system" && message.Content != "" {
			return true
		}
	}
	return false
}

func requestNeedsImageInputs(req CompletionRequest) bool {
	for _, message := range req.Messages {
		if len(message.ImageURLs) > 0 {
			return true
		}
	}
	return false
}

func hardcodedStructuredModelForProvider(providerName string) string {
	switch providerName {
	case "openai":
		return "gpt-5.4-mini"
	case "deepseek":
		return "deepseek-chat"
	case "openrouter":
		return "qwen/qwen3-max"
	case "google":
		return "gemini-3-flash-preview"
	case "anthropic":
		return "claude-haiku-4-5-20251001"
	default:
		return ""
	}
}

func (r *Router) defaultModelForProvider(providerName string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return strings.TrimSpace(r.defaultModels[providerName])
}

func (r *Router) structuredDefaultModelForProvider(providerName string) string {
	if model := r.defaultModelForProvider(providerName); model != "" {
		return model
	}
	return hardcodedStructuredModelForProvider(providerName)
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

func validateStructuredJSONPayload(raw json.RawMessage, spec *StructuredOutputSpec) error {
	if spec == nil {
		return nil
	}

	result, err := gojsonschema.Validate(
		gojsonschema.NewBytesLoader(spec.JSONSchema),
		gojsonschema.NewBytesLoader(raw),
	)
	if err != nil {
		return fmt.Errorf("validate structured output schema: %w", err)
	}
	if result.Valid() {
		return nil
	}

	errors := result.Errors()
	if len(errors) == 0 {
		return fmt.Errorf("structured output does not match schema")
	}
	return fmt.Errorf("structured output does not match schema: %s", errors[0])
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

package piai

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StreamFunction is the provider contract. Once invoked, request/model/
// runtime failures are encoded in the returned stream as a terminal error
// event — never returned as a Go error or panic. Cancellation comes from ctx
// and terminates the stream with stopReason "aborted".
type StreamFunction func(ctx context.Context, model Model, c Context, opts *StreamOptions) *EventStream

type registeredProvider struct {
	stream   StreamFunction
	sourceID string
}

var (
	registryMu sync.RWMutex
	registry   = map[string]registeredProvider{}
)

// RegisterProvider registers a StreamFunction for an API shape. sourceID
// groups registrations for UnregisterProviders (test cleanup).
func RegisterProvider(api string, stream StreamFunction, sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[api] = registeredProvider{stream: stream, sourceID: sourceID}
}

// UnregisterProviders removes all providers registered under sourceID.
func UnregisterProviders(sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for api, entry := range registry {
		if entry.sourceID == sourceID {
			delete(registry, api)
		}
	}
}

// Stream starts a completion against model.API's registered provider. A
// missing provider is reported as a terminal error event on the returned
// stream, keeping one failure path for callers.
func Stream(ctx context.Context, model Model, c Context, opts *StreamOptions) *EventStream {
	registryMu.RLock()
	entry, ok := registry[model.API]
	registryMu.RUnlock()
	if !ok {
		s := NewEventStream()
		s.endWithError(AssistantMessage{
			API:          model.API,
			Provider:     model.Provider,
			Model:        model.ID,
			StopReason:   StopReasonError,
			ErrorMessage: fmt.Sprintf("no API provider registered for api: %s", model.API),
			Timestamp:    time.Now(),
		})
		return s
	}
	return entry.stream(ctx, model, c, opts)
}

// Complete runs Stream to completion and returns the final message.
func Complete(ctx context.Context, model Model, c Context, opts *StreamOptions) (AssistantMessage, error) {
	return Stream(ctx, model, c, opts).Result()
}

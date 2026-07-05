package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type StreamFunction func(ctx context.Context, model Model, c Context, opts *StreamOptions) *EventStream

type registeredProvider struct {
	stream   StreamFunction
	sourceID string
}

var (
	registryMu sync.RWMutex
	registry   = map[string]registeredProvider{}
)

func RegisterProvider(api string, stream StreamFunction, sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[api] = registeredProvider{stream: stream, sourceID: sourceID}
}

func UnregisterProviders(sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for api, entry := range registry {
		if entry.sourceID == sourceID {
			delete(registry, api)
		}
	}
}

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

func Complete(ctx context.Context, model Model, c Context, opts *StreamOptions) (AssistantMessage, error) {
	return Stream(ctx, model, c, opts).Result()
}

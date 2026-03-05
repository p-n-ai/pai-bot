package ai

import (
	"context"
	"errors"
	"fmt"
)

// StructuredProvider is an optional provider capability for native structured responses.
type StructuredProvider interface {
	CompleteStructured(ctx context.Context, req InvocationRequest) (CompletionResponse, error)
}

// ProviderCapabilities tracks optional provider features used by routing.
type ProviderCapabilities struct {
	StructuredOutput bool
}

// ErrStructuredRouteUnavailable indicates there is no structured-capable route for the request.
var ErrStructuredRouteUnavailable = errors.New("structured-capable route unavailable")

// StructuredRouteError explains why a structured-capable route could not be selected.
type StructuredRouteError struct {
	SelectedProvider         string
	OpenAIFallbackConfigured bool
	OpenAIFallbackStructured bool
}

func (e *StructuredRouteError) Error() string {
	return fmt.Sprintf(
		"structured route unavailable: selected_provider=%q openai_configured=%t openai_structured=%t",
		e.SelectedProvider,
		e.OpenAIFallbackConfigured,
		e.OpenAIFallbackStructured,
	)
}

func (e *StructuredRouteError) Unwrap() error {
	return ErrStructuredRouteUnavailable
}

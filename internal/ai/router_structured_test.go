package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestRouterInvoke_StructuredRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		setup                  func(*ai.Router) (*structuredTestProvider, *structuredTestProvider)
		wantContent            string
		wantSelectedStructured int
		wantOpenAIStructured   int
		wantErrIsUnavailable   bool
		wantErrType            bool
	}{
		{
			name: "selected provider supports structured",
			setup: func(r *ai.Router) (*structuredTestProvider, *structuredTestProvider) {
				selected := &structuredTestProvider{structuredResponse: "selected-structured"}
				openai := &structuredTestProvider{structuredResponse: "openai-structured"}
				r.Register("anthropic", selected)
				r.Register("openai", openai)
				return selected, openai
			},
			wantContent:            "selected-structured",
			wantSelectedStructured: 1,
			wantOpenAIStructured:   0,
		},
		{
			name: "selected provider lacks structured, fallback to openai structured",
			setup: func(r *ai.Router) (*structuredTestProvider, *structuredTestProvider) {
				r.Register("anthropic", ai.NewMockProvider("anthropic-text"))
				openai := &structuredTestProvider{structuredResponse: "openai-structured"}
				r.Register("openai", openai)
				return nil, openai
			},
			wantContent:          "openai-structured",
			wantOpenAIStructured: 1,
		},
		{
			name: "no structured-capable route returns typed error",
			setup: func(r *ai.Router) (*structuredTestProvider, *structuredTestProvider) {
				r.Register("anthropic", ai.NewMockProvider("anthropic-text"))
				r.Register("openai", ai.NewMockProvider("openai-text"))
				return nil, nil
			},
			wantErrIsUnavailable: true,
			wantErrType:          true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter()
			selectedStructured, openaiStructured := tt.setup(router)

			resp, err := router.Invoke(
				context.Background(),
				ai.NewInvocationRequest(
					[]ai.Message{{Role: "user", Content: "grade"}},
					ai.WithResponseSchema("grade_result", map[string]any{
						"type": "object",
					}, true),
				),
			)

			if tt.wantErrIsUnavailable {
				if !errors.Is(err, ai.ErrStructuredRouteUnavailable) {
					t.Fatalf("Invoke() error = %v, want errors.Is(_, ErrStructuredRouteUnavailable)", err)
				}

				var typedErr *ai.StructuredRouteError
				if !errors.As(err, &typedErr) {
					t.Fatalf("Invoke() error = %T, want *StructuredRouteError", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Invoke() error = %v", err)
			}
			if resp.Content != tt.wantContent {
				t.Fatalf("response content = %q, want %q", resp.Content, tt.wantContent)
			}

			if selectedStructured != nil && selectedStructured.structuredCalls != tt.wantSelectedStructured {
				t.Fatalf("selected structured calls = %d, want %d", selectedStructured.structuredCalls, tt.wantSelectedStructured)
			}
			if openaiStructured != nil && openaiStructured.structuredCalls != tt.wantOpenAIStructured {
				t.Fatalf("openai structured calls = %d, want %d", openaiStructured.structuredCalls, tt.wantOpenAIStructured)
			}
		})
	}
}

type structuredTestProvider struct {
	structuredResponse string
	structuredErr      error
	structuredCalls    int
}

func (p *structuredTestProvider) Complete(_ context.Context, _ ai.CompletionRequest) (ai.CompletionResponse, error) {
	return ai.CompletionResponse{
		Content: p.structuredResponse,
		Model:   "structured-test",
	}, nil
}

func (p *structuredTestProvider) CompleteStructured(_ context.Context, _ ai.InvocationRequest) (ai.CompletionResponse, error) {
	p.structuredCalls++
	if p.structuredErr != nil {
		return ai.CompletionResponse{}, p.structuredErr
	}
	return ai.CompletionResponse{
		Content: p.structuredResponse,
		Model:   "structured-test",
	}, nil
}

func (p *structuredTestProvider) StreamComplete(_ context.Context, _ ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (p *structuredTestProvider) Models() []ai.ModelInfo {
	return nil
}

func (p *structuredTestProvider) HealthCheck(_ context.Context) error {
	return nil
}

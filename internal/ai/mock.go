package ai

import "context"

// MockProvider is a test double for AI providers.
type MockProvider struct {
	Response    string
	Err         error
	LastRequest *CompletionRequest // captures the last request for inspection
}

// NewMockProvider creates a MockProvider that returns the given response.
func NewMockProvider(response string) *MockProvider {
	return &MockProvider{Response: response}
}

func (m *MockProvider) Complete(_ context.Context, req CompletionRequest) (CompletionResponse, error) {
	m.LastRequest = &req
	if m.Err != nil {
		return CompletionResponse{}, m.Err
	}
	return CompletionResponse{
		Content:      m.Response,
		Model:        "mock",
		InputTokens:  10,
		OutputTokens: len(m.Response),
	}, nil
}

func (m *MockProvider) StreamComplete(_ context.Context, _ CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- StreamChunk{Content: m.Response, Done: true}
	}()
	return ch, nil
}

func (m *MockProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "mock", Name: "Mock Model", MaxTokens: 4096, Description: "Test mock"},
	}
}

func (m *MockProvider) HealthCheck(_ context.Context) error {
	return m.Err
}

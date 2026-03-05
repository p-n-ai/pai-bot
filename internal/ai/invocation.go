package ai

import "context"

// ResponseFormat describes the requested response mode for an invocation.
type ResponseFormat string

const (
	// ResponseFormatText requests a plain text response.
	ResponseFormatText ResponseFormat = "text"
	// ResponseFormatJSONSchema requests a JSON response constrained by schema.
	ResponseFormatJSONSchema ResponseFormat = "json_schema"
)

// ResponseSchema describes the expected JSON schema response contract.
type ResponseSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

// InvocationRequest is a schema-first internal request contract.
type InvocationRequest struct {
	Messages       []Message       `json:"messages"`
	Model          string          `json:"model,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	Task           TaskType        `json:"task,omitempty"`
	ResponseFormat ResponseFormat  `json:"response_format,omitempty"`
	Schema         *ResponseSchema `json:"schema,omitempty"`
}

// CallOption configures an InvocationRequest.
type CallOption func(*InvocationRequest)

// NewInvocationRequest builds an invocation request with defaults and overrides.
func NewInvocationRequest(messages []Message, opts ...CallOption) InvocationRequest {
	req := InvocationRequest{
		Messages:       cloneMessages(messages),
		Task:           TaskTeaching,
		ResponseFormat: ResponseFormatText,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&req)
		}
	}
	return req
}

// WithTask sets the task type.
func WithTask(task TaskType) CallOption {
	return func(req *InvocationRequest) {
		req.Task = task
	}
}

// WithModel sets the preferred model.
func WithModel(model string) CallOption {
	return func(req *InvocationRequest) {
		req.Model = model
	}
}

// WithMaxTokens sets the response token cap.
func WithMaxTokens(maxTokens int) CallOption {
	return func(req *InvocationRequest) {
		req.MaxTokens = maxTokens
	}
}

// WithTemperature sets the sampling temperature.
func WithTemperature(temperature float64) CallOption {
	return func(req *InvocationRequest) {
		req.Temperature = temperature
	}
}

// WithResponseSchema sets JSON schema response mode.
func WithResponseSchema(name string, schema map[string]any, strict bool) CallOption {
	return func(req *InvocationRequest) {
		req.ResponseFormat = ResponseFormatJSONSchema
		req.Schema = &ResponseSchema{
			Name:   name,
			Schema: cloneSchema(schema),
			Strict: strict,
		}
	}
}

// ToCompletionRequest converts invocation requests to the existing completion contract.
func (req InvocationRequest) ToCompletionRequest() CompletionRequest {
	return CompletionRequest{
		Messages:    cloneMessages(req.Messages),
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Task:        req.Task,
	}
}

// Invoke executes an invocation request while keeping legacy completion behavior.
func (r *Router) Invoke(ctx context.Context, req InvocationRequest) (CompletionResponse, error) {
	if req.ResponseFormat == ResponseFormatJSONSchema {
		return r.completeStructured(ctx, req)
	}
	return r.Complete(ctx, req.ToCompletionRequest())
}

// Call provides SDK-style invocation ergonomics with options.
func (r *Router) Call(ctx context.Context, messages []Message, opts ...CallOption) (CompletionResponse, error) {
	req := NewInvocationRequest(messages, opts...)
	return r.Invoke(ctx, req)
}

func cloneMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	out := make([]Message, len(messages))
	for i, message := range messages {
		out[i] = message
		if len(message.ImageURLs) > 0 {
			out[i].ImageURLs = append([]string(nil), message.ImageURLs...)
		}
	}
	return out
}

func cloneSchema(schema map[string]any) map[string]any {
	if len(schema) == 0 {
		return nil
	}
	out := make(map[string]any, len(schema))
	for key, value := range schema {
		out[key] = value
	}
	return out
}

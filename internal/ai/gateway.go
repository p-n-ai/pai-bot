// Package ai provides a provider-agnostic AI gateway with task-based routing.
package ai

import "context"

// TaskType defines the kind of AI task for routing purposes.
type TaskType int

const (
	TaskTeaching TaskType = iota
	TaskGrading
	TaskNudge
	TaskAnalysis
)

func (t TaskType) String() string {
	switch t {
	case TaskTeaching:
		return "teaching"
	case TaskGrading:
		return "grading"
	case TaskNudge:
		return "nudge"
	case TaskAnalysis:
		return "analysis"
	default:
		return "unknown"
	}
}

// Message represents a chat message.
type Message struct {
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	ImageURLs []string `json:"image_urls,omitempty"`
}

// CompletionRequest is the input to an AI completion.
type CompletionRequest struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Task        TaskType  `json:"task,omitempty"`
}

// CompletionResponse is the output from an AI completion.
type CompletionResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// TotalTokens returns the sum of input and output tokens.
func (r CompletionResponse) TotalTokens() int {
	return r.InputTokens + r.OutputTokens
}

// StreamChunk represents a streaming response chunk.
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// ModelInfo describes an available model.
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MaxTokens   int    `json:"max_tokens"`
	Description string `json:"description"`
}

// Provider is the interface all AI providers must implement.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
	Models() []ModelInfo
	HealthCheck(ctx context.Context) error
}

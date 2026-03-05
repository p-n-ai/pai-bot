package ai_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestNewInvocationRequest_Defaults(t *testing.T) {
	input := []ai.Message{
		{Role: "user", Content: "hello"},
	}

	req := ai.NewInvocationRequest(input)

	if req.Task != ai.TaskTeaching {
		t.Fatalf("Task = %v, want %v", req.Task, ai.TaskTeaching)
	}
	if req.ResponseFormat != ai.ResponseFormatText {
		t.Fatalf("ResponseFormat = %q, want %q", req.ResponseFormat, ai.ResponseFormatText)
	}
	if req.Schema != nil {
		t.Fatalf("Schema = %#v, want nil", req.Schema)
	}
	if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
		t.Fatalf("Messages = %#v, want user hello message", req.Messages)
	}

	input[0].Content = "changed"
	if req.Messages[0].Content != "hello" {
		t.Fatalf("Messages should be copied; got %q", req.Messages[0].Content)
	}
}

func TestNewInvocationRequest_Overrides(t *testing.T) {
	req := ai.NewInvocationRequest(
		[]ai.Message{{Role: "user", Content: "grade this"}},
		ai.WithTask(ai.TaskGrading),
		ai.WithModel("gpt-4o-mini"),
		ai.WithMaxTokens(250),
		ai.WithTemperature(0.2),
		ai.WithResponseSchema("grading_result", map[string]any{
			"type": "object",
		}, true),
	)

	if req.Task != ai.TaskGrading {
		t.Fatalf("Task = %v, want %v", req.Task, ai.TaskGrading)
	}
	if req.Model != "gpt-4o-mini" {
		t.Fatalf("Model = %q, want gpt-4o-mini", req.Model)
	}
	if req.MaxTokens != 250 {
		t.Fatalf("MaxTokens = %d, want 250", req.MaxTokens)
	}
	if req.Temperature != 0.2 {
		t.Fatalf("Temperature = %v, want 0.2", req.Temperature)
	}
	if req.ResponseFormat != ai.ResponseFormatJSONSchema {
		t.Fatalf("ResponseFormat = %q, want %q", req.ResponseFormat, ai.ResponseFormatJSONSchema)
	}
	if req.Schema == nil {
		t.Fatal("Schema = nil, want non-nil")
	}
	if req.Schema.Name != "grading_result" {
		t.Fatalf("Schema.Name = %q, want grading_result", req.Schema.Name)
	}
	if req.Schema.Strict != true {
		t.Fatalf("Schema.Strict = %v, want true", req.Schema.Strict)
	}
	if req.Schema.Schema["type"] != "object" {
		t.Fatalf("Schema.Schema[type] = %#v, want object", req.Schema.Schema["type"])
	}
}

func TestRouterCall_BackwardCompatibleCompletePath(t *testing.T) {
	router := ai.NewRouter()
	mock := ai.NewMockProvider("ok")
	router.Register("mock", mock)

	_, err := router.Call(
		context.Background(),
		[]ai.Message{{Role: "user", Content: "hello"}},
		ai.WithTask(ai.TaskAnalysis),
		ai.WithModel("test-model"),
		ai.WithMaxTokens(42),
		ai.WithTemperature(0.9),
	)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if mock.LastRequest == nil {
		t.Fatal("LastRequest = nil, want captured completion request")
	}
	if mock.LastRequest.Task != ai.TaskAnalysis {
		t.Fatalf("LastRequest.Task = %v, want %v", mock.LastRequest.Task, ai.TaskAnalysis)
	}
	if mock.LastRequest.Model != "test-model" {
		t.Fatalf("LastRequest.Model = %q, want test-model", mock.LastRequest.Model)
	}
	if mock.LastRequest.MaxTokens != 42 {
		t.Fatalf("LastRequest.MaxTokens = %d, want 42", mock.LastRequest.MaxTokens)
	}
	if mock.LastRequest.Temperature != 0.9 {
		t.Fatalf("LastRequest.Temperature = %v, want 0.9", mock.LastRequest.Temperature)
	}
	if len(mock.LastRequest.Messages) != 1 || mock.LastRequest.Messages[0].Content != "hello" {
		t.Fatalf("LastRequest.Messages = %#v, want hello", mock.LastRequest.Messages)
	}
}

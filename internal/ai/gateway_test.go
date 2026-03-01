package ai_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestMockProvider_Complete(t *testing.T) {
	mock := ai.NewMockProvider("test response")

	resp, err := mock.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "test response" {
		t.Errorf("Content = %q, want %q", resp.Content, "test response")
	}
	if resp.Model != "mock" {
		t.Errorf("Model = %q, want %q", resp.Model, "mock")
	}
}

func TestMockProvider_HealthCheck(t *testing.T) {
	mock := ai.NewMockProvider("response")
	if err := mock.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestMockProvider_Models(t *testing.T) {
	mock := ai.NewMockProvider("response")
	models := mock.Models()
	if len(models) == 0 {
		t.Error("Models() returned empty")
	}
}

func TestTaskType_String(t *testing.T) {
	tests := []struct {
		task     ai.TaskType
		expected string
	}{
		{ai.TaskTeaching, "teaching"},
		{ai.TaskGrading, "grading"},
		{ai.TaskNudge, "nudge"},
		{ai.TaskAnalysis, "analysis"},
	}
	for _, tt := range tests {
		if tt.task.String() != tt.expected {
			t.Errorf("TaskType.String() = %q, want %q", tt.task.String(), tt.expected)
		}
	}
}

func TestCompletionResponse_TotalTokens(t *testing.T) {
	resp := ai.CompletionResponse{InputTokens: 100, OutputTokens: 50}
	if got := resp.TotalTokens(); got != 150 {
		t.Errorf("TotalTokens() = %d, want 150", got)
	}
}

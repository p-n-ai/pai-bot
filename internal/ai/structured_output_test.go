package ai_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestRouter_PassesStructuredOutputSpecToProvider(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider(`{"final_answer":"ok"}`)
	router.Register("mock", mock)

	schema := json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`)

	_, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "tutor_response",
			JSONSchema: schema,
			Strict:     true,
		},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if mock.LastRequest == nil {
		t.Fatal("expected mock to capture LastRequest")
	}
	if mock.LastRequest.StructuredOutput == nil {
		t.Fatal("expected StructuredOutput spec to be forwarded")
	}
	if mock.LastRequest.StructuredOutput.Name != "tutor_response" {
		t.Fatalf("StructuredOutput.Name = %q, want tutor_response", mock.LastRequest.StructuredOutput.Name)
	}
	if string(mock.LastRequest.StructuredOutput.JSONSchema) != string(schema) {
		t.Fatalf("StructuredOutput.JSONSchema mismatch")
	}
	if !mock.LastRequest.StructuredOutput.Strict {
		t.Fatal("expected StructuredOutput.Strict = true")
	}
}

package ai_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

type structuredReply struct {
	FinalAnswer string `json:"final_answer"`
}

func TestRouter_CompleteJSON_ParsesOutputAndDefaultsToCheapModel(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider(`{"final_answer":"12"}`)
	router.Register("openai", mock)

	var out structuredReply
	resp, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "grade this"}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
			Strict:     true,
		},
	}, &out)
	if err != nil {
		t.Fatalf("CompleteJSON() error = %v", err)
	}

	if out.FinalAnswer != "12" {
		t.Fatalf("parsed output = %#v, want final_answer 12", out)
	}
	if mock.LastRequest == nil {
		t.Fatal("expected provider to capture request")
	}
	if mock.LastRequest.Model != "gpt-4o-mini" {
		t.Fatalf("default structured model = %q, want gpt-4o-mini", mock.LastRequest.Model)
	}
	if string(resp.StructuredOutput) != `{"final_answer":"12"}` {
		t.Fatalf("StructuredOutput = %s, want raw JSON payload", string(resp.StructuredOutput))
	}
}

func TestRouter_CompleteJSON_RequiresStructuredOutputSpec(t *testing.T) {
	router := newTestRouter()
	router.Register("openai", ai.NewMockProvider(`{"final_answer":"12"}`))

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "grade this"}},
	}, &out)
	if err == nil {
		t.Fatal("CompleteJSON() should require structured output spec")
	}
	if !strings.Contains(err.Error(), "structured output") {
		t.Fatalf("error = %v, want structured output validation error", err)
	}
}

func TestRouter_CompleteJSON_FallsBackWhenProviderReturnsInvalidJSON(t *testing.T) {
	router := newTestRouter()
	invalid := ai.NewMockProvider("not json")
	valid := ai.NewMockProvider(`{"final_answer":"fallback"}`)
	router.Register("openai", invalid)
	router.Register("ollama", valid)

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "grade this"}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
		},
	}, &out)
	if err != nil {
		t.Fatalf("CompleteJSON() error = %v", err)
	}

	if out.FinalAnswer != "fallback" {
		t.Fatalf("parsed output = %#v, want fallback response", out)
	}
	if invalid.LastRequest == nil || invalid.LastRequest.Model != "gpt-4o-mini" {
		t.Fatalf("first provider should receive cheap OpenAI model, got %#v", invalid.LastRequest)
	}
	if valid.LastRequest == nil || valid.LastRequest.Model != "llama3:8b" {
		t.Fatalf("fallback provider should receive cheap Ollama model, got %#v", valid.LastRequest)
	}
}

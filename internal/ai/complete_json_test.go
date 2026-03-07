package ai_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	router.Register("openrouter", valid)

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
	if valid.LastRequest == nil || valid.LastRequest.Model != "qwen/qwen-2.5-72b-instruct" {
		t.Fatalf("fallback provider should receive cheap OpenRouter model, got %#v", valid.LastRequest)
	}
}

func TestRouter_CompleteJSON_FallsBackWhenProviderReturnsSchemaInvalidJSON(t *testing.T) {
	router := newTestRouter()
	invalid := ai.NewMockProvider(`{}`)
	valid := ai.NewMockProvider(`{"final_answer":"fallback"}`)
	router.Register("openai", invalid)
	router.Register("openrouter", valid)

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
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

	if out.FinalAnswer != "fallback" {
		t.Fatalf("parsed output = %#v, want fallback response", out)
	}
}

func TestRouter_CompleteJSON_SkipsProvidersWithoutStructuredOutputSupport(t *testing.T) {
	router := newTestRouter()
	skipped := ai.NewMockProvider(`{"final_answer":"wrong"}`)
	supported := ai.NewMockProvider(`{"final_answer":"ok"}`)
	router.Register("ollama", skipped)
	router.Register("openrouter", supported)

	var out structuredReply
	resp, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "grade this"}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
		},
	}, &out)
	if err != nil {
		t.Fatalf("CompleteJSON() error = %v", err)
	}

	if skipped.LastRequest != nil {
		t.Fatalf("unsupported provider should be skipped, got request %#v", skipped.LastRequest)
	}
	if supported.LastRequest == nil {
		t.Fatal("expected supported provider to receive request")
	}
	if out.FinalAnswer != "ok" {
		t.Fatalf("parsed output = %#v, want ok", out)
	}
	if string(resp.StructuredOutput) != `{"final_answer":"ok"}` {
		t.Fatalf("StructuredOutput = %s, want supported provider payload", string(resp.StructuredOutput))
	}
}

func TestRouter_CompleteJSON_UsesGoogleStructuredDefaults(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider(`{"final_answer":"ok"}`)
	router.Register("google", mock)

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

	if mock.LastRequest == nil {
		t.Fatal("expected provider to capture request")
	}
	if mock.LastRequest.Model != "gemini-2.5-flash" {
		t.Fatalf("default structured model = %q, want gemini-2.5-flash", mock.LastRequest.Model)
	}
	if out.FinalAnswer != "ok" {
		t.Fatalf("parsed output = %#v, want ok", out)
	}
}

func TestRouter_CompleteJSON_UsesAnthropicStructuredDefaults(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider(`{"final_answer":"ok"}`)
	router.Register("anthropic", mock)

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

	if mock.LastRequest == nil {
		t.Fatal("expected provider to capture request")
	}
	if mock.LastRequest.Model != "claude-haiku-4-5-20251001" {
		t.Fatalf("default structured model = %q, want claude-haiku-4-5-20251001", mock.LastRequest.Model)
	}
	if out.FinalAnswer != "ok" {
		t.Fatalf("parsed output = %#v, want ok", out)
	}
}

func TestRouter_CompleteJSON_UsesGoogleForStructuredSystemPrompt(t *testing.T) {
	router := newTestRouter()
	google := ai.NewMockProvider(`{"final_answer":"google"}`)
	openai := ai.NewMockProvider(`{"final_answer":"openai"}`)
	router.Register("google", google)
	router.Register("openai", openai)

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "system", Content: "Follow the grading rubric exactly."},
			{Role: "user", Content: "grade this"},
		},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
		},
	}, &out)
	if err != nil {
		t.Fatalf("CompleteJSON() error = %v", err)
	}

	if google.LastRequest == nil {
		t.Fatal("expected google provider to receive request")
	}
	if openai.LastRequest != nil {
		t.Fatalf("fallback provider should not be used, got %#v", openai.LastRequest)
	}
	if out.FinalAnswer != "google" {
		t.Fatalf("parsed output = %#v, want google", out)
	}
}

func TestRouter_CompleteJSON_UsesGoogleForStructuredImageRequest(t *testing.T) {
	router := newTestRouter()
	google := ai.NewMockProvider(`{"final_answer":"google"}`)
	openai := ai.NewMockProvider(`{"final_answer":"openai"}`)
	router.Register("google", google)
	router.Register("openai", openai)

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{
			Role:      "user",
			Content:   "grade this image",
			ImageURLs: []string{"data:image/png;base64,AAEC"},
		}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
		},
	}, &out)
	if err != nil {
		t.Fatalf("CompleteJSON() error = %v", err)
	}

	if google.LastRequest == nil {
		t.Fatal("expected google provider to receive request")
	}
	if openai.LastRequest != nil {
		t.Fatalf("fallback provider should not be used, got %#v", openai.LastRequest)
	}
	if out.FinalAnswer != "google" {
		t.Fatalf("parsed output = %#v, want google", out)
	}
}

func TestRouter_CompleteJSON_UsesAnthropicForStructuredImageRequest(t *testing.T) {
	router := newTestRouter()
	anthropic := ai.NewMockProvider(`{"final_answer":"anthropic"}`)
	openai := ai.NewMockProvider(`{"final_answer":"openai"}`)
	router.Register("anthropic", anthropic)
	router.Register("openai", openai)

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{
			Role:      "user",
			Content:   "grade this image",
			ImageURLs: []string{"https://example.com/image.png"},
		}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
		},
	}, &out)
	if err != nil {
		t.Fatalf("CompleteJSON() error = %v", err)
	}

	if anthropic.LastRequest == nil {
		t.Fatal("expected anthropic provider to receive request")
	}
	if openai.LastRequest != nil {
		t.Fatalf("fallback provider should not be used, got %#v", openai.LastRequest)
	}
	if out.FinalAnswer != "anthropic" {
		t.Fatalf("parsed output = %#v, want anthropic", out)
	}
}

func TestRouter_CompleteJSON_SkipsProvidersWithoutRequiredStructuredCapabilities(t *testing.T) {
	router := newTestRouter()
	skipped := ai.NewMockProvider(`{"final_answer":"wrong"}`)
	supported := ai.NewMockProvider(`{"final_answer":"ok"}`)
	router.Register("custom", skipped)
	router.Register("openai", supported)

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "system", Content: "Follow the rubric."},
			{Role: "user", Content: "grade this", ImageURLs: []string{"data:image/png;base64,AAEC"}},
		},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
		},
	}, &out)
	if err != nil {
		t.Fatalf("CompleteJSON() error = %v", err)
	}

	if skipped.LastRequest != nil {
		t.Fatalf("provider without required capabilities should be skipped, got %#v", skipped.LastRequest)
	}
	if supported.LastRequest == nil {
		t.Fatal("expected supported provider to receive request")
	}
	if out.FinalAnswer != "ok" {
		t.Fatalf("parsed output = %#v, want ok", out)
	}
}

func TestRouter_CompleteJSON_FallsBackWhenGoogleReturnsInvalidJSON(t *testing.T) {
	router := newTestRouter()
	invalid := ai.NewMockProvider("not json")
	valid := ai.NewMockProvider(`{"final_answer":"fallback"}`)
	router.Register("google", invalid)
	router.Register("openai", valid)

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
		t.Fatalf("parsed output = %#v, want fallback", out)
	}
}

func TestRouter_CompleteJSON_FallsBackWhenAnthropicReturnsSchemaInvalidJSON(t *testing.T) {
	router := newTestRouter()
	invalid := ai.NewMockProvider(`{}`)
	valid := ai.NewMockProvider(`{"final_answer":"fallback"}`)
	router.Register("anthropic", invalid)
	router.Register("openai", valid)

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
		t.Fatalf("parsed output = %#v, want fallback", out)
	}
}

func TestRouter_CompleteJSON_RejectsMalformedSchemaBeforeProviderCall(t *testing.T) {
	router := newTestRouter()
	mock := ai.NewMockProvider(`{"final_answer":"ok"}`)
	router.Register("openai", mock)

	var out structuredReply
	_, err := router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "grade this"}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object",`),
		},
	}, &out)
	if err == nil {
		t.Fatal("CompleteJSON() should reject malformed schema")
	}
	if mock.LastRequest != nil {
		t.Fatalf("provider should not be called when schema is malformed, got %#v", mock.LastRequest)
	}
}

func TestRouter_CompleteJSON_InvalidJSONOpensStructuredCircuitOnly(t *testing.T) {
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{1 * time.Millisecond},
		BreakerFailureThreshold: 1,
		BreakerCooldown:         50 * time.Millisecond,
	})

	primary := &capturingProvider{response: "not json"}
	secondary := &capturingProvider{response: `{"final_answer":"fallback"}`}
	router.Register("openai", primary)
	router.Register("openrouter", secondary)

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

	if primary.calls == 0 {
		t.Fatal("expected primary structured provider to be attempted")
	}

	primary.response = `{"final_answer":"should stay skipped"}`
	secondary.response = `{"final_answer":"fallback-again"}`

	out = structuredReply{}
	_, err = router.CompleteJSON(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "grade this again"}},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "grading_result",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`),
		},
	}, &out)
	if err != nil {
		t.Fatalf("second CompleteJSON() error = %v", err)
	}
	if out.FinalAnswer != "fallback-again" {
		t.Fatalf("second parsed output = %#v, want fallback-again", out)
	}
	if primary.calls != 1 {
		t.Fatalf("structured circuit should skip primary on second CompleteJSON; calls = %d, want 1", primary.calls)
	}

	primary.response = "primary text"
	secondary.response = "secondary text"

	resp, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "plain text"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "primary text" {
		t.Fatalf("plain response = %q, want primary provider to remain available", resp.Content)
	}
}

type capturingProvider struct {
	response string
	calls    int
}

func (p *capturingProvider) Complete(_ context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	p.calls++
	return ai.CompletionResponse{
		Content:      p.response,
		Model:        req.Model,
		InputTokens:  1,
		OutputTokens: 1,
	}, nil
}

func (p *capturingProvider) StreamComplete(_ context.Context, _ ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, nil
}

func (p *capturingProvider) Models() []ai.ModelInfo {
	return nil
}

func (p *capturingProvider) HealthCheck(_ context.Context) error {
	return nil
}

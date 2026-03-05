package ai

import (
	"encoding/json"
	"net/http"
	"testing"
)

var testStructuredSchema = json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`)

func assertJSONSchemaResponseFormat(t *testing.T, captured map[string]any) {
	t.Helper()

	raw, ok := captured["response_format"]
	if !ok {
		t.Fatalf("expected request to include response_format, got keys: %v", captured)
	}
	rf, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("response_format type = %T, want map[string]any", raw)
	}
	if rf["type"] != "json_schema" {
		t.Fatalf("response_format.type = %#v, want json_schema", rf["type"])
	}
	jsonSchema, ok := rf["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("response_format.json_schema type = %T, want map[string]any", rf["json_schema"])
	}
	if jsonSchema["name"] != "tutor_response" {
		t.Fatalf("response_format.json_schema.name = %#v, want tutor_response", jsonSchema["name"])
	}
	if strict, ok := jsonSchema["strict"].(bool); !ok || !strict {
		t.Fatalf("response_format.json_schema.strict = %#v, want true", jsonSchema["strict"])
	}
	if _, ok := jsonSchema["schema"]; !ok {
		t.Fatalf("response_format.json_schema.schema missing")
	}
}

func writeOpenAITextResponse(
	t *testing.T,
	w http.ResponseWriter,
	content, model string,
	promptTokens, completionTokens int,
) {
	t.Helper()

	resp := openaiResponse{
		Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{
			{Message: struct {
				Content string `json:"content"`
			}{Content: content}},
		},
		Model: model,
	}
	resp.Usage.PromptTokens = promptTokens
	resp.Usage.CompletionTokens = completionTokens

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

package ai

import (
	"encoding/json"
	"net/http"
	"testing"
)

var testStructuredSchema = json.RawMessage(`{"type":"object","properties":{"final_answer":{"type":"string"}},"required":["final_answer"]}`)

func assertJSONSchemaResponseFormat(t *testing.T, captured map[string]any) {
	t.Helper()

	rf, ok := captured["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("response_format missing or invalid: %#v", captured["response_format"])
	}
	if rf["type"] != "json_schema" {
		t.Fatalf("response_format.type = %#v, want json_schema", rf["type"])
	}
	jsonSchema, ok := rf["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("response_format.json_schema type = %T, want map[string]any", rf["json_schema"])
	}
	if jsonSchema["name"] != "tutor_response" || jsonSchema["strict"] != true {
		t.Fatalf("unexpected response_format.json_schema metadata: %#v", jsonSchema)
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

	resp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{"content": content},
			},
		},
		"model": model,
		"usage": map[string]int{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
		},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

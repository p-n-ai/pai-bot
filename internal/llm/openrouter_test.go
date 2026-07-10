package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/OpenRouterTeam/go-sdk/models/sdkerrors"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

func openRouterModel(baseURL string) llm.Model {
	return llm.Model{
		ID:        "openai/gpt-test",
		Name:      "GPT Test via OpenRouter",
		API:       llm.APIOpenRouterChat,
		Provider:  "openrouter",
		BaseURL:   baseURL + "/v1",
		Reasoning: true,
		Cost:      llm.Cost{Input: 1, Output: 2, CacheRead: 0.5, CacheWrite: 1.5},
	}
}

func openRouterChunk(body string) string { return "data: " + body }

func TestOpenRouterStreamsNativeTextReasoningUsageAndRequest(t *testing.T) {
	srv, captured := sseServer(t, []string{
		openRouterChunk(`{"id":"or-1","model":"anthropic/claude-routed","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"role":"assistant","content":"Ans","reasoning":"Think"},"finish_reason":null}]}`),
		openRouterChunk(`{"id":"or-1","model":"anthropic/claude-routed","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"wer","reasoning":" more"},"finish_reason":null}]}`),
		openRouterChunk(`{"id":"or-1","model":"anthropic/claude-routed","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`),
		openRouterChunk(`{"id":"or-1","model":"anthropic/claude-routed","object":"chat.completion.chunk","created":1,"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":10,"total_tokens":110,"prompt_tokens_details":{"cached_tokens":40,"cache_write_tokens":20},"cost":0.00123}}`),
		"data: [DONE]",
	})

	temperature := 0.3
	model := openRouterModel(srv.URL)
	model.ID = "~anthropic/claude-test"
	stream := llm.StreamOpenRouterChat(
		context.Background(),
		model,
		llm.Context{SystemPrompt: "Be brief.", Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{
			APIKey:          "sk-or-test",
			Temperature:     &temperature,
			MaxTokens:       128,
			SessionID:       "session-1",
			CacheRetention:  llm.CacheRetentionLong,
			ReasoningEffort: llm.ReasoningEffortMinimal,
			Headers: map[string]string{
				"http-referer": "https://review.example",
				"x-title":      "Review Bot",
				"x-test":       "native",
			},
		},
	)
	events := collectEvents(stream)
	msg, err := stream.Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	if msg.ResponseID != "or-1" || msg.Model != "~anthropic/claude-test" || msg.ResponseModel != "anthropic/claude-routed" || msg.StopReason != llm.StopReasonStop {
		t.Fatalf("message identity = %+v", msg)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("content blocks = %d: %#v", len(msg.Content), msg.Content)
	}
	text, ok := msg.Content[0].(llm.TextContent)
	if !ok || text.Text != "Answer" {
		t.Fatalf("text = %#v", msg.Content[0])
	}
	thinking, ok := msg.Content[1].(llm.ThinkingContent)
	if !ok || thinking.Thinking != "Think more" {
		t.Fatalf("thinking = %#v", msg.Content[1])
	}
	if msg.Usage.Input != 40 || msg.Usage.CacheRead != 40 || msg.Usage.CacheWrite != 20 || msg.Usage.Output != 10 {
		t.Fatalf("usage = %+v", msg.Usage)
	}
	requireTotalTokensEqualsComponents(t, msg.Usage)
	if msg.Usage.Cost.Total != 0.00123 {
		t.Fatalf("billed cost = %+v", msg.Usage.Cost)
	}
	if msg.Usage.Cost.Input == 0 || msg.Usage.Cost.Output == 0 || msg.Usage.Cost.CacheRead == 0 || msg.Usage.Cost.CacheWrite == 0 {
		t.Fatalf("cost components = %+v", msg.Usage.Cost)
	}
	components := msg.Usage.Cost.Input + msg.Usage.Cost.Output + msg.Usage.Cost.CacheRead + msg.Usage.Cost.CacheWrite
	if delta := components - msg.Usage.Cost.Total; delta < -1e-12 || delta > 1e-12 {
		t.Fatalf("cost components = %f, total = %f", components, msg.Usage.Cost.Total)
	}

	wantEvents := []llm.EventType{
		llm.EventStart,
		llm.EventTextStart, llm.EventTextDelta,
		llm.EventThinkingStart, llm.EventThinkingDelta,
		llm.EventTextDelta, llm.EventThinkingDelta,
		llm.EventTextEnd, llm.EventThinkingEnd,
		llm.EventDone,
	}
	if got := eventTypes(events); !equalTypes(got, wantEvents...) {
		t.Fatalf("events = %v, want %v", got, wantEvents)
	}

	if captured.method != http.MethodPost || captured.path != "/v1/chat/completions" {
		t.Fatalf("request = %s %s", captured.method, captured.path)
	}
	if captured.headers.Get("Authorization") != "Bearer sk-or-test" {
		t.Fatalf("auth header = %q", captured.headers.Get("Authorization"))
	}
	if captured.headers.Get("HTTP-Referer") != "https://review.example" || captured.headers.Get("X-Title") != "Review Bot" {
		t.Fatalf("attribution headers = %+v", captured.headers)
	}
	if captured.headers.Get("X-Test") != "native" || captured.headers.Get("Accept") != "text/event-stream" {
		t.Fatalf("request headers = %+v", captured.headers)
	}
	if captured.body["model"] != "~anthropic/claude-test" || captured.body["stream"] != true {
		t.Fatalf("body = %+v", captured.body)
	}
	if captured.body["temperature"] != 0.3 || captured.body["max_completion_tokens"] != float64(128) {
		t.Fatalf("params = %+v", captured.body)
	}
	if captured.body["reasoning_effort"] != "minimal" {
		t.Fatalf("reasoning_effort = %#v", captured.body["reasoning_effort"])
	}
	if captured.body["session_id"] != "session-1" {
		t.Fatalf("session_id = %+v", captured.body["session_id"])
	}
	if _, ok := captured.body["tools"]; ok {
		t.Fatalf("tools must be omitted when no tools are configured: %+v", captured.body["tools"])
	}
	if _, ok := captured.body["response_format"]; ok {
		t.Fatalf("response_format must be omitted when structured output is not configured: %+v", captured.body["response_format"])
	}
	cache := captured.body["cache_control"].(map[string]any)
	if cache["type"] != "ephemeral" || cache["ttl"] != "1h" {
		t.Fatalf("cache_control = %+v", cache)
	}
	streamOptions := captured.body["stream_options"].(map[string]any)
	if streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %+v", streamOptions)
	}
	messages := captured.body["messages"].([]any)
	first := messages[0].(map[string]any)
	if first["role"] != "developer" || first["content"] != "Be brief." {
		t.Fatalf("developer message = %+v", first)
	}
}

func TestOpenRouterPreservesOrderedSystemMessages(t *testing.T) {
	srv, captured := sseServer(t, []string{
		openRouterChunk(`{"id":"or-ordered","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})

	_, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{
			SystemPrompt: "leading",
			Messages: []llm.Message{
				llm.SystemMessage{Content: "before"},
				llm.UserText("question"),
				llm.SystemMessage{Content: "between"},
				llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "answer"}}},
				llm.SystemMessage{Content: "after"},
			},
		},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	messages := captured.body["messages"].([]any)
	want := []struct {
		role    string
		content string
	}{
		{role: "developer", content: "leading"},
		{role: "developer", content: "before"},
		{role: "user", content: "question"},
		{role: "developer", content: "between"},
		{role: "assistant", content: "answer"},
		{role: "developer", content: "after"},
	}
	if len(messages) != len(want) {
		t.Fatalf("messages = %#v, want %d entries", messages, len(want))
	}
	for i, expected := range want {
		message := messages[i].(map[string]any)
		if message["role"] != expected.role || message["content"] != expected.content {
			t.Fatalf("message %d = %#v, want role %q content %q", i, message, expected.role, expected.content)
		}
	}
}

func TestOpenRouterSendsStructuredOutputResponseFormat(t *testing.T) {
	srv, captured := sseServer(t, []string{
		openRouterChunk(`{"id":"or-structured","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"{\"answer\":\"ok\"}"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})
	schema := json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`)

	_, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("return JSON")}},
		&llm.StreamOptions{
			APIKey: "sk-or-test",
			StructuredOutput: &llm.StructuredOutputSpec{
				Name:       "tutor_response",
				JSONSchema: schema,
				Strict:     true,
			},
		},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	var decodedSchema map[string]any
	if err := json.Unmarshal(schema, &decodedSchema); err != nil {
		t.Fatalf("decode expected schema: %v", err)
	}
	want := map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "tutor_response",
			"schema": decodedSchema,
			"strict": true,
		},
	}
	if got := captured.body["response_format"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("response_format = %#v, want %#v", got, want)
	}
}

func TestOpenRouterRejectsInvalidStructuredOutputBeforeRequest(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, openRouterChunk(`{"id":"unexpected","choices":[{"delta":{"content":"unexpected"},"finish_reason":"stop"}]}`)+"\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	tests := []struct {
		name string
		spec llm.StructuredOutputSpec
		want string
	}{
		{name: "missing name", spec: llm.StructuredOutputSpec{JSONSchema: json.RawMessage(`{}`)}, want: "name is required"},
		{name: "invalid name", spec: llm.StructuredOutputSpec{Name: "invalid name", JSONSchema: json.RawMessage(`{}`)}, want: "name must match"},
		{name: "name too long", spec: llm.StructuredOutputSpec{Name: strings.Repeat("a", 65), JSONSchema: json.RawMessage(`{}`)}, want: "name must match"},
		{name: "missing schema", spec: llm.StructuredOutputSpec{Name: "reply"}, want: "JSON schema is required"},
		{name: "malformed schema", spec: llm.StructuredOutputSpec{Name: "reply", JSONSchema: json.RawMessage(`{"schema-secret-marker":`)}, want: "must contain valid JSON"},
		{name: "multiple schemas", spec: llm.StructuredOutputSpec{Name: "reply", JSONSchema: json.RawMessage(`{} {}`)}, want: "must contain valid JSON"},
		{name: "array schema", spec: llm.StructuredOutputSpec{Name: "reply", JSONSchema: json.RawMessage(`[]`)}, want: "must be a JSON object"},
		{name: "null schema", spec: llm.StructuredOutputSpec{Name: "reply", JSONSchema: json.RawMessage(`null`)}, want: "must be a JSON object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := requests.Load()
			msg, err := llm.StreamOpenRouterChat(
				context.Background(),
				openRouterModel(srv.URL),
				llm.Context{Messages: []llm.Message{llm.UserText("return JSON")}},
				&llm.StreamOptions{APIKey: "sk-secret-that-must-not-leak", StructuredOutput: &tt.spec},
			).Result()
			if err == nil || !strings.Contains(msg.ErrorMessage, tt.want) {
				t.Fatalf("expected %q error, got message=%+v err=%v", tt.want, msg, err)
			}
			if strings.Contains(msg.ErrorMessage, "schema-secret-marker") || strings.Contains(msg.ErrorMessage, "sk-secret-that-must-not-leak") {
				t.Fatalf("error leaks request data: %q", msg.ErrorMessage)
			}
			if got := requests.Load(); got != before {
				t.Fatalf("requests = %d, want %d", got, before)
			}
		})
	}
}

func TestOpenRouterSendsShortCacheTTL(t *testing.T) {
	srv, captured := sseServer(t, []string{
		openRouterChunk(`{"id":"or-cache","model":"anthropic/claude-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})
	model := openRouterModel(srv.URL)
	model.ID = "anthropic/claude-test"
	_, err := llm.StreamOpenRouterChat(
		context.Background(),
		model,
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test", CacheRetention: llm.CacheRetentionShort},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	cache := captured.body["cache_control"].(map[string]any)
	if cache["type"] != "ephemeral" || cache["ttl"] != "5m" {
		t.Fatalf("cache_control = %+v", cache)
	}
}

func TestOpenRouterCoalescesMixedParallelToolDeltas(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-tools","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"answer","reasoning":"think","tool_calls":[{"index":0,"id":"read-first","type":"function","function":{"name":"read","arguments":"{\"path\":\"README"}},{"index":1,"id":"grep-first","type":"function","function":{"name":"grep","arguments":"{\"pattern\":\"TODO"}}]},"finish_reason":null}]}`),
		openRouterChunk(`{"id":"or-tools","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"id":"grep-changed","type":"function","function":{"arguments":"\",\"path\":\"src\"}"}},{"index":0,"id":"read-changed","type":"function","function":{"arguments":".md\"}"}}]},"finish_reason":"tool_calls"}]}`),
		"data: [DONE]",
	})

	stream := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("use tools")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	)
	events := collectEvents(stream)
	msg, err := stream.Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if msg.StopReason != llm.StopReasonToolUse || len(msg.Content) != 4 {
		t.Fatalf("message = %+v", msg)
	}
	readCall, ok := msg.Content[2].(llm.ToolCall)
	if !ok || readCall.ID != "read-first" || readCall.Name != "read" || readCall.Arguments["path"] != "README.md" {
		t.Fatalf("read call = %#v", msg.Content[2])
	}
	grepCall, ok := msg.Content[3].(llm.ToolCall)
	if !ok || grepCall.ID != "grep-first" || grepCall.Name != "grep" || grepCall.Arguments["pattern"] != "TODO" || grepCall.Arguments["path"] != "src" {
		t.Fatalf("grep call = %#v", msg.Content[3])
	}

	starts := 0
	deltas := 0
	ends := 0
	for _, event := range events {
		switch event.Type {
		case llm.EventToolCallStart:
			starts++
		case llm.EventToolCallDelta:
			deltas++
		case llm.EventToolCallEnd:
			ends++
		}
	}
	if starts != 2 || deltas != 4 || ends != 2 {
		t.Fatalf("tool events: starts=%d deltas=%d ends=%d", starts, deltas, ends)
	}
}

func TestOpenRouterSeparatesIndexlessParallelToolCalls(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-tools","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"id":"first","type":"function","function":{"name":"read","arguments":"{\"path\":\"README.md\"}"}},{"id":"second","type":"function","function":{"name":"grep","arguments":"{\"pattern\":\"TODO\"}"}}]},"finish_reason":"tool_calls"}]}`),
		"data: [DONE]",
	})

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("use tools")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("content = %#v", msg.Content)
	}
	first := msg.Content[0].(llm.ToolCall)
	second := msg.Content[1].(llm.ToolCall)
	if first.ID != "first" || first.Name != "read" || first.Arguments["path"] != "README.md" {
		t.Fatalf("first = %#v", first)
	}
	if second.ID != "second" || second.Name != "grep" || second.Arguments["pattern"] != "TODO" {
		t.Fatalf("second = %#v", second)
	}
}

func TestOpenRouterCoalescesIndexlessToolCallByID(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-tools","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"id":"call-1","type":"function","function":{"name":"read","arguments":"{\"path\":\"READ"}}]},"finish_reason":null}]}`),
		openRouterChunk(`{"id":"or-tools","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"id":"call-1","type":"function","function":{"arguments":"ME.md\"}"}}]},"finish_reason":"tool_calls"}]}`),
		"data: [DONE]",
	})

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("use tools")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("content = %#v", msg.Content)
	}
	call := msg.Content[0].(llm.ToolCall)
	if call.ID != "call-1" || call.Name != "read" || call.Arguments["path"] != "README.md" {
		t.Fatalf("call = %#v", call)
	}
}

func TestOpenRouterPreservesOrderedReasoningDetails(t *testing.T) {
	srv, captured := sseServer(t, []string{
		openRouterChunk(`{"id":"or-reasoning","model":"google/gemini-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"reasoning_details":[{"type":"reasoning.summary","summary":"brief","index":0},{"type":"reasoning.text","text":"step","signature":"text-signature","id":"reasoning-1","index":1},{"type":"reasoning.encrypted","id":"call-1","data":"encrypted-signature","index":2}]},"finish_reason":null}]}`),
		openRouterChunk(`{"id":"or-reasoning","model":"google/gemini-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"inspect","arguments":"{\"path\":\"README.md\"}"}}]},"finish_reason":"tool_calls"}]}`),
		"data: [DONE]",
	})
	model := openRouterModel(srv.URL)
	model.ID = "google/gemini-test"
	firstUser := llm.UserText("inspect the readme")
	first, err := llm.StreamOpenRouterChat(
		context.Background(),
		model,
		llm.Context{Messages: []llm.Message{firstUser}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil {
		t.Fatalf("first Result: %v", err)
	}
	wantDetails := []any{
		map[string]any{"type": "reasoning.summary", "summary": "brief", "index": float64(0)},
		map[string]any{"type": "reasoning.text", "text": "step", "signature": "text-signature", "id": "reasoning-1", "index": float64(1)},
		map[string]any{"type": "reasoning.encrypted", "id": "call-1", "data": "encrypted-signature", "index": float64(2)},
	}
	encoded, err := json.Marshal(first.ReasoningDetails)
	if err != nil {
		t.Fatalf("marshal reasoning details: %v", err)
	}
	var gotDetails []any
	if err := json.Unmarshal(encoded, &gotDetails); err != nil {
		t.Fatalf("decode reasoning details: %v", err)
	}
	if !reflect.DeepEqual(gotDetails, wantDetails) {
		t.Fatalf("reasoning details = %+v, want %+v", gotDetails, wantDetails)
	}

	_, err = llm.StreamOpenRouterChat(
		context.Background(),
		model,
		llm.Context{Messages: []llm.Message{
			firstUser,
			first,
			llm.ToolResultMessage{ToolCallID: "call-1", ToolName: "inspect", Content: []llm.UserContent{llm.TextContent{Text: "done"}}},
		}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil {
		t.Fatalf("second Result: %v", err)
	}

	messages := captured.body["messages"].([]any)
	assistant := messages[1].(map[string]any)
	details := assistant["reasoning_details"].([]any)
	if !reflect.DeepEqual(details, wantDetails) {
		t.Fatalf("reasoning_details = %+v, want %+v", details, wantDetails)
	}
}

func TestReasoningDetailRejectsInvalidKnownPayload(t *testing.T) {
	var detail llm.ReasoningDetail
	err := json.Unmarshal([]byte(`{"type":"reasoning.text","text":{"secret":"TOP_SECRET"}}`), &detail)
	if err == nil {
		t.Fatal("expected invalid reasoning text error")
	}
	if strings.Contains(err.Error(), "TOP_SECRET") {
		t.Fatalf("validation error leaks raw payload: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"type":"reasoning.encrypted"}`), &detail); err == nil {
		t.Fatal("expected missing encrypted reasoning data error")
	}

	if err := json.Unmarshal([]byte(`{"type":"reasoning.future","payload":{"nested":true}}`), &detail); err != nil {
		t.Fatalf("unknown reasoning variant must remain opaque: %v", err)
	}
}

func TestOpenRouterStreamsRefusalAsText(t *testing.T) {
	srv, captured := sseServer(t, []string{
		openRouterChunk(`{"id":"or-refusal","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"refusal":"I can"},"finish_reason":null}]}`),
		openRouterChunk(`{"id":"or-refusal","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"refusal":"not help with that."},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})
	model := openRouterModel(srv.URL)
	firstUser := llm.UserText("unsafe request")
	stream := llm.StreamOpenRouterChat(
		context.Background(),
		model,
		llm.Context{Messages: []llm.Message{firstUser}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	)
	events := collectEvents(stream)
	first, err := stream.Result()
	if err != nil {
		t.Fatalf("first Result: %v", err)
	}
	if len(first.Content) != 1 {
		t.Fatalf("content = %#v", first.Content)
	}
	text, ok := first.Content[0].(llm.TextContent)
	if !ok || text.Text != "I cannot help with that." {
		t.Fatalf("text = %#v", first.Content[0])
	}
	wantEvents := []llm.EventType{
		llm.EventStart,
		llm.EventTextStart,
		llm.EventTextDelta,
		llm.EventTextDelta,
		llm.EventTextEnd,
		llm.EventDone,
	}
	if got := eventTypes(events); !equalTypes(got, wantEvents...) {
		t.Fatalf("events = %v, want %v", got, wantEvents)
	}

	_, err = llm.StreamOpenRouterChat(
		context.Background(),
		model,
		llm.Context{Messages: []llm.Message{firstUser, first, llm.UserText("try another")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil {
		t.Fatalf("second Result: %v", err)
	}
	assistant := captured.body["messages"].([]any)[1].(map[string]any)
	if assistant["content"] != "I cannot help with that." {
		t.Fatalf("assistant = %+v", assistant)
	}
}

func TestOpenRouterConvertsMessagesToolsAndImages(t *testing.T) {
	srv, captured := sseServer(t, []string{
		openRouterChunk(`{"id":"or-convert","model":"deepseek/deepseek-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})

	model := openRouterModel(srv.URL)
	model.ID = "deepseek/deepseek-test"
	prior := llm.AssistantMessage{Content: []llm.AssistantContent{
		llm.TextContent{Text: "calling"},
		llm.ThinkingContent{Thinking: "planning"},
		llm.ToolCall{ID: "call-1", Name: "inspect", Arguments: map[string]any{"path": "README.md"}},
	}}
	tool := llm.Tool{
		Name:        "inspect",
		Description: "Inspect a file",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}
	conversation := llm.Context{
		SystemPrompt: "Use evidence.",
		Messages: []llm.Message{
			llm.UserMessage{Content: []llm.UserContent{
				llm.TextContent{Text: "look"},
				llm.ImageContent{MimeType: "image/png", Data: "abcd"},
				llm.ImageURLContent{URL: "HTTPS://images.example/cat.png?size=large#view"},
			}},
			prior,
			llm.ToolResultMessage{ToolCallID: "call-1", ToolName: "inspect", Content: []llm.UserContent{
				llm.TextContent{Text: "found"},
				llm.ImageContent{MimeType: "image/jpeg", Data: "efgh"},
				llm.ImageURLContent{URL: "http://images.example/result.jpg"},
			}},
		},
		Tools: []llm.Tool{tool},
	}

	if _, err := llm.StreamOpenRouterChat(context.Background(), model, conversation, &llm.StreamOptions{APIKey: "sk-or-test", CacheRetention: llm.CacheRetentionShort}).Result(); err != nil {
		t.Fatalf("Result: %v", err)
	}

	messages := captured.body["messages"].([]any)
	if len(messages) != 4 {
		t.Fatalf("messages = %d: %+v", len(messages), messages)
	}
	if messages[0].(map[string]any)["role"] != "system" {
		t.Fatalf("system message = %+v", messages[0])
	}
	if _, ok := captured.body["cache_control"]; ok {
		t.Fatalf("cache_control must be omitted for %s", model.ID)
	}
	userParts := messages[1].(map[string]any)["content"].([]any)
	userImage := userParts[1].(map[string]any)["image_url"].(map[string]any)
	if !strings.HasPrefix(userImage["url"].(string), "data:image/png;base64,") {
		t.Fatalf("user image = %+v", userImage)
	}
	userRemoteImage := userParts[2].(map[string]any)["image_url"].(map[string]any)
	if userRemoteImage["url"] != "HTTPS://images.example/cat.png?size=large#view" {
		t.Fatalf("remote user image = %+v", userRemoteImage)
	}
	assistant := messages[2].(map[string]any)
	if assistant["content"] != "calling" || assistant["reasoning"] != "planning" {
		t.Fatalf("assistant = %+v", assistant)
	}
	toolCalls := assistant["tool_calls"].([]any)
	function := toolCalls[0].(map[string]any)["function"].(map[string]any)
	if function["name"] != "inspect" || !strings.Contains(function["arguments"].(string), `"path":"README.md"`) {
		t.Fatalf("tool calls = %+v", toolCalls)
	}
	toolResultParts := messages[3].(map[string]any)["content"].([]any)
	toolImage := toolResultParts[1].(map[string]any)["image_url"].(map[string]any)
	if !strings.HasPrefix(toolImage["url"].(string), "data:image/jpeg;base64,") {
		t.Fatalf("tool image = %+v", toolImage)
	}
	toolRemoteImage := toolResultParts[2].(map[string]any)["image_url"].(map[string]any)
	if toolRemoteImage["url"] != "http://images.example/result.jpg" {
		t.Fatalf("remote tool image = %+v", toolRemoteImage)
	}

	tools := captured.body["tools"].([]any)
	toolFunction := tools[0].(map[string]any)["function"].(map[string]any)
	parameters := toolFunction["parameters"].(map[string]any)
	if toolFunction["name"] != "inspect" || parameters["type"] != "object" {
		t.Fatalf("tools = %+v", tools)
	}
}

func TestOpenRouterRejectsInvalidImageURLsBeforeRequest(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(srv.Close)

	invalid := []string{
		"",
		"images.example/cat.png",
		"data:image/png;base64,abcd",
		"ftp://images.example/cat.png",
		"https://user:secret@images.example/cat.png",
		"https://:443/cat.png",
		"://malformed",
	}
	for _, imageURL := range invalid {
		t.Run(imageURL, func(t *testing.T) {
			msg, err := llm.StreamOpenRouterChat(
				context.Background(),
				openRouterModel(srv.URL),
				llm.Context{Messages: []llm.Message{llm.UserMessage{Content: []llm.UserContent{
					llm.ImageURLContent{URL: imageURL},
				}}}},
				&llm.StreamOptions{APIKey: "sk-or-test"},
			).Result()
			if err == nil || msg.ErrorMessage != "openrouter-chat: image URL must be an absolute HTTP(S) URL without credentials" {
				t.Fatalf("expected safe image URL error, got %+v err=%v", msg, err)
			}
			if strings.Contains(msg.ErrorMessage, "secret") {
				t.Fatalf("error leaks URL credentials: %q", msg.ErrorMessage)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("requests = %d, want 0", requests.Load())
	}
}

func TestOpenRouterRejectsInvalidToolSchema(t *testing.T) {
	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel("http://127.0.0.1:0"),
		llm.Context{
			Messages: []llm.Message{llm.UserText("hi")},
			Tools:    []llm.Tool{{Name: "bad", Parameters: json.RawMessage(`[]`)}},
		},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, "parameters must be a JSON object") {
		t.Fatalf("expected schema error, got %+v err=%v", msg, err)
	}
}

func TestOpenRouterRejectsUnencodableToolArguments(t *testing.T) {
	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel("http://127.0.0.1:0"),
		llm.Context{Messages: []llm.Message{
			llm.UserText("call the tool"),
			llm.AssistantMessage{Content: []llm.AssistantContent{
				llm.ToolCall{ID: "call-1", Name: "bad", Arguments: map[string]any{"value": func() {}}},
			}},
		}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, `tool call "bad" arguments`) || !strings.Contains(msg.ErrorMessage, "unsupported type") {
		t.Fatalf("expected argument encoding error, got %+v err=%v", msg, err)
	}
	var unsupportedType *json.UnsupportedTypeError
	if !errors.As(err, &unsupportedType) {
		t.Fatalf("expected wrapped UnsupportedTypeError, got %v", err)
	}
}

func TestOpenRouterRejectsMalformedStreamedToolArguments(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-bad-tool","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"bad","arguments":"{not-json"}}]},"finish_reason":"tool_calls"}]}`),
		"data: [DONE]",
	})
	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("call the tool")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) || msg.StopReason != llm.StopReasonError {
		t.Fatalf("expected malformed tool arguments error, got %+v err=%v", msg, err)
	}
	if !strings.Contains(msg.ErrorMessage, `tool call "bad" arguments`) {
		t.Fatalf("errorMessage = %q", msg.ErrorMessage)
	}
}

func TestOpenRouterInlineStreamError(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-error","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[],"error":{"code":429,"message":"provider overloaded"}}`),
		"data: [DONE]",
	})

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, "provider overloaded") {
		t.Fatalf("expected inline stream error, got %+v err=%v", msg, err)
	}
}

func TestOpenRouterFinishReasonMapping(t *testing.T) {
	cases := []struct {
		finish  string
		want    llm.StopReason
		wantErr bool
	}{
		{"length", llm.StopReasonLength, false},
		{"content_filter", llm.StopReasonError, true},
		{"error", llm.StopReasonError, true},
	}
	for _, tc := range cases {
		t.Run(tc.finish, func(t *testing.T) {
			srv, _ := sseServer(t, []string{
				openRouterChunk(`{"id":"or-finish","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"x"},"finish_reason":"` + tc.finish + `"}]}`),
				"data: [DONE]",
			})
			stream := llm.StreamOpenRouterChat(
				context.Background(),
				openRouterModel(srv.URL),
				llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
				&llm.StreamOptions{APIKey: "sk-or-test"},
			)
			events := collectEvents(stream)
			msg, err := stream.Result()
			if tc.wantErr {
				if err == nil || msg.StopReason != llm.StopReasonError || !strings.Contains(msg.ErrorMessage, tc.finish) {
					t.Fatalf("message = %+v err=%v", msg, err)
				}
				if got := eventTypes(events); !equalTypes(got, llm.EventStart, llm.EventTextStart, llm.EventTextDelta, llm.EventTextEnd, llm.EventError) {
					t.Fatalf("events = %v", got)
				}
				return
			}
			if err != nil || msg.StopReason != tc.want {
				t.Fatalf("message = %+v err=%v", msg, err)
			}
		})
	}
}

func TestOpenRouterHTTPErrorBecomesTerminalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"rate limited"}}`, http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	var streamErr *llm.StreamError
	if !errors.As(err, &streamErr) || streamErr.Reason != llm.StopReasonError {
		t.Fatalf("expected StreamError, got %v", err)
	}
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected SDK APIError cause, got %v", err)
	}
	if apiErr.RawResponse != nil {
		t.Fatalf("SDK error exposes raw response request")
	}
	if !strings.Contains(msg.ErrorMessage, "429") || !strings.Contains(msg.ErrorMessage, "rate limited") {
		t.Fatalf("errorMessage = %q", msg.ErrorMessage)
	}
}

func TestOpenRouterBoundsErrorBodies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", 2<<20)))
	}))
	t.Cleanup(srv.Close)

	_, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected SDK APIError cause, got %v", err)
	}
	if len(apiErr.Body) > 64<<10 {
		t.Fatalf("error body length = %d", len(apiErr.Body))
	}
}

func TestOpenRouterRejectsTransportOwnedHeaders(t *testing.T) {
	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel("http://127.0.0.1:0"),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test", Headers: map[string]string{"authorization": "Bearer attacker"}},
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, `header "Authorization" is managed by the transport`) {
		t.Fatalf("expected reserved header error, got %+v err=%v", msg, err)
	}
}

func TestOpenRouterStreamParseErrorDoesNotLeakPayload(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-bad-stream","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"reasoning_details":[{"type":"reasoning.text","text":{"secret":"TOP_SECRET"}}]},"finish_reason":null}]}`),
		"data: [DONE]",
	})
	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err == nil {
		t.Fatalf("expected stream parse error, got %+v err=%v", msg, err)
	}
	if strings.Contains(msg.ErrorMessage, "TOP_SECRET") || strings.Contains(err.Error(), "TOP_SECRET") {
		t.Fatalf("stream error leaks payload: msg=%q err=%v", msg.ErrorMessage, err)
	}
}

func TestOpenRouterMissingFinishReasonIsError(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-no-finish","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"partial"},"finish_reason":null}]}`),
		"data: [DONE]",
	})

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, "finish_reason") {
		t.Fatalf("expected finish_reason error, got %+v err=%v", msg, err)
	}
}

func TestOpenRouterReportsTruncationAfterFinishReason(t *testing.T) {
	finish := openRouterChunk(`{"id":"or-truncated","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`) + "\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Content-Length", fmt.Sprint(len(finish)+100))
		_, _ = io.WriteString(w, finish)
	}))
	t.Cleanup(srv.Close)

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, "reading stream") {
		t.Fatalf("expected stream read error, got %+v err=%v", msg, err)
	}
}

func TestOpenRouterAcceptsLargeSSEEvent(t *testing.T) {
	want := strings.Repeat("x", 70<<10)
	chunk, err := json.Marshal(map[string]any{
		"id":      "or-large",
		"model":   "openai/gpt-test",
		"object":  "chat.completion.chunk",
		"created": 1,
		"choices": []any{map[string]any{
			"index":         0,
			"delta":         map[string]any{"content": want},
			"finish_reason": "stop",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	srv, _ := sseServer(t, []string{openRouterChunk(string(chunk)), "data: [DONE]"})

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if got := textOf(t, msg); got != want {
		t.Fatalf("content length = %d, want %d", len(got), len(want))
	}
}

func TestOpenRouterJoinsMultilineSSEData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"id\":\"or-multiline\",\"model\":\"openai/gpt-test\",\n")
		_, _ = io.WriteString(w, "data: \"object\":\"chat.completion.chunk\",\"created\":1,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil || textOf(t, msg) != "ok" {
		t.Fatalf("message = %+v err=%v", msg, err)
	}
}

func TestOpenRouterAcceptsUTF8BOM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "\uFEFF"+openRouterChunk(`{"id":"or-bom","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`)+"\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil || textOf(t, msg) != "ok" {
		t.Fatalf("message = %+v err=%v", msg, err)
	}
}

func TestOpenRouterStopsAtDoneSentinel(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-done","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
		strings.Repeat("x", 2<<20),
	})

	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	).Result()
	if err != nil || textOf(t, msg) != "ok" {
		t.Fatalf("message = %+v err=%v", msg, err)
	}
}

func TestOpenRouterMissingAPIKeyIsError(t *testing.T) {
	msg, err := llm.StreamOpenRouterChat(
		context.Background(),
		openRouterModel("http://127.0.0.1:0"),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		nil,
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, "no API key") {
		t.Fatalf("expected missing-key error, got %+v err=%v", msg, err)
	}
}

func TestOpenRouterPreCanceledRequestPreservesContextCause(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	msg, err := llm.StreamOpenRouterChat(
		ctx,
		openRouterModel("http://127.0.0.1:0"),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		nil,
	).Result()
	if msg.StopReason != llm.StopReasonAborted || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected aborted context cause, got %+v err=%v", msg, err)
	}
	if !strings.Contains(msg.ErrorMessage, "no API key") {
		t.Fatalf("errorMessage = %q", msg.ErrorMessage)
	}
}

func TestOpenRouterAbortMidStream(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(openRouterChunk(`{"id":"or-abort","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"partial"},"finish_reason":null}]}`) + "\n\n"))
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	stream := llm.StreamOpenRouterChat(
		ctx,
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	)
	<-started
	cancel()
	msg, err := stream.Result()
	var streamErr *llm.StreamError
	if !errors.As(err, &streamErr) || streamErr.Reason != llm.StopReasonAborted {
		t.Fatalf("expected aborted StreamError, got %v (msg=%+v)", err, msg)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped context cancellation, got %v", err)
	}
}

func TestOpenRouterRegistersInRegistry(t *testing.T) {
	srv, _ := sseServer(t, []string{
		openRouterChunk(`{"id":"or-registry","model":"openai/gpt-test","object":"chat.completion.chunk","created":1,"choices":[{"index":0,"delta":{"content":"via registry"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})
	llm.RegisterOpenAICompletions()
	llm.RegisterOpenRouterChat()
	if llm.APIOpenRouterChat == llm.APIOpenAICompletions {
		t.Fatal("OpenRouter must have a distinct registry key")
	}

	msg, err := llm.Complete(
		context.Background(),
		openRouterModel(srv.URL),
		llm.Context{Messages: []llm.Message{llm.UserText("hi")}},
		&llm.StreamOptions{APIKey: "sk-or-test"},
	)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := textOf(t, msg); got != "via registry" {
		t.Fatalf("text = %q", got)
	}
}

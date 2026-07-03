package piai_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/piai"
)

type capturedRequest struct {
	body    map[string]any
	headers http.Header
}

func sseServer(t *testing.T, lines []string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.headers = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&captured.body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, line := range lines {
			_, _ = w.Write([]byte(line + "\n\n"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

func openAIModel(baseURL string) piai.Model {
	return piai.Model{
		ID:       "gpt-test",
		Name:     "GPT Test",
		API:      piai.APIOpenAICompletions,
		Provider: "openai",
		BaseURL:  baseURL + "/v1",
		Cost:     piai.Cost{Input: 1, Output: 2, CacheRead: 0.5, CacheWrite: 1.5},
	}
}

func chunk(s string) string { return "data: " + s }

func TestOpenAIStreamsTextAndUsage(t *testing.T) {
	srv, captured := sseServer(t, []string{
		chunk(`{"id":"chatcmpl-1","choices":[{"delta":{"content":"Hel"}}]}`),
		chunk(`{"id":"chatcmpl-1","choices":[{"delta":{"content":"lo"}}]}`),
		chunk(`{"id":"chatcmpl-1","choices":[{"delta":{},"finish_reason":"stop"}]}`),
		chunk(`{"id":"chatcmpl-1","choices":[],"usage":{"prompt_tokens":100,"completion_tokens":10,"prompt_tokens_details":{"cached_tokens":40}}}`),
		"data: [DONE]",
	})

	temp := 0.3
	msg, err := piai.StreamOpenAICompletions(context.Background(), openAIModel(srv.URL),
		piai.Context{SystemPrompt: "Be brief.", Messages: []piai.Message{piai.UserText("hi")}},
		&piai.StreamOptions{APIKey: "sk-test", Temperature: &temp, MaxTokens: 128},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	if got := textOf(t, msg); got != "Hello" {
		t.Fatalf("text = %q", got)
	}
	if msg.ResponseID != "chatcmpl-1" || msg.StopReason != piai.StopReasonStop {
		t.Fatalf("responseID=%q stopReason=%q", msg.ResponseID, msg.StopReason)
	}

	if msg.Usage.Input != 60 || msg.Usage.CacheRead != 40 || msg.Usage.Output != 10 {
		t.Fatalf("usage = %+v", msg.Usage)
	}
	requireTotalTokensEqualsComponents(t, msg.Usage)

	if msg.Usage.Cost.Total == 0 || msg.Usage.Cost.Input != 60.0/1e6 {
		t.Fatalf("cost = %+v", msg.Usage.Cost)
	}

	if captured.headers.Get("Authorization") != "Bearer sk-test" {
		t.Fatalf("auth header = %q", captured.headers.Get("Authorization"))
	}
	if captured.body["model"] != "gpt-test" || captured.body["stream"] != true {
		t.Fatalf("body = %+v", captured.body)
	}
	if captured.body["temperature"] != 0.3 || captured.body["max_completion_tokens"] != float64(128) {
		t.Fatalf("params = %+v", captured.body)
	}
	if _, hasStore := captured.body["store"]; hasStore {
		t.Fatal("store must only be sent to api.openai.com")
	}
	msgs := captured.body["messages"].([]any)
	first := msgs[0].(map[string]any)
	if first["role"] != "system" || first["content"] != "Be brief." {
		t.Fatalf("system message = %+v", first)
	}
}

func TestOpenAIStreamsToolCallDeltas(t *testing.T) {
	srv, captured := sseServer(t, []string{
		chunk(`{"id":"c","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"echo","arguments":""}}]}}]}`),
		chunk(`{"id":"c","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"text\":"}}]}}]}`),
		chunk(`{"id":"c","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"hi\"}"}}]}}]}`),
		chunk(`{"id":"c","choices":[{"delta":{},"finish_reason":"tool_calls"}]}`),
		"data: [DONE]",
	})

	tool := piai.Tool{Name: "echo", Description: "Echo", Parameters: json.RawMessage(`{"type":"object"}`)}
	stream := piai.StreamOpenAICompletions(context.Background(), openAIModel(srv.URL),
		piai.Context{Messages: []piai.Message{piai.UserText("hi")}, Tools: []piai.Tool{tool}},
		&piai.StreamOptions{APIKey: "sk-test"},
	)
	events := collectEvents(stream)
	msg, err := stream.Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}

	if msg.StopReason != piai.StopReasonToolUse {
		t.Fatalf("stopReason = %q", msg.StopReason)
	}
	tc, ok := msg.Content[0].(piai.ToolCall)
	if !ok || tc.ID != "call_1" || tc.Name != "echo" || tc.Arguments["text"] != "hi" {
		t.Fatalf("toolCall = %#v", msg.Content[0])
	}
	for _, want := range []piai.EventType{piai.EventToolCallStart, piai.EventToolCallDelta, piai.EventToolCallEnd, piai.EventDone} {
		if !containsType(events, want) {
			t.Fatalf("missing %q in %v", want, eventTypes(events))
		}
	}

	tools := captured.body["tools"].([]any)
	fn := tools[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "echo" {
		t.Fatalf("tools = %+v", tools)
	}
}

func TestOpenAIStreamsMultipleToolCallsByIndex(t *testing.T) {
	srv, _ := sseServer(t, []string{
		chunk(`{"id":"c","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"a","arguments":"{}"}}]}}]}`),
		chunk(`{"id":"c","choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_2","function":{"name":"b","arguments":"{}"}}]}}]}`),
		chunk(`{"id":"c","choices":[{"delta":{},"finish_reason":"tool_calls"}]}`),
		"data: [DONE]",
	})

	msg, err := piai.StreamOpenAICompletions(context.Background(), openAIModel(srv.URL),
		piai.Context{Messages: []piai.Message{piai.UserText("hi")}},
		&piai.StreamOptions{APIKey: "sk-test"},
	).Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("content blocks = %d", len(msg.Content))
	}
	a, b := msg.Content[0].(piai.ToolCall), msg.Content[1].(piai.ToolCall)
	if a.ID != "call_1" || b.ID != "call_2" {
		t.Fatalf("ids = %q, %q", a.ID, b.ID)
	}
}

func TestOpenAIFinishReasonMapping(t *testing.T) {
	cases := []struct {
		finish  string
		want    piai.StopReason
		wantErr bool
	}{
		{"length", piai.StopReasonLength, false},
		{"content_filter", piai.StopReasonError, true},
	}
	for _, tc := range cases {
		srv, _ := sseServer(t, []string{
			chunk(`{"id":"c","choices":[{"delta":{"content":"x"},"finish_reason":"` + tc.finish + `"}]}`),
			"data: [DONE]",
		})
		msg, err := piai.StreamOpenAICompletions(context.Background(), openAIModel(srv.URL),
			piai.Context{Messages: []piai.Message{piai.UserText("hi")}},
			&piai.StreamOptions{APIKey: "sk-test"},
		).Result()
		if tc.wantErr {
			if err == nil || msg.StopReason != piai.StopReasonError {
				t.Fatalf("%s: expected error terminal, got %+v", tc.finish, msg)
			}
			if !strings.Contains(msg.ErrorMessage, tc.finish) {
				t.Fatalf("%s: errorMessage = %q", tc.finish, msg.ErrorMessage)
			}
		} else {
			if err != nil || msg.StopReason != tc.want {
				t.Fatalf("%s: stopReason = %q err=%v", tc.finish, msg.StopReason, err)
			}
		}
	}
}

func TestOpenAIStreamWithoutFinishReasonIsError(t *testing.T) {
	srv, _ := sseServer(t, []string{
		chunk(`{"id":"c","choices":[{"delta":{"content":"x"}}]}`),
		"data: [DONE]",
	})
	msg, err := piai.StreamOpenAICompletions(context.Background(), openAIModel(srv.URL),
		piai.Context{Messages: []piai.Message{piai.UserText("hi")}},
		&piai.StreamOptions{APIKey: "sk-test"},
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, "finish_reason") {
		t.Fatalf("expected finish_reason error, got %+v err=%v", msg, err)
	}
}

func TestOpenAIHTTPErrorBecomesTerminalErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"bad model"}}`, http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	msg, err := piai.StreamOpenAICompletions(context.Background(), openAIModel(srv.URL),
		piai.Context{Messages: []piai.Message{piai.UserText("hi")}},
		&piai.StreamOptions{APIKey: "sk-test"},
	).Result()
	var streamErr *piai.StreamError
	if !errors.As(err, &streamErr) || streamErr.Reason != piai.StopReasonError {
		t.Fatalf("expected StreamError, got %v", err)
	}
	if !strings.Contains(msg.ErrorMessage, "HTTP 400") || !strings.Contains(msg.ErrorMessage, "bad model") {
		t.Fatalf("errorMessage = %q", msg.ErrorMessage)
	}
}

func TestOpenAIMissingAPIKeyIsError(t *testing.T) {
	msg, err := piai.StreamOpenAICompletions(context.Background(), openAIModel("http://localhost:0"),
		piai.Context{Messages: []piai.Message{piai.UserText("hi")}}, nil,
	).Result()
	if err == nil || !strings.Contains(msg.ErrorMessage, "no API key") {
		t.Fatalf("expected missing-key error, got %+v err=%v", msg, err)
	}
}

func TestOpenAIAbortMidStream(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(chunk(`{"id":"c","choices":[{"delta":{"content":"x"}}]}`) + "\n\n"))
		w.(http.Flusher).Flush()
		<-release
	}))
	t.Cleanup(func() { close(release); srv.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	stream := piai.StreamOpenAICompletions(ctx, openAIModel(srv.URL),
		piai.Context{Messages: []piai.Message{piai.UserText("hi")}},
		&piai.StreamOptions{APIKey: "sk-test"},
	)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	msg, err := stream.Result()
	var streamErr *piai.StreamError
	if !errors.As(err, &streamErr) || streamErr.Reason != piai.StopReasonAborted {
		t.Fatalf("expected aborted, got %v (msg=%+v)", err, msg)
	}
}

func TestOpenAIMessageConversion(t *testing.T) {
	srv, captured := sseServer(t, []string{
		chunk(`{"id":"c","choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})

	prior := piai.FauxAssistantMessage(
		piai.FauxText("calling"),
		piai.ToolCall{ID: "call_1", Name: "echo", Arguments: map[string]any{"text": "hi"}},
	)
	aborted := piai.FauxAssistantMessage()
	c := piai.Context{
		Messages: []piai.Message{
			piai.UserMessage{Content: []piai.UserContent{
				piai.TextContent{Text: "look"},
				piai.ImageContent{MimeType: "image/png", Data: "abcd"},
			}},
			prior,
			piai.ToolResultMessage{ToolCallID: "call_1", ToolName: "echo",
				Content: []piai.UserContent{piai.TextContent{Text: "echoed"}}},
			aborted,
			piai.UserText("next"),
		},
	}
	if _, err := piai.StreamOpenAICompletions(context.Background(), openAIModel(srv.URL), c,
		&piai.StreamOptions{APIKey: "sk-test"}).Result(); err != nil {
		t.Fatalf("Result: %v", err)
	}

	msgs := captured.body["messages"].([]any)
	if len(msgs) != 4 {
		t.Fatalf("messages = %d: %+v", len(msgs), msgs)
	}
	userParts := msgs[0].(map[string]any)["content"].([]any)
	img := userParts[1].(map[string]any)["image_url"].(map[string]any)
	if !strings.HasPrefix(img["url"].(string), "data:image/png;base64,") {
		t.Fatalf("image part = %+v", userParts[1])
	}
	assistant := msgs[1].(map[string]any)
	toolCalls := assistant["tool_calls"].([]any)
	fn := toolCalls[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "echo" || !strings.Contains(fn["arguments"].(string), `"text":"hi"`) {
		t.Fatalf("tool_calls = %+v", toolCalls)
	}
	toolMsg := msgs[2].(map[string]any)
	if toolMsg["role"] != "tool" || toolMsg["tool_call_id"] != "call_1" || toolMsg["content"] != "echoed" {
		t.Fatalf("tool message = %+v", toolMsg)
	}

	if tools, ok := captured.body["tools"].([]any); !ok || len(tools) != 0 {
		t.Fatalf("tools = %+v", captured.body["tools"])
	}
}

func TestOpenAIDeveloperRoleForReasoningModels(t *testing.T) {
	srv, captured := sseServer(t, []string{
		chunk(`{"id":"c","choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})
	model := openAIModel(srv.URL)
	model.Reasoning = true

	if _, err := piai.StreamOpenAICompletions(context.Background(), model,
		piai.Context{SystemPrompt: "sys", Messages: []piai.Message{piai.UserText("hi")}},
		&piai.StreamOptions{APIKey: "sk-test"}).Result(); err != nil {
		t.Fatalf("Result: %v", err)
	}
	first := captured.body["messages"].([]any)[0].(map[string]any)
	if first["role"] != "developer" {
		t.Fatalf("system role = %q, want developer", first["role"])
	}
}

func TestOpenAIRegistersInRegistry(t *testing.T) {
	srv, _ := sseServer(t, []string{
		chunk(`{"id":"c","choices":[{"delta":{"content":"via registry"},"finish_reason":"stop"}]}`),
		"data: [DONE]",
	})
	piai.RegisterOpenAICompletions()

	msg, err := piai.Complete(context.Background(), openAIModel(srv.URL),
		piai.Context{Messages: []piai.Message{piai.UserText("hi")}},
		&piai.StreamOptions{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := textOf(t, msg); got != "via registry" {
		t.Fatalf("text = %q", got)
	}
}

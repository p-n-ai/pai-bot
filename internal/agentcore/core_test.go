package agentcore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func TestCoreDirectAnswer(t *testing.T) {
	model := &scriptedModel{replies: []llm.AssistantMessage{{Content: []llm.AssistantContent{llm.TextContent{Text: "Hello"}}}}}
	core, err := New(model, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Run(context.Background(), llm.Context{Messages: []llm.Message{llm.UserText("Hi")}})
	if err != nil {
		t.Fatal(err)
	}
	if got := FinalText(result.Final); got != "Hello" {
		t.Fatalf("FinalText = %q", got)
	}
	if result.ModelCalls != 1 || result.ToolCalls != 0 {
		t.Fatalf("counts = %d, %d", result.ModelCalls, result.ToolCalls)
	}
}

func TestCoreToolCallThenFinalReplyPreservesNativeTranscript(t *testing.T) {
	model := &scriptedModel{replies: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{llm.ToolCall{ID: "call-1", Name: "lookup", Arguments: map[string]any{"query": "algebra"}}}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "Here is the answer."}}},
	}}
	tool := &recordingTool{}
	core, err := New(model, []Tool{tool}, 3)
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Run(context.Background(), llm.Context{Messages: []llm.Message{llm.UserText("Help")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Transcript) != 4 {
		t.Fatalf("transcript length = %d, want 4", len(result.Transcript))
	}
	toolResult, ok := result.Transcript[2].(llm.ToolResultMessage)
	if !ok || toolResult.ToolCallID != "call-1" || toolResult.ToolName != "lookup" || toolResult.IsError {
		t.Fatalf("tool result = %#v", result.Transcript[2])
	}
	if len(model.contexts) != 2 || len(model.contexts[1].Messages) != 3 {
		t.Fatalf("second model context = %#v", model.contexts)
	}
}

func TestCoreUnknownToolReturnsRecoverableToolError(t *testing.T) {
	model := &scriptedModel{replies: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{llm.ToolCall{ID: "missing-1", Name: "missing"}}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "Recovered"}}},
	}}
	core, _ := New(model, nil, 3)
	result, err := core.Run(context.Background(), llm.Context{})
	if err != nil {
		t.Fatal(err)
	}
	toolResult := result.Transcript[1].(llm.ToolResultMessage)
	if !toolResult.IsError {
		t.Fatal("unknown tool result was not marked as error")
	}
}

func TestCoreStopsAtModelCallLimit(t *testing.T) {
	model := &scriptedModel{replies: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{llm.ToolCall{ID: "call-1", Name: "lookup"}}},
		{Content: []llm.AssistantContent{llm.ToolCall{ID: "call-2", Name: "lookup"}}},
	}}
	core, _ := New(model, []Tool{&recordingTool{}}, 2)
	if _, err := core.Run(context.Background(), llm.Context{}); err == nil {
		t.Fatal("model-call limit was not enforced")
	}
}

type scriptedModel struct {
	replies  []llm.AssistantMessage
	contexts []llm.Context
}

func (m *scriptedModel) Complete(_ context.Context, ctx llm.Context) (llm.AssistantMessage, error) {
	m.contexts = append(m.contexts, ctx)
	reply := m.replies[0]
	m.replies = m.replies[1:]
	return reply, nil
}

type recordingTool struct{}

func (*recordingTool) Definition() llm.Tool {
	return llm.Tool{Name: "lookup", Parameters: json.RawMessage(`{"type":"object"}`)}
}
func (*recordingTool) Execute(_ context.Context, _ llm.ToolCall) llm.ToolResultMessage {
	return llm.ToolResultMessage{Content: []llm.UserContent{llm.TextContent{Text: "result"}}}
}

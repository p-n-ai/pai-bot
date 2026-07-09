package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func requireTotalTokensEqualsComponents(t *testing.T, u llm.Usage) {
	t.Helper()
	computed := u.Input + u.Output + u.CacheRead + u.CacheWrite
	if u.TotalTokens != computed {
		t.Fatalf("totalTokens = %d, want sum of components %d (%+v)", u.TotalTokens, computed, u)
	}
}

func userContext(text string) llm.Context {
	return llm.Context{Messages: []llm.Message{llm.UserText(text)}}
}

func textOf(t *testing.T, msg llm.AssistantMessage) string {
	t.Helper()
	if len(msg.Content) != 1 {
		t.Fatalf("expected single content block, got %d", len(msg.Content))
	}
	block, ok := msg.Content[0].(llm.TextContent)
	if !ok {
		t.Fatalf("expected text block, got %T", msg.Content[0])
	}
	return block.Text
}

func TestFauxRegistersProviderAndEstimatesUsage(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("hello world")))

	c := llm.Context{SystemPrompt: "Be concise.", Messages: []llm.Message{llm.UserText("hi there")}}
	msg, err := llm.Complete(context.Background(), f.Model(), c, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := textOf(t, msg); got != "hello world" {
		t.Fatalf("content = %q", got)
	}
	if msg.Usage.Input == 0 || msg.Usage.Output == 0 {
		t.Fatalf("expected non-zero usage, got %+v", msg.Usage)
	}
	if msg.Usage.TotalTokens != msg.Usage.Input+msg.Usage.Output {
		t.Fatalf("totalTokens = %d, want input+output", msg.Usage.TotalTokens)
	}
	requireTotalTokensEqualsComponents(t, msg.Usage)
	if f.CallCount() != 1 {
		t.Fatalf("callCount = %d", f.CallCount())
	}
}

func TestFauxHelperBlocks(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	reply := llm.FauxAssistantMessage(
		llm.FauxThinking("think"),
		llm.FauxToolCall("echo", map[string]any{"text": "hi"}),
		llm.FauxText("done"),
	)
	reply.StopReason = llm.StopReasonToolUse
	f.SetResponses(llm.FauxRespond(reply))

	msg, err := llm.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if msg.StopReason != llm.StopReasonToolUse {
		t.Fatalf("stopReason = %q", msg.StopReason)
	}
	if len(msg.Content) != 3 {
		t.Fatalf("content blocks = %d", len(msg.Content))
	}
	if th, ok := msg.Content[0].(llm.ThinkingContent); !ok || th.Thinking != "think" {
		t.Fatalf("block 0 = %#v", msg.Content[0])
	}
	tc, ok := msg.Content[1].(llm.ToolCall)
	if !ok || tc.Name != "echo" || tc.ID == "" || tc.Arguments["text"] != "hi" {
		t.Fatalf("block 1 = %#v", msg.Content[1])
	}
	if txt, ok := msg.Content[2].(llm.TextContent); !ok || txt.Text != "done" {
		t.Fatalf("block 2 = %#v", msg.Content[2])
	}
	requireTotalTokensEqualsComponents(t, msg.Usage)
}

func TestFauxMultipleModelsAndModelAwareFactories(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{
		Models: []llm.FauxModel{
			{ID: "faux-fast", Name: "Faux Fast"},
			{ID: "faux-thinker", Name: "Faux Thinker", Reasoning: true},
		},
	})
	defer f.Unregister()
	factory := func(_ llm.Context, _ *llm.StreamOptions, _ int, model llm.Model) (llm.AssistantMessage, error) {
		return llm.FauxAssistantText(fmt.Sprintf("%s:%t", model.ID, model.Reasoning)), nil
	}
	f.SetResponses(factory, factory)

	fast, ok := f.ModelByID("faux-fast")
	if !ok || fast.Reasoning {
		t.Fatalf("faux-fast = %+v ok=%v", fast, ok)
	}
	thinker, ok := f.ModelByID("faux-thinker")
	if !ok || !thinker.Reasoning {
		t.Fatalf("faux-thinker = %+v ok=%v", thinker, ok)
	}
	if f.Model().ID != "faux-fast" {
		t.Fatalf("first model = %q", f.Model().ID)
	}

	fastMsg, err := llm.Complete(context.Background(), fast, userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete fast: %v", err)
	}
	thinkerMsg, err := llm.Complete(context.Background(), thinker, userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete thinker: %v", err)
	}
	if textOf(t, fastMsg) != "faux-fast:false" || textOf(t, thinkerMsg) != "faux-thinker:true" {
		t.Fatalf("responses = %q, %q", textOf(t, fastMsg), textOf(t, thinkerMsg))
	}
}

func TestFauxRewritesAPIProviderModel(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{
		API:      "faux:test",
		Provider: "faux-provider",
		Models:   []llm.FauxModel{{ID: "faux-model"}},
	})
	defer f.Unregister()
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("hello")))

	msg, err := llm.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if msg.API != "faux:test" || msg.Provider != "faux-provider" || msg.Model != "faux-model" {
		t.Fatalf("api/provider/model = %q/%q/%q", msg.API, msg.Provider, msg.Model)
	}
}

func TestFauxConsumesQueueInOrderAndErrorsWhenExhausted(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		llm.FauxRespond(llm.FauxAssistantText("first")),
		llm.FauxRespond(llm.FauxAssistantText("second")),
	)

	c := userContext("hi")
	first, err := llm.Complete(context.Background(), f.Model(), c, nil)
	if err != nil || textOf(t, first) != "first" {
		t.Fatalf("first = %q err=%v", textOf(t, first), err)
	}
	second, err := llm.Complete(context.Background(), f.Model(), c, nil)
	if err != nil || textOf(t, second) != "second" {
		t.Fatalf("second = %q err=%v", textOf(t, second), err)
	}
	exhausted, err := llm.Complete(context.Background(), f.Model(), c, nil)
	if err == nil {
		t.Fatal("expected error when queue exhausted")
	}
	if exhausted.StopReason != llm.StopReasonError || exhausted.ErrorMessage != "no more faux responses queued" {
		t.Fatalf("exhausted = %+v", exhausted)
	}
	requireTotalTokensEqualsComponents(t, exhausted.Usage)
	if f.PendingResponses() != 0 || f.CallCount() != 3 {
		t.Fatalf("pending=%d callCount=%d", f.PendingResponses(), f.CallCount())
	}
}

func TestFauxReplaceAndAppendResponses(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	c := userContext("hi")
	mustText := func(want string) {
		t.Helper()
		msg, err := llm.Complete(context.Background(), f.Model(), c, nil)
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if got := textOf(t, msg); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}

	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("first")))
	mustText("first")
	if f.PendingResponses() != 0 {
		t.Fatalf("pending = %d", f.PendingResponses())
	}

	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("second")))
	if f.PendingResponses() != 1 {
		t.Fatalf("pending = %d", f.PendingResponses())
	}
	mustText("second")

	f.AppendResponses(
		llm.FauxRespond(llm.FauxAssistantText("third")),
		llm.FauxRespond(llm.FauxAssistantText("fourth")),
	)
	if f.PendingResponses() != 2 {
		t.Fatalf("pending = %d", f.PendingResponses())
	}
	mustText("third")
	mustText("fourth")
}

func TestFauxFactorySeesContextAndCallCount(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(func(c llm.Context, _ *llm.StreamOptions, callCount int, _ llm.Model) (llm.AssistantMessage, error) {
		return llm.FauxAssistantText(fmt.Sprintf("%d:%d", len(c.Messages), callCount)), nil
	})

	msg, err := llm.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := textOf(t, msg); got != "1:1" {
		t.Fatalf("got %q", got)
	}
}

func TestFauxFactoryErrorBecomesTerminalErrorEvent(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(func(llm.Context, *llm.StreamOptions, int, llm.Model) (llm.AssistantMessage, error) {
		return llm.AssistantMessage{}, errors.New("boom")
	})

	events := collectEvents(llm.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	if len(events) != 1 || events[0].Type != llm.EventError {
		t.Fatalf("events = %v", eventTypes(events))
	}
	if events[0].Message.StopReason != llm.StopReasonError || events[0].Message.ErrorMessage != "boom" {
		t.Fatalf("terminal = %+v", events[0].Message)
	}
}

func TestFauxEstimatesTokensFromSerializedContext(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("done")))

	tool := llm.Tool{
		Name:        "echo",
		Description: "Echo back text",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`),
	}
	prior := llm.FauxAssistantText("prior")
	c := llm.Context{
		SystemPrompt: "sys",
		Messages: []llm.Message{
			llm.UserMessage{
				Content: []llm.UserContent{
					llm.TextContent{Text: "hello"},
					llm.ImageContent{MimeType: "image/png", Data: "abcd"},
				},
			},
			prior,
			llm.ToolResultMessage{
				ToolCallID: "tool-1",
				ToolName:   "echo",
				Content:    []llm.UserContent{llm.TextContent{Text: "tool out"}},
			},
		},
		Tools: []llm.Tool{tool},
	}

	msg, err := llm.Complete(context.Background(), f.Model(), c, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	toolsJSON, _ := json.Marshal([]llm.Tool{tool})
	promptText := strings.Join([]string{
		"system:sys",
		"user:hello\n[image:image/png:4]",
		"assistant:prior",
		"toolResult:echo\ntool out",
		"tools:" + string(toolsJSON),
	}, "\n\n")
	wantInput := (len(promptText) + 3) / 4
	wantOutput := (len("done") + 3) / 4

	if msg.Usage.Input != wantInput || msg.Usage.Output != wantOutput {
		t.Fatalf("usage = %+v, want input %d output %d", msg.Usage, wantInput, wantOutput)
	}
	if msg.Usage.CacheRead != 0 || msg.Usage.CacheWrite != 0 {
		t.Fatalf("expected no cache activity: %+v", msg.Usage)
	}
	requireTotalTokensEqualsComponents(t, msg.Usage)
}

func TestFauxSimulatesPromptCachingPerSession(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		llm.FauxRespond(llm.FauxAssistantText("first")),
		llm.FauxRespond(llm.FauxAssistantText("second")),
	)

	c := llm.Context{SystemPrompt: "Be concise.", Messages: []llm.Message{llm.UserText("hello")}}
	opts := &llm.StreamOptions{SessionID: "session-1", CacheRetention: llm.CacheRetentionShort}

	first, err := llm.Complete(context.Background(), f.Model(), c, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if first.Usage.CacheRead != 0 || first.Usage.CacheWrite == 0 {
		t.Fatalf("first usage = %+v", first.Usage)
	}
	requireTotalTokensEqualsComponents(t, first.Usage)

	c.Messages = append(c.Messages, first, llm.UserText("follow up"))
	second, err := llm.Complete(context.Background(), f.Model(), c, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if second.Usage.CacheRead == 0 {
		t.Fatalf("second usage = %+v, want cacheRead > 0", second.Usage)
	}
	requireTotalTokensEqualsComponents(t, second.Usage)
}

func TestFauxDoesNotShareCacheAcrossSessionsOrWithoutSessionID(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		llm.FauxRespond(llm.FauxAssistantText("first")),
		llm.FauxRespond(llm.FauxAssistantText("second")),
		llm.FauxRespond(llm.FauxAssistantText("third")),
	)

	c := userContext("hello")
	first, err := llm.Complete(context.Background(), f.Model(), c,
		&llm.StreamOptions{SessionID: "session-1", CacheRetention: llm.CacheRetentionShort})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if first.Usage.CacheWrite == 0 {
		t.Fatalf("first usage = %+v", first.Usage)
	}

	c.Messages = append(c.Messages, first, llm.UserText("follow up"))
	second, err := llm.Complete(context.Background(), f.Model(), c,
		&llm.StreamOptions{SessionID: "session-2", CacheRetention: llm.CacheRetentionShort})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if second.Usage.CacheRead != 0 || second.Usage.CacheWrite == 0 {
		t.Fatalf("second usage = %+v", second.Usage)
	}

	third, err := llm.Complete(context.Background(), f.Model(), c, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if third.Usage.CacheRead != 0 || third.Usage.CacheWrite != 0 {
		t.Fatalf("third usage = %+v", third.Usage)
	}
}

func TestFauxNoCachingWhenRetentionNone(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		llm.FauxRespond(llm.FauxAssistantText("first")),
		llm.FauxRespond(llm.FauxAssistantText("second")),
	)

	c := userContext("hello")
	opts := &llm.StreamOptions{SessionID: "session-1", CacheRetention: llm.CacheRetentionNone}
	if _, err := llm.Complete(context.Background(), f.Model(), c, opts); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	c.Messages = append(c.Messages, llm.FauxAssistantText("first"), llm.UserText("follow up"))
	second, err := llm.Complete(context.Background(), f.Model(), c, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if second.Usage.CacheRead != 0 || second.Usage.CacheWrite != 0 {
		t.Fatalf("usage = %+v", second.Usage)
	}
}

func TestFauxStreamsThinkingTextAndToolCallDeltas(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	reply := llm.FauxAssistantMessage(
		llm.FauxThinking("thinking text"),
		llm.FauxText("answer text"),
		llm.ToolCall{ID: "tool-1", Name: "echo", Arguments: map[string]any{"text": "hi", "count": float64(12)}},
	)
	reply.StopReason = llm.StopReasonToolUse
	f.SetResponses(llm.FauxRespond(reply))

	var toolCallDeltas []string
	events := collectEvents(llm.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	for _, ev := range events {
		if ev.Type == llm.EventToolCallDelta {
			toolCallDeltas = append(toolCallDeltas, ev.Delta)
		}
	}

	for _, want := range []llm.EventType{
		llm.EventThinkingStart, llm.EventThinkingDelta,
		llm.EventTextStart, llm.EventTextDelta,
		llm.EventToolCallStart, llm.EventToolCallDelta, llm.EventToolCallEnd,
	} {
		if !containsType(events, want) {
			t.Fatalf("missing event %q in %v", want, eventTypes(events))
		}
	}
	if len(toolCallDeltas) < 2 {
		t.Fatalf("toolcall deltas = %d, want > 1", len(toolCallDeltas))
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(strings.Join(toolCallDeltas, "")), &args); err != nil {
		t.Fatalf("joined deltas not valid JSON: %v", err)
	}
	if args["text"] != "hi" || args["count"] != float64(12) {
		t.Fatalf("args = %+v", args)
	}
}

func TestFauxStreamsExactEventOrderForFixedChunks(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{TokenSizeMin: 1, TokenSizeMax: 1})
	defer f.Unregister()
	reply := llm.FauxAssistantMessage(
		llm.FauxThinking("go"),
		llm.FauxText("ok"),
		llm.ToolCall{ID: "tool-1", Name: "echo", Arguments: map[string]any{}},
	)
	reply.StopReason = llm.StopReasonToolUse
	f.SetResponses(llm.FauxRespond(reply))

	events := collectEvents(llm.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	if !equalTypes(eventTypes(events),
		llm.EventStart,
		llm.EventThinkingStart, llm.EventThinkingDelta, llm.EventThinkingEnd,
		llm.EventTextStart, llm.EventTextDelta, llm.EventTextEnd,
		llm.EventToolCallStart, llm.EventToolCallDelta, llm.EventToolCallEnd,
		llm.EventDone,
	) {
		t.Fatalf("event order = %v", eventTypes(events))
	}
}

func TestFauxStreamsRefusal(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{TokenSizeMin: 1, TokenSizeMax: 1})
	defer f.Unregister()
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantMessage(llm.RefusalContent{Refusal: "no"})))

	stream := llm.Stream(context.Background(), f.Model(), userContext("hi"), nil)
	events := collectEvents(stream)
	msg, err := stream.Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if refusal, ok := msg.Content[0].(llm.RefusalContent); !ok || refusal.Refusal != "no" {
		t.Fatalf("refusal = %#v", msg.Content[0])
	}
	if !equalTypes(eventTypes(events),
		llm.EventStart,
		llm.EventRefusalStart, llm.EventRefusalDelta, llm.EventRefusalEnd,
		llm.EventDone,
	) {
		t.Fatalf("event order = %v", eventTypes(events))
	}
}

func TestFauxStreamsMultipleToolCalls(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	defer f.Unregister()
	reply := llm.FauxAssistantMessage(
		llm.ToolCall{ID: "tool-1", Name: "echo", Arguments: map[string]any{"text": "one"}},
		llm.ToolCall{ID: "tool-2", Name: "echo", Arguments: map[string]any{"text": "two"}},
	)
	reply.StopReason = llm.StopReasonToolUse
	f.SetResponses(llm.FauxRespond(reply))

	events := collectEvents(llm.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	starts, ends := 0, 0
	for _, ev := range events {
		switch ev.Type {
		case llm.EventToolCallStart:
			starts++
		case llm.EventToolCallEnd:
			ends++
		}
	}
	if starts != 2 || ends != 2 {
		t.Fatalf("toolcall starts=%d ends=%d", starts, ends)
	}
}

func TestFauxExplicitErrorMessageStreamsAsTerminalError(t *testing.T) {
	for _, tc := range []struct {
		reason  llm.StopReason
		errText string
	}{
		{llm.StopReasonError, "upstream failed"},
		{llm.StopReasonAborted, "request was aborted"},
	} {
		f := llm.RegisterFauxProvider(llm.FauxOptions{TokenSizeMin: 2, TokenSizeMax: 2})
		reply := llm.FauxAssistantText("partial")
		reply.StopReason = tc.reason
		reply.ErrorMessage = tc.errText
		f.SetResponses(llm.FauxRespond(reply))

		events := collectEvents(llm.Stream(context.Background(), f.Model(), userContext("hi"), nil))
		if !equalTypes(eventTypes(events),
			llm.EventStart, llm.EventTextStart, llm.EventTextDelta, llm.EventTextEnd, llm.EventError,
		) {
			t.Fatalf("%s: event order = %v", tc.reason, eventTypes(events))
		}
		terminal := events[len(events)-1]
		if terminal.Reason != tc.reason || terminal.Message.StopReason != tc.reason || terminal.Message.ErrorMessage != tc.errText {
			t.Fatalf("%s: terminal = %+v", tc.reason, terminal.Message)
		}
		f.Unregister()
	}
}

func TestFauxAbortBeforeFirstChunk(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{TokensPerSecond: 50, TokenSizeMin: 3, TokenSizeMax: 3})
	defer f.Unregister()
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("abcdefghijklmnopqrstuvwxyz")))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := llm.Stream(ctx, f.Model(), userContext("hi"), nil)
	events := collectEvents(stream)
	if len(events) != 1 || events[0].Type != llm.EventError {
		t.Fatalf("events = %v", eventTypes(events))
	}
	if events[0].Reason != llm.StopReasonAborted || events[0].Message.StopReason != llm.StopReasonAborted {
		t.Fatalf("terminal = %+v", events[0].Message)
	}

	_, err := stream.Result()
	var streamErr *llm.StreamError
	if !errors.As(err, &streamErr) || streamErr.Reason != llm.StopReasonAborted {
		t.Fatalf("expected aborted StreamError, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped context cancellation, got %v", err)
	}
}

func TestFauxAbortMidTextStream(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{TokensPerSecond: 100, TokenSizeMin: 3, TokenSizeMax: 3})
	defer f.Unregister()
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("abcdefghijklmnopqrstuvwxyz")))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var events []llm.AssistantMessageEvent
	textDeltas := 0
	for ev := range llm.Stream(ctx, f.Model(), userContext("hi"), nil).Events() {
		events = append(events, ev)
		if ev.Type == llm.EventTextDelta {
			textDeltas++
			cancel()
		}
	}

	if textDeltas != 1 {
		t.Fatalf("text deltas = %d, want 1", textDeltas)
	}
	if !containsType(events, llm.EventTextStart) || !containsType(events, llm.EventError) {
		t.Fatalf("events = %v", eventTypes(events))
	}
	if containsType(events, llm.EventTextEnd) {
		t.Fatalf("unexpected text_end after abort: %v", eventTypes(events))
	}
}

func TestFauxUnregister(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{})
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText("hello")))
	f.Unregister()

	msg, err := llm.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err == nil {
		t.Fatal("expected error for unregistered provider")
	}
	want := "no API provider registered for api: " + f.API
	if msg.ErrorMessage != want {
		t.Fatalf("errorMessage = %q, want %q", msg.ErrorMessage, want)
	}
}

func TestFauxPacingSpreadsDeltasOverTime(t *testing.T) {
	f := llm.RegisterFauxProvider(llm.FauxOptions{TokensPerSecond: 200, TokenSizeMin: 2, TokenSizeMax: 2})
	defer f.Unregister()
	f.SetResponses(llm.FauxRespond(llm.FauxAssistantText(strings.Repeat("a", 32))))

	start := time.Now()
	if _, err := llm.Complete(context.Background(), f.Model(), userContext("hi"), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("elapsed = %v, expected pacing delay", elapsed)
	}
}

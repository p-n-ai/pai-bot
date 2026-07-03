// Port of pi-ai's faux-provider.test.ts plus the total-tokens invariant from
// total-tokens.test.ts: totalTokens must equal input+output+cacheRead+cacheWrite
// on every response — it is the base for context-size accounting.
package piai_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/piai"
)

func requireTotalTokensEqualsComponents(t *testing.T, u piai.Usage) {
	t.Helper()
	computed := u.Input + u.Output + u.CacheRead + u.CacheWrite
	if u.TotalTokens != computed {
		t.Fatalf("totalTokens = %d, want sum of components %d (%+v)", u.TotalTokens, computed, u)
	}
}

func userContext(text string) piai.Context {
	return piai.Context{Messages: []piai.Message{piai.UserText(text)}}
}

func textOf(t *testing.T, msg piai.AssistantMessage) string {
	t.Helper()
	if len(msg.Content) != 1 {
		t.Fatalf("expected single content block, got %d", len(msg.Content))
	}
	block, ok := msg.Content[0].(piai.TextContent)
	if !ok {
		t.Fatalf("expected text block, got %T", msg.Content[0])
	}
	return block.Text
}

func TestFauxRegistersProviderAndEstimatesUsage(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("hello world")))

	c := piai.Context{SystemPrompt: "Be concise.", Messages: []piai.Message{piai.UserText("hi there")}}
	msg, err := piai.Complete(context.Background(), f.Model(), c, nil)
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
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	reply := piai.FauxAssistantMessage(
		piai.FauxThinking("think"),
		piai.FauxToolCall("echo", map[string]any{"text": "hi"}),
		piai.FauxText("done"),
	)
	reply.StopReason = piai.StopReasonToolUse
	f.SetResponses(piai.FauxRespond(reply))

	msg, err := piai.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if msg.StopReason != piai.StopReasonToolUse {
		t.Fatalf("stopReason = %q", msg.StopReason)
	}
	if len(msg.Content) != 3 {
		t.Fatalf("content blocks = %d", len(msg.Content))
	}
	if th, ok := msg.Content[0].(piai.ThinkingContent); !ok || th.Thinking != "think" {
		t.Fatalf("block 0 = %#v", msg.Content[0])
	}
	tc, ok := msg.Content[1].(piai.ToolCall)
	if !ok || tc.Name != "echo" || tc.ID == "" || tc.Arguments["text"] != "hi" {
		t.Fatalf("block 1 = %#v", msg.Content[1])
	}
	if txt, ok := msg.Content[2].(piai.TextContent); !ok || txt.Text != "done" {
		t.Fatalf("block 2 = %#v", msg.Content[2])
	}
	requireTotalTokensEqualsComponents(t, msg.Usage)
}

func TestFauxMultipleModelsAndModelAwareFactories(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{
		Models: []piai.FauxModel{
			{ID: "faux-fast", Name: "Faux Fast"},
			{ID: "faux-thinker", Name: "Faux Thinker", Reasoning: true},
		},
	})
	defer f.Unregister()
	factory := func(_ piai.Context, _ *piai.StreamOptions, _ int, model piai.Model) (piai.AssistantMessage, error) {
		return piai.FauxAssistantText(fmt.Sprintf("%s:%t", model.ID, model.Reasoning)), nil
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

	fastMsg, err := piai.Complete(context.Background(), fast, userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete fast: %v", err)
	}
	thinkerMsg, err := piai.Complete(context.Background(), thinker, userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete thinker: %v", err)
	}
	if textOf(t, fastMsg) != "faux-fast:false" || textOf(t, thinkerMsg) != "faux-thinker:true" {
		t.Fatalf("responses = %q, %q", textOf(t, fastMsg), textOf(t, thinkerMsg))
	}
}

func TestFauxRewritesAPIProviderModel(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{
		API:      "faux:test",
		Provider: "faux-provider",
		Models:   []piai.FauxModel{{ID: "faux-model"}},
	})
	defer f.Unregister()
	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("hello")))

	msg, err := piai.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if msg.API != "faux:test" || msg.Provider != "faux-provider" || msg.Model != "faux-model" {
		t.Fatalf("api/provider/model = %q/%q/%q", msg.API, msg.Provider, msg.Model)
	}
}

func TestFauxConsumesQueueInOrderAndErrorsWhenExhausted(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		piai.FauxRespond(piai.FauxAssistantText("first")),
		piai.FauxRespond(piai.FauxAssistantText("second")),
	)

	c := userContext("hi")
	first, err := piai.Complete(context.Background(), f.Model(), c, nil)
	if err != nil || textOf(t, first) != "first" {
		t.Fatalf("first = %q err=%v", textOf(t, first), err)
	}
	second, err := piai.Complete(context.Background(), f.Model(), c, nil)
	if err != nil || textOf(t, second) != "second" {
		t.Fatalf("second = %q err=%v", textOf(t, second), err)
	}
	exhausted, err := piai.Complete(context.Background(), f.Model(), c, nil)
	if err == nil {
		t.Fatal("expected error when queue exhausted")
	}
	if exhausted.StopReason != piai.StopReasonError || exhausted.ErrorMessage != "no more faux responses queued" {
		t.Fatalf("exhausted = %+v", exhausted)
	}
	requireTotalTokensEqualsComponents(t, exhausted.Usage)
	if f.PendingResponses() != 0 || f.CallCount() != 3 {
		t.Fatalf("pending=%d callCount=%d", f.PendingResponses(), f.CallCount())
	}
}

func TestFauxReplaceAndAppendResponses(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	c := userContext("hi")
	mustText := func(want string) {
		t.Helper()
		msg, err := piai.Complete(context.Background(), f.Model(), c, nil)
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if got := textOf(t, msg); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}

	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("first")))
	mustText("first")
	if f.PendingResponses() != 0 {
		t.Fatalf("pending = %d", f.PendingResponses())
	}

	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("second")))
	if f.PendingResponses() != 1 {
		t.Fatalf("pending = %d", f.PendingResponses())
	}
	mustText("second")

	f.AppendResponses(
		piai.FauxRespond(piai.FauxAssistantText("third")),
		piai.FauxRespond(piai.FauxAssistantText("fourth")),
	)
	if f.PendingResponses() != 2 {
		t.Fatalf("pending = %d", f.PendingResponses())
	}
	mustText("third")
	mustText("fourth")
}

func TestFauxFactorySeesContextAndCallCount(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(func(c piai.Context, _ *piai.StreamOptions, callCount int, _ piai.Model) (piai.AssistantMessage, error) {
		return piai.FauxAssistantText(fmt.Sprintf("%d:%d", len(c.Messages), callCount)), nil
	})

	msg, err := piai.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := textOf(t, msg); got != "1:1" {
		t.Fatalf("got %q", got)
	}
}

func TestFauxFactoryErrorBecomesTerminalErrorEvent(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(func(piai.Context, *piai.StreamOptions, int, piai.Model) (piai.AssistantMessage, error) {
		return piai.AssistantMessage{}, errors.New("boom")
	})

	events := collectEvents(piai.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	if len(events) != 1 || events[0].Type != piai.EventError {
		t.Fatalf("events = %v", eventTypes(events))
	}
	if events[0].Message.StopReason != piai.StopReasonError || events[0].Message.ErrorMessage != "boom" {
		t.Fatalf("terminal = %+v", events[0].Message)
	}
}

func TestFauxEstimatesTokensFromSerializedContext(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("done")))

	tool := piai.Tool{
		Name:        "echo",
		Description: "Echo back text",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`),
	}
	prior := piai.FauxAssistantText("prior")
	c := piai.Context{
		SystemPrompt: "sys",
		Messages: []piai.Message{
			piai.UserMessage{
				Content: []piai.UserContent{
					piai.TextContent{Text: "hello"},
					piai.ImageContent{MimeType: "image/png", Data: "abcd"},
				},
			},
			prior,
			piai.ToolResultMessage{
				ToolCallID: "tool-1",
				ToolName:   "echo",
				Content:    []piai.UserContent{piai.TextContent{Text: "tool out"}},
			},
		},
		Tools: []piai.Tool{tool},
	}

	msg, err := piai.Complete(context.Background(), f.Model(), c, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	toolsJSON, _ := json.Marshal([]piai.Tool{tool})
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
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		piai.FauxRespond(piai.FauxAssistantText("first")),
		piai.FauxRespond(piai.FauxAssistantText("second")),
	)

	c := piai.Context{SystemPrompt: "Be concise.", Messages: []piai.Message{piai.UserText("hello")}}
	opts := &piai.StreamOptions{SessionID: "session-1", CacheRetention: piai.CacheRetentionShort}

	first, err := piai.Complete(context.Background(), f.Model(), c, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if first.Usage.CacheRead != 0 || first.Usage.CacheWrite == 0 {
		t.Fatalf("first usage = %+v", first.Usage)
	}
	requireTotalTokensEqualsComponents(t, first.Usage)

	c.Messages = append(c.Messages, first, piai.UserText("follow up"))
	second, err := piai.Complete(context.Background(), f.Model(), c, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if second.Usage.CacheRead == 0 {
		t.Fatalf("second usage = %+v, want cacheRead > 0", second.Usage)
	}
	requireTotalTokensEqualsComponents(t, second.Usage)
}

func TestFauxDoesNotShareCacheAcrossSessionsOrWithoutSessionID(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		piai.FauxRespond(piai.FauxAssistantText("first")),
		piai.FauxRespond(piai.FauxAssistantText("second")),
		piai.FauxRespond(piai.FauxAssistantText("third")),
	)

	c := userContext("hello")
	first, err := piai.Complete(context.Background(), f.Model(), c,
		&piai.StreamOptions{SessionID: "session-1", CacheRetention: piai.CacheRetentionShort})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if first.Usage.CacheWrite == 0 {
		t.Fatalf("first usage = %+v", first.Usage)
	}

	c.Messages = append(c.Messages, first, piai.UserText("follow up"))
	second, err := piai.Complete(context.Background(), f.Model(), c,
		&piai.StreamOptions{SessionID: "session-2", CacheRetention: piai.CacheRetentionShort})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if second.Usage.CacheRead != 0 || second.Usage.CacheWrite == 0 {
		t.Fatalf("second usage = %+v", second.Usage)
	}

	third, err := piai.Complete(context.Background(), f.Model(), c, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if third.Usage.CacheRead != 0 || third.Usage.CacheWrite != 0 {
		t.Fatalf("third usage = %+v", third.Usage)
	}
}

func TestFauxNoCachingWhenRetentionNone(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	f.SetResponses(
		piai.FauxRespond(piai.FauxAssistantText("first")),
		piai.FauxRespond(piai.FauxAssistantText("second")),
	)

	c := userContext("hello")
	opts := &piai.StreamOptions{SessionID: "session-1", CacheRetention: piai.CacheRetentionNone}
	if _, err := piai.Complete(context.Background(), f.Model(), c, opts); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	c.Messages = append(c.Messages, piai.FauxAssistantText("first"), piai.UserText("follow up"))
	second, err := piai.Complete(context.Background(), f.Model(), c, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if second.Usage.CacheRead != 0 || second.Usage.CacheWrite != 0 {
		t.Fatalf("usage = %+v", second.Usage)
	}
}

func TestFauxStreamsThinkingTextAndToolCallDeltas(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	reply := piai.FauxAssistantMessage(
		piai.FauxThinking("thinking text"),
		piai.FauxText("answer text"),
		piai.ToolCall{ID: "tool-1", Name: "echo", Arguments: map[string]any{"text": "hi", "count": float64(12)}},
	)
	reply.StopReason = piai.StopReasonToolUse
	f.SetResponses(piai.FauxRespond(reply))

	var toolCallDeltas []string
	events := collectEvents(piai.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	for _, ev := range events {
		if ev.Type == piai.EventToolCallDelta {
			toolCallDeltas = append(toolCallDeltas, ev.Delta)
		}
	}

	for _, want := range []piai.EventType{
		piai.EventThinkingStart, piai.EventThinkingDelta,
		piai.EventTextStart, piai.EventTextDelta,
		piai.EventToolCallStart, piai.EventToolCallDelta, piai.EventToolCallEnd,
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
	f := piai.RegisterFauxProvider(piai.FauxOptions{TokenSizeMin: 1, TokenSizeMax: 1})
	defer f.Unregister()
	reply := piai.FauxAssistantMessage(
		piai.FauxThinking("go"),
		piai.FauxText("ok"),
		piai.ToolCall{ID: "tool-1", Name: "echo", Arguments: map[string]any{}},
	)
	reply.StopReason = piai.StopReasonToolUse
	f.SetResponses(piai.FauxRespond(reply))

	events := collectEvents(piai.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	if !equalTypes(eventTypes(events),
		piai.EventStart,
		piai.EventThinkingStart, piai.EventThinkingDelta, piai.EventThinkingEnd,
		piai.EventTextStart, piai.EventTextDelta, piai.EventTextEnd,
		piai.EventToolCallStart, piai.EventToolCallDelta, piai.EventToolCallEnd,
		piai.EventDone,
	) {
		t.Fatalf("event order = %v", eventTypes(events))
	}
}

func TestFauxStreamsMultipleToolCalls(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	defer f.Unregister()
	reply := piai.FauxAssistantMessage(
		piai.ToolCall{ID: "tool-1", Name: "echo", Arguments: map[string]any{"text": "one"}},
		piai.ToolCall{ID: "tool-2", Name: "echo", Arguments: map[string]any{"text": "two"}},
	)
	reply.StopReason = piai.StopReasonToolUse
	f.SetResponses(piai.FauxRespond(reply))

	events := collectEvents(piai.Stream(context.Background(), f.Model(), userContext("hi"), nil))
	starts, ends := 0, 0
	for _, ev := range events {
		switch ev.Type {
		case piai.EventToolCallStart:
			starts++
		case piai.EventToolCallEnd:
			ends++
		}
	}
	if starts != 2 || ends != 2 {
		t.Fatalf("toolcall starts=%d ends=%d", starts, ends)
	}
}

func TestFauxExplicitErrorMessageStreamsAsTerminalError(t *testing.T) {
	for _, tc := range []struct {
		reason  piai.StopReason
		errText string
	}{
		{piai.StopReasonError, "upstream failed"},
		{piai.StopReasonAborted, "request was aborted"},
	} {
		f := piai.RegisterFauxProvider(piai.FauxOptions{TokenSizeMin: 2, TokenSizeMax: 2})
		reply := piai.FauxAssistantText("partial")
		reply.StopReason = tc.reason
		reply.ErrorMessage = tc.errText
		f.SetResponses(piai.FauxRespond(reply))

		events := collectEvents(piai.Stream(context.Background(), f.Model(), userContext("hi"), nil))
		if !equalTypes(eventTypes(events),
			piai.EventStart, piai.EventTextStart, piai.EventTextDelta, piai.EventTextEnd, piai.EventError,
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
	f := piai.RegisterFauxProvider(piai.FauxOptions{TokensPerSecond: 50, TokenSizeMin: 3, TokenSizeMax: 3})
	defer f.Unregister()
	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("abcdefghijklmnopqrstuvwxyz")))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := piai.Stream(ctx, f.Model(), userContext("hi"), nil)
	events := collectEvents(stream)
	if len(events) != 1 || events[0].Type != piai.EventError {
		t.Fatalf("events = %v", eventTypes(events))
	}
	if events[0].Reason != piai.StopReasonAborted || events[0].Message.StopReason != piai.StopReasonAborted {
		t.Fatalf("terminal = %+v", events[0].Message)
	}

	// Cancellation is classified: the error channel says aborted, not error.
	_, err := stream.Result()
	var streamErr *piai.StreamError
	if !errors.As(err, &streamErr) || streamErr.Reason != piai.StopReasonAborted {
		t.Fatalf("expected aborted StreamError, got %v", err)
	}
}

func TestFauxAbortMidTextStream(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{TokensPerSecond: 100, TokenSizeMin: 3, TokenSizeMax: 3})
	defer f.Unregister()
	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("abcdefghijklmnopqrstuvwxyz")))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var events []piai.AssistantMessageEvent
	textDeltas := 0
	for ev := range piai.Stream(ctx, f.Model(), userContext("hi"), nil).Events() {
		events = append(events, ev)
		if ev.Type == piai.EventTextDelta {
			textDeltas++
			cancel()
		}
	}

	if textDeltas != 1 {
		t.Fatalf("text deltas = %d, want 1", textDeltas)
	}
	if !containsType(events, piai.EventTextStart) || !containsType(events, piai.EventError) {
		t.Fatalf("events = %v", eventTypes(events))
	}
	if containsType(events, piai.EventTextEnd) {
		t.Fatalf("unexpected text_end after abort: %v", eventTypes(events))
	}
}

func TestFauxUnregister(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{})
	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText("hello")))
	f.Unregister()

	msg, err := piai.Complete(context.Background(), f.Model(), userContext("hi"), nil)
	if err == nil {
		t.Fatal("expected error for unregistered provider")
	}
	want := "no API provider registered for api: " + f.API
	if msg.ErrorMessage != want {
		t.Fatalf("errorMessage = %q, want %q", msg.ErrorMessage, want)
	}
}

func TestFauxPacingSpreadsDeltasOverTime(t *testing.T) {
	f := piai.RegisterFauxProvider(piai.FauxOptions{TokensPerSecond: 200, TokenSizeMin: 2, TokenSizeMax: 2})
	defer f.Unregister()
	f.SetResponses(piai.FauxRespond(piai.FauxAssistantText(strings.Repeat("a", 32))))

	start := time.Now()
	if _, err := piai.Complete(context.Background(), f.Model(), userContext("hi"), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	// 32 chars → 4 chunks of 2 tokens at 200 tok/s → ≥ 40ms total.
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("elapsed = %v, expected pacing delay", elapsed)
	}
}

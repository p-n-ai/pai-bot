package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func TestCompleteNativeKeepsFallbackInRouter(t *testing.T) {
	router := NewRouter()
	router.Register("text-only", NewMockProvider("flattened"))
	native := &nativeTestProvider{reply: llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "native"}}}}
	router.Register("native", native)
	reply, err := router.CompleteNative(context.Background(), NativeCompletionRequest{Context: llm.Context{Messages: []llm.Message{llm.UserText("hi")}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Content) != 1 || reply.Content[0].(llm.TextContent).Text != "native" {
		t.Fatalf("reply = %#v", reply)
	}
	if native.calls != 1 {
		t.Fatalf("native calls = %d", native.calls)
	}
}

func TestCompleteNativeFallsBackAfterNativeProviderFailure(t *testing.T) {
	router := NewRouterWithConfig(RouterConfig{RetryBackoff: []time.Duration{time.Nanosecond}})
	primary := &nativeTestProvider{err: errors.New("primary unavailable")}
	secondary := &nativeTestProvider{reply: llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "fallback"}}}}
	router.Register("primary", primary)
	router.Register("secondary", secondary)
	reply, err := router.CompleteNative(context.Background(), NativeCompletionRequest{Context: llm.Context{Messages: []llm.Message{llm.UserText("hi")}}})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Content[0].(llm.TextContent).Text != "fallback" {
		t.Fatalf("reply = %#v", reply)
	}
	if primary.calls != 2 || secondary.calls != 1 {
		t.Fatalf("calls = primary %d, secondary %d", primary.calls, secondary.calls)
	}
}

type nativeTestProvider struct {
	reply llm.AssistantMessage
	err   error
	calls int
}

func (p *nativeTestProvider) CompleteNative(context.Context, NativeCompletionRequest) (llm.AssistantMessage, error) {
	p.calls++
	return p.reply, p.err
}
func (*nativeTestProvider) Complete(context.Context, CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{}, errors.New("unexpected")
}
func (*nativeTestProvider) StreamComplete(context.Context, CompletionRequest) (<-chan StreamChunk, error) {
	return nil, errors.New("unexpected")
}
func (*nativeTestProvider) Models() []ModelInfo               { return nil }
func (*nativeTestProvider) HealthCheck(context.Context) error { return nil }

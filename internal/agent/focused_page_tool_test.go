package agent

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

func TestProcessTurnNativeToolThenFinalReplyPersistsOnlyConversationText(t *testing.T) {
	store := NewMemoryStore()
	if err := store.SetUserName("learner-1", "Aina"); err != nil {
		t.Fatal(err)
	}
	pageService, err := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	provider := &nativeScriptProvider{replies: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{llm.ToolCall{ID: "page-1", Name: createFocusedPageToolName, Arguments: map[string]any{"message": "You completed your goal report."}}}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "I made a focused page for your report."}}, Model: "test-model"},
	}}
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{time.Millisecond}})
	router.Register("native", provider)
	engine := NewEngine(EngineConfig{AIRouter: router, Store: store, TenantID: "tenant-1", FocusedPages: pageService, RatingPromptEvery: 100})

	result, err := engine.ProcessTurn(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1", Text: "Give me a report on my goal"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "I made a focused page for your report." {
		t.Fatalf("text = %q", result.Text)
	}
	if result.FocusedPage == nil {
		t.Fatal("focused page artifact is nil")
	}
	parsed, _ := url.Parse(result.FocusedPage.URL)
	if parsed.Fragment == "" {
		t.Fatal("artifact capability is empty")
	}

	conv, ok := store.GetActiveConversation("learner-1")
	if !ok || len(conv.Messages) != 2 {
		t.Fatalf("conversation = %#v", conv)
	}
	if conv.Messages[0].Role != "user" || conv.Messages[1].Role != "assistant" {
		t.Fatalf("stored roles = %#v", conv.Messages)
	}
	for _, message := range conv.Messages {
		if strings.Contains(message.Content, parsed.Fragment) || strings.Contains(message.Content, result.FocusedPage.URL) {
			t.Fatal("capability leaked into conversation history")
		}
	}
	for _, ctx := range provider.contexts {
		for _, message := range ctx.Messages {
			if strings.Contains(messageText(message), parsed.Fragment) || strings.Contains(messageText(message), result.FocusedPage.URL) {
				t.Fatal("capability leaked into model context")
			}
		}
	}
	deliverer := &retryDeliverer{err: errors.New("telegram unavailable")}
	engine.SetTurnDeliverer(deliverer)
	if err := engine.DeliverTurn(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1"}, result); err == nil {
		t.Fatal("first delivery unexpectedly succeeded")
	}
	deliverer.err = nil
	if err := engine.DeliverTurn(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1"}, result); err != nil {
		t.Fatalf("retry delivery error = %v", err)
	}
	if _, err := pageService.Redeem(context.Background(), result.FocusedPage.PublicID, parsed.Fragment); err != nil {
		t.Fatalf("page inactive after delivery failure: %v", err)
	}
}

func TestProcessTurnNativeDirectAnswerProducesNoArtifact(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetUserName("learner-1", "Aina")
	pageService, _ := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	provider := &nativeScriptProvider{replies: []llm.AssistantMessage{{Content: []llm.AssistantContent{llm.TextContent{Text: "Plain tutor reply"}}}}}
	router := ai.NewRouter()
	router.Register("native", provider)
	engine := NewEngine(EngineConfig{AIRouter: router, Store: store, TenantID: "tenant-1", FocusedPages: pageService, RatingPromptEvery: 100})
	result, err := engine.ProcessTurn(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1", Text: "Explain algebra"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Plain tutor reply" || result.FocusedPage != nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestFocusedPagesFallBackToTextWhenNoNativeProviderIsConfigured(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetUserName("learner-1", "Aina")
	pageService, _ := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	provider := &orderedProvider{calls: make(chan int, 1)}
	router := ai.NewRouter()
	router.Register("text", provider)
	engine := NewEngine(EngineConfig{AIRouter: router, Store: store, FocusedPages: pageService, RatingPromptEvery: 100})

	result, err := engine.ProcessTurn(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1", Text: "Explain algebra"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Serialized reply" || result.FocusedPage != nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestProcessAndDeliverRetriesTheSameFocusedPageArtifact(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetUserName("learner-1", "Aina")
	pageService, _ := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	provider := &nativeScriptProvider{replies: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{llm.ToolCall{ID: "page-1", Name: createFocusedPageToolName, Arguments: map[string]any{"message": "Goal report"}}}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "Your report is ready."}}},
	}}
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{time.Millisecond}})
	router.Register("native", provider)
	deliverer := &flakyTurnDeliverer{failures: 2}
	engine := NewEngine(EngineConfig{AIRouter: router, Store: store, TenantID: "tenant-1", FocusedPages: pageService, RatingPromptEvery: 100, TurnDeliverer: deliverer})
	engine.deliveryRetryBackoff = []time.Duration{0, 0}

	result, err := engine.ProcessAndDeliver(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1", Text: "Show my goal report"})
	if err != nil {
		t.Fatal(err)
	}
	if result.FocusedPage == nil || len(deliverer.results) != 3 {
		t.Fatalf("result = %#v, deliveries = %d", result, len(deliverer.results))
	}
	for _, delivered := range deliverer.results {
		if delivered.FocusedPage == nil || delivered.FocusedPage.URL != result.FocusedPage.URL || delivered.Text != result.Text {
			t.Fatalf("delivery changed assembled result: %#v", delivered)
		}
	}
	if len(provider.contexts) != 2 {
		t.Fatalf("model calls = %d, want 2", len(provider.contexts))
	}
}

func TestCreateFocusedPageToolIsIdempotentAndEnforcesOneArtifact(t *testing.T) {
	service, _ := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	tool := &createFocusedPageTool{service: service, input: focusedpage.CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "turn-1", RecipientName: "Aina"}}
	first := tool.Execute(context.Background(), llm.ToolCall{Arguments: map[string]any{"message": "Goal summary"}})
	artifact := tool.artifact
	duplicate := tool.Execute(context.Background(), llm.ToolCall{Arguments: map[string]any{"message": "Goal summary"}})
	second := tool.Execute(context.Background(), llm.ToolCall{Arguments: map[string]any{"message": "Different report"}})
	if first.IsError || duplicate.IsError {
		t.Fatalf("idempotent results = %#v %#v", first, duplicate)
	}
	if artifact == nil || tool.artifact != artifact {
		t.Fatal("duplicate execution replaced artifact")
	}
	if !second.IsError {
		t.Fatal("second semantic artifact was accepted")
	}
}

func TestProcessTurnSerializesActiveConversation(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetUserName("learner-1", "Aina")
	provider := &concurrencyProbeProvider{entered: make(chan struct{}), release: make(chan struct{})}
	router := ai.NewRouter()
	router.Register("probe", provider)
	engine := NewEngine(EngineConfig{AIRouter: router, Store: store, RatingPromptEvery: 100})

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_, _ = engine.ProcessTurn(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1", Text: "Explain algebra"})
		}()
	}
	<-provider.entered
	close(provider.release)
	wg.Wait()
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if provider.maxActive != 1 {
		t.Fatalf("max concurrent model calls = %d, want 1", provider.maxActive)
	}
}

func TestProcessAndDeliverSerializesThroughChannelDelivery(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetUserName("learner-1", "Aina")
	provider := &orderedProvider{calls: make(chan int, 2)}
	router := ai.NewRouter()
	router.Register("ordered", provider)
	deliverer := &blockingFirstDeliverer{entered: make(chan int, 2), releaseFirst: make(chan struct{})}
	engine := NewEngine(EngineConfig{AIRouter: router, Store: store, RatingPromptEvery: 100, TurnDeliverer: deliverer})
	message := chat.InboundMessage{Channel: "telegram", UserID: "learner-1", Text: "Explain algebra"}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = engine.ProcessAndDeliver(context.Background(), message)
	}()
	if call := <-provider.calls; call != 1 {
		t.Fatalf("first provider call = %d", call)
	}
	if delivery := <-deliverer.entered; delivery != 1 {
		t.Fatalf("first delivery = %d", delivery)
	}

	secondStarted := make(chan struct{})
	go func() {
		defer wg.Done()
		close(secondStarted)
		_, _ = engine.ProcessAndDeliver(context.Background(), message)
	}()
	<-secondStarted
	select {
	case call := <-provider.calls:
		t.Fatalf("second turn reached provider before first delivery completed: call %d", call)
	case <-time.After(50 * time.Millisecond):
	}

	close(deliverer.releaseFirst)
	if call := <-provider.calls; call != 2 {
		t.Fatalf("second provider call = %d", call)
	}
	if delivery := <-deliverer.entered; delivery != 2 {
		t.Fatalf("second delivery = %d", delivery)
	}
	wg.Wait()
}

func TestFocusedPageCreationFailureReturnsFinalTextWithoutArtifactOrLeak(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetUserName("learner-1", "Aina")
	pageService, err := focusedpage.NewService(failingFocusedPageStore{}, "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	provider := &nativeScriptProvider{replies: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{llm.ToolCall{ID: "page-1", Name: createFocusedPageToolName, Arguments: map[string]any{"message": "Private goal report"}}}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "Your report is ready here in chat."}}},
	}}
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{time.Millisecond}})
	router.Register("native", provider)
	engine := NewEngine(EngineConfig{AIRouter: router, Store: store, TenantID: "tenant-1", FocusedPages: pageService, RatingPromptEvery: 100})

	result, err := engine.ProcessTurn(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "learner-1", Text: "Show my goal report"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Your report is ready here in chat." || result.FocusedPage != nil {
		t.Fatalf("result = %#v", result)
	}
	secondContext := provider.contexts[1]
	toolResult, ok := secondContext.Messages[len(secondContext.Messages)-1].(llm.ToolResultMessage)
	if !ok || !toolResult.IsError {
		t.Fatalf("tool result = %#v", secondContext.Messages[len(secondContext.Messages)-1])
	}
	toolText := messageText(toolResult)
	if toolText != "Focused page creation failed." || strings.Contains(toolText, "storage-secret") || strings.Contains(toolText, "http") {
		t.Fatalf("unsafe tool error = %q", toolText)
	}
	conv, ok := store.GetActiveConversation("learner-1")
	if !ok || len(conv.Messages) != 2 || conv.Messages[1].Content != result.Text {
		t.Fatalf("conversation = %#v", conv)
	}
}

type nativeScriptProvider struct {
	replies  []llm.AssistantMessage
	contexts []llm.Context
}

type retryDeliverer struct {
	err error
}

func (d *retryDeliverer) DeliverTurn(context.Context, chat.InboundMessage, TurnResult) error {
	return d.err
}

type flakyTurnDeliverer struct {
	failures int
	results  []TurnResult
}

func (d *flakyTurnDeliverer) DeliverTurn(_ context.Context, _ chat.InboundMessage, result TurnResult) error {
	d.results = append(d.results, result)
	if len(d.results) <= d.failures {
		return errors.New("telegram unavailable")
	}
	return nil
}

type concurrencyProbeProvider struct {
	mu        sync.Mutex
	active    int
	maxActive int
	entered   chan struct{}
	release   chan struct{}
	once      sync.Once
}

type orderedProvider struct {
	mu    sync.Mutex
	next  int
	calls chan int
}

func (p *orderedProvider) Complete(context.Context, ai.CompletionRequest) (ai.CompletionResponse, error) {
	p.mu.Lock()
	p.next++
	call := p.next
	p.mu.Unlock()
	p.calls <- call
	return ai.CompletionResponse{Content: "Serialized reply"}, nil
}
func (*orderedProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("unexpected stream")
}
func (*orderedProvider) Models() []ai.ModelInfo            { return nil }
func (*orderedProvider) HealthCheck(context.Context) error { return nil }

type blockingFirstDeliverer struct {
	mu           sync.Mutex
	calls        int
	entered      chan int
	releaseFirst chan struct{}
}

func (d *blockingFirstDeliverer) DeliverTurn(context.Context, chat.InboundMessage, TurnResult) error {
	d.mu.Lock()
	d.calls++
	call := d.calls
	d.mu.Unlock()
	d.entered <- call
	if call == 1 {
		<-d.releaseFirst
	}
	return nil
}

type failingFocusedPageStore struct{}

func (failingFocusedPageStore) CreateOrGet(context.Context, focusedpage.CreateRecord) (focusedpage.Page, error) {
	return focusedpage.Page{}, errors.New("storage-secret https://private.example")
}
func (failingFocusedPageStore) Redeem(context.Context, string, []byte, time.Time) (focusedpage.Page, error) {
	return focusedpage.Page{}, errors.New("unexpected redeem")
}
func (failingFocusedPageStore) Revoke(context.Context, string, string, string, time.Time) error {
	return errors.New("unexpected revoke")
}

func (p *concurrencyProbeProvider) Complete(context.Context, ai.CompletionRequest) (ai.CompletionResponse, error) {
	p.mu.Lock()
	p.active++
	if p.active > p.maxActive {
		p.maxActive = p.active
	}
	p.once.Do(func() { close(p.entered) })
	p.mu.Unlock()
	<-p.release
	p.mu.Lock()
	p.active--
	p.mu.Unlock()
	return ai.CompletionResponse{Content: "Serialized reply"}, nil
}
func (*concurrencyProbeProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("unexpected stream")
}
func (*concurrencyProbeProvider) Models() []ai.ModelInfo            { return nil }
func (*concurrencyProbeProvider) HealthCheck(context.Context) error { return nil }

func (p *nativeScriptProvider) CompleteNative(_ context.Context, req ai.NativeCompletionRequest) (llm.AssistantMessage, error) {
	p.contexts = append(p.contexts, req.Context)
	if len(p.replies) == 0 {
		return llm.AssistantMessage{}, errors.New("unexpected model call")
	}
	reply := p.replies[0]
	p.replies = p.replies[1:]
	return reply, nil
}
func (*nativeScriptProvider) Complete(context.Context, ai.CompletionRequest) (ai.CompletionResponse, error) {
	return ai.CompletionResponse{}, errors.New("flattened completion called")
}
func (*nativeScriptProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("unexpected stream")
}
func (*nativeScriptProvider) Models() []ai.ModelInfo            { return nil }
func (*nativeScriptProvider) HealthCheck(context.Context) error { return nil }

func messageText(message llm.Message) string {
	var parts []string
	switch value := message.(type) {
	case llm.UserMessage:
		for _, content := range value.Content {
			if text, ok := content.(llm.TextContent); ok {
				parts = append(parts, text.Text)
			}
		}
	case llm.AssistantMessage:
		for _, content := range value.Content {
			if text, ok := content.(llm.TextContent); ok {
				parts = append(parts, text.Text)
			}
		}
	case llm.ToolResultMessage:
		for _, content := range value.Content {
			if text, ok := content.(llm.TextContent); ok {
				parts = append(parts, text.Text)
			}
		}
	}
	return strings.Join(parts, " ")
}

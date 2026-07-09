package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	fauxDefaultProvider     = "faux"
	fauxDefaultModelID      = "faux-1"
	fauxDefaultModelName    = "Faux Model"
	fauxDefaultBaseURL      = "http://localhost:0"
	fauxDefaultMinTokenSize = 3
	fauxDefaultMaxTokenSize = 5
)

var fauxCounter atomic.Int64

type FauxStep func(c Context, opts *StreamOptions, callCount int, model Model) (AssistantMessage, error)

func FauxRespond(msg AssistantMessage) FauxStep {
	return func(Context, *StreamOptions, int, Model) (AssistantMessage, error) { return msg, nil }
}

func FauxText(text string) TextContent { return TextContent{Text: text} }

func FauxThinking(thinking string) ThinkingContent { return ThinkingContent{Thinking: thinking} }

func FauxToolCall(name string, arguments map[string]any) ToolCall {
	return ToolCall{
		ID:        fmt.Sprintf("tool:%d", fauxCounter.Add(1)),
		Name:      name,
		Arguments: arguments,
	}
}

func FauxAssistantMessage(blocks ...AssistantContent) AssistantMessage {
	return AssistantMessage{
		Content:    blocks,
		API:        "faux",
		Provider:   fauxDefaultProvider,
		Model:      fauxDefaultModelID,
		StopReason: StopReasonStop,
		Timestamp:  time.Now(),
	}
}

func FauxAssistantText(text string) AssistantMessage {
	return FauxAssistantMessage(FauxText(text))
}

type FauxModel struct {
	ID        string
	Name      string
	Reasoning bool
}

type FauxOptions struct {
	API             string
	Provider        string
	Models          []FauxModel
	TokensPerSecond float64
	TokenSizeMin    int
	TokenSizeMax    int
}

type FauxProvider struct {
	API      string
	Models   []Model
	sourceID string
	provider string
	minTok   int
	maxTok   int
	tps      float64

	mu          sync.Mutex
	pending     []FauxStep
	callCount   int
	promptCache map[string]string
}

func RegisterFauxProvider(opts FauxOptions) *FauxProvider {
	n := fauxCounter.Add(1)
	api := opts.API
	if api == "" {
		api = fmt.Sprintf("faux:%d", n)
	}
	provider := opts.Provider
	if provider == "" {
		provider = fauxDefaultProvider
	}
	minTok := opts.TokenSizeMin
	if minTok < 1 {
		minTok = fauxDefaultMinTokenSize
	}
	maxTok := opts.TokenSizeMax
	if maxTok < minTok {
		maxTok = max(minTok, fauxDefaultMaxTokenSize)
	}
	defs := opts.Models
	if len(defs) == 0 {
		defs = []FauxModel{{ID: fauxDefaultModelID, Name: fauxDefaultModelName}}
	}
	f := &FauxProvider{
		API:         api,
		sourceID:    fmt.Sprintf("faux-provider:%d", n),
		provider:    provider,
		minTok:      minTok,
		maxTok:      maxTok,
		tps:         opts.TokensPerSecond,
		promptCache: map[string]string{},
	}
	for _, def := range defs {
		name := def.Name
		if name == "" {
			name = def.ID
		}
		f.Models = append(f.Models, Model{
			ID:            def.ID,
			Name:          name,
			API:           api,
			Provider:      provider,
			BaseURL:       fauxDefaultBaseURL,
			Reasoning:     def.Reasoning,
			ContextWindow: 128000,
			MaxTokens:     16384,
		})
	}
	RegisterProvider(api, f.stream, f.sourceID)
	return f
}

func (f *FauxProvider) Model() Model { return f.Models[0] }

func (f *FauxProvider) ModelByID(id string) (Model, bool) {
	for _, m := range f.Models {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}

func (f *FauxProvider) SetResponses(steps ...FauxStep) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = append([]FauxStep(nil), steps...)
}

func (f *FauxProvider) AppendResponses(steps ...FauxStep) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = append(f.pending, steps...)
}

func (f *FauxProvider) PendingResponses() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pending)
}

func (f *FauxProvider) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

func (f *FauxProvider) Unregister() { UnregisterProviders(f.sourceID) }

func (f *FauxProvider) stream(ctx context.Context, model Model, c Context, opts *StreamOptions) *EventStream {
	s := NewEventStream()
	fail := func(err error) {
		cause := err
		msg := f.errorMessage(model, err.Error())
		if ctxErr := ctx.Err(); ctxErr != nil {
			msg.StopReason = StopReasonAborted
			if !errors.Is(err, ctxErr) {
				cause = errors.Join(ctxErr, err)
			}
		}
		s.endWithError(msg, cause)
	}
	f.mu.Lock()
	var step FauxStep
	if len(f.pending) > 0 {
		step = f.pending[0]
		f.pending = f.pending[1:]
	}
	f.callCount++
	count := f.callCount
	f.mu.Unlock()

	go func() {
		if step == nil {
			err := fmt.Errorf("no more faux responses queued")
			fail(err)
			return
		}
		resolved, err := step(c, opts, count, model)
		if err != nil {
			fail(err)
			return
		}
		resolved.API = f.API
		resolved.Provider = f.provider
		resolved.Model = model.ID
		if resolved.Timestamp.IsZero() {
			resolved.Timestamp = time.Now()
		}
		resolved, err = f.withUsageEstimate(resolved, c, opts)
		if err != nil {
			fail(err)
			return
		}
		if err := f.streamWithDeltas(ctx, s, resolved); err != nil {
			fail(err)
		}
	}()
	return s
}

func (f *FauxProvider) errorMessage(model Model, errText string) AssistantMessage {
	return AssistantMessage{
		API:          f.API,
		Provider:     f.provider,
		Model:        model.ID,
		StopReason:   StopReasonError,
		ErrorMessage: errText,
		Timestamp:    time.Now(),
	}
}

func (f *FauxProvider) streamWithDeltas(ctx context.Context, s *EventStream, msg AssistantMessage) error {
	partial := msg
	partial.Content = nil
	abort := func() {
		aborted := partial
		aborted.StopReason = StopReasonAborted
		aborted.ErrorMessage = "request was aborted"
		aborted.Timestamp = time.Now()
		s.Push(AssistantMessageEvent{Type: EventError, Reason: StopReasonAborted, Message: &aborted, Err: ctx.Err()})
	}
	push := func(typ EventType, mutate func(*AssistantMessageEvent)) {
		snapshot := partial
		snapshot.Content = append([]AssistantContent(nil), partial.Content...)
		ev := AssistantMessageEvent{Type: typ, Partial: &snapshot}
		if mutate != nil {
			mutate(&ev)
		}
		s.Push(ev)
	}

	if ctx.Err() != nil {
		abort()
		return nil
	}
	push(EventStart, nil)

	for i, block := range msg.Content {
		switch b := block.(type) {
		case ThinkingContent:
			partial.Content = append(partial.Content, ThinkingContent{})
			push(EventThinkingStart, func(ev *AssistantMessageEvent) { ev.ContentIndex = i })
			written := 0
			for _, chunk := range f.splitByTokenSize(b.Thinking) {
				f.pace(chunk)
				if ctx.Err() != nil {
					abort()
					return nil
				}
				written += len(chunk)
				partial.Content[i] = ThinkingContent{Thinking: b.Thinking[:written]}
				push(EventThinkingDelta, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.Delta = chunk })
			}
			push(EventThinkingEnd, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.Content = b.Thinking })
		case TextContent:
			partial.Content = append(partial.Content, TextContent{})
			push(EventTextStart, func(ev *AssistantMessageEvent) { ev.ContentIndex = i })
			written := 0
			for _, chunk := range f.splitByTokenSize(b.Text) {
				f.pace(chunk)
				if ctx.Err() != nil {
					abort()
					return nil
				}
				written += len(chunk)
				partial.Content[i] = TextContent{Text: b.Text[:written]}
				push(EventTextDelta, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.Delta = chunk })
			}
			push(EventTextEnd, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.Content = b.Text })
		case RefusalContent:
			partial.Content = append(partial.Content, RefusalContent{})
			push(EventRefusalStart, func(ev *AssistantMessageEvent) { ev.ContentIndex = i })
			written := 0
			for _, chunk := range f.splitByTokenSize(b.Refusal) {
				f.pace(chunk)
				if ctx.Err() != nil {
					abort()
					return nil
				}
				written += len(chunk)
				partial.Content[i] = RefusalContent{Refusal: b.Refusal[:written]}
				push(EventRefusalDelta, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.Delta = chunk })
			}
			push(EventRefusalEnd, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.Content = b.Refusal })
		case ToolCall:
			args, err := marshalToolArguments(b.Arguments)
			if err != nil {
				return fmt.Errorf("faux: tool call %q arguments: %w", b.Name, err)
			}
			partial.Content = append(partial.Content, ToolCall{ID: b.ID, Name: b.Name, Arguments: map[string]any{}})
			push(EventToolCallStart, func(ev *AssistantMessageEvent) { ev.ContentIndex = i })
			for _, chunk := range f.splitByTokenSize(args) {
				f.pace(chunk)
				if ctx.Err() != nil {
					abort()
					return nil
				}
				push(EventToolCallDelta, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.Delta = chunk })
			}
			partial.Content[i] = b
			toolCall := b
			push(EventToolCallEnd, func(ev *AssistantMessageEvent) { ev.ContentIndex = i; ev.ToolCall = &toolCall })
		default:

			panic(fmt.Sprintf("llm: unhandled assistant content block %T", block))
		}
	}

	final := msg
	if final.StopReason == StopReasonError || final.StopReason == StopReasonAborted {
		s.Push(AssistantMessageEvent{Type: EventError, Reason: final.StopReason, Message: &final})
		return nil
	}
	s.Push(AssistantMessageEvent{Type: EventDone, Reason: final.StopReason, Message: &final})
	return nil
}

func (f *FauxProvider) pace(chunk string) {
	if f.tps <= 0 {
		return
	}
	time.Sleep(time.Duration(float64(estimateTokens(chunk)) / f.tps * float64(time.Second)))
}

func (f *FauxProvider) splitByTokenSize(text string) []string {
	var chunks []string
	for i := 0; i < len(text); {
		tok := f.minTok + rand.IntN(f.maxTok-f.minTok+1)
		size := max(1, tok*4)
		end := min(len(text), i+size)
		chunks = append(chunks, text[i:end])
		i = end
	}
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	return chunks
}

func (f *FauxProvider) withUsageEstimate(msg AssistantMessage, c Context, opts *StreamOptions) (AssistantMessage, error) {
	promptText, err := serializeContext(c)
	if err != nil {
		return AssistantMessage{}, fmt.Errorf("faux: serialize context: %w", err)
	}
	outputText, err := assistantContentToText(msg.Content)
	if err != nil {
		return AssistantMessage{}, fmt.Errorf("faux: serialize response: %w", err)
	}
	promptTokens := estimateTokens(promptText)
	outputTokens := estimateTokens(outputText)
	input := promptTokens
	cacheRead, cacheWrite := 0, 0

	if opts != nil && opts.SessionID != "" && opts.CacheRetention != CacheRetentionNone {
		f.mu.Lock()
		previous := f.promptCache[opts.SessionID]
		f.promptCache[opts.SessionID] = promptText
		f.mu.Unlock()
		if previous != "" {
			cached := commonPrefixLen(previous, promptText)
			cacheRead = estimateTokens(previous[:cached])
			cacheWrite = estimateTokens(promptText[cached:])
			input = max(0, promptTokens-cacheRead)
		} else {
			cacheWrite = promptTokens
		}
	}

	msg.Usage = Usage{
		Input:       input,
		Output:      outputTokens,
		CacheRead:   cacheRead,
		CacheWrite:  cacheWrite,
		TotalTokens: input + outputTokens + cacheRead + cacheWrite,
	}
	return msg, nil
}

func estimateTokens(text string) int { return (len(text) + 3) / 4 }

func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func userContentToText(content []UserContent) string {
	parts := make([]string, 0, len(content))
	for _, block := range content {
		switch b := block.(type) {
		case TextContent:
			parts = append(parts, b.Text)
		case ImageContent:
			parts = append(parts, fmt.Sprintf("[image:%s:%d]", b.MimeType, len(b.Data)))
		}
	}
	return strings.Join(parts, "\n")
}

func assistantContentToText(content []AssistantContent) (string, error) {
	parts := make([]string, 0, len(content))
	for _, block := range content {
		switch b := block.(type) {
		case TextContent:
			parts = append(parts, b.Text)
		case ThinkingContent:
			parts = append(parts, b.Thinking)
		case RefusalContent:
			parts = append(parts, b.Refusal)
		case ToolCall:
			args, err := marshalToolArguments(b.Arguments)
			if err != nil {
				return "", fmt.Errorf("tool call %q arguments: %w", b.Name, err)
			}
			parts = append(parts, b.Name+":"+args)
		}
	}
	return strings.Join(parts, "\n"), nil
}

func messageToText(m Message) (role, text string, err error) {
	switch msg := m.(type) {
	case UserMessage:
		return "user", userContentToText(msg.Content), nil
	case AssistantMessage:
		text, err := assistantContentToText(msg.Content)
		return "assistant", text, err
	case ToolResultMessage:
		parts := []string{msg.ToolName}
		for _, block := range msg.Content {
			parts = append(parts, userContentToText([]UserContent{block}))
		}
		return "toolResult", strings.Join(parts, "\n"), nil
	}
	return "", "", nil
}

func serializeContext(c Context) (string, error) {
	var parts []string
	if c.SystemPrompt != "" {
		parts = append(parts, "system:"+c.SystemPrompt)
	}
	for _, m := range c.Messages {
		role, text, err := messageToText(m)
		if err != nil {
			return "", err
		}
		parts = append(parts, role+":"+text)
	}
	if len(c.Tools) > 0 {
		tools, err := json.Marshal(c.Tools)
		if err != nil {
			return "", fmt.Errorf("tools: %w", err)
		}
		parts = append(parts, "tools:"+string(tools))
	}
	return strings.Join(parts, "\n\n"), nil
}

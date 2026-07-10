package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const APIOpenAICompletions = "openai-completions"

func RegisterOpenAICompletions() {
	RegisterProvider(APIOpenAICompletions, StreamOpenAICompletions, "builtin:openai-completions")
}

var openAIHTTPClient = &http.Client{}

func StreamOpenAICompletions(ctx context.Context, model Model, c Context, opts *StreamOptions) *EventStream {
	s := NewEventStream()
	go func() {
		out := AssistantMessage{
			API:        model.API,
			Provider:   model.Provider,
			Model:      model.ID,
			StopReason: StopReasonStop,
			Timestamp:  time.Now(),
		}
		fail := func(err error) {
			var cause error
			out.StopReason, cause = classifyStreamFailure(ctx, err)
			out.ErrorMessage = err.Error()
			out.Timestamp = time.Now()
			s.Push(AssistantMessageEvent{Type: EventError, Reason: out.StopReason, Message: &out, Err: cause})
		}

		if opts == nil || opts.APIKey == "" {
			fail(fmt.Errorf("no API key for provider: %s", model.Provider))
			return
		}
		body, err := buildOpenAIRequest(model, c, opts)
		if err != nil {
			fail(err)
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint(model.BaseURL), bytes.NewReader(body))
		if err != nil {
			fail(err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+opts.APIKey)
		req.Header.Set("Accept", "text/event-stream")
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}

		resp, err := openAIHTTPClient.Do(req)
		if err != nil {
			fail(err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			fail(fmt.Errorf("openai-completions: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg))))
			return
		}

		if err := consumeOpenAIStream(ctx, s, &out, model, resp.Body); err != nil {
			fail(err)
		}
	}()
	return s
}

func openAIEndpoint(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/chat/completions"
}

type oaMessage struct {
	Role       string       `json:"role"`
	Content    any          `json:"content,omitempty"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type oaToolCall struct {
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Function oaFunction `json:"function"`
}

type oaFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaTextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type oaImagePart struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

func buildOpenAIRequest(model Model, c Context, opts *StreamOptions) ([]byte, error) {
	messages, err := convertOpenAIMessages(model, c)
	if err != nil {
		return nil, err
	}
	params := map[string]any{
		"model":          model.ID,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if strings.Contains(model.BaseURL, "api.openai.com") {
		params["store"] = false
	}
	if opts.Temperature != nil {
		params["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens > 0 {
		params["max_completion_tokens"] = opts.MaxTokens
	}
	if len(c.Tools) > 0 {
		params["tools"] = convertOpenAITools(c.Tools)
	} else if hasToolHistory(c.Messages) {

		params["tools"] = []oaTool{}
	}
	return json.Marshal(params)
}

func hasToolHistory(messages []Message) bool {
	for _, m := range messages {
		switch msg := m.(type) {
		case ToolResultMessage:
			return true
		case AssistantMessage:
			for _, b := range msg.Content {
				if _, ok := b.(ToolCall); ok {
					return true
				}
			}
		}
	}
	return false
}

func convertOpenAITools(tools []Tool) []oaTool {
	out := make([]oaTool, len(tools))
	for i, t := range tools {
		out[i].Type = "function"
		out[i].Function.Name = t.Name
		out[i].Function.Description = t.Description
		out[i].Function.Parameters = t.Parameters
	}
	return out
}

func convertOpenAIMessages(model Model, c Context) ([]oaMessage, error) {
	var params []oaMessage
	if c.SystemPrompt != "" {
		role := "system"
		if model.Reasoning {
			role = "developer"
		}
		params = append(params, oaMessage{Role: role, Content: c.SystemPrompt})
	}
	for _, m := range c.Messages {
		switch msg := m.(type) {
		case SystemMessage:
			return nil, fmt.Errorf("openai-completions: ordered system messages are unsupported")
		case UserMessage:
			converted, err := convertOpenAIUserMessage(msg)
			if err != nil {
				return nil, err
			}
			params = append(params, converted...)
		case AssistantMessage:
			var texts []string
			var toolCalls []oaToolCall
			for _, b := range msg.Content {
				switch block := b.(type) {
				case TextContent:
					if strings.TrimSpace(block.Text) != "" {
						texts = append(texts, block.Text)
					}
				case ToolCall:
					args, err := marshalToolArguments(block.Arguments)
					if err != nil {
						return nil, fmt.Errorf("openai-completions: tool call %q arguments: %w", block.Name, err)
					}
					toolCalls = append(toolCalls, oaToolCall{
						ID: block.ID, Type: "function",
						Function: oaFunction{Name: block.Name, Arguments: args},
					})
				case ThinkingContent:

				}
			}
			am := oaMessage{Role: "assistant", ToolCalls: toolCalls}
			if text := strings.Join(texts, ""); text != "" {
				am.Content = text
			}

			if am.Content == nil && len(am.ToolCalls) == 0 {
				continue
			}
			params = append(params, am)
		case ToolResultMessage:
			var texts []string
			for _, b := range msg.Content {
				if _, ok := b.(ImageURLContent); ok {
					return nil, fmt.Errorf("openai-completions: remote image URL content is unsupported")
				}
				if t, ok := b.(TextContent); ok {
					texts = append(texts, t.Text)
				}
			}
			content := strings.Join(texts, "\n")
			if content == "" {
				content = "(no text result)"
			}
			params = append(params, oaMessage{Role: "tool", Content: content, ToolCallID: msg.ToolCallID})
		}
	}
	return params, nil
}

func convertOpenAIUserMessage(msg UserMessage) ([]oaMessage, error) {
	onlyText := true
	for _, b := range msg.Content {
		if _, ok := b.(TextContent); !ok {
			onlyText = false
			break
		}
	}
	if onlyText {
		var texts []string
		for _, b := range msg.Content {
			texts = append(texts, b.(TextContent).Text)
		}
		return []oaMessage{{Role: "user", Content: strings.Join(texts, "\n")}}, nil
	}
	var parts []any
	for _, b := range msg.Content {
		switch block := b.(type) {
		case TextContent:
			parts = append(parts, oaTextPart{Type: "text", Text: block.Text})
		case ImageContent:
			p := oaImagePart{Type: "image_url"}
			p.ImageURL.URL = "data:" + block.MimeType + ";base64," + block.Data
			parts = append(parts, p)
		case ImageURLContent:
			return nil, fmt.Errorf("openai-completions: remote image URL content is unsupported")
		}
	}
	if len(parts) == 0 {
		return nil, nil
	}
	return []oaMessage{{Role: "user", Content: parts}}, nil
}

type oaChunk struct {
	ID      string     `json:"id"`
	Model   string     `json:"model"`
	Usage   *oaUsage   `json:"usage"`
	Choices []oaChoice `json:"choices"`
}

type oaUsage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	PromptTokensDetails *struct {
		CachedTokens     int `json:"cached_tokens"`
		CacheWriteTokens int `json:"cache_write_tokens"`
	} `json:"prompt_tokens_details"`
}

type oaChoice struct {
	FinishReason string `json:"finish_reason"`
	Delta        struct {
		Content   string `json:"content"`
		Refusal   string `json:"refusal"`
		ToolCalls []struct {
			Index    *int   `json:"index"`
			ID       string `json:"id"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"delta"`
}

type streamingToolCall struct {
	contentIndex int
	partialArgs  strings.Builder
}

func consumeOpenAIStream(ctx context.Context, s *EventStream, out *AssistantMessage, model Model, body io.Reader) error {
	s.Push(AssistantMessageEvent{Type: EventStart, Partial: snapshot(out)})

	textIndex := -1
	toolByStreamIndex := map[int]*streamingToolCall{}
	var toolOrder []*streamingToolCall
	var currentTool *streamingToolCall
	hasFinish := false

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk oaChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("openai-completions: bad SSE chunk: %w", err)
		}

		if out.ResponseID == "" {
			out.ResponseID = chunk.ID
		}
		if out.ResponseModel == "" && chunk.Model != "" && chunk.Model != model.ID {
			out.ResponseModel = chunk.Model
		}
		if chunk.Usage != nil {
			out.Usage = parseOpenAIUsage(*chunk.Usage, model)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]

		if choice.FinishReason != "" {
			reason, errMsg := mapOpenAIStopReason(choice.FinishReason)
			out.StopReason = reason
			out.ErrorMessage = errMsg
			hasFinish = true
		}

		textDelta := choice.Delta.Content + choice.Delta.Refusal
		if textDelta != "" {
			if textIndex == -1 {
				out.Content = append(out.Content, TextContent{})
				textIndex = len(out.Content) - 1
				s.Push(AssistantMessageEvent{Type: EventTextStart, ContentIndex: textIndex, Partial: snapshot(out)})
			}
			block := out.Content[textIndex].(TextContent)
			block.Text += textDelta
			out.Content[textIndex] = block
			s.Push(AssistantMessageEvent{Type: EventTextDelta, ContentIndex: textIndex, Delta: textDelta, Partial: snapshot(out)})
		}

		for _, tc := range choice.Delta.ToolCalls {
			tool := currentTool
			if tc.Index != nil {
				if existing, ok := toolByStreamIndex[*tc.Index]; ok {
					tool = existing
				} else {
					tool = nil
				}
			} else if tc.ID != "" {
				tool = nil
			}
			if tool == nil {
				out.Content = append(out.Content, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: map[string]any{}})
				tool = &streamingToolCall{contentIndex: len(out.Content) - 1}
				if tc.Index != nil {
					toolByStreamIndex[*tc.Index] = tool
				}
				toolOrder = append(toolOrder, tool)
				s.Push(AssistantMessageEvent{Type: EventToolCallStart, ContentIndex: tool.contentIndex, Partial: snapshot(out)})
			}
			currentTool = tool
			block := out.Content[tool.contentIndex].(ToolCall)
			if block.ID == "" && tc.ID != "" {
				block.ID = tc.ID
			}
			if block.Name == "" && tc.Function.Name != "" {
				block.Name = tc.Function.Name
			}
			out.Content[tool.contentIndex] = block
			if tc.Function.Arguments != "" {
				tool.partialArgs.WriteString(tc.Function.Arguments)
			}
			s.Push(AssistantMessageEvent{Type: EventToolCallDelta, ContentIndex: tool.contentIndex, Delta: tc.Function.Arguments, Partial: snapshot(out)})
		}
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("openai-completions: reading stream: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if textIndex != -1 {
		text := out.Content[textIndex].(TextContent).Text
		s.Push(AssistantMessageEvent{Type: EventTextEnd, ContentIndex: textIndex, Content: text, Partial: snapshot(out)})
	}
	for _, tool := range toolOrder {
		block := out.Content[tool.contentIndex].(ToolCall)
		args, err := parseToolArguments(tool.partialArgs.String())
		if err != nil {
			return fmt.Errorf("openai-completions: tool call %q arguments: %w", block.Name, err)
		}
		block.Arguments = args
		out.Content[tool.contentIndex] = block
		s.Push(AssistantMessageEvent{Type: EventToolCallEnd, ContentIndex: tool.contentIndex, ToolCall: &block, Partial: snapshot(out)})
	}

	if out.StopReason == StopReasonError {
		return fmt.Errorf("%s", out.ErrorMessage)
	}
	if !hasFinish {
		return fmt.Errorf("openai-completions: stream ended without finish_reason")
	}
	out.Timestamp = time.Now()
	s.Push(AssistantMessageEvent{Type: EventDone, Reason: out.StopReason, Message: out})
	return nil
}

func snapshot(out *AssistantMessage) *AssistantMessage {
	cp := *out
	cp.Content = append([]AssistantContent(nil), out.Content...)
	cp.ReasoningDetails = append([]ReasoningDetail(nil), out.ReasoningDetails...)
	return &cp
}

func parseOpenAIUsage(raw oaUsage, model Model) Usage {
	cacheRead, cacheWrite := 0, 0
	if raw.PromptTokensDetails != nil {
		cacheRead = raw.PromptTokensDetails.CachedTokens
		cacheWrite = raw.PromptTokensDetails.CacheWriteTokens
	}
	input := max(0, raw.PromptTokens-cacheRead-cacheWrite)
	u := Usage{
		Input:       input,
		Output:      raw.CompletionTokens,
		CacheRead:   cacheRead,
		CacheWrite:  cacheWrite,
		TotalTokens: input + raw.CompletionTokens + cacheRead + cacheWrite,
	}
	perTok := func(rate float64, tokens int) float64 { return rate * float64(tokens) / 1e6 }
	u.Cost = Cost{
		Input:      perTok(model.Cost.Input, u.Input),
		Output:     perTok(model.Cost.Output, u.Output),
		CacheRead:  perTok(model.Cost.CacheRead, u.CacheRead),
		CacheWrite: perTok(model.Cost.CacheWrite, u.CacheWrite),
	}
	u.Cost.Total = u.Cost.Input + u.Cost.Output + u.Cost.CacheRead + u.Cost.CacheWrite
	return u
}

func mapOpenAIStopReason(reason string) (StopReason, string) {
	switch reason {
	case "stop", "end":
		return StopReasonStop, ""
	case "length":
		return StopReasonLength, ""
	case "tool_calls", "function_call":
		return StopReasonToolUse, ""
	default:
		return StopReasonError, "provider finish_reason: " + reason
	}
}

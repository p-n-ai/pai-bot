package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/models/operations"
	"github.com/OpenRouterTeam/go-sdk/models/sdkerrors"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
	"github.com/OpenRouterTeam/go-sdk/retry"
	sdkstream "github.com/OpenRouterTeam/go-sdk/types/stream"
)

const APIOpenRouterChat = "openrouter-chat"

const (
	openRouterDefaultBaseURL = "https://openrouter.ai/api/v1"
	openRouterHTTPReferer    = "https://pandai.org"
	openRouterTitle          = "P&AI Bot"
)

var openRouterHTTPClient = &http.Client{}

func RegisterOpenRouterChat() {
	RegisterProvider(APIOpenRouterChat, StreamOpenRouterChat, "builtin:openrouter-chat")
}

func StreamOpenRouterChat(ctx context.Context, model Model, c Context, opts *StreamOptions) *EventStream {
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
			cause := err
			out.StopReason = StopReasonError
			if ctxErr := ctx.Err(); ctxErr != nil {
				out.StopReason = StopReasonAborted
				if !errors.Is(err, ctxErr) {
					cause = errors.Join(ctxErr, err)
				}
			}
			out.ErrorMessage = err.Error()
			out.Timestamp = time.Now()
			s.Push(AssistantMessageEvent{Type: EventError, Reason: out.StopReason, Message: &out, Err: cause})
		}

		if opts == nil || opts.APIKey == "" {
			fail(fmt.Errorf("no API key for provider: %s", model.Provider))
			return
		}
		req, err := buildOpenRouterRequest(model, c, opts)
		if err != nil {
			fail(err)
			return
		}

		clientOpts := []openrouter.SDKOption{
			openrouter.WithSecurity(opts.APIKey),
			openrouter.WithClient(openRouterHTTPClient),
			openrouter.WithRetryConfig(retry.Config{Strategy: "none"}),
		}
		baseURL := model.BaseURL
		if baseURL == "" {
			baseURL = openRouterDefaultBaseURL
		}
		clientOpts = append(clientOpts, openrouter.WithServerURL(baseURL))

		headers := map[string]string{
			http.CanonicalHeaderKey("HTTP-Referer"): openRouterHTTPReferer,
			http.CanonicalHeaderKey("X-Title"):      openRouterTitle,
		}
		for k, v := range opts.Headers {
			headers[http.CanonicalHeaderKey(k)] = v
		}

		client := openrouter.New(clientOpts...)
		resp, err := client.Chat.Send(
			ctx,
			req,
			nil,
			operations.WithAcceptHeaderOverride(operations.AcceptHeaderEnumTextEventStream),
			operations.WithSetHeaders(headers),
		)
		if err != nil {
			fail(sanitizeOpenRouterError(err))
			return
		}
		if resp == nil || resp.EventStream == nil {
			fail(fmt.Errorf("openrouter-chat: expected streaming response"))
			return
		}
		defer func() { _ = resp.EventStream.Close() }()

		if err := consumeOpenRouterStream(ctx, s, &out, model, resp.EventStream); err != nil {
			fail(err)
		}
	}()
	return s
}

func sanitizeOpenRouterError(err error) error {
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	return sdkerrors.NewAPIError(apiErr.Message, apiErr.StatusCode, apiErr.Body, nil)
}

func parseReasoningDetail(data []byte) (ReasoningDetail, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return ReasoningDetail{}, fmt.Errorf("openrouter-chat: reasoning detail must be a JSON object: %w", err)
	}
	if fields == nil {
		return ReasoningDetail{}, fmt.Errorf("openrouter-chat: reasoning detail must be a JSON object")
	}
	rawType, ok := fields["type"]
	if !ok {
		return ReasoningDetail{}, fmt.Errorf("openrouter-chat: reasoning detail type is required")
	}
	var detailType string
	if err := json.Unmarshal(rawType, &detailType); err != nil || detailType == "" {
		return ReasoningDetail{}, fmt.Errorf("openrouter-chat: reasoning detail type must be a non-empty string")
	}
	requiredField := ""
	switch detailType {
	case "reasoning.encrypted":
		requiredField = "data"
	case "reasoning.summary":
		requiredField = "summary"
	case "reasoning.text":
	default:
		return ReasoningDetail{raw: append(json.RawMessage(nil), data...)}, nil
	}
	if requiredField != "" {
		raw, ok := fields[requiredField]
		var value *string
		if !ok || json.Unmarshal(raw, &value) != nil || value == nil {
			return ReasoningDetail{}, fmt.Errorf("openrouter-chat: invalid %s detail", detailType)
		}
	}
	var detail components.ReasoningDetailUnion
	if err := json.Unmarshal(data, &detail); err != nil {
		return ReasoningDetail{}, fmt.Errorf("openrouter-chat: invalid %s detail", detailType)
	}
	return ReasoningDetail{raw: append(json.RawMessage(nil), data...)}, nil
}

func buildOpenRouterRequest(model Model, c Context, opts *StreamOptions) (components.ChatRequest, error) {
	messages, err := convertOpenRouterMessages(model, c)
	if err != nil {
		return components.ChatRequest{}, err
	}
	tools, err := convertOpenRouterTools(c.Tools)
	if err != nil {
		return components.ChatRequest{}, err
	}

	streamOptions := components.ChatStreamOptions{IncludeUsage: openrouter.Pointer(true)}
	req := components.ChatRequest{
		Messages:      messages,
		Model:         openrouter.Pointer(model.ID),
		Stream:        openrouter.Pointer(true),
		StreamOptions: optionalnullable.From(&streamOptions),
		Tools:         tools,
	}
	if opts.Temperature != nil {
		req.Temperature = optionalnullable.From(opts.Temperature)
	}
	if opts.MaxTokens > 0 {
		req.MaxCompletionTokens = optionalnullable.From(openrouter.Pointer(int64(opts.MaxTokens)))
	}
	if opts.SessionID != "" {
		req.SessionID = openrouter.Pointer(opts.SessionID)
	}
	if openRouterSupportsCacheControl(model) {
		switch opts.CacheRetention {
		case CacheRetentionShort:
			ttl := components.AnthropicCacheControlTTLFivem
			req.CacheControl = &components.AnthropicCacheControlDirective{
				TTL:  &ttl,
				Type: components.AnthropicCacheControlDirectiveTypeEphemeral,
			}
		case CacheRetentionLong:
			ttl := components.AnthropicCacheControlTTLOneh
			req.CacheControl = &components.AnthropicCacheControlDirective{
				TTL:  &ttl,
				Type: components.AnthropicCacheControlDirectiveTypeEphemeral,
			}
		}
	}
	return req, nil
}

func openRouterSupportsCacheControl(model Model) bool {
	return strings.HasPrefix(strings.TrimPrefix(model.ID, "~"), "anthropic/")
}

func convertOpenRouterMessages(model Model, c Context) ([]components.ChatMessages, error) {
	messages := make([]components.ChatMessages, 0, len(c.Messages)+1)
	if c.SystemPrompt != "" {
		if openRouterUsesDeveloperRole(model) {
			messages = append(messages, components.CreateChatMessagesDeveloper(components.ChatDeveloperMessage{
				Content: components.CreateChatDeveloperMessageContentStr(c.SystemPrompt),
			}))
		} else {
			messages = append(messages, components.CreateChatMessagesSystem(components.ChatSystemMessage{
				Content: components.CreateChatSystemMessageContentStr(c.SystemPrompt),
			}))
		}
	}

	for _, m := range c.Messages {
		switch msg := m.(type) {
		case UserMessage:
			messages = append(messages, components.CreateChatMessagesUser(components.ChatUserMessage{
				Content: openRouterUserContent(msg.Content),
			}))
		case AssistantMessage:
			assistant, ok, err := openRouterAssistantMessage(msg)
			if err != nil {
				return nil, err
			}
			if ok {
				messages = append(messages, components.CreateChatMessagesAssistant(assistant))
			}
		case ToolResultMessage:
			messages = append(messages, components.CreateChatMessagesTool(components.ChatToolMessage{
				Content:    openRouterToolContent(msg.Content),
				ToolCallID: msg.ToolCallID,
			}))
		}
	}
	return messages, nil
}

func openRouterUsesDeveloperRole(model Model) bool {
	modelID := strings.TrimPrefix(model.ID, "~")
	return model.Reasoning && (strings.HasPrefix(modelID, "openai/") || strings.HasPrefix(modelID, "anthropic/"))
}

func openRouterUserContent(content []UserContent) components.ChatUserMessageContent {
	if text, ok := openRouterTextOnly(content); ok {
		return components.CreateChatUserMessageContentStr(text)
	}
	return components.CreateChatUserMessageContentArrayOfChatContentItems(openRouterContentItems(content))
}

func openRouterToolContent(content []UserContent) components.ChatToolMessageContent {
	if len(content) == 0 {
		return components.CreateChatToolMessageContentStr("(no text result)")
	}
	if text, ok := openRouterTextOnly(content); ok {
		if text == "" {
			text = "(no text result)"
		}
		return components.CreateChatToolMessageContentStr(text)
	}
	return components.CreateChatToolMessageContentArrayOfChatContentItems(openRouterContentItems(content))
}

func openRouterTextOnly(content []UserContent) (string, bool) {
	texts := make([]string, 0, len(content))
	for _, item := range content {
		text, ok := item.(TextContent)
		if !ok {
			return "", false
		}
		texts = append(texts, text.Text)
	}
	return strings.Join(texts, "\n"), true
}

func openRouterContentItems(content []UserContent) []components.ChatContentItems {
	items := make([]components.ChatContentItems, 0, len(content))
	for _, item := range content {
		switch block := item.(type) {
		case TextContent:
			items = append(items, components.CreateChatContentItemsText(components.ChatContentText{Text: block.Text}))
		case ImageContent:
			items = append(items, components.CreateChatContentItemsImageURL(components.ChatContentImage{
				ImageURL: components.ChatContentImageImageURL{
					URL: "data:" + block.MimeType + ";base64," + block.Data,
				},
			}))
		}
	}
	return items
}

func openRouterAssistantMessage(msg AssistantMessage) (components.ChatAssistantMessage, bool, error) {
	texts := make([]string, 0, len(msg.Content))
	thoughts := make([]string, 0, len(msg.Content))
	refusals := make([]string, 0, len(msg.Content))
	toolCalls := make([]components.ChatToolCall, 0, len(msg.Content))
	reasoningDetails := make([]components.ReasoningDetailUnion, 0, len(msg.ReasoningDetails))
	for i, raw := range msg.ReasoningDetails {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return components.ChatAssistantMessage{}, false, fmt.Errorf("openrouter-chat: reasoning detail %d: %w", i, err)
		}
		var detail components.ReasoningDetailUnion
		if err := json.Unmarshal(encoded, &detail); err != nil {
			return components.ChatAssistantMessage{}, false, fmt.Errorf("openrouter-chat: reasoning detail %d: %w", i, err)
		}
		reasoningDetails = append(reasoningDetails, detail)
	}
	for _, item := range msg.Content {
		switch block := item.(type) {
		case TextContent:
			if strings.TrimSpace(block.Text) != "" {
				texts = append(texts, block.Text)
			}
		case ThinkingContent:
			if strings.TrimSpace(block.Thinking) != "" {
				thoughts = append(thoughts, block.Thinking)
			}
		case RefusalContent:
			if strings.TrimSpace(block.Refusal) != "" {
				refusals = append(refusals, block.Refusal)
			}
		case ToolCall:
			args, err := marshalToolArguments(block.Arguments)
			if err != nil {
				return components.ChatAssistantMessage{}, false, fmt.Errorf("openrouter-chat: tool call %q arguments: %w", block.Name, err)
			}
			toolCalls = append(toolCalls, components.ChatToolCall{
				ID:   block.ID,
				Type: components.ChatToolCallTypeFunction,
				Function: components.ChatToolCallFunction{
					Name:      block.Name,
					Arguments: args,
				},
			})
		}
	}

	assistant := components.ChatAssistantMessage{ReasoningDetails: reasoningDetails, ToolCalls: toolCalls}
	if text := strings.Join(texts, ""); text != "" {
		content := components.CreateChatAssistantMessageContentStr(text)
		assistant.Content = optionalnullable.From(&content)
	}
	if reasoning := strings.Join(thoughts, ""); reasoning != "" {
		assistant.Reasoning = optionalnullable.From(&reasoning)
	}
	if refusal := strings.Join(refusals, ""); refusal != "" {
		assistant.Refusal = optionalnullable.From(&refusal)
	}
	return assistant, assistant.Content.IsSet() || assistant.Reasoning.IsSet() || assistant.Refusal.IsSet() || len(assistant.ReasoningDetails) > 0 || len(assistant.ToolCalls) > 0, nil
}

func convertOpenRouterTools(tools []Tool) ([]components.ChatFunctionTool, error) {
	converted := make([]components.ChatFunctionTool, 0, len(tools))
	for _, tool := range tools {
		parameters := map[string]any{}
		if len(tool.Parameters) > 0 {
			var decoded any
			if err := json.Unmarshal(tool.Parameters, &decoded); err != nil {
				return nil, fmt.Errorf("openrouter-chat: tool %q parameters: %w", tool.Name, err)
			}
			var ok bool
			parameters, ok = decoded.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("openrouter-chat: tool %q parameters must be a JSON object", tool.Name)
			}
		}

		fn := components.ChatFunctionToolFunction{
			Type: components.ChatFunctionToolTypeFunction,
			Function: components.ChatFunctionToolFunctionFunction{
				Description: openrouter.Pointer(tool.Description),
				Name:        tool.Name,
				Parameters:  parameters,
			},
		}
		converted = append(converted, components.CreateChatFunctionToolChatFunctionToolFunction(fn))
	}
	return converted, nil
}

type openRouterStreamingToolCall struct {
	contentIndex int
	partialArgs  strings.Builder
}

func consumeOpenRouterStream(
	ctx context.Context,
	s *EventStream,
	out *AssistantMessage,
	model Model,
	stream *sdkstream.EventStream[components.ChatStreamingResponse],
) error {
	s.Push(AssistantMessageEvent{Type: EventStart, Partial: snapshot(out)})

	textIndex := -1
	thinkingIndex := -1
	refusalIndex := -1
	toolByStreamIndex := map[int64]*openRouterStreamingToolCall{}
	toolByContentIndex := map[int]*openRouterStreamingToolCall{}
	hasFinish := false

	for stream.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		event := stream.Value()
		if event == nil {
			continue
		}
		chunk := event.Data
		if out.ResponseID == "" && chunk.ID != "" {
			out.ResponseID = chunk.ID
		}
		if out.ResponseModel == "" && chunk.Model != "" && chunk.Model != model.ID {
			out.ResponseModel = chunk.Model
		}
		if chunk.Usage != nil {
			out.Usage = parseOpenRouterUsage(*chunk.Usage, model)
		}
		if chunk.Error != nil {
			return fmt.Errorf("openrouter-chat: stream error %d: %s", chunk.Error.Code, chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		if choice.FinishReason != nil {
			reason, errMsg := mapOpenRouterStopReason(*choice.FinishReason)
			out.StopReason = reason
			out.ErrorMessage = errMsg
			hasFinish = true
		}

		if text, ok := choice.Delta.Content.Get(); ok && text != nil && *text != "" {
			if textIndex == -1 {
				out.Content = append(out.Content, TextContent{})
				textIndex = len(out.Content) - 1
				s.Push(AssistantMessageEvent{Type: EventTextStart, ContentIndex: textIndex, Partial: snapshot(out)})
			}
			block := out.Content[textIndex].(TextContent)
			block.Text += *text
			out.Content[textIndex] = block
			s.Push(AssistantMessageEvent{Type: EventTextDelta, ContentIndex: textIndex, Delta: *text, Partial: snapshot(out)})
		}

		if reasoning, ok := choice.Delta.Reasoning.Get(); ok && reasoning != nil && *reasoning != "" {
			if thinkingIndex == -1 {
				out.Content = append(out.Content, ThinkingContent{})
				thinkingIndex = len(out.Content) - 1
				s.Push(AssistantMessageEvent{Type: EventThinkingStart, ContentIndex: thinkingIndex, Partial: snapshot(out)})
			}
			block := out.Content[thinkingIndex].(ThinkingContent)
			block.Thinking += *reasoning
			out.Content[thinkingIndex] = block
			s.Push(AssistantMessageEvent{Type: EventThinkingDelta, ContentIndex: thinkingIndex, Delta: *reasoning, Partial: snapshot(out)})
		}

		if refusal, ok := choice.Delta.Refusal.Get(); ok && refusal != nil && *refusal != "" {
			if refusalIndex == -1 {
				out.Content = append(out.Content, RefusalContent{})
				refusalIndex = len(out.Content) - 1
				s.Push(AssistantMessageEvent{Type: EventRefusalStart, ContentIndex: refusalIndex, Partial: snapshot(out)})
			}
			block := out.Content[refusalIndex].(RefusalContent)
			block.Refusal += *refusal
			out.Content[refusalIndex] = block
			s.Push(AssistantMessageEvent{Type: EventRefusalDelta, ContentIndex: refusalIndex, Delta: *refusal, Partial: snapshot(out)})
		}

		for _, tc := range choice.Delta.ToolCalls {
			tool := toolByStreamIndex[tc.Index]
			if tool == nil {
				id := ""
				name := ""
				if tc.ID != nil {
					id = *tc.ID
				}
				if tc.Function != nil && tc.Function.Name != nil {
					name = *tc.Function.Name
				}
				out.Content = append(out.Content, ToolCall{ID: id, Name: name, Arguments: map[string]any{}})
				tool = &openRouterStreamingToolCall{contentIndex: len(out.Content) - 1}
				toolByStreamIndex[tc.Index] = tool
				toolByContentIndex[tool.contentIndex] = tool
				s.Push(AssistantMessageEvent{Type: EventToolCallStart, ContentIndex: tool.contentIndex, Partial: snapshot(out)})
			}

			block := out.Content[tool.contentIndex].(ToolCall)
			if block.ID == "" && tc.ID != nil {
				block.ID = *tc.ID
			}
			if block.Name == "" && tc.Function != nil && tc.Function.Name != nil {
				block.Name = *tc.Function.Name
			}
			args := ""
			if tc.Function != nil && tc.Function.Arguments != nil {
				args = *tc.Function.Arguments
				tool.partialArgs.WriteString(args)
			}
			out.Content[tool.contentIndex] = block
			s.Push(AssistantMessageEvent{Type: EventToolCallDelta, ContentIndex: tool.contentIndex, Delta: args, Partial: snapshot(out)})
		}

		for _, detail := range choice.Delta.ReasoningDetails {
			encoded, err := json.Marshal(detail)
			if err != nil {
				return fmt.Errorf("openrouter-chat: encode reasoning detail: %w", err)
			}
			parsed, err := parseReasoningDetail(encoded)
			if err != nil {
				return fmt.Errorf("openrouter-chat: decode reasoning detail: %w", err)
			}
			out.ReasoningDetails = append(out.ReasoningDetails, parsed)
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	for i, content := range out.Content {
		switch block := content.(type) {
		case TextContent:
			s.Push(AssistantMessageEvent{Type: EventTextEnd, ContentIndex: i, Content: block.Text, Partial: snapshot(out)})
		case ThinkingContent:
			s.Push(AssistantMessageEvent{Type: EventThinkingEnd, ContentIndex: i, Content: block.Thinking, Partial: snapshot(out)})
		case RefusalContent:
			s.Push(AssistantMessageEvent{Type: EventRefusalEnd, ContentIndex: i, Content: block.Refusal, Partial: snapshot(out)})
		case ToolCall:
			tool := toolByContentIndex[i]
			if tool != nil {
				args, err := parseToolArguments(tool.partialArgs.String())
				if err != nil {
					return fmt.Errorf("openrouter-chat: tool call %q arguments: %w", block.Name, err)
				}
				block.Arguments = args
				out.Content[i] = block
			}
			s.Push(AssistantMessageEvent{Type: EventToolCallEnd, ContentIndex: i, ToolCall: &block, Partial: snapshot(out)})
		}
	}

	if out.StopReason == StopReasonError {
		return fmt.Errorf("%s", out.ErrorMessage)
	}
	if !hasFinish {
		return fmt.Errorf("openrouter-chat: stream ended without finish_reason")
	}
	out.Timestamp = time.Now()
	s.Push(AssistantMessageEvent{Type: EventDone, Reason: out.StopReason, Message: out})
	return nil
}

func parseOpenRouterUsage(raw components.ChatUsage, model Model) Usage {
	cacheRead := 0
	cacheWrite := 0
	if details, ok := raw.PromptTokensDetails.Get(); ok && details != nil {
		if details.CachedTokens != nil {
			cacheRead = int(*details.CachedTokens)
		}
		if details.CacheWriteTokens != nil {
			cacheWrite = int(*details.CacheWriteTokens)
		}
	}
	input := max(0, int(raw.PromptTokens)-cacheRead-cacheWrite)
	usage := Usage{
		Input:       input,
		Output:      int(raw.CompletionTokens),
		CacheRead:   cacheRead,
		CacheWrite:  cacheWrite,
		TotalTokens: input + int(raw.CompletionTokens) + cacheRead + cacheWrite,
	}
	perToken := func(rate float64, tokens int) float64 { return rate * float64(tokens) / 1e6 }
	usage.Cost = Cost{
		Input:      perToken(model.Cost.Input, usage.Input),
		Output:     perToken(model.Cost.Output, usage.Output),
		CacheRead:  perToken(model.Cost.CacheRead, usage.CacheRead),
		CacheWrite: perToken(model.Cost.CacheWrite, usage.CacheWrite),
	}
	usage.Cost.Total = usage.Cost.Input + usage.Cost.Output + usage.Cost.CacheRead + usage.Cost.CacheWrite
	if cost, ok := raw.Cost.Get(); ok && cost != nil {
		if usage.Cost.Total > 0 {
			scale := *cost / usage.Cost.Total
			usage.Cost.Input *= scale
			usage.Cost.Output *= scale
			usage.Cost.CacheRead *= scale
			usage.Cost.CacheWrite *= scale
		}
		usage.Cost.Total = *cost
	}
	return usage
}

func mapOpenRouterStopReason(reason components.ChatFinishReasonEnum) (StopReason, string) {
	switch reason {
	case components.ChatFinishReasonEnumStop:
		return StopReasonStop, ""
	case components.ChatFinishReasonEnumLength:
		return StopReasonLength, ""
	case components.ChatFinishReasonEnumToolCalls:
		return StopReasonToolUse, ""
	default:
		return StopReasonError, "provider finish_reason: " + string(reason)
	}
}

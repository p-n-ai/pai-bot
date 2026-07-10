package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/models/sdkerrors"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
)

const APIOpenRouterChat = "openrouter-chat"

const (
	openRouterDefaultBaseURL = "https://openrouter.ai/api/v1"
	openRouterHTTPReferer    = "https://pandai.org"
	openRouterTitle          = "P&AI Bot"
	openRouterMaxErrorBody   = 64 << 10
)

var openRouterHTTPClient = &http.Client{}

var openRouterSSEBoundary = regexp.MustCompile(`\r\n\r\n|\r\n\r|\r\n\n|\r\r\n|\n\r\n|\r\r|\n\r|\n\n`)
var openRouterStructuredOutputName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

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
		req, err := buildOpenRouterRequest(model, c, opts)
		if err != nil {
			fail(err)
			return
		}

		baseURL := model.BaseURL
		if baseURL == "" {
			baseURL = openRouterDefaultBaseURL
		}
		headers, err := openRouterHeaders(opts.Headers)
		if err != nil {
			fail(err)
			return
		}
		body, err := json.Marshal(req)
		if err != nil {
			fail(fmt.Errorf("openrouter-chat: encode request: %w", err))
			return
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterEndpoint(baseURL), bytes.NewReader(body))
		if err != nil {
			fail(err)
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+opts.APIKey)
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Content-Type", "application/json")
		for name, value := range headers {
			httpReq.Header.Set(name, value)
		}

		resp, err := openRouterHTTPClient.Do(httpReq)
		if err != nil {
			fail(err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		mediaType, _, mediaErr := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if resp.StatusCode != http.StatusOK || mediaErr != nil || !strings.EqualFold(mediaType, "text/event-stream") {
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, openRouterMaxErrorBody))
			message := "openrouter-chat request failed"
			if resp.StatusCode == http.StatusOK {
				message = fmt.Sprintf("unknown content-type received: %s", resp.Header.Get("Content-Type"))
			}
			fail(sdkerrors.NewAPIError(message, resp.StatusCode, strings.TrimSpace(string(raw)), nil))
			return
		}

		if err := consumeOpenRouterStream(ctx, s, &out, model, resp.Body); err != nil {
			fail(err)
		}
	}()
	return s
}

func openRouterHeaders(custom map[string]string) (map[string]string, error) {
	headers := map[string]string{
		http.CanonicalHeaderKey("HTTP-Referer"): openRouterHTTPReferer,
		http.CanonicalHeaderKey("X-Title"):      openRouterTitle,
	}
	for name, value := range custom {
		canonical := http.CanonicalHeaderKey(name)
		switch canonical {
		case "Accept", "Authorization", "Content-Length", "Content-Type", "Host", "Transfer-Encoding":
			return nil, fmt.Errorf("openrouter-chat: header %q is managed by the transport", canonical)
		}
		headers[canonical] = value
	}
	return headers, nil
}

func openRouterEndpoint(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/chat/completions"
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
	responseFormat, err := openRouterJSONSchemaResponseFormat(opts.StructuredOutput)
	if err != nil {
		return components.ChatRequest{}, err
	}

	streamOptions := components.ChatStreamOptions{IncludeUsage: openrouter.Pointer(true)}
	req := components.ChatRequest{
		Messages:       messages,
		Model:          openrouter.Pointer(model.ID),
		ResponseFormat: responseFormat,
		Stream:         openrouter.Pointer(true),
		StreamOptions:  optionalnullable.From(&streamOptions),
		Tools:          tools,
	}
	if opts.Temperature != nil {
		req.Temperature = optionalnullable.From(opts.Temperature)
	}
	if opts.MaxTokens > 0 {
		req.MaxCompletionTokens = optionalnullable.From(openrouter.Pointer(int64(opts.MaxTokens)))
	}
	if opts.ReasoningEffort != "" {
		effort := components.ChatRequestReasoningEffort(opts.ReasoningEffort)
		req.ReasoningEffort = optionalnullable.From(&effort)
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

func openRouterJSONSchemaResponseFormat(spec *StructuredOutputSpec) (*components.ResponseFormat, error) {
	if spec == nil {
		return nil, nil
	}
	if spec.Name == "" {
		return nil, fmt.Errorf("openrouter-chat: structured output name is required")
	}
	if !openRouterStructuredOutputName.MatchString(spec.Name) {
		return nil, fmt.Errorf("openrouter-chat: structured output name must match [A-Za-z0-9_-]{1,64}")
	}

	rawSchema := bytes.TrimSpace(spec.JSONSchema)
	if len(rawSchema) == 0 {
		return nil, fmt.Errorf("openrouter-chat: structured output JSON schema is required")
	}

	decoder := json.NewDecoder(bytes.NewReader(rawSchema))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil || len(bytes.TrimSpace(rawSchema[decoder.InputOffset():])) != 0 {
		return nil, fmt.Errorf("openrouter-chat: structured output JSON schema must contain valid JSON")
	}
	schema, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("openrouter-chat: structured output JSON schema must be a JSON object")
	}

	jsonSchema := components.ChatJSONSchemaConfig{
		Name:   spec.Name,
		Schema: schema,
	}
	if spec.Strict {
		jsonSchema.Strict = optionalnullable.From(openrouter.Pointer(true))
	}
	responseFormat := components.CreateResponseFormatJSONSchema(components.ChatFormatJSONSchemaConfig{
		JSONSchema: jsonSchema,
	})
	return &responseFormat, nil
}

func openRouterSupportsCacheControl(model Model) bool {
	return strings.HasPrefix(strings.TrimPrefix(model.ID, "~"), "anthropic/")
}

func convertOpenRouterMessages(model Model, c Context) ([]components.ChatMessages, error) {
	messages := make([]components.ChatMessages, 0, len(c.Messages)+1)
	if c.SystemPrompt != "" {
		messages = append(messages, openRouterSystemMessage(model, c.SystemPrompt))
	}

	for _, m := range c.Messages {
		switch msg := m.(type) {
		case SystemMessage:
			messages = append(messages, openRouterSystemMessage(model, msg.Content))
		case UserMessage:
			content, err := openRouterUserContent(msg.Content)
			if err != nil {
				return nil, err
			}
			messages = append(messages, components.CreateChatMessagesUser(components.ChatUserMessage{
				Content: content,
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
			content, err := openRouterToolContent(msg.Content)
			if err != nil {
				return nil, err
			}
			messages = append(messages, components.CreateChatMessagesTool(components.ChatToolMessage{
				Content:    content,
				ToolCallID: msg.ToolCallID,
			}))
		}
	}
	return messages, nil
}

func openRouterSystemMessage(model Model, content string) components.ChatMessages {
	if openRouterUsesDeveloperRole(model) {
		return components.CreateChatMessagesDeveloper(components.ChatDeveloperMessage{
			Content: components.CreateChatDeveloperMessageContentStr(content),
		})
	}
	return components.CreateChatMessagesSystem(components.ChatSystemMessage{
		Content: components.CreateChatSystemMessageContentStr(content),
	})
}

func openRouterUsesDeveloperRole(model Model) bool {
	modelID := strings.TrimPrefix(model.ID, "~")
	return model.Reasoning && (strings.HasPrefix(modelID, "openai/") || strings.HasPrefix(modelID, "anthropic/"))
}

func openRouterUserContent(content []UserContent) (components.ChatUserMessageContent, error) {
	if text, ok := openRouterTextOnly(content); ok {
		return components.CreateChatUserMessageContentStr(text), nil
	}
	items, err := openRouterContentItems(content)
	if err != nil {
		return components.ChatUserMessageContent{}, err
	}
	return components.CreateChatUserMessageContentArrayOfChatContentItems(items), nil
}

func openRouterToolContent(content []UserContent) (components.ChatToolMessageContent, error) {
	if len(content) == 0 {
		return components.CreateChatToolMessageContentStr("(no text result)"), nil
	}
	if text, ok := openRouterTextOnly(content); ok {
		if text == "" {
			text = "(no text result)"
		}
		return components.CreateChatToolMessageContentStr(text), nil
	}
	items, err := openRouterContentItems(content)
	if err != nil {
		return components.ChatToolMessageContent{}, err
	}
	return components.CreateChatToolMessageContentArrayOfChatContentItems(items), nil
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

func openRouterContentItems(content []UserContent) ([]components.ChatContentItems, error) {
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
		case ImageURLContent:
			if err := validateOpenRouterImageURL(block.URL); err != nil {
				return nil, err
			}
			items = append(items, components.CreateChatContentItemsImageURL(components.ChatContentImage{
				ImageURL: components.ChatContentImageImageURL{URL: block.URL},
			}))
		}
	}
	return items, nil
}

func validateOpenRouterImageURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || raw == "" || !parsed.IsAbs() || parsed.Hostname() == "" || parsed.User != nil ||
		(!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return fmt.Errorf("openrouter-chat: image URL must be an absolute HTTP(S) URL without credentials")
	}
	return nil
}

func openRouterAssistantMessage(msg AssistantMessage) (components.ChatAssistantMessage, bool, error) {
	texts := make([]string, 0, len(msg.Content))
	thoughts := make([]string, 0, len(msg.Content))
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
	return assistant, assistant.Content.IsSet() || assistant.Reasoning.IsSet() || len(assistant.ReasoningDetails) > 0 || len(assistant.ToolCalls) > 0, nil
}

func convertOpenRouterTools(tools []Tool) ([]components.ChatFunctionTool, error) {
	if len(tools) == 0 {
		return nil, nil
	}
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

type openRouterStreamChunk struct {
	ID      string                           `json:"id"`
	Model   string                           `json:"model"`
	Choices []openRouterStreamChoice         `json:"choices"`
	Usage   *components.ChatUsage            `json:"usage"`
	Error   *components.ChatStreamChunkError `json:"error"`
}

type openRouterStreamChoice struct {
	Delta        openRouterStreamDelta            `json:"delta"`
	FinishReason *components.ChatFinishReasonEnum `json:"finish_reason"`
}

type openRouterStreamDelta struct {
	Content          string                     `json:"content"`
	Reasoning        string                     `json:"reasoning"`
	ReasoningDetails []json.RawMessage          `json:"reasoning_details"`
	Refusal          string                     `json:"refusal"`
	ToolCalls        []openRouterStreamToolCall `json:"tool_calls"`
}

type openRouterStreamToolCall struct {
	Index    *int64                       `json:"index"`
	ID       string                       `json:"id"`
	Function openRouterStreamToolFunction `json:"function"`
}

type openRouterStreamToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func splitOpenRouterSSEEvents(data []byte, atEOF bool) (int, []byte, error) {
	if len(data) == 0 && atEOF {
		return 0, nil, nil
	}
	if boundary := openRouterSSEBoundary.FindIndex(data); boundary != nil {
		return boundary[1], data[:boundary[0]], nil
	}
	if atEOF {
		return len(data), bytes.TrimRight(data, "\r\n"), nil
	}
	return 0, nil, nil
}

func openRouterSSEData(event []byte) (string, bool) {
	text := strings.ReplaceAll(string(event), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimPrefix(text, "\uFEFF")
	var data strings.Builder
	found := false
	for _, line := range strings.Split(text, "\n") {
		value, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		if found {
			data.WriteByte('\n')
		}
		data.WriteString(strings.TrimPrefix(value, " "))
		found = true
	}
	return data.String(), found
}

func consumeOpenRouterStream(
	ctx context.Context,
	s *EventStream,
	out *AssistantMessage,
	model Model,
	body io.Reader,
) error {
	s.Push(AssistantMessageEvent{Type: EventStart, Partial: snapshot(out)})

	textIndex := -1
	thinkingIndex := -1
	toolByStreamIndex := map[int64]*openRouterStreamingToolCall{}
	toolByID := map[string]*openRouterStreamingToolCall{}
	toolByContentIndex := map[int]*openRouterStreamingToolCall{}
	var currentTool *openRouterStreamingToolCall
	hasFinish := false

	scanner := bufio.NewScanner(body)
	scanner.Split(splitOpenRouterSSEEvents)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		data, ok := openRouterSSEData(scanner.Bytes())
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}
		var chunk openRouterStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("openrouter-chat: invalid event stream")
		}
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

		if reasoning := choice.Delta.Reasoning; reasoning != "" {
			if thinkingIndex == -1 {
				out.Content = append(out.Content, ThinkingContent{})
				thinkingIndex = len(out.Content) - 1
				s.Push(AssistantMessageEvent{Type: EventThinkingStart, ContentIndex: thinkingIndex, Partial: snapshot(out)})
			}
			block := out.Content[thinkingIndex].(ThinkingContent)
			block.Thinking += reasoning
			out.Content[thinkingIndex] = block
			s.Push(AssistantMessageEvent{Type: EventThinkingDelta, ContentIndex: thinkingIndex, Delta: reasoning, Partial: snapshot(out)})
		}

		for _, tc := range choice.Delta.ToolCalls {
			tool := currentTool
			streamIndex := tc.Index
			if streamIndex != nil {
				tool = toolByStreamIndex[*streamIndex]
			} else if tc.ID != "" {
				tool = toolByID[tc.ID]
			}
			if tool == nil {
				out.Content = append(out.Content, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: map[string]any{}})
				tool = &openRouterStreamingToolCall{contentIndex: len(out.Content) - 1}
				if streamIndex != nil {
					toolByStreamIndex[*streamIndex] = tool
				}
				if tc.ID != "" {
					toolByID[tc.ID] = tool
				}
				toolByContentIndex[tool.contentIndex] = tool
				s.Push(AssistantMessageEvent{Type: EventToolCallStart, ContentIndex: tool.contentIndex, Partial: snapshot(out)})
			}
			currentTool = tool

			block := out.Content[tool.contentIndex].(ToolCall)
			if block.ID == "" {
				block.ID = tc.ID
			}
			if tc.ID != "" {
				toolByID[tc.ID] = tool
			}
			if block.Name == "" {
				block.Name = tc.Function.Name
			}
			args := tc.Function.Arguments
			if args != "" {
				tool.partialArgs.WriteString(args)
			}
			out.Content[tool.contentIndex] = block
			s.Push(AssistantMessageEvent{Type: EventToolCallDelta, ContentIndex: tool.contentIndex, Delta: args, Partial: snapshot(out)})
		}

		for _, detail := range choice.Delta.ReasoningDetails {
			parsed, err := parseReasoningDetail(detail)
			if err != nil {
				return fmt.Errorf("openrouter-chat: decode reasoning detail: %w", err)
			}
			out.ReasoningDetails = append(out.ReasoningDetails, parsed)
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("openrouter-chat: reading stream: %w", err)
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

package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonError   StopReason = "error"
	StopReasonAborted StopReason = "aborted"
)

type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

type Usage struct {
	Input       int  `json:"input"`
	Output      int  `json:"output"`
	CacheRead   int  `json:"cacheRead"`
	CacheWrite  int  `json:"cacheWrite"`
	TotalTokens int  `json:"totalTokens"`
	Cost        Cost `json:"cost"`
}

type TextContent struct {
	Text string `json:"text"`
}

type ImageContent struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

type ThinkingContent struct {
	Thinking string `json:"thinking"`

	Signature string `json:"thinkingSignature,omitempty"`
	Redacted  bool   `json:"redacted,omitempty"`
}

type RefusalContent struct {
	Refusal string `json:"refusal"`
}

type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func marshalToolArguments(arguments map[string]any) (string, error) {
	if arguments == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(arguments)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func parseToolArguments(encoded string) (map[string]any, error) {
	if strings.TrimSpace(encoded) == "" {
		return map[string]any{}, nil
	}
	var arguments map[string]any
	if err := json.Unmarshal([]byte(encoded), &arguments); err != nil {
		return nil, err
	}
	if arguments == nil {
		return nil, fmt.Errorf("arguments must be a JSON object")
	}
	return arguments, nil
}

type ReasoningDetail struct {
	raw json.RawMessage
}

func parseReasoningDetail(data []byte) (ReasoningDetail, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ReasoningDetail{}, fmt.Errorf("reasoning detail must be a JSON object: %w", err)
	}
	if envelope.Type == "" {
		return ReasoningDetail{}, fmt.Errorf("reasoning detail must have a type")
	}
	return ReasoningDetail{raw: append(json.RawMessage(nil), data...)}, nil
}

func (d ReasoningDetail) MarshalJSON() ([]byte, error) {
	if len(d.raw) == 0 {
		return nil, fmt.Errorf("reasoning detail is empty")
	}
	return append([]byte(nil), d.raw...), nil
}

func (d *ReasoningDetail) UnmarshalJSON(data []byte) error {
	parsed, err := parseReasoningDetail(data)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

type UserContent interface{ isUserContent() }

type AssistantContent interface{ isAssistantContent() }

func (TextContent) isUserContent()          {}
func (ImageContent) isUserContent()         {}
func (TextContent) isAssistantContent()     {}
func (ThinkingContent) isAssistantContent() {}
func (RefusalContent) isAssistantContent()  {}
func (ToolCall) isAssistantContent()        {}

type Message interface{ isMessage() }

type UserMessage struct {
	Content   []UserContent
	Timestamp time.Time
}

type AssistantMessage struct {
	Content          []AssistantContent
	ReasoningDetails []ReasoningDetail
	API              string
	Provider         string
	Model            string
	ResponseModel    string
	ResponseID       string
	Usage            Usage
	StopReason       StopReason
	ErrorMessage     string
	Timestamp        time.Time
}

type ToolResultMessage struct {
	ToolCallID string
	ToolName   string
	Content    []UserContent
	IsError    bool
	Timestamp  time.Time
}

func (UserMessage) isMessage()       {}
func (AssistantMessage) isMessage()  {}
func (ToolResultMessage) isMessage() {}

func UserText(text string) UserMessage {
	return UserMessage{Content: []UserContent{TextContent{Text: text}}, Timestamp: time.Now()}
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type Context struct {
	SystemPrompt string
	Messages     []Message
	Tools        []Tool
}

type CacheRetention string

const (
	CacheRetentionNone  CacheRetention = "none"
	CacheRetentionShort CacheRetention = "short"
	CacheRetentionLong  CacheRetention = "long"
)

type StreamOptions struct {
	Temperature    *float64
	MaxTokens      int
	APIKey         string
	SessionID      string
	CacheRetention CacheRetention
	Headers        map[string]string
}

type Model struct {
	ID            string
	Name          string
	API           string
	Provider      string
	BaseURL       string
	Reasoning     bool
	Cost          Cost
	ContextWindow int
	MaxTokens     int
}

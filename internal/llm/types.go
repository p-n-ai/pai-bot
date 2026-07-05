package llm

import (
	"encoding/json"
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

type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type UserContent interface{ isUserContent() }

type AssistantContent interface{ isAssistantContent() }

func (TextContent) isUserContent()          {}
func (ImageContent) isUserContent()         {}
func (TextContent) isAssistantContent()     {}
func (ThinkingContent) isAssistantContent() {}
func (ToolCall) isAssistantContent()        {}

type Message interface{ isMessage() }

type UserMessage struct {
	Content   []UserContent
	Timestamp time.Time
}

type AssistantMessage struct {
	Content      []AssistantContent
	API          string
	Provider     string
	Model        string
	ResponseID   string
	Usage        Usage
	StopReason   StopReason
	ErrorMessage string
	Timestamp    time.Time
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

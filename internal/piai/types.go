// Package piai is a Go port of pi-ai (@earendil-works/pi-ai), a unified
// multi-provider LLM API: Stream(model, context) → event stream →
// AssistantMessage. See NOTICE for attribution.
package piai

import (
	"encoding/json"
	"time"
)

// StopReason mirrors pi-ai's stop reasons. "error" and "aborted" terminate a
// stream via an error event; the others via a done event.
type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonError   StopReason = "error"
	StopReasonAborted StopReason = "aborted"
)

// Cost is USD cost per usage component.
type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

// Usage is token usage for one assistant response. TotalTokens is the total
// processed by the LLM — input (with cache) plus output — and must equal
// Input + Output + CacheRead + CacheWrite.
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
	Data     string `json:"data"` // base64
	MimeType string `json:"mimeType"`
}

type ThinkingContent struct {
	Thinking string `json:"thinking"`
	// Signature: opaque provider payload replayed for multi-turn continuity.
	Signature string `json:"thinkingSignature,omitempty"`
	Redacted  bool   `json:"redacted,omitempty"`
}

type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// UserContent is TextContent or ImageContent.
type UserContent interface{ isUserContent() }

// AssistantContent is TextContent, ThinkingContent, or ToolCall.
type AssistantContent interface{ isAssistantContent() }

func (TextContent) isUserContent()          {}
func (ImageContent) isUserContent()         {}
func (TextContent) isAssistantContent()     {}
func (ThinkingContent) isAssistantContent() {}
func (ToolCall) isAssistantContent()        {}

// Message is UserMessage, AssistantMessage, or ToolResultMessage.
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

// UserText builds a plain-text user message.
func UserText(text string) UserMessage {
	return UserMessage{Content: []UserContent{TextContent{Text: text}}, Timestamp: time.Now()}
}

// Tool describes a callable tool. Parameters is a JSON Schema document.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Context is the full request context for one completion.
type Context struct {
	SystemPrompt string
	Messages     []Message
	Tools        []Tool
}

// CacheRetention is the prompt-cache retention preference. Zero value means
// the provider default ("short"); "none" disables caching.
type CacheRetention string

const (
	CacheRetentionNone  CacheRetention = "none"
	CacheRetentionShort CacheRetention = "short"
	CacheRetentionLong  CacheRetention = "long"
)

// StreamOptions are per-request options shared by all providers. Cancellation
// rides the context.Context passed to StreamFunction, not an option. Fields
// grow as adapters need them.
type StreamOptions struct {
	Temperature    *float64
	MaxTokens      int
	APIKey         string
	SessionID      string
	CacheRetention CacheRetention
	Headers        map[string]string
}

// Model identifies a concrete model behind an API shape.
type Model struct {
	ID            string
	Name          string
	API           string // API wire shape, e.g. "openai-completions"
	Provider      string // provider slug, e.g. "openai"
	BaseURL       string
	Reasoning     bool
	Cost          Cost // $/million tokens per component
	ContextWindow int
	MaxTokens     int
}

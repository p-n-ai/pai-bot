package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/jsonobject"
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

type ImageURLContent struct {
	URL string `json:"url"`
}

type ThinkingContent struct {
	Thinking string `json:"thinking"`

	Signature string `json:"thinkingSignature,omitempty"`
	Redacted  bool   `json:"redacted,omitempty"`
}

type ToolCall struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Arguments jsonobject.Object `json:"arguments"`
}

func NewToolArguments(fields ...jsonobject.Field) jsonobject.Object {
	return jsonobject.New(fields...)
}

func ToolArgumentsFrom[T any](value T) jsonobject.Object {
	return jsonobject.From(value)
}

func ToolArgument[T any](name string, value T) jsonobject.Field {
	return jsonobject.Member(name, value)
}

func ToolArgumentValue[T any](arguments jsonobject.Object, name string) (T, bool, error) {
	return jsonobject.Get[T](arguments, name)
}

func ToolArgumentValueOrZero[T any](arguments jsonobject.Object, name string) T {
	value, _, _ := jsonobject.Get[T](arguments, name)
	return value
}

func marshalToolArguments(arguments jsonobject.Object) (string, error) {
	encoded, err := json.Marshal(arguments)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func parseToolArguments(encoded string) (jsonobject.Object, error) {
	if strings.TrimSpace(encoded) == "" {
		return jsonobject.New(), nil
	}
	return jsonobject.Parse([]byte(encoded))
}

type ReasoningDetail struct {
	raw json.RawMessage
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
func (ImageURLContent) isUserContent()      {}
func (TextContent) isAssistantContent()     {}
func (ThinkingContent) isAssistantContent() {}
func (ToolCall) isAssistantContent()        {}

type Message interface{ isMessage() }

type SystemMessage struct {
	Content string
}

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

func (SystemMessage) isMessage()     {}
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

type StructuredOutputSpec struct {
	Name       string
	JSONSchema json.RawMessage
	Strict     bool
}

type ReasoningEffort string

const ReasoningEffortMinimal ReasoningEffort = "minimal"

type StreamOptions struct {
	Temperature      *float64
	MaxTokens        int
	APIKey           string
	SessionID        string
	CacheRetention   CacheRetention
	ReasoningEffort  ReasoningEffort
	Headers          map[string]string
	StructuredOutput *StructuredOutputSpec
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

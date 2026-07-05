//go:build integration
// +build integration

package llm_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func runningInCI() bool {
	ci := strings.TrimSpace(os.Getenv("CI"))
	if ci != "" && ci != "0" && !strings.EqualFold(ci, "false") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")), "true")
}

func liveOpenAIKey(t *testing.T) string {
	t.Helper()
	if runningInCI() {
		t.Skip("skipping live OpenAI integration tests on CI")
	}
	key := strings.TrimSpace(os.Getenv("LEARN_AI_OPENAI_API_KEY"))
	if key == "" {
		t.Skip("LEARN_AI_OPENAI_API_KEY is not set; skipping live OpenAI integration tests")
	}
	return key
}

func liveOpenAIModel() llm.Model {
	return llm.Model{
		ID:       "gpt-4o-mini",
		Name:     "GPT-4o mini",
		API:      llm.APIOpenAICompletions,
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
	}
}

func retryLive(t *testing.T, scenario func(ctx context.Context) error) {
	t.Helper()
	var err error
	for attempt := 1; attempt <= 4; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		err = scenario(ctx)
		cancel()
		if err == nil {
			return
		}
		t.Logf("attempt %d failed: %v", attempt, err)
	}
	t.Fatalf("all attempts failed, last error: %v", err)
}

func mathOperationTool() llm.Tool {
	return llm.Tool{
		Name:        "math_operation",
		Description: "Perform basic arithmetic operations",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"a": {"type": "number", "description": "First number"},
				"b": {"type": "number", "description": "Second number"},
				"operation": {"type": "string", "enum": ["add", "subtract", "multiply", "divide"], "description": "The operation to perform. One of 'add', 'subtract', 'multiply', 'divide'."}
			},
			"required": ["a", "b", "operation"]
		}`),
	}
}

func joinedText(msg llm.AssistantMessage) string {
	var b strings.Builder
	for _, block := range msg.Content {
		if text, ok := block.(llm.TextContent); ok {
			b.WriteString(text.Text)
		}
	}
	return b.String()
}

func checkLiveTextResponse(msg llm.AssistantMessage, want string) error {
	if len(msg.Content) == 0 {
		return fmt.Errorf("empty content")
	}
	if msg.Usage.Input+msg.Usage.CacheRead <= 0 {
		return fmt.Errorf("no input tokens: %+v", msg.Usage)
	}
	if msg.Usage.Output <= 0 {
		return fmt.Errorf("no output tokens: %+v", msg.Usage)
	}
	if msg.ErrorMessage != "" {
		return fmt.Errorf("errorMessage = %q", msg.ErrorMessage)
	}
	if got := joinedText(msg); !strings.Contains(got, want) {
		return fmt.Errorf("text = %q, want substring %q", got, want)
	}
	return nil
}

func TestOpenAILiveBasicTextGeneration(t *testing.T) {
	key := liveOpenAIKey(t)
	model := liveOpenAIModel()
	retryLive(t, func(ctx context.Context) error {
		c := llm.Context{
			SystemPrompt: "You are a helpful assistant. Be concise.",
			Messages:     []llm.Message{llm.UserText("Reply with exactly: 'Hello test successful'")},
		}
		first, err := llm.StreamOpenAICompletions(ctx, model, c, &llm.StreamOptions{APIKey: key}).Result()
		if err != nil {
			return err
		}
		if err := checkLiveTextResponse(first, "Hello test successful"); err != nil {
			return err
		}
		c.Messages = append(c.Messages, first, llm.UserText("Now say 'Goodbye test successful'"))
		second, err := llm.StreamOpenAICompletions(ctx, model, c, &llm.StreamOptions{APIKey: key}).Result()
		if err != nil {
			return err
		}
		return checkLiveTextResponse(second, "Goodbye test successful")
	})
}

func TestOpenAILiveToolCalling(t *testing.T) {
	key := liveOpenAIKey(t)
	model := liveOpenAIModel()
	retryLive(t, func(ctx context.Context) error {
		stream := llm.StreamOpenAICompletions(ctx, model, llm.Context{
			SystemPrompt: "You are a helpful assistant that uses tools when asked.",
			Messages:     []llm.Message{llm.UserText("Calculate 15 + 27 using the math_operation tool.")},
			Tools:        []llm.Tool{mathOperationTool()},
		}, &llm.StreamOptions{APIKey: key})

		var hasStart, hasDelta, hasEnd bool
		var accumulated strings.Builder
		index := -1
		for ev := range stream.Events() {
			switch ev.Type {
			case llm.EventToolCallStart:
				hasStart = true
				index = ev.ContentIndex
				block, ok := ev.Partial.Content[ev.ContentIndex].(llm.ToolCall)
				if !ok || block.Name != "math_operation" || block.ID == "" {
					return fmt.Errorf("toolcall_start block = %#v", ev.Partial.Content[ev.ContentIndex])
				}
			case llm.EventToolCallDelta:
				hasDelta = true
				if ev.ContentIndex != index {
					return fmt.Errorf("toolcall_delta contentIndex = %d, want %d", ev.ContentIndex, index)
				}
				block, ok := ev.Partial.Content[ev.ContentIndex].(llm.ToolCall)
				if !ok || block.Name != "math_operation" || block.Arguments == nil {
					return fmt.Errorf("toolcall_delta block = %#v", ev.Partial.Content[ev.ContentIndex])
				}
				accumulated.WriteString(ev.Delta)
			case llm.EventToolCallEnd:
				hasEnd = true
				if ev.ContentIndex != index {
					return fmt.Errorf("toolcall_end contentIndex = %d, want %d", ev.ContentIndex, index)
				}
				block, ok := ev.Partial.Content[ev.ContentIndex].(llm.ToolCall)
				if !ok || block.Name != "math_operation" {
					return fmt.Errorf("toolcall_end block = %#v", ev.Partial.Content[ev.ContentIndex])
				}
				var raw map[string]any
				if err := json.Unmarshal([]byte(accumulated.String()), &raw); err != nil {
					return fmt.Errorf("accumulated args %q: %w", accumulated.String(), err)
				}
				if block.Arguments == nil {
					return fmt.Errorf("toolcall_end arguments undefined")
				}
				if block.Arguments["a"] != float64(15) || block.Arguments["b"] != float64(27) {
					return fmt.Errorf("arguments = %+v", block.Arguments)
				}
				switch block.Arguments["operation"] {
				case "add", "subtract", "multiply", "divide":
				default:
					return fmt.Errorf("operation = %v", block.Arguments["operation"])
				}
			}
		}
		msg, err := stream.Result()
		if err != nil {
			return err
		}
		if !hasStart || !hasDelta || !hasEnd {
			return fmt.Errorf("events: start=%t delta=%t end=%t", hasStart, hasDelta, hasEnd)
		}
		if msg.StopReason != llm.StopReasonToolUse {
			return fmt.Errorf("stopReason = %q", msg.StopReason)
		}
		for _, block := range msg.Content {
			if tc, ok := block.(llm.ToolCall); ok {
				if tc.Name != "math_operation" || tc.ID == "" {
					return fmt.Errorf("toolCall = %+v", tc)
				}
				return nil
			}
		}
		return fmt.Errorf("no tool call found in response")
	})
}

func TestOpenAILiveStreaming(t *testing.T) {
	key := liveOpenAIKey(t)
	model := liveOpenAIModel()
	retryLive(t, func(ctx context.Context) error {
		stream := llm.StreamOpenAICompletions(ctx, model, llm.Context{
			SystemPrompt: "You are a helpful assistant.",
			Messages:     []llm.Message{llm.UserText("Count from 1 to 3")},
		}, &llm.StreamOptions{APIKey: key})

		var started, completed bool
		var chunks strings.Builder
		for ev := range stream.Events() {
			switch ev.Type {
			case llm.EventTextStart:
				started = true
			case llm.EventTextDelta:
				chunks.WriteString(ev.Delta)
			case llm.EventTextEnd:
				completed = true
			}
		}
		msg, err := stream.Result()
		if err != nil {
			return err
		}
		if !started || chunks.Len() == 0 || !completed {
			return fmt.Errorf("started=%t chunks=%d completed=%t", started, chunks.Len(), completed)
		}
		for _, block := range msg.Content {
			if _, ok := block.(llm.TextContent); ok {
				return nil
			}
		}
		return fmt.Errorf("no text block in response")
	})
}

func TestOpenAILiveImageInput(t *testing.T) {
	key := liveOpenAIKey(t)
	model := liveOpenAIModel()
	png, err := os.ReadFile("testdata/red-circle.png")
	if err != nil {
		t.Fatalf("read image: %v", err)
	}
	image := llm.ImageContent{Data: base64.StdEncoding.EncodeToString(png), MimeType: "image/png"}
	retryLive(t, func(ctx context.Context) error {
		msg, err := llm.StreamOpenAICompletions(ctx, model, llm.Context{
			SystemPrompt: "You are a helpful assistant.",
			Messages: []llm.Message{llm.UserMessage{Content: []llm.UserContent{
				llm.TextContent{Text: "What do you see in this image? Please describe the shape (circle, rectangle, square, triangle, ...) and color (red, blue, green, ...). You MUST reply in English."},
				image,
			}}},
		}, &llm.StreamOptions{APIKey: key}).Result()
		if err != nil {
			return err
		}
		if len(msg.Content) == 0 {
			return fmt.Errorf("empty content")
		}
		for _, block := range msg.Content {
			if text, ok := block.(llm.TextContent); ok {
				got := strings.ToLower(text.Text)
				if !strings.Contains(got, "red") || !strings.Contains(got, "circle") {
					return fmt.Errorf("text = %q", text.Text)
				}
				return nil
			}
		}
		return fmt.Errorf("no text block in response")
	})
}

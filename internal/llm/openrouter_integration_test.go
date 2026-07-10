//go:build integration
// +build integration

package llm_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func liveOpenRouterKey(t *testing.T) string {
	t.Helper()
	if runningInCI() {
		t.Skip("skipping live OpenRouter integration tests on CI")
	}
	key := strings.TrimSpace(os.Getenv("LEARN_AI_OPENROUTER_API_KEY"))
	if key == "" {
		t.Skip("LEARN_AI_OPENROUTER_API_KEY is not set; skipping live OpenRouter integration tests")
	}
	return key
}

func liveOpenRouterModel() llm.Model {
	modelID := strings.TrimSpace(os.Getenv("LEARN_AI_OPENROUTER_MODEL"))
	if modelID == "" {
		modelID = "openai/gpt-4o-mini"
	}
	return llm.Model{
		ID:       modelID,
		Name:     modelID,
		API:      llm.APIOpenRouterChat,
		Provider: "openrouter",
		BaseURL:  "https://openrouter.ai/api/v1",
	}
}

func TestOpenRouterLiveStreaming(t *testing.T) {
	key := liveOpenRouterKey(t)
	model := liveOpenRouterModel()
	retryLive(t, func(ctx context.Context) error {
		stream := llm.StreamOpenRouterChat(
			ctx,
			model,
			llm.Context{Messages: []llm.Message{llm.UserText("Reply with exactly: OR_NATIVE_OK")}},
			&llm.StreamOptions{APIKey: key, MaxTokens: 32},
		)
		events := collectEvents(stream)
		msg, err := stream.Result()
		if err != nil {
			return err
		}
		if msg.StopReason != llm.StopReasonStop {
			return fmt.Errorf("stop reason = %q", msg.StopReason)
		}
		if !strings.Contains(joinedText(msg), "OR_NATIVE_OK") {
			return fmt.Errorf("response text = %q", joinedText(msg))
		}
		if !containsType(events, llm.EventTextDelta) || !containsType(events, llm.EventDone) {
			return fmt.Errorf("stream events = %v", eventTypes(events))
		}
		return nil
	})
}

func TestOpenRouterLiveTextOnlyAssistantHistory(t *testing.T) {
	key := liveOpenRouterKey(t)
	model := liveOpenRouterModel()
	retryLive(t, func(ctx context.Context) error {
		firstUser := llm.UserText("Remember the code word COBALT-7319. Reply only ACK.")
		first, err := llm.StreamOpenRouterChat(
			ctx,
			model,
			llm.Context{Messages: []llm.Message{firstUser}},
			&llm.StreamOptions{APIKey: key, MaxTokens: 32},
		).Result()
		if err != nil {
			return err
		}
		textOnlyHistory := llm.AssistantMessage{
			Content: []llm.AssistantContent{llm.TextContent{Text: joinedText(first)}},
		}
		second, err := llm.StreamOpenRouterChat(
			ctx,
			model,
			llm.Context{Messages: []llm.Message{
				firstUser,
				textOnlyHistory,
				llm.UserText("What code word did I ask you to remember? Reply only with the code word."),
			}},
			&llm.StreamOptions{APIKey: key, MaxTokens: 32},
		).Result()
		if err != nil {
			return err
		}
		if !strings.Contains(joinedText(second), "COBALT-7319") {
			return fmt.Errorf("multi-turn response = %q", joinedText(second))
		}
		return nil
	})
}

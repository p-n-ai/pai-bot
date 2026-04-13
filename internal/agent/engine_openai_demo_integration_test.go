// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package agent_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestOpenAILiveConversationDemo(t *testing.T) {
	if runningInCI() {
		t.Skip("skipping live OpenAI demo on CI")
	}

	apiKey := strings.TrimSpace(os.Getenv("LEARN_AI_OPENAI_API_KEY"))
	if apiKey == "" {
		t.Skip("LEARN_AI_OPENAI_API_KEY is not set; skipping live OpenAI demo")
	}

	timeout := 45 * time.Second
	provider := ai.NewOpenAIProvider(apiKey, ai.WithHTTPClient(&http.Client{Timeout: timeout}))
	router := ai.NewRouterWithConfig(ai.RouterConfig{
		RetryBackoff:            []time.Duration{250 * time.Millisecond, 500 * time.Millisecond},
		BreakerFailureThreshold: 2,
		BreakerCooldown:         2 * time.Second,
	})
	router.Register("openai", provider)

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: router,
		Store:    agent.NewMemoryStore(),
	})

	turns := []string{
		"Saya pelajar Tingkatan 1. Tolong ajar 2x + 4 = 10.",
		"Kenapa kita tolak 4 dahulu?",
	}

	userID := "live-demo-c01"
	for i, turn := range turns {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		resp, err := engine.ProcessMessage(ctx, chat.InboundMessage{
			Channel: "telegram",
			UserID:  userID,
			Text:    turn,
		})
		cancel()
		if err != nil {
			t.Fatalf("turn %d ProcessMessage error: %v", i+1, err)
		}
		if strings.TrimSpace(resp) == "" {
			t.Fatalf("turn %d returned empty response", i+1)
		}

		t.Logf("turn %d user:\n%s", i+1, turn)
		t.Logf("turn %d assistant:\n%s", i+1, resp)
	}
}

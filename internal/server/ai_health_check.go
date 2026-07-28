// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

// NewAIHealthCheck verifies the routed AI completion path used by learners.
func NewAIHealthCheck(router *ai.Router) func(context.Context) error {
	return func(ctx context.Context) error {
		if router == nil {
			return fmt.Errorf("AI completion health check failed: router unavailable")
		}

		response, err := router.ProbePrimaryCompletion(ctx, ai.CompletionRequest{
			Messages: []ai.Message{
				{Role: "user", Content: "Reply with only OK."},
			},
			MaxTokens:   8,
			Temperature: 0,
			Task:        ai.TaskAnalysis,
		})
		if err != nil {
			return fmt.Errorf("AI completion health check failed: %w", err)
		}
		if strings.TrimSpace(response.Content) == "" {
			return fmt.Errorf("AI completion health check failed: empty response")
		}
		if isTechnicalIssueMessage(response.Content) {
			return fmt.Errorf("AI completion health check failed: fallback response")
		}
		return nil
	}
}

func isTechnicalIssueMessage(response string) bool {
	response = strings.TrimSpace(response)
	for _, locale := range []string{"en", "ms", "zh"} {
		if response == i18n.S(locale, i18n.MsgTechnicalIssue) {
			return true
		}
	}
	return false
}

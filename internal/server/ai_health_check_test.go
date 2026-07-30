// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

func TestAIHealthCheckUsesCompletionPath(t *testing.T) {
	tests := []struct {
		name     string
		response string
		err      error
		wantErr  bool
	}{
		{name: "operational", response: "OK"},
		{name: "empty response", response: " ", wantErr: true},
		{
			name:     "technical issue fallback",
			response: i18n.S("en", i18n.MsgTechnicalIssue),
			wantErr:  true,
		},
		{name: "provider failure", err: context.DeadlineExceeded, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := ai.NewRouter()
			router.Register("provider", &ai.MockProvider{
				Response: test.response,
				Err:      test.err,
			})

			err := NewAIHealthCheck(router)(context.Background())
			if (err != nil) != test.wantErr {
				t.Fatalf("NewAIHealthCheck() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestAIHealthCheckSendsBoundedDeterministicProbe(t *testing.T) {
	provider := &ai.MockProvider{Response: "OK"}
	router := ai.NewRouter()
	router.Register("provider", provider)

	if err := NewAIHealthCheck(router)(context.Background()); err != nil {
		t.Fatalf("NewAIHealthCheck() error = %v", err)
	}
	if provider.LastRequest == nil {
		t.Fatal("AI health check did not send a completion request")
	}
	request := provider.LastRequest
	if len(request.Messages) != 1 ||
		request.Messages[0].Role != "user" ||
		request.Messages[0].Content != "Reply with only OK." {
		t.Fatalf("AI health request messages = %#v", request.Messages)
	}
	if request.MaxTokens != 8 || request.Temperature != 0 || request.Task != ai.TaskAnalysis {
		t.Fatalf(
			"AI health request maxTokens=%d temperature=%v task=%v",
			request.MaxTokens,
			request.Temperature,
			request.Task,
		)
	}
}

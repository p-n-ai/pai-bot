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

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func TestCurriculumLookupToolRejectsTopicBelowAITeachingQuality(t *testing.T) {
	tool := curriculumLookupTool{loader: newUnlockCurriculumLoader(t)}

	result, err := tool.Execute(context.Background(), llm.ToolCall{
		Arguments: map[string]any{"topic_id": "F1-02"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("Execute() = %#v, want lower-quality topic rejected", result)
	}
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalchat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

type stubProcessor struct {
	responses []agent.TurnResult
	err       error
	messages  []chat.InboundMessage
}

func (s *stubProcessor) ProcessTurn(_ context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
	s.messages = append(s.messages, msg)
	if s.err != nil {
		return agent.TurnResult{}, s.err
	}
	if len(s.responses) == 0 {
		return agent.TurnResult{}, nil
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]
	return resp, nil
}

func TestRun_ForwardsConversationAndPrintsReplies(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("hello bot\nhow are you?\n/exit\n")
	processor := &stubProcessor{
		responses: []agent.TurnResult{
			{Text: "Hi there"},
			{Text: "I'm ready to help"},
		},
	}

	err := Run(context.Background(), input, &output, processor, Config{
		UserID:  "cli-user",
		Channel: "terminal",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(processor.messages) != 2 {
		t.Fatalf("processed messages = %d, want 2", len(processor.messages))
	}
	if processor.messages[0].UserID != "cli-user" {
		t.Fatalf("first UserID = %q, want cli-user", processor.messages[0].UserID)
	}
	if processor.messages[0].Channel != "terminal" {
		t.Fatalf("first Channel = %q, want terminal", processor.messages[0].Channel)
	}
	if processor.messages[0].Text != "hello bot" {
		t.Fatalf("first Text = %q, want hello bot", processor.messages[0].Text)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "Terminal chat ready") {
		t.Fatalf("output = %q, want welcome text", rendered)
	}
	if !strings.Contains(rendered, "P&AI> Hi there") {
		t.Fatalf("output = %q, want first reply", rendered)
	}
	if !strings.Contains(rendered, "P&AI> I'm ready to help") {
		t.Fatalf("output = %q, want second reply", rendered)
	}
	if !strings.Contains(rendered, "Session ended.") {
		t.Fatalf("output = %q, want session end text", rendered)
	}
}

func TestRun_PrintsFocusedPageAfterTutorText(t *testing.T) {
	var output strings.Builder
	processor := &stubProcessor{responses: []agent.TurnResult{{
		Text:        "Your report is ready.",
		FocusedPage: &focusedpage.Artifact{URL: "https://pages.example/a/page-1#private-capability"},
	}}}

	err := Run(context.Background(), strings.NewReader("make my report\n/exit\n"), &output, processor, Config{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "P&AI> Your report is ready.\nFocused page> https://pages.example/a/page-1#private-capability\n") {
		t.Fatalf("output = %q, want tutor text followed by focused-page URL", rendered)
	}
}

func TestRun_ProcessorErrorIsPrintedAndLoopContinues(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("hello\n/exit\n")
	processor := &stubProcessor{err: errors.New("boom")}

	err := Run(context.Background(), input, &output, processor, Config{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "Error: boom") {
		t.Fatalf("output = %q, want processor error", rendered)
	}
	if len(processor.messages) != 1 {
		t.Fatalf("processed messages = %d, want 1", len(processor.messages))
	}
}

func TestRun_IgnoresBlankLinesAndQuitAlias(t *testing.T) {
	var output strings.Builder
	input := strings.NewReader("\n   \n/quit\n")
	processor := &stubProcessor{}

	err := Run(context.Background(), input, &output, processor, Config{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(processor.messages) != 0 {
		t.Fatalf("processed messages = %d, want 0", len(processor.messages))
	}
}

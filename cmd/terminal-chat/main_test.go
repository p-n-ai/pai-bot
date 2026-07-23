// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
	"github.com/p-n-ai/pai-bot/internal/server"
)

type turnProcessorStub struct {
	result agent.TurnResult
}

func (s turnProcessorStub) ProcessTurn(context.Context, chat.InboundMessage) (agent.TurnResult, error) {
	return s.result, nil
}

func TestWriteConversationHistoryTightensExistingFilePermissions(t *testing.T) {
	path := t.TempDir() + "/history.json"
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	history := newConversationHistory("u1", "terminal")
	history.Turns = append(history.Turns, conversationTurnJSON{
		UserID:  "u1",
		Channel: "terminal",
		Role:    "student",
		Text:    "hi",
	})
	if err := writeConversationHistory(path, history, 0); err != nil {
		t.Fatalf("writeConversationHistory() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("history permissions = %v, want 0600", got)
	}
}

func TestHistoryProcessorPreservesArtifactWithoutRecordingCapabilityURL(t *testing.T) {
	artifact := &focusedpage.Artifact{URL: "https://pages.example/a/page-1#private-capability"}
	history := newConversationHistory("u1", "telegram")
	processor := &historyProcessor{
		inner:   turnProcessorStub{result: agent.TurnResult{Text: "Your report is ready.", FocusedPage: artifact}},
		history: history,
	}

	result, err := processor.ProcessTurn(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "u1",
		Text:    "make my report",
	})
	if err != nil {
		t.Fatalf("ProcessTurn() error = %v", err)
	}
	if result.FocusedPage != artifact {
		t.Fatal("ProcessTurn() did not preserve the focused-page artifact")
	}

	snapshot := history.snapshot(0)
	if len(snapshot.Turns) != 2 || snapshot.Turns[1].Text != "Your report is ready." {
		t.Fatalf("history turns = %#v, want student and assistant text", snapshot.Turns)
	}
	for _, turn := range snapshot.Turns {
		if strings.Contains(turn.Text, "private-capability") || strings.Contains(turn.Text, artifact.URL) {
			t.Fatalf("history leaked focused-page capability in %#v", turn)
		}
	}
}

func TestWriteWSResponseWithFocusedPage(t *testing.T) {
	var output strings.Builder
	err := writeWSResponse(&output, "", wsOutboundMsg{
		Type:        "response",
		Text:        "Your report is ready.",
		FocusedPage: &wsFocusedPage{URL: "https://pages.example/a/page-1#private-capability"},
	})
	if err != nil {
		t.Fatalf("writeWSResponse() error = %v", err)
	}
	if got, want := output.String(), "Your report is ready.\nFocused page: https://pages.example/a/page-1#private-capability\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriteWSResponseWithoutFocusedPagePreservesPlainText(t *testing.T) {
	var output strings.Builder
	if err := writeWSResponse(&output, "", wsOutboundMsg{Type: "response", Text: "Plain tutor reply"}); err != nil {
		t.Fatalf("writeWSResponse() error = %v", err)
	}
	if got, want := output.String(), "Plain tutor reply\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWSClientOnceRendersTurnWithAndWithoutFocusedPage(t *testing.T) {
	tests := []struct {
		name   string
		result agent.TurnResult
		want   string
	}{
		{
			name:   "plain text",
			result: agent.TurnResult{Text: "Plain tutor reply"},
			want:   "Plain tutor reply\n",
		},
		{
			name: "focused page",
			result: agent.TurnResult{
				Text: "Your report is ready.",
				FocusedPage: &focusedpage.Artifact{
					URL: "https://pages.example/a/page-1#private-capability",
				},
			},
			want: "Your report is ready.\nFocused page: https://pages.example/a/page-1#private-capability\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := chat.NewWSChannel()
			gateway := chat.NewGateway()
			gateway.Register("websocket", channel)
			deliverer := server.NewGatewayTurnDeliverer(gateway, agent.NewMemoryStore())
			if err := channel.Start(context.Background(), func(msg chat.InboundMessage) {
				_ = deliverer.DeliverTurn(context.Background(), msg, tt.result)
			}); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			server := httptest.NewServer(channel.Handler())
			t.Cleanup(func() {
				_ = channel.Stop()
				server.Close()
			})

			var output strings.Builder
			serverURL := "ws" + strings.TrimPrefix(server.URL, "http")
			if err := runWSClientOnceTo(serverURL, "terminal-user", "make my report", &output); err != nil {
				t.Fatalf("runWSClientOnceTo() error = %v", err)
			}
			if got := output.String(); got != tt.want {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

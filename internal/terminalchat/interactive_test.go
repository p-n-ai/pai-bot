// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalchat

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

type stubInteractiveSession struct {
	mu      sync.Mutex
	process func(context.Context, chat.InboundMessage) (agent.TurnResult, error)
	reload  func(context.Context) (InteractiveStatus, error)
	status  InteractiveStatus
}

func (s *stubInteractiveSession) ProcessTurn(ctx context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
	return s.process(ctx, msg)
}

func (s *stubInteractiveSession) New(context.Context) (InteractiveStatus, error) {
	return s.Status(), nil
}

func (s *stubInteractiveSession) Reload(ctx context.Context) (InteractiveStatus, error) {
	if s.reload != nil {
		return s.reload(ctx)
	}
	return s.Status(), nil
}

func (s *stubInteractiveSession) SelectCharacter(_ context.Context, id string) (InteractiveStatus, error) {
	s.mu.Lock()
	s.status.CharacterID = id
	status := s.status
	s.mu.Unlock()
	return status, nil
}

func (s *stubInteractiveSession) Status() InteractiveStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (*stubInteractiveSession) CandidateChanged() (bool, error) {
	return false, nil
}

func TestRunInteractiveQueuesMessagesFIFO(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close() }()
	started := make(chan string, 3)
	releaseFirst := make(chan struct{})
	session := &stubInteractiveSession{
		status: InteractiveStatus{
			Provider:      "codex",
			Memory:        true,
			CharacterID:   "default",
			CandidateHash: "sha256:test",
		},
		process: func(_ context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
			started <- msg.Text
			if msg.Text == "first" {
				<-releaseFirst
			}
			return agent.TurnResult{Text: "reply:" + msg.Text}, nil
		},
	}
	var output strings.Builder
	runDone := make(chan error, 1)
	go func() {
		runDone <- RunInteractive(t.Context(), reader, &output, session, InteractiveConfig{})
	}()

	writeInteractiveTestLine(t, writer, "first")
	requireInteractiveTestValue(t, started, "first")
	writeInteractiveTestLine(t, writer, "second")
	writeInteractiveTestLine(t, writer, "third")
	if err := writer.Close(); err != nil {
		t.Fatalf("close input: %v", err)
	}

	select {
	case got := <-started:
		t.Fatalf("queued turn %q started before first completed", got)
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseFirst)
	requireInteractiveTestValue(t, started, "second")
	requireInteractiveTestValue(t, started, "third")
	requireInteractiveTestDone(t, runDone)

	rendered := output.String()
	for _, want := range []string{
		"[queued #1]",
		"[queued #2]",
		"P&AI> reply:first",
		"P&AI> reply:second",
		"P&AI> reply:third",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output = %q, want %q", rendered, want)
		}
	}
	if strings.Index(rendered, "reply:first") > strings.Index(rendered, "reply:second") ||
		strings.Index(rendered, "reply:second") > strings.Index(rendered, "reply:third") {
		t.Fatalf("output replies are not FIFO: %q", rendered)
	}
}

func TestRunInteractiveInterruptSuppressesStaleReply(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close() }()
	firstStarted := make(chan struct{})
	firstCanceled := make(chan struct{})
	session := &stubInteractiveSession{
		status: InteractiveStatus{
			Provider:      "codex",
			Memory:        true,
			CharacterID:   "default",
			CandidateHash: "sha256:test",
		},
		process: func(ctx context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
			if msg.Text == "long answer" {
				close(firstStarted)
				<-ctx.Done()
				close(firstCanceled)
				return agent.TurnResult{Text: "stale reply"}, nil
			}
			return agent.TurnResult{Text: "fresh reply"}, nil
		},
	}
	var output strings.Builder
	runDone := make(chan error, 1)
	go func() {
		runDone <- RunInteractive(t.Context(), reader, &output, session, InteractiveConfig{})
	}()

	writeInteractiveTestLine(t, writer, "long answer")
	requireInteractiveTestSignal(t, firstStarted, "first turn did not start")
	writeInteractiveTestLine(t, writer, "/interrupt actually, be brief")
	if err := writer.Close(); err != nil {
		t.Fatalf("close input: %v", err)
	}
	requireInteractiveTestSignal(t, firstCanceled, "first turn was not canceled")
	requireInteractiveTestDone(t, runDone)

	rendered := output.String()
	if strings.Contains(rendered, "stale reply") {
		t.Fatalf("interrupted response leaked: %q", rendered)
	}
	if !strings.Contains(rendered, "[interrupted]") || !strings.Contains(rendered, "P&AI> fresh reply") {
		t.Fatalf("output = %q, want interruption marker and fresh replacement", rendered)
	}
}

func TestRunInteractiveReloadDrainsActiveTurnAndDropsQueue(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close() }()
	firstStarted := make(chan struct{})
	firstCanceled := make(chan struct{})
	releaseFirst := make(chan struct{})
	reloadCalled := make(chan struct{})
	processed := make(chan string, 3)
	session := &stubInteractiveSession{
		status: InteractiveStatus{
			Provider:      "codex",
			Memory:        true,
			CharacterID:   "default",
			CandidateHash: "sha256:old",
		},
		process: func(ctx context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
			processed <- msg.Text
			if msg.Text == "long answer" {
				close(firstStarted)
				<-ctx.Done()
				close(firstCanceled)
				<-releaseFirst
				return agent.TurnResult{Text: "stale reply"}, nil
			}
			return agent.TurnResult{Text: "unexpected reply"}, nil
		},
		reload: func(context.Context) (InteractiveStatus, error) {
			close(reloadCalled)
			return InteractiveStatus{
				Provider:      "codex",
				Memory:        true,
				CharacterID:   "default",
				CandidateHash: "sha256:new",
			}, nil
		},
	}
	var output strings.Builder
	runDone := make(chan error, 1)
	go func() {
		runDone <- RunInteractive(t.Context(), reader, &output, session, InteractiveConfig{})
	}()

	writeInteractiveTestLine(t, writer, "long answer")
	requireInteractiveTestSignal(t, firstStarted, "first turn did not start")
	writeInteractiveTestLine(t, writer, "queued stale follow-up")
	writeInteractiveTestLine(t, writer, "/reload")
	requireInteractiveTestSignal(t, firstCanceled, "reload did not cancel the active turn")
	select {
	case <-reloadCalled:
		t.Fatal("reload ran before the canceled processor drained")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseFirst)
	requireInteractiveTestSignal(t, reloadCalled, "reload did not run after the processor drained")
	if err := writer.Close(); err != nil {
		t.Fatalf("close input: %v", err)
	}
	requireInteractiveTestDone(t, runDone)

	close(processed)
	var messages []string
	for message := range processed {
		messages = append(messages, message)
	}
	if len(messages) != 1 || messages[0] != "long answer" {
		t.Fatalf("processed messages = %#v, want only the canceled active turn", messages)
	}
	rendered := output.String()
	for _, want := range []string{
		"[queued #1]",
		"[interrupted]",
		"Candidate reloaded into a fresh session.",
		"prompt=sha256:new",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output = %q, want %q", rendered, want)
		}
	}
	if strings.Contains(rendered, "stale reply") {
		t.Fatalf("reloaded session exposed stale reply: %q", rendered)
	}
}

func TestRunInteractivePassesUnknownSlashCommandToProcessor(t *testing.T) {
	processed := make(chan chat.InboundMessage, 1)
	session := &stubInteractiveSession{
		status: InteractiveStatus{
			Provider:      "codex",
			Memory:        true,
			CharacterID:   "default",
			CandidateHash: "sha256:test",
		},
		process: func(_ context.Context, msg chat.InboundMessage) (agent.TurnResult, error) {
			processed <- msg
			return agent.TurnResult{Text: "challenge accepted"}, nil
		},
	}
	var output strings.Builder

	err := RunInteractive(
		t.Context(),
		strings.NewReader("/challenge friend\n"),
		&output,
		session,
		InteractiveConfig{},
	)
	if err != nil {
		t.Fatalf("RunInteractive() error = %v", err)
	}
	message := <-processed
	if message.Text != "/challenge friend" || message.UserID != "terminal-user" || message.Channel != "terminal" {
		t.Fatalf("processed message = %#v, want unchanged slash command with defaults", message)
	}
	if !strings.Contains(output.String(), "P&AI> challenge accepted") {
		t.Fatalf("output = %q, want processor reply", output.String())
	}
}

func writeInteractiveTestLine(t *testing.T, writer io.Writer, text string) {
	t.Helper()
	if _, err := io.WriteString(writer, text+"\n"); err != nil {
		t.Fatalf("write %q: %v", text, err)
	}
}

func requireInteractiveTestValue(t *testing.T, values <-chan string, want string) {
	t.Helper()
	select {
	case got := <-values:
		if got != want {
			t.Fatalf("started = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %q", want)
	}
}

func requireInteractiveTestSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

func requireInteractiveTestDone(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunInteractive() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RunInteractive() did not finish")
	}
}

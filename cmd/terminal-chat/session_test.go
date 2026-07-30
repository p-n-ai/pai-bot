// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/terminalchat"
)

type recordingTerminalProcessor struct {
	mu       sync.Mutex
	prompt   string
	messages []chat.InboundMessage
}

func (p *recordingTerminalProcessor) ProcessTurn(
	_ context.Context,
	message chat.InboundMessage,
) (agent.TurnResult, error) {
	p.mu.Lock()
	p.messages = append(p.messages, message)
	p.mu.Unlock()
	return agent.TurnResult{Text: p.prompt}, nil
}

func (p *recordingTerminalProcessor) snapshot() []chat.InboundMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]chat.InboundMessage(nil), p.messages...)
}

func TestInteractiveTerminalSessionNewPreservesSelectedCharacter(t *testing.T) {
	directory := t.TempDir()
	writeTerminalSessionFile(t, directory, `prompt: candidate one
default: aina
characters:
  - id: aina
    first_name: Aina
    language: en
  - id: faris
    first_name: Faris
    username: faris_student
    language: ms
`)

	var processors []*recordingTerminalProcessor
	session, err := newInteractiveTerminalSession(
		terminalchat.NewCandidateSource(directory),
		"codex",
		func(candidate terminalchat.Candidate) (terminalchat.Processor, error) {
			processor := &recordingTerminalProcessor{prompt: candidate.Prompt}
			processors = append(processors, processor)
			return processor, nil
		},
	)
	if err != nil {
		t.Fatalf("newInteractiveTerminalSession() error = %v", err)
	}
	if _, err := session.SelectCharacter(t.Context(), "faris"); err != nil {
		t.Fatalf("SelectCharacter() error = %v", err)
	}
	if _, err := session.ProcessTurn(t.Context(), chat.InboundMessage{Text: "first"}); err != nil {
		t.Fatalf("first ProcessTurn() error = %v", err)
	}
	if _, err := session.New(t.Context()); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := session.ProcessTurn(t.Context(), chat.InboundMessage{Text: "second"}); err != nil {
		t.Fatalf("second ProcessTurn() error = %v", err)
	}

	if len(processors) != 3 {
		t.Fatalf("processors built = %d, want initial, character, and new sessions", len(processors))
	}
	for index, processor := range processors[1:] {
		messages := processor.snapshot()
		if len(messages) != 1 {
			t.Fatalf("processor %d messages = %#v, want one", index+2, messages)
		}
		message := messages[0]
		if message.FirstName != "Faris" || message.Username != "faris_student" || message.Language != "ms" {
			t.Fatalf("processor %d character = %#v, want Faris profile", index+2, message)
		}
	}
	if status := session.Status(); status.CharacterID != "faris" || !status.Memory || status.Provider != "codex" {
		t.Fatalf("Status() = %#v, want preserved Faris Codex memory session", status)
	}
}

func TestInteractiveTerminalSessionReloadIsAtomicAndUsesCandidateDefault(t *testing.T) {
	directory := t.TempDir()
	writeTerminalSessionFile(t, directory, `prompt: old candidate
default: aina
characters:
  - id: aina
  - id: faris
`)

	var processors []*recordingTerminalProcessor
	session, err := newInteractiveTerminalSession(
		terminalchat.NewCandidateSource(directory),
		"codex",
		func(candidate terminalchat.Candidate) (terminalchat.Processor, error) {
			processor := &recordingTerminalProcessor{prompt: candidate.Prompt}
			processors = append(processors, processor)
			return processor, nil
		},
	)
	if err != nil {
		t.Fatalf("newInteractiveTerminalSession() error = %v", err)
	}
	if _, err := session.SelectCharacter(t.Context(), "faris"); err != nil {
		t.Fatalf("SelectCharacter() error = %v", err)
	}
	before := session.Status()

	writeTerminalSessionFile(t, directory, `prompt: new candidate
default: aina
characters:
  - id: aina
    langauge: en
`)
	if _, err := session.Reload(t.Context()); err == nil || !strings.Contains(err.Error(), "langauge") {
		t.Fatalf("Reload() error = %v, want invalid candidate failure", err)
	}
	if after := session.Status(); after != before {
		t.Fatalf("status after failed reload = %#v, want unchanged %#v", after, before)
	}
	result, err := session.ProcessTurn(t.Context(), chat.InboundMessage{Text: "still old"})
	if err != nil {
		t.Fatalf("ProcessTurn() after failed reload error = %v", err)
	}
	if result.Text != "old candidate" {
		t.Fatalf("reply after failed reload = %q, want old candidate", result.Text)
	}

	writeTerminalSessionFile(t, directory, `prompt: new candidate
default: aina
characters:
  - id: aina
    first_name: Aina
    language: en
  - id: faris
`)
	reloaded, err := session.Reload(t.Context())
	if err != nil {
		t.Fatalf("Reload(valid) error = %v", err)
	}
	if reloaded.CharacterID != "aina" {
		t.Fatalf("reloaded character = %q, want candidate default aina", reloaded.CharacterID)
	}
	if reloaded.CandidateHash == before.CandidateHash {
		t.Fatalf("candidate hash did not change: %q", reloaded.CandidateHash)
	}
	result, err = session.ProcessTurn(t.Context(), chat.InboundMessage{Text: "now new"})
	if err != nil {
		t.Fatalf("ProcessTurn() after reload error = %v", err)
	}
	if result.Text != "new candidate" {
		t.Fatalf("reply after valid reload = %q, want new candidate", result.Text)
	}
}

func TestInteractiveTerminalSessionCharacterFactoryFailureIsAtomic(t *testing.T) {
	directory := t.TempDir()
	writeTerminalSessionFile(t, directory, `prompt: candidate
default: aina
characters:
  - id: aina
    first_name: Aina
  - id: faris
    first_name: Faris
`)

	processor := &recordingTerminalProcessor{prompt: "active reply"}
	factoryCalls := 0
	session, err := newInteractiveTerminalSession(
		terminalchat.NewCandidateSource(directory),
		"codex",
		func(terminalchat.Candidate) (terminalchat.Processor, error) {
			factoryCalls++
			if factoryCalls > 1 {
				return nil, errors.New("processor unavailable")
			}
			return processor, nil
		},
	)
	if err != nil {
		t.Fatalf("newInteractiveTerminalSession() error = %v", err)
	}
	before := session.Status()

	if _, err := session.SelectCharacter(t.Context(), "faris"); err == nil ||
		!strings.Contains(err.Error(), "processor unavailable") {
		t.Fatalf("SelectCharacter() error = %v, want processor failure", err)
	}
	if after := session.Status(); after != before {
		t.Fatalf("status after failed selection = %#v, want unchanged %#v", after, before)
	}
	result, err := session.ProcessTurn(t.Context(), chat.InboundMessage{Text: "hello"})
	if err != nil {
		t.Fatalf("ProcessTurn() error = %v", err)
	}
	if result.Text != "active reply" {
		t.Fatalf("ProcessTurn() reply = %q, want active processor reply", result.Text)
	}
	messages := processor.snapshot()
	if len(messages) != 1 || messages[0].FirstName != "Aina" {
		t.Fatalf("active processor messages = %#v, want original Aina character", messages)
	}
}

func writeTerminalSessionFile(t *testing.T, directory, content string) {
	t.Helper()
	const filename = "candidate.yaml"
	if err := os.WriteFile(filepath.Join(directory, filename), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

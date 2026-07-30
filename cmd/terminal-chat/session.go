// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/terminalchat"
)

type terminalProcessorFactory func(terminalchat.Candidate) (terminalchat.Processor, error)

type interactiveTerminalSession struct {
	mu        sync.RWMutex
	source    terminalchat.CandidateSource
	factory   terminalProcessorFactory
	processor terminalchat.Processor
	candidate terminalchat.Candidate
	character terminalchat.Character
	provider  string
}

func newInteractiveTerminalSession(
	source terminalchat.CandidateSource,
	provider string,
	factory terminalProcessorFactory,
) (*interactiveTerminalSession, error) {
	if factory == nil {
		return nil, errors.New("processor factory is required")
	}
	candidate, err := source.Load()
	if err != nil {
		return nil, err
	}
	processor, err := factory(candidate)
	if err != nil {
		return nil, err
	}
	return &interactiveTerminalSession{
		source:    source,
		factory:   factory,
		processor: processor,
		candidate: candidate,
		character: candidate.DefaultCharacter(),
		provider:  strings.ToLower(strings.TrimSpace(provider)),
	}, nil
}

func (s *interactiveTerminalSession) ProcessTurn(
	ctx context.Context,
	message chat.InboundMessage,
) (agent.TurnResult, error) {
	s.mu.RLock()
	processor := s.processor
	character := s.character
	s.mu.RUnlock()
	if processor == nil {
		return agent.TurnResult{}, errors.New("interactive processor is not configured")
	}
	message.FirstName = character.FirstName
	message.Username = character.Username
	message.Language = character.Language
	return processor.ProcessTurn(ctx, message)
}

func (s *interactiveTerminalSession) New(context.Context) (terminalchat.InteractiveStatus, error) {
	s.mu.RLock()
	candidate := s.candidate
	character := s.character
	s.mu.RUnlock()

	processor, err := s.factory(candidate)
	if err != nil {
		return terminalchat.InteractiveStatus{}, err
	}
	s.mu.Lock()
	s.processor = processor
	s.character = character
	status := s.statusLocked()
	s.mu.Unlock()
	return status, nil
}

func (s *interactiveTerminalSession) Reload(context.Context) (terminalchat.InteractiveStatus, error) {
	candidate, err := s.source.Load()
	if err != nil {
		return terminalchat.InteractiveStatus{}, err
	}
	processor, err := s.factory(candidate)
	if err != nil {
		return terminalchat.InteractiveStatus{}, err
	}
	s.mu.Lock()
	s.processor = processor
	s.candidate = candidate
	s.character = candidate.DefaultCharacter()
	status := s.statusLocked()
	s.mu.Unlock()
	return status, nil
}

func (s *interactiveTerminalSession) SelectCharacter(
	_ context.Context,
	id string,
) (terminalchat.InteractiveStatus, error) {
	s.mu.RLock()
	candidate := s.candidate
	s.mu.RUnlock()
	character, ok := candidate.Character(id)
	if !ok {
		return terminalchat.InteractiveStatus{}, fmt.Errorf("unknown character %q", strings.TrimSpace(id))
	}
	processor, err := s.factory(candidate)
	if err != nil {
		return terminalchat.InteractiveStatus{}, err
	}
	s.mu.Lock()
	s.processor = processor
	s.character = character
	status := s.statusLocked()
	s.mu.Unlock()
	return status, nil
}

func (s *interactiveTerminalSession) Status() terminalchat.InteractiveStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

func (s *interactiveTerminalSession) CandidateChanged() (bool, error) {
	s.mu.RLock()
	activeHash := s.candidate.Hash
	s.mu.RUnlock()
	return s.source.Changed(activeHash)
}

func (s *interactiveTerminalSession) statusLocked() terminalchat.InteractiveStatus {
	return terminalchat.InteractiveStatus{
		Provider:      s.provider,
		Memory:        true,
		CharacterID:   s.character.ID,
		CandidateHash: s.candidate.Hash,
	}
}

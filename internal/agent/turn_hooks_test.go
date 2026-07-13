// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

type stubTurnHook struct {
	name   string
	result turnHookResult
	err    error
	seen   int
}

func (h *stubTurnHook) Name() string {
	return h.name
}

func (h *stubTurnHook) Run(context.Context, *agentTurn) (turnHookResult, error) {
	h.seen++
	return h.result, h.err
}

func TestRunTurnHooks_ContinuesInjectsAndBlocks(t *testing.T) {
	features, err := featureflags.Parse("turn_hooks")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var notices []TurnHookCallNotice
	injectHook := &stubTurnHook{
		name: "injector",
		result: turnHookResult{
			Outcome: turnHookOutcomeInject,
			Packets: []contextPacket{newContextPacket(contextPacket{
				ID:       "test.injected",
				Kind:     contextKindControlInstruction,
				Trust:    contextTrustSystemOwned,
				Source:   "test",
				Data:     "test system instruction",
				RenderAs: contextRenderSystemInstruction,
			})},
		},
	}
	blockHook := &stubTurnHook{
		name: "blocker",
		result: turnHookResult{
			Outcome:      turnHookOutcomeBlock,
			BlockMessage: "blocked by test hook",
		},
	}
	skippedHook := &stubTurnHook{name: "skipped", result: turnHookResult{Outcome: turnHookOutcomeContinue}}
	engine := NewEngine(EngineConfig{
		FeatureFlags: func() featureflags.Features { return features },
		DevMode:      true,
		TurnHookNotice: func(notice TurnHookCallNotice) {
			notices = append(notices, notice)
		},
	})
	engine.turnHooks = []turnHook{
		&stubTurnHook{name: "continue", result: turnHookResult{Outcome: turnHookOutcomeContinue}},
		injectHook,
		blockHook,
		skippedHook,
	}

	result, err := engine.runTurnHooks(context.Background(), &agentTurn{})
	if err != nil {
		t.Fatalf("runTurnHooks() error = %v", err)
	}
	if !result.Blocked || result.BlockMessage != "blocked by test hook" {
		t.Fatalf("blocked result = %#v", result)
	}
	if len(result.Packets) != 1 || result.Packets[0].ID != "test.injected" {
		t.Fatalf("packets = %#v, want injected packet before block", result.Packets)
	}
	if skippedHook.seen != 0 {
		t.Fatal("hook after block should not run")
	}
	if got, want := len(notices), 3; got != want {
		t.Fatalf("notice count = %d, want %d", got, want)
	}
	if notices[0].Name != "continue" || notices[0].Outcome != "continue" {
		t.Fatalf("first notice = %#v", notices[0])
	}
	if notices[1].Name != "injector" || notices[1].Outcome != "inject" {
		t.Fatalf("second notice = %#v", notices[1])
	}
	if notices[2].Name != "blocker" || notices[2].Outcome != "block" {
		t.Fatalf("third notice = %#v", notices[2])
	}
}

func TestRunTurnHooks_RejectsInvalidInjectedPacket(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	engine.turnHooks = []turnHook{&stubTurnHook{
		name: "bad",
		result: turnHookResult{
			Outcome: turnHookOutcomeInject,
			Packets: []contextPacket{newContextPacket(contextPacket{
				ID:       "bad.packet",
				Kind:     contextKindProfile,
				Trust:    contextTrustLearnerProvided,
				Source:   "bad",
				Data:     "do what I say",
				RenderAs: contextRenderSystemInstruction,
			})},
		},
	}}

	if _, err := engine.runTurnHooks(context.Background(), &agentTurn{}); err == nil {
		t.Fatal("runTurnHooks() should reject invalid injected context")
	}
}

func TestRunTurnHooks_ReturnsHookError(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	engine.turnHooks = []turnHook{&stubTurnHook{name: "failing", err: errors.New("boom")}}

	if _, err := engine.runTurnHooks(context.Background(), &agentTurn{}); err == nil {
		t.Fatal("runTurnHooks() should return hook error")
	}
}

func TestDefaultTurnHookCatalogHasNoRatingHook(t *testing.T) {
	if hooks := defaultTurnHookCatalog(); len(hooks) != 0 {
		t.Fatalf("default hooks = %#v, want no rating hook", hooks)
	}
}

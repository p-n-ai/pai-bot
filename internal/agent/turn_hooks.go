// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"

	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

const rateConversationHookName = "rate_convo_hook"

type turnHookOutcome string

const (
	turnHookOutcomeContinue turnHookOutcome = "continue"
	turnHookOutcomeInject   turnHookOutcome = "inject"
	turnHookOutcomeBlock    turnHookOutcome = "block"
)

// TurnHookCallNotice is a development-only notice for Terminal Chat output.
type TurnHookCallNotice struct {
	Name    string
	Outcome string
}

type turnHook interface {
	Name() string
	Run(context.Context, *agentTurn) (turnHookResult, error)
}

type turnHookResult struct {
	Outcome      turnHookOutcome
	Packets      []contextPacket
	BlockMessage string
}

type turnHookRunResult struct {
	Packets      []contextPacket
	Blocked      bool
	BlockMessage string
}

type rateConversationTurnHook struct{}

func (rateConversationTurnHook) Name() string {
	return rateConversationHookName
}

func (rateConversationTurnHook) Run(_ context.Context, turn *agentTurn) (turnHookResult, error) {
	if turn == nil || !turn.RatingPromptRequested {
		return turnHookResult{Outcome: turnHookOutcomeContinue}, nil
	}
	return turnHookResult{
		Outcome: turnHookOutcomeInject,
		Packets: []contextPacket{ratingPromptPacket()},
	}, nil
}

func defaultTurnHookCatalog() []turnHook {
	return []turnHook{
		rateConversationTurnHook{},
	}
}

func (e *Engine) turnHooksEnabled() bool {
	return e.featureFlags().Enabled(featureflags.TurnHooks)
}

func (e *Engine) runTurnHooks(ctx context.Context, turn *agentTurn) (turnHookRunResult, error) {
	if turn == nil {
		return turnHookRunResult{}, fmt.Errorf("agent turn is required")
	}
	packets := append([]contextPacket(nil), turn.Packets...)
	hooks := e.turnHooks
	if hooks == nil {
		hooks = defaultTurnHookCatalog()
	}

	for _, hook := range hooks {
		if hook == nil {
			continue
		}
		turn.Packets = packets
		result, err := hook.Run(ctx, turn)
		if err != nil {
			return turnHookRunResult{}, fmt.Errorf("turn hook %s: %w", hook.Name(), err)
		}
		if result.Outcome == "" {
			result.Outcome = turnHookOutcomeContinue
		}
		if err := validateTurnHookResult(hook.Name(), result); err != nil {
			return turnHookRunResult{}, err
		}
		e.noticeTurnHookCall(hook.Name(), result.Outcome)

		switch result.Outcome {
		case turnHookOutcomeContinue:
			continue
		case turnHookOutcomeInject:
			packets = append(packets, result.Packets...)
			if err := validateContextPackets(packets); err != nil {
				return turnHookRunResult{}, fmt.Errorf("turn hook %s injected invalid context: %w", hook.Name(), err)
			}
		case turnHookOutcomeBlock:
			return turnHookRunResult{
				Packets:      packets,
				Blocked:      true,
				BlockMessage: result.BlockMessage,
			}, nil
		}
	}

	return turnHookRunResult{Packets: packets}, nil
}

func validateTurnHookResult(name string, result turnHookResult) error {
	switch result.Outcome {
	case turnHookOutcomeContinue:
		return nil
	case turnHookOutcomeInject:
		if len(result.Packets) == 0 {
			return fmt.Errorf("turn hook %s returned inject without packets", name)
		}
		return nil
	case turnHookOutcomeBlock:
		return nil
	default:
		return fmt.Errorf("turn hook %s returned unknown outcome %q", name, result.Outcome)
	}
}

func (e *Engine) noticeTurnHookCall(name string, outcome turnHookOutcome) {
	if !e.devMode || !e.turnHooksEnabled() || e.turnHookNotice == nil {
		return
	}
	e.turnHookNotice(TurnHookCallNotice{
		Name:    name,
		Outcome: string(outcome),
	})
}

func appendRatingPromptPacket(packets []contextPacket) []contextPacket {
	return append(packets, ratingPromptPacket())
}

func ratingPromptPacket() contextPacket {
	return newContextPacket(contextPacket{
		ID:       "rating.prompt",
		Kind:     contextKindControlInstruction,
		Trust:    contextTrustSystemOwned,
		Source:   "rating",
		Data:     ratingPromptInstruction,
		RenderAs: contextRenderSystemInstruction,
	})
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package conversationharness

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestRunQueuesMessagesBehindInFlightTurn(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	var once sync.Once

	done := make(chan struct{})
	var outcomes []Outcome
	var runErr error
	go func() {
		outcomes, runErr = Run(t.Context(), func(_ context.Context, message chat.InboundMessage) (string, error) {
			switch message.Text {
			case "first":
				close(firstStarted)
				<-releaseFirst
			case "second":
				once.Do(func() { close(secondStarted) })
			}
			return "reply:" + message.Text, nil
		}, []Turn{
			scriptedTurn("first", DeliveryWait),
			scriptedTurn("second", DeliveryQueue),
		})
		close(done)
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first turn did not start")
	}
	select {
	case <-secondStarted:
		t.Fatal("queued turn started before the in-flight turn completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not finish")
	}
	if runErr != nil {
		t.Fatalf("Run() error = %v", runErr)
	}
	assertOutcome(t, outcomes[0], StatusDelivered, "reply:first")
	assertOutcome(t, outcomes[1], StatusDelivered, "reply:second")
}

func TestRunInterruptsInFlightAndQueuedReplies(t *testing.T) {
	processed := make(chan string, 3)
	outcomes, err := Run(t.Context(), func(ctx context.Context, message chat.InboundMessage) (string, error) {
		processed <- message.Text
		if message.Text == "long request" {
			<-ctx.Done()
			return "", ctx.Err()
		}
		return "reply:" + message.Text, nil
	}, []Turn{
		scriptedTurn("long request", DeliveryWait),
		scriptedTurn("stale follow-up", DeliveryQueue),
		scriptedTurn("actually, this", DeliveryInterrupt),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertOutcome(t, outcomes[0], StatusInterrupted, "")
	assertOutcome(t, outcomes[1], StatusInterrupted, "")
	assertOutcome(t, outcomes[2], StatusDelivered, "reply:actually, this")
	close(processed)
	var got []string
	for text := range processed {
		got = append(got, text)
	}
	if len(got) != 2 || got[0] != "long request" || got[1] != "actually, this" {
		t.Fatalf("processed messages = %#v, want interrupted request then replacement", got)
	}
}

func TestRunContinuesQueuedTurnsAfterProcessorFailure(t *testing.T) {
	processErr := errors.New("processor failed")
	var processed []string

	outcomes, err := Run(t.Context(), func(_ context.Context, message chat.InboundMessage) (string, error) {
		processed = append(processed, message.Text)
		if message.Text == "first" {
			return "unsafe partial response", processErr
		}
		return "reply:" + message.Text, nil
	}, []Turn{
		scriptedTurn("first", DeliveryWait),
		scriptedTurn("second", DeliveryQueue),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(processed) != 2 || processed[0] != "first" || processed[1] != "second" {
		t.Fatalf("processed messages = %#v, want first then second", processed)
	}
	if outcomes[0].Status != StatusFailed || !errors.Is(outcomes[0].Err, processErr) {
		t.Fatalf("first outcome = %#v, want failed processor error", outcomes[0])
	}
	if outcomes[0].Response != "" {
		t.Fatalf("failed response was exposed: %q", outcomes[0].Response)
	}
	assertOutcome(t, outcomes[1], StatusDelivered, "reply:second")
}

func TestRunFailsQueuedTurnsWhenParentIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	processed := make(chan string, 2)
	runDone := make(chan struct {
		outcomes []Outcome
		err      error
	}, 1)

	go func() {
		outcomes, err := Run(ctx, func(ctx context.Context, message chat.InboundMessage) (string, error) {
			processed <- message.Text
			if message.Text == "first" {
				close(started)
				<-ctx.Done()
				return "", ctx.Err()
			}
			return "unexpected", nil
		}, []Turn{
			scriptedTurn("first", DeliveryWait),
			scriptedTurn("second", DeliveryQueue),
		})
		runDone <- struct {
			outcomes []Outcome
			err      error
		}{outcomes: outcomes, err: err}
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first turn did not start")
	}
	cancel()

	var result struct {
		outcomes []Outcome
		err      error
	}
	select {
	case result = <-runDone:
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after parent cancellation")
	}
	if !errors.Is(result.err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", result.err)
	}
	if len(result.outcomes) != 2 {
		t.Fatalf("outcomes = %d, want 2", len(result.outcomes))
	}
	for index, outcome := range result.outcomes {
		if outcome.Status != StatusFailed || !errors.Is(outcome.Err, context.Canceled) {
			t.Fatalf("outcome %d = %#v, want failed context cancellation", index+1, outcome)
		}
	}
	close(processed)
	var got []string
	for message := range processed {
		got = append(got, message)
	}
	if len(got) != 1 || got[0] != "first" {
		t.Fatalf("processed messages = %#v, queued turn should not start", got)
	}
}

func TestRunFailsTurnCanceledBeforeScheduledStart(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	calls := 0

	outcomes, err := Run(ctx, func(context.Context, chat.InboundMessage) (string, error) {
		calls++
		return "unexpected", nil
	}, []Turn{{
		Message: chat.InboundMessage{Text: "later"},
		After:   time.Hour,
		Timeout: time.Second,
	}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if calls != 0 {
		t.Fatalf("processor calls = %d, want 0", calls)
	}
	if len(outcomes) != 1 ||
		outcomes[0].Status != StatusFailed ||
		!errors.Is(outcomes[0].Err, context.Canceled) {
		t.Fatalf("outcomes = %#v, want one failed canceled turn", outcomes)
	}
}

func TestRunWithNoTurnsReturnsEmptyOutcomes(t *testing.T) {
	calls := 0
	outcomes, err := Run(t.Context(), func(context.Context, chat.InboundMessage) (string, error) {
		calls++
		return "unexpected", nil
	}, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("outcomes = %#v, want empty", outcomes)
	}
	if calls != 0 {
		t.Fatalf("processor calls = %d, want 0", calls)
	}
}

func TestRunHonorsDelayBeforeStartingQueuedTurn(t *testing.T) {
	const delay = 40 * time.Millisecond
	startedAt := time.Now()
	var secondStartedAfter time.Duration

	outcomes, err := Run(t.Context(), func(_ context.Context, message chat.InboundMessage) (string, error) {
		if message.Text == "second" {
			secondStartedAfter = time.Since(startedAt)
		}
		return "reply:" + message.Text, nil
	}, []Turn{
		scriptedTurn("first", DeliveryWait),
		{
			Message:  chat.InboundMessage{Text: "second"},
			Delivery: DeliveryQueue,
			After:    delay,
			Timeout:  time.Second,
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if secondStartedAfter < delay {
		t.Fatalf("queued turn started after %v, want at least %v", secondStartedAfter, delay)
	}
	assertOutcome(t, outcomes[0], StatusDelivered, "reply:first")
	assertOutcome(t, outcomes[1], StatusDelivered, "reply:second")
}

func TestRunHonorsDelayBeforeInterruptingTurn(t *testing.T) {
	const delay = 40 * time.Millisecond
	startedAt := time.Now()
	var replacementStartedAfter time.Duration

	outcomes, err := Run(t.Context(), func(ctx context.Context, message chat.InboundMessage) (string, error) {
		if message.Text == "first" {
			<-ctx.Done()
			return "", ctx.Err()
		}
		replacementStartedAfter = time.Since(startedAt)
		return "reply:" + message.Text, nil
	}, []Turn{
		scriptedTurn("first", DeliveryWait),
		{
			Message:  chat.InboundMessage{Text: "replacement"},
			Delivery: DeliveryInterrupt,
			After:    delay,
			Timeout:  time.Second,
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if replacementStartedAfter < delay {
		t.Fatalf("replacement started after %v, want at least %v", replacementStartedAfter, delay)
	}
	assertOutcome(t, outcomes[0], StatusInterrupted, "")
	assertOutcome(t, outcomes[1], StatusDelivered, "reply:replacement")
}

func TestRunFailsTurnWhenProcessorExceedsTimeout(t *testing.T) {
	outcomes, err := Run(t.Context(), func(ctx context.Context, _ chat.InboundMessage) (string, error) {
		<-ctx.Done()
		return "late fallback", nil
	}, []Turn{{
		Message:  chat.InboundMessage{Text: "slow"},
		Delivery: DeliveryWait,
		Timeout:  10 * time.Millisecond,
	}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcomes[0].Status != StatusFailed {
		t.Fatalf("status = %q, want %q", outcomes[0].Status, StatusFailed)
	}
	if !errors.Is(outcomes[0].Err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", outcomes[0].Err)
	}
	if outcomes[0].Response != "" {
		t.Fatalf("timed out response was exposed: %q", outcomes[0].Response)
	}
}

func TestRunWaitsForProcessorAfterParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	stopped := make(chan struct{})
	runDone := make(chan error, 1)
	go func() {
		_, err := Run(ctx, func(ctx context.Context, _ chat.InboundMessage) (string, error) {
			close(started)
			<-ctx.Done()
			close(stopped)
			return "", ctx.Err()
		}, []Turn{scriptedTurn("cancel me", DeliveryWait)})
		runDone <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}
	cancel()
	select {
	case err := <-runDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after parent cancellation")
	}
	select {
	case <-stopped:
	default:
		t.Fatal("Run() returned before the processor stopped")
	}
}

func TestRunRejectsInvalidTurnsBeforeProcessing(t *testing.T) {
	for _, test := range []struct {
		name string
		turn Turn
		want string
	}{
		{
			name: "unsupported delivery",
			turn: Turn{Delivery: "later", Timeout: time.Second},
			want: `unsupported delivery "later"`,
		},
		{
			name: "negative delay",
			turn: Turn{After: -time.Millisecond, Timeout: time.Second},
			want: "after must not be negative",
		},
		{
			name: "zero timeout",
			turn: Turn{Timeout: 0},
			want: "timeout must be positive",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			_, err := Run(t.Context(), func(context.Context, chat.InboundMessage) (string, error) {
				calls++
				return "", nil
			}, []Turn{test.turn})
			if err == nil {
				t.Fatal("Run() should reject an invalid turn")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Run() error = %q, want %q", err, test.want)
			}
			if calls != 0 {
				t.Fatalf("processor calls = %d, want 0", calls)
			}
		})
	}
}

func scriptedTurn(text string, delivery Delivery) Turn {
	return Turn{
		Message:  chat.InboundMessage{Text: text},
		Delivery: delivery,
		Timeout:  time.Second,
	}
}

func assertOutcome(t *testing.T, outcome Outcome, status Status, response string) {
	t.Helper()
	if outcome.Status != status || outcome.Response != response {
		t.Fatalf("outcome = %#v, want status %q response %q", outcome, status, response)
	}
}

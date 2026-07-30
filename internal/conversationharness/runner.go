// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package conversationharness coordinates scripted conversation timing for
// local quality evaluation.
package conversationharness

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

// Delivery describes how a learner message relates to unfinished tutor work.
type Delivery string

const (
	// DeliveryWait waits for all earlier messages before starting the turn.
	DeliveryWait Delivery = "wait"
	// DeliveryQueue preserves unfinished earlier messages and appends the turn.
	DeliveryQueue Delivery = "queue"
	// DeliveryInterrupt suppresses unfinished earlier replies and replaces them.
	DeliveryInterrupt Delivery = "interrupt"
)

// Status describes the caller-visible result of one scripted turn.
type Status string

const (
	// StatusDelivered means the processor completed and its response is visible.
	StatusDelivered Status = "delivered"
	// StatusInterrupted means a newer turn suppressed the response.
	StatusInterrupted Status = "interrupted"
	// StatusFailed means the processor or its turn context failed.
	StatusFailed Status = "failed"
)

// Processor handles one learner message. It must stop when ctx is canceled.
type Processor func(ctx context.Context, message chat.InboundMessage) (string, error)

// Turn is one timed learner message in a scripted conversation.
type Turn struct {
	Message  chat.InboundMessage
	Delivery Delivery
	After    time.Duration
	Timeout  time.Duration
}

// Outcome records the caller-visible result of one scripted turn.
type Outcome struct {
	Status   Status
	Response string
	Err      error
	Duration time.Duration
}

type processResult struct {
	response   string
	err        error
	contextErr error
	duration   time.Duration
}

type runningTurn struct {
	index       int
	cancel      context.CancelFunc
	done        <-chan processResult
	interrupted bool
}

type coordinator struct {
	ctx      context.Context
	process  Processor
	turns    []Turn
	outcomes []Outcome
	current  *runningTurn
	pending  []int
}

// Run processes turns according to their wait, queue, and interrupt semantics.
// It returns outcomes in fixture order and does not leave processor work running.
func Run(ctx context.Context, process Processor, turns []Turn) ([]Outcome, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if process == nil {
		return nil, errors.New("processor is required")
	}
	if err := validateTurns(turns); err != nil {
		return nil, err
	}

	c := coordinator{
		ctx:      ctx,
		process:  process,
		turns:    turns,
		outcomes: make([]Outcome, len(turns)),
	}
	for index, turn := range turns {
		if err := c.waitDelay(turn.After); err != nil {
			c.stop()
			return c.outcomes, err
		}
		switch normalizedDelivery(turn.Delivery) {
		case DeliveryWait:
			if err := c.drain(); err != nil {
				c.stop()
				return c.outcomes, err
			}
		case DeliveryInterrupt:
			if err := c.interrupt(); err != nil {
				c.stop()
				return c.outcomes, err
			}
		case DeliveryQueue:
		}
		c.submit(index)
	}
	if err := c.drain(); err != nil {
		c.stop()
		return c.outcomes, err
	}
	return c.outcomes, nil
}

func validateTurns(turns []Turn) error {
	for index, turn := range turns {
		switch normalizedDelivery(turn.Delivery) {
		case DeliveryWait, DeliveryQueue, DeliveryInterrupt:
		default:
			return fmt.Errorf("turn %d: unsupported delivery %q", index+1, turn.Delivery)
		}
		if turn.After < 0 {
			return fmt.Errorf("turn %d: after must not be negative", index+1)
		}
		if turn.Timeout <= 0 {
			return fmt.Errorf("turn %d: timeout must be positive", index+1)
		}
	}
	return nil
}

func normalizedDelivery(delivery Delivery) Delivery {
	if delivery == "" {
		return DeliveryWait
	}
	return delivery
}

func (c *coordinator) submit(index int) {
	if c.current != nil {
		c.pending = append(c.pending, index)
		return
	}
	c.start(index)
}

func (c *coordinator) start(index int) {
	turn := c.turns[index]
	turnCtx, cancel := context.WithTimeout(c.ctx, turn.Timeout)
	done := make(chan processResult, 1)
	startedAt := time.Now()
	go func() {
		response, err := c.process(turnCtx, turn.Message)
		done <- processResult{
			response:   response,
			err:        err,
			contextErr: turnCtx.Err(),
			duration:   time.Since(startedAt),
		}
	}()
	c.current = &runningTurn{
		index:  index,
		cancel: cancel,
		done:   done,
	}
}

func (c *coordinator) waitDelay(delay time.Duration) error {
	if delay == 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		if c.current == nil {
			select {
			case <-timer.C:
				return nil
			case <-c.ctx.Done():
				return c.ctx.Err()
			}
		}
		select {
		case result := <-c.current.done:
			c.finishCurrent(result)
		case <-timer.C:
			return nil
		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

func (c *coordinator) drain() error {
	for c.current != nil {
		select {
		case result := <-c.current.done:
			c.finishCurrent(result)
		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
	return nil
}

func (c *coordinator) interrupt() error {
	for _, index := range c.pending {
		c.outcomes[index] = Outcome{Status: StatusInterrupted}
	}
	c.pending = nil
	if c.current == nil {
		return nil
	}
	c.current.interrupted = true
	c.current.cancel()
	select {
	case result := <-c.current.done:
		c.finishCurrent(result)
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *coordinator) finishCurrent(result processResult) {
	current := c.current
	current.cancel()
	outcome := Outcome{
		Response: result.response,
		Err:      result.err,
		Duration: result.duration,
	}
	switch {
	case current.interrupted:
		outcome.Status = StatusInterrupted
		outcome.Response = ""
		outcome.Err = nil
	case result.contextErr != nil:
		outcome.Status = StatusFailed
		outcome.Response = ""
		outcome.Err = result.contextErr
	case result.err != nil:
		outcome.Status = StatusFailed
		outcome.Response = ""
	default:
		outcome.Status = StatusDelivered
	}
	c.outcomes[current.index] = outcome
	c.current = nil
	if len(c.pending) == 0 {
		return
	}
	next := c.pending[0]
	c.pending = c.pending[1:]
	c.start(next)
}

func (c *coordinator) stop() {
	c.pending = nil
	if c.current != nil {
		c.current.cancel()
		result := <-c.current.done
		c.finishCurrent(result)
	}
	for index, outcome := range c.outcomes {
		if outcome.Status == "" {
			c.outcomes[index] = Outcome{Status: StatusFailed, Err: c.ctx.Err()}
		}
	}
}

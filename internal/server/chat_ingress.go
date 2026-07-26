// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

const chatIngressDedupeTTL = 10 * time.Minute

// ChatIngress serializes inbound adapter deliveries and suppresses webhook
// retries without conflating delivery IDs from different channels.
type ChatIngress struct {
	queue       chan chat.InboundMessage
	slots       chan struct{}
	process     func(context.Context, chat.InboundMessage)
	workerCount int
	maxSeen     int
}

func NewChatIngress(
	capacity int,
	process func(context.Context, chat.InboundMessage),
) (*ChatIngress, error) {
	if capacity <= 0 {
		return nil, errors.New("chat ingress capacity must be positive")
	}
	if process == nil {
		return nil, errors.New("chat ingress processor is required")
	}
	return &ChatIngress{
		queue:       make(chan chat.InboundMessage, capacity),
		slots:       make(chan struct{}, capacity),
		process:     process,
		workerCount: min(capacity, 16),
		maxSeen:     max(capacity*16, 256),
	}, nil
}

func newChatIngress(
	capacity int,
	process func(context.Context, chat.InboundMessage),
) (*ChatIngress, error) {
	return NewChatIngress(capacity, process)
}

func (i *ChatIngress) Enqueue(ctx context.Context, message chat.InboundMessage) error {
	select {
	case i.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case i.queue <- message:
		return nil
	case <-ctx.Done():
		<-i.slots
		return ctx.Err()
	}
}

func (i *ChatIngress) Run(ctx context.Context) {
	seen := make(map[string]time.Time)
	active := make(map[string]bool)
	pending := make(map[string][]chat.InboundMessage)
	jobs := make(chan chatIngressJob, cap(i.queue))
	completed := make(chan string, i.workerCount)
	var workers sync.WaitGroup
	for range i.workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			i.runWorker(ctx, jobs, completed)
		}()
	}
	defer func() {
		close(jobs)
		workers.Wait()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case key := <-completed:
			<-i.slots
			queued := pending[key]
			if len(queued) == 0 {
				delete(active, key)
				delete(pending, key)
				continue
			}
			next := queued[0]
			pending[key] = queued[1:]
			if !sendChatIngressJob(ctx, jobs, chatIngressJob{key: key, message: next}) {
				return
			}
		case message := <-i.queue:
			if duplicateDelivery(seen, message, time.Now(), i.maxSeen) {
				<-i.slots
				continue
			}
			key := chatIngressRoute(message)
			if active[key] {
				pending[key] = append(pending[key], message)
				continue
			}
			active[key] = true
			if !sendChatIngressJob(ctx, jobs, chatIngressJob{key: key, message: message}) {
				return
			}
		}
	}
}

type chatIngressJob struct {
	key     string
	message chat.InboundMessage
}

func sendChatIngressJob(ctx context.Context, jobs chan<- chatIngressJob, job chatIngressJob) bool {
	select {
	case jobs <- job:
		return true
	case <-ctx.Done():
		return false
	}
}

func (i *ChatIngress) runWorker(
	ctx context.Context,
	jobs <-chan chatIngressJob,
	completed chan<- string,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			i.process(ctx, job.message)
			select {
			case completed <- job.key:
			case <-ctx.Done():
				return
			}
		}
	}
}

func chatIngressRoute(message chat.InboundMessage) string {
	return message.Channel + "\x00" + message.DestinationID()
}

func duplicateDelivery(
	seen map[string]time.Time,
	message chat.InboundMessage,
	now time.Time,
	maxSeen int,
) bool {
	if message.DeliveryID == "" {
		return false
	}
	key := message.Channel + "\x00" + message.DeliveryID
	if expiresAt, ok := seen[key]; ok && expiresAt.After(now) {
		return true
	}
	if len(seen) >= maxSeen {
		for delivery, expiresAt := range seen {
			if !expiresAt.After(now) {
				delete(seen, delivery)
			}
		}
		if len(seen) >= maxSeen {
			var oldestKey string
			var oldestExpiry time.Time
			for delivery, expiresAt := range seen {
				if oldestKey == "" || expiresAt.Before(oldestExpiry) {
					oldestKey = delivery
					oldestExpiry = expiresAt
				}
			}
			delete(seen, oldestKey)
		}
	}
	seen[key] = now.Add(chatIngressDedupeTTL)
	return false
}

package piai

import (
	"fmt"
	"iter"
	"sync"
)

// EventType tags AssistantMessageEvent. Streams emit start first, then
// per-block start/delta/end triples, and terminate with exactly one done or
// error event.
type EventType string

const (
	EventStart         EventType = "start"
	EventTextStart     EventType = "text_start"
	EventTextDelta     EventType = "text_delta"
	EventTextEnd       EventType = "text_end"
	EventThinkingStart EventType = "thinking_start"
	EventThinkingDelta EventType = "thinking_delta"
	EventThinkingEnd   EventType = "thinking_end"
	EventToolCallStart EventType = "toolcall_start"
	EventToolCallDelta EventType = "toolcall_delta"
	EventToolCallEnd   EventType = "toolcall_end"
	EventDone          EventType = "done"
	EventError         EventType = "error"
)

// AssistantMessageEvent is one streaming event. Which fields are set depends
// on Type: deltas carry Delta, *_end carries Content or ToolCall, done/error
// carry Reason and the final Message. Partial is the message assembled so far.
type AssistantMessageEvent struct {
	Type         EventType
	ContentIndex int
	Delta        string
	Content      string
	ToolCall     *ToolCall
	Partial      *AssistantMessage
	Reason       StopReason
	Message      *AssistantMessage // final message on done/error
}

// EventStream carries AssistantMessageEvents from a provider to a consumer.
// The queue is unbounded so Push never blocks and Result can be awaited
// without draining events. A done or error event completes the stream;
// further pushes are dropped.
type EventStream struct {
	mu     sync.Mutex
	cond   *sync.Cond
	queue  []AssistantMessageEvent
	closed bool
	final  AssistantMessage
	done   chan struct{}
}

func NewEventStream() *EventStream {
	s := &EventStream{done: make(chan struct{})}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Push appends an event. A done or error event must carry Message and
// completes the stream.
func (s *EventStream) Push(ev AssistantMessageEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.queue = append(s.queue, ev)
	if ev.Type == EventDone || ev.Type == EventError {
		s.closed = true
		s.final = *ev.Message
		close(s.done)
	}
	s.cond.Broadcast()
}

// Events iterates all events in order, ending after the terminal event.
func (s *EventStream) Events() iter.Seq[AssistantMessageEvent] {
	return func(yield func(AssistantMessageEvent) bool) {
		i := 0
		for {
			s.mu.Lock()
			for i >= len(s.queue) && !s.closed {
				s.cond.Wait()
			}
			if i >= len(s.queue) {
				s.mu.Unlock()
				return
			}
			ev := s.queue[i]
			i++
			s.mu.Unlock()
			if !yield(ev) {
				return
			}
		}
	}
}

// Result blocks until the stream completes and returns the final
// AssistantMessage. The error is non-nil when the stream terminated with
// stopReason "error" or "aborted"; the message still carries the details.
func (s *EventStream) Result() (AssistantMessage, error) {
	<-s.done
	if s.final.StopReason == StopReasonError || s.final.StopReason == StopReasonAborted {
		return s.final, fmt.Errorf("piai: %s: %s", s.final.StopReason, s.final.ErrorMessage)
	}
	return s.final, nil
}

// endWithError terminates the stream with an error-shaped AssistantMessage.
func (s *EventStream) endWithError(msg AssistantMessage) {
	reason := msg.StopReason
	if reason != StopReasonAborted {
		reason = StopReasonError
		msg.StopReason = StopReasonError
	}
	s.Push(AssistantMessageEvent{Type: EventError, Reason: reason, Message: &msg})
}

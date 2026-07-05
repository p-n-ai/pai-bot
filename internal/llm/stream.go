package llm

import (
	"fmt"
	"iter"
	"sync"
)

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

type AssistantMessageEvent struct {
	Type         EventType
	ContentIndex int
	Delta        string
	Content      string
	ToolCall     *ToolCall
	Partial      *AssistantMessage
	Reason       StopReason
	Message      *AssistantMessage
}

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

type StreamError struct {
	Reason  StopReason
	Message string
}

func (e *StreamError) Error() string { return fmt.Sprintf("llm: %s: %s", e.Reason, e.Message) }

func (s *EventStream) Result() (AssistantMessage, error) {
	<-s.done
	if s.final.StopReason == StopReasonError || s.final.StopReason == StopReasonAborted {
		return s.final, &StreamError{Reason: s.final.StopReason, Message: s.final.ErrorMessage}
	}
	return s.final, nil
}

func (s *EventStream) endWithError(msg AssistantMessage) {
	reason := msg.StopReason
	if reason != StopReasonAborted {
		reason = StopReasonError
		msg.StopReason = StopReasonError
	}
	s.Push(AssistantMessageEvent{Type: EventError, Reason: reason, Message: &msg})
}

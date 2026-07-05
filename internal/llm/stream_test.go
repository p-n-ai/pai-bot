package llm_test

import (
	"errors"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func collectEvents(s *llm.EventStream) []llm.AssistantMessageEvent {
	var events []llm.AssistantMessageEvent
	for ev := range s.Events() {
		events = append(events, ev)
	}
	return events
}

func eventTypes(events []llm.AssistantMessageEvent) []llm.EventType {
	types := make([]llm.EventType, len(events))
	for i, ev := range events {
		types[i] = ev.Type
	}
	return types
}

func equalTypes(a []llm.EventType, b ...llm.EventType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsType(events []llm.AssistantMessageEvent, typ llm.EventType) bool {
	for _, ev := range events {
		if ev.Type == typ {
			return true
		}
	}
	return false
}

func TestEventStreamDeliversEventsInOrder(t *testing.T) {
	s := llm.NewEventStream()
	final := llm.FauxAssistantText("hi")
	s.Push(llm.AssistantMessageEvent{Type: llm.EventStart, Partial: &llm.AssistantMessage{}})
	s.Push(llm.AssistantMessageEvent{Type: llm.EventTextDelta, Delta: "hi"})
	s.Push(llm.AssistantMessageEvent{Type: llm.EventDone, Reason: llm.StopReasonStop, Message: &final})

	events := collectEvents(s)
	if !equalTypes(eventTypes(events), llm.EventStart, llm.EventTextDelta, llm.EventDone) {
		t.Fatalf("unexpected event order: %v", eventTypes(events))
	}
}

func TestEventStreamResultWithoutDraining(t *testing.T) {
	s := llm.NewEventStream()
	final := llm.FauxAssistantText("done")
	go func() {
		s.Push(llm.AssistantMessageEvent{Type: llm.EventStart, Partial: &llm.AssistantMessage{}})
		s.Push(llm.AssistantMessageEvent{Type: llm.EventTextDelta, Delta: "done"})
		s.Push(llm.AssistantMessageEvent{Type: llm.EventDone, Reason: llm.StopReasonStop, Message: &final})
	}()

	msg, err := s.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.StopReason != llm.StopReasonStop {
		t.Fatalf("stopReason = %q, want stop", msg.StopReason)
	}

	if got := eventTypes(collectEvents(s)); !equalTypes(got, llm.EventStart, llm.EventTextDelta, llm.EventDone) {
		t.Fatalf("late consumer events: %v", got)
	}
}

func TestEventStreamErrorTerminal(t *testing.T) {
	s := llm.NewEventStream()
	errMsg := llm.FauxAssistantText("")
	errMsg.StopReason = llm.StopReasonError
	errMsg.ErrorMessage = "upstream failed"
	s.Push(llm.AssistantMessageEvent{Type: llm.EventError, Reason: llm.StopReasonError, Message: &errMsg})

	msg, err := s.Result()
	var streamErr *llm.StreamError
	if !errors.As(err, &streamErr) {
		t.Fatalf("expected *StreamError from Result, got %v", err)
	}
	if streamErr.Reason != llm.StopReasonError || streamErr.Message != "upstream failed" {
		t.Fatalf("stream error = %+v", streamErr)
	}
	if msg.StopReason != llm.StopReasonError || msg.ErrorMessage != "upstream failed" {
		t.Fatalf("final message = %+v", msg)
	}
}

func TestEventStreamIgnoresPushesAfterTerminal(t *testing.T) {
	s := llm.NewEventStream()
	final := llm.FauxAssistantText("hi")
	s.Push(llm.AssistantMessageEvent{Type: llm.EventDone, Reason: llm.StopReasonStop, Message: &final})
	s.Push(llm.AssistantMessageEvent{Type: llm.EventTextDelta, Delta: "late"})

	if events := collectEvents(s); len(events) != 1 || events[0].Type != llm.EventDone {
		t.Fatalf("expected single done event, got %v", eventTypes(events))
	}
}

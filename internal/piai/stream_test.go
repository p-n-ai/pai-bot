package piai_test

import (
	"errors"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/piai"
)

func collectEvents(s *piai.EventStream) []piai.AssistantMessageEvent {
	var events []piai.AssistantMessageEvent
	for ev := range s.Events() {
		events = append(events, ev)
	}
	return events
}

func eventTypes(events []piai.AssistantMessageEvent) []piai.EventType {
	types := make([]piai.EventType, len(events))
	for i, ev := range events {
		types[i] = ev.Type
	}
	return types
}

func equalTypes(a []piai.EventType, b ...piai.EventType) bool {
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

func containsType(events []piai.AssistantMessageEvent, typ piai.EventType) bool {
	for _, ev := range events {
		if ev.Type == typ {
			return true
		}
	}
	return false
}

func TestEventStreamDeliversEventsInOrder(t *testing.T) {
	s := piai.NewEventStream()
	final := piai.FauxAssistantText("hi")
	s.Push(piai.AssistantMessageEvent{Type: piai.EventStart, Partial: &piai.AssistantMessage{}})
	s.Push(piai.AssistantMessageEvent{Type: piai.EventTextDelta, Delta: "hi"})
	s.Push(piai.AssistantMessageEvent{Type: piai.EventDone, Reason: piai.StopReasonStop, Message: &final})

	events := collectEvents(s)
	if !equalTypes(eventTypes(events), piai.EventStart, piai.EventTextDelta, piai.EventDone) {
		t.Fatalf("unexpected event order: %v", eventTypes(events))
	}
}

func TestEventStreamResultWithoutDraining(t *testing.T) {
	s := piai.NewEventStream()
	final := piai.FauxAssistantText("done")
	go func() {
		s.Push(piai.AssistantMessageEvent{Type: piai.EventStart, Partial: &piai.AssistantMessage{}})
		s.Push(piai.AssistantMessageEvent{Type: piai.EventTextDelta, Delta: "done"})
		s.Push(piai.AssistantMessageEvent{Type: piai.EventDone, Reason: piai.StopReasonStop, Message: &final})
	}()

	// No consumer drains events; Result must still return.
	msg, err := s.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.StopReason != piai.StopReasonStop {
		t.Fatalf("stopReason = %q, want stop", msg.StopReason)
	}

	// A late consumer still sees every event.
	if got := eventTypes(collectEvents(s)); !equalTypes(got, piai.EventStart, piai.EventTextDelta, piai.EventDone) {
		t.Fatalf("late consumer events: %v", got)
	}
}

func TestEventStreamErrorTerminal(t *testing.T) {
	s := piai.NewEventStream()
	errMsg := piai.FauxAssistantText("")
	errMsg.StopReason = piai.StopReasonError
	errMsg.ErrorMessage = "upstream failed"
	s.Push(piai.AssistantMessageEvent{Type: piai.EventError, Reason: piai.StopReasonError, Message: &errMsg})

	msg, err := s.Result()
	var streamErr *piai.StreamError
	if !errors.As(err, &streamErr) {
		t.Fatalf("expected *StreamError from Result, got %v", err)
	}
	if streamErr.Reason != piai.StopReasonError || streamErr.Message != "upstream failed" {
		t.Fatalf("stream error = %+v", streamErr)
	}
	if msg.StopReason != piai.StopReasonError || msg.ErrorMessage != "upstream failed" {
		t.Fatalf("final message = %+v", msg)
	}
}

func TestEventStreamIgnoresPushesAfterTerminal(t *testing.T) {
	s := piai.NewEventStream()
	final := piai.FauxAssistantText("hi")
	s.Push(piai.AssistantMessageEvent{Type: piai.EventDone, Reason: piai.StopReasonStop, Message: &final})
	s.Push(piai.AssistantMessageEvent{Type: piai.EventTextDelta, Delta: "late"})

	if events := collectEvents(s); len(events) != 1 || events[0].Type != piai.EventDone {
		t.Fatalf("expected single done event, got %v", eventTypes(events))
	}
}

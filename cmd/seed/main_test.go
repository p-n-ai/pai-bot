package main

import (
	"testing"
	"time"
)

func TestParseBudgetWindowDefaultsToCurrentUTCMonth(t *testing.T) {
	start, end, err := parseBudgetWindow("", "")
	if err != nil {
		t.Fatalf("parseBudgetWindow() error = %v", err)
	}
	if !end.After(start) {
		t.Fatalf("window = %v -> %v, want end after start", start, end)
	}
	if start.Location() != time.UTC || end.Location() != time.UTC {
		t.Fatalf("window locations = %v / %v, want UTC", start.Location(), end.Location())
	}
	if start.Day() != 1 || start.Hour() != 0 || start.Minute() != 0 {
		t.Fatalf("start = %v, want first day of month at 00:00 UTC", start)
	}
}

func TestParseBudgetWindowAcceptsExplicitRange(t *testing.T) {
	start, end, err := parseBudgetWindow("2026-04-01T00:00:00Z", "2026-05-01T00:00:00Z")
	if err != nil {
		t.Fatalf("parseBudgetWindow() error = %v", err)
	}
	if got := start.Format(time.RFC3339); got != "2026-04-01T00:00:00Z" {
		t.Fatalf("start = %q, want explicit value", got)
	}
	if got := end.Format(time.RFC3339); got != "2026-05-01T00:00:00Z" {
		t.Fatalf("end = %q, want explicit value", got)
	}
}

func TestParseBudgetWindowRejectsInvalidRange(t *testing.T) {
	if _, _, err := parseBudgetWindow("2026-05-01T00:00:00Z", "2026-04-01T00:00:00Z"); err == nil {
		t.Fatal("parseBudgetWindow() error = nil, want invalid range error")
	}
}

package adminapi

import (
	"testing"
	"time"
)

func TestFormFromClassID(t *testing.T) {
	tests := []struct {
		classID string
		want    string
	}{
		{classID: "form-1-algebra", want: "Form 1"},
		{classID: "form-2-algebra", want: "Form 2"},
		{classID: "form-3-algebra", want: "Form 3"},
		{classID: "all-students", want: ""},
	}

	for _, tt := range tests {
		if got := formFromClassID(tt.classID); got != tt.want {
			t.Fatalf("formFromClassID(%q) = %q, want %q", tt.classID, got, tt.want)
		}
	}
}

func TestComputeStreakSummary(t *testing.T) {
	base := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	dates := []time.Time{
		base,
		base.Add(-24 * time.Hour),
		base.Add(-48 * time.Hour),
		base.Add(-24 * time.Hour * 5),
		base.Add(-24 * time.Hour * 6),
	}

	current, longest := computeStreakSummary(dates)
	if current != 3 {
		t.Fatalf("current = %d, want 3", current)
	}
	if longest != 3 {
		t.Fatalf("longest = %d, want 3", longest)
	}
}

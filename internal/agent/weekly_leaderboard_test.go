package agent

import (
	"testing"
	"time"
)

func TestTimeUntilNextWeekday(t *testing.T) {
	d := timeUntilNextWeekday(time.Monday, 8, 0)

	if d <= 0 {
		t.Fatalf("timeUntilNextWeekday returned %v, want positive duration", d)
	}
	if d > 7*24*time.Hour {
		t.Fatalf("timeUntilNextWeekday returned %v, want <= 7 days", d)
	}
}

func TestTimeUntilNextWeekday_AlwaysInFuture(t *testing.T) {
	// Regardless of when this test runs, the result should always be in the future.
	for _, day := range []time.Weekday{time.Monday, time.Wednesday, time.Friday, time.Sunday} {
		d := timeUntilNextWeekday(day, 12, 0)
		if d <= 0 {
			t.Fatalf("timeUntilNextWeekday(%v, 12, 0) = %v, want positive", day, d)
		}
	}
}

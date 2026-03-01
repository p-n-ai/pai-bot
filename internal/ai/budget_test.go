package ai

import (
	"testing"
)

func TestInMemoryBudget_NoBudgetSet(t *testing.T) {
	b := NewInMemoryBudget()

	ok, err := b.Check("tenant1", "user1")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !ok {
		t.Error("Check() = false, want true (no budget means unlimited)")
	}
}

func TestInMemoryBudget_WithinBudget(t *testing.T) {
	b := NewInMemoryBudget()
	b.SetBudget("tenant1", "user1", 1000)

	if err := b.Record("tenant1", "user1", 500); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	ok, err := b.Check("tenant1", "user1")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !ok {
		t.Error("Check() = false, want true (500 < 1000)")
	}
}

func TestInMemoryBudget_OverBudget(t *testing.T) {
	b := NewInMemoryBudget()
	b.SetBudget("tenant1", "user1", 100)

	if err := b.Record("tenant1", "user1", 150); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	ok, err := b.Check("tenant1", "user1")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if ok {
		t.Error("Check() = true, want false (150 >= 100)")
	}
}

func TestInMemoryBudget_ExactBudget(t *testing.T) {
	b := NewInMemoryBudget()
	b.SetBudget("tenant1", "user1", 100)

	if err := b.Record("tenant1", "user1", 100); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	ok, err := b.Check("tenant1", "user1")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if ok {
		t.Error("Check() = true, want false (100 >= 100, budget exhausted)")
	}
}

func TestInMemoryBudget_MultipleRecords(t *testing.T) {
	b := NewInMemoryBudget()
	b.SetBudget("tenant1", "user1", 1000)

	records := []int{100, 200, 300}
	for _, tokens := range records {
		if err := b.Record("tenant1", "user1", tokens); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	used, budget, err := b.Usage("tenant1", "user1")
	if err != nil {
		t.Fatalf("Usage() error = %v", err)
	}
	if used != 600 {
		t.Errorf("used = %d, want 600", used)
	}
	if budget != 1000 {
		t.Errorf("budget = %d, want 1000", budget)
	}
}

func TestInMemoryBudget_NegativeTokens(t *testing.T) {
	b := NewInMemoryBudget()

	err := b.Record("tenant1", "user1", -10)
	if err == nil {
		t.Fatal("Record() should return error for negative tokens")
	}
}

func TestInMemoryBudget_IsolatedUsers(t *testing.T) {
	b := NewInMemoryBudget()
	b.SetBudget("tenant1", "user1", 100)
	b.SetBudget("tenant1", "user2", 200)

	b.Record("tenant1", "user1", 90)
	b.Record("tenant1", "user2", 50)

	ok1, _ := b.Check("tenant1", "user1")
	ok2, _ := b.Check("tenant1", "user2")

	if !ok1 {
		t.Error("user1 should be within budget (90 < 100)")
	}
	if !ok2 {
		t.Error("user2 should be within budget (50 < 200)")
	}
}

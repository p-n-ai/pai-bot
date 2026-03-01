package ai

import (
	"fmt"
	"sync"
)

// BudgetChecker checks and records token usage against budgets.
type BudgetChecker interface {
	// Check returns true if the tenant/user has budget remaining.
	Check(tenantID, userID string) (bool, error)
	// Record records token usage for a tenant/user.
	Record(tenantID, userID string, tokens int) error
	// Usage returns current usage for a tenant/user.
	Usage(tenantID, userID string) (used int64, budget int64, err error)
}

// InMemoryBudget is a simple in-memory budget tracker for development.
// Production will use Dragonfly for real-time tracking with periodic PostgreSQL sync.
type InMemoryBudget struct {
	mu      sync.RWMutex
	budgets map[string]int64 // key -> budget limit
	usage   map[string]int64 // key -> tokens used
}

// NewInMemoryBudget creates a new in-memory budget tracker.
func NewInMemoryBudget() *InMemoryBudget {
	return &InMemoryBudget{
		budgets: make(map[string]int64),
		usage:   make(map[string]int64),
	}
}

// SetBudget sets the token budget for a tenant/user.
func (b *InMemoryBudget) SetBudget(tenantID, userID string, tokens int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.budgets[budgetKey(tenantID, userID)] = tokens
}

func (b *InMemoryBudget) Check(tenantID, userID string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	key := budgetKey(tenantID, userID)
	budget, hasBudget := b.budgets[key]
	if !hasBudget {
		// No budget set means unlimited.
		return true, nil
	}

	used := b.usage[key]
	return used < budget, nil
}

func (b *InMemoryBudget) Record(tenantID, userID string, tokens int) error {
	if tokens < 0 {
		return fmt.Errorf("tokens must be non-negative, got %d", tokens)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	key := budgetKey(tenantID, userID)
	b.usage[key] += int64(tokens)
	return nil
}

func (b *InMemoryBudget) Usage(tenantID, userID string) (int64, int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	key := budgetKey(tenantID, userID)
	return b.usage[key], b.budgets[key], nil
}

func budgetKey(tenantID, userID string) string {
	return tenantID + ":" + userID
}

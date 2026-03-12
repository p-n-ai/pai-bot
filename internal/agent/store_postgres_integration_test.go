//go:build integration
// +build integration

package agent

import (
	"context"
	"testing"
)

func TestPostgresStore_ResetProfileClearsFormAndLanguage(t *testing.T) {
	ctx := context.Background()
	pool, _ := startSchedulerPostgres(t, ctx)

	store, err := NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}

	userID := "store-reset-profile-user"
	if err := store.SetUserForm(userID, "2"); err != nil {
		t.Fatalf("SetUserForm() error = %v", err)
	}
	if err := store.SetUserPreferredLanguage(userID, "en"); err != nil {
		t.Fatalf("SetUserPreferredLanguage() error = %v", err)
	}

	if err := store.SetUserForm(userID, ""); err != nil {
		t.Fatalf("SetUserForm(clear) error = %v", err)
	}
	if err := store.SetUserPreferredLanguage(userID, ""); err != nil {
		t.Fatalf("SetUserPreferredLanguage(clear) error = %v", err)
	}

	if form, ok := store.GetUserForm(userID); ok || form != "" {
		t.Fatalf("GetUserForm() = %q, %v, want empty, false", form, ok)
	}
	if lang, ok := store.GetUserPreferredLanguage(userID); ok || lang != "" {
		t.Fatalf("GetUserPreferredLanguage() = %q, %v, want empty, false", lang, ok)
	}
}

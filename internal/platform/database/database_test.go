package database

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid", "postgres://user:pass@localhost:5432/db", false},
		{"empty", "", true},
		{"invalid", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew_UnreachableHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping unreachable host test in short mode")
	}

	ctx := t.Context()
	_, err := New(ctx, "postgres://user:pass@localhost:59999/nonexistent?connect_timeout=1", 5, 1)
	if err == nil {
		t.Fatal("New() should return error for unreachable host")
	}
}

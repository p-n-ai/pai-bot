package cache

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid-redis", "redis://localhost:6379", false},
		{"valid-with-db", "redis://localhost:6379/0", false},
		{"empty", "", true},
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
	_, err := New(ctx, "redis://localhost:59999")
	if err == nil {
		t.Fatal("New() should return error for unreachable host")
	}
}

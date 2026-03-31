package group

import (
	"strings"
	"testing"
)

const validCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func TestGenerateJoinCode_Length(t *testing.T) {
	code, err := GenerateJoinCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Errorf("len = %d, want 6", len(code))
	}
}

func TestGenerateJoinCode_Charset(t *testing.T) {
	for i := 0; i < 100; i++ {
		code, _ := GenerateJoinCode()
		for _, c := range code {
			if !strings.ContainsRune(validCharset, c) {
				t.Errorf("invalid char %c in code %s", c, code)
			}
		}
	}
}

func TestGenerateJoinCode_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		code, _ := GenerateJoinCode()
		if seen[code] {
			t.Fatalf("duplicate: %s", code)
		}
		seen[code] = true
	}
}

func TestNormalizeJoinCode(t *testing.T) {
	tests := []struct{ in, want string }{
		{"abc123", "ABC123"},
		{"  XYZ  ", "XYZ"},
	}
	for _, tt := range tests {
		got := NormalizeJoinCode(tt.in)
		if got != tt.want {
			t.Errorf("NormalizeJoinCode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

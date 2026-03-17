package auth

import "testing"

func TestNormalizeIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim and lowercase email", input: "  Teacher@Example.COM ", want: "teacher@example.com"},
		{name: "preserve internal spacing", input: "Parent One@example.com", want: "parent one@example.com"},
		{name: "empty", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeIdentifier(tt.input); got != tt.want {
				t.Fatalf("NormalizeIdentifier() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHashAndComparePassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if err := ComparePassword(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("ComparePassword() error = %v", err)
	}

	if err := ComparePassword(hash, "wrong password"); err == nil {
		t.Fatal("ComparePassword() should fail for mismatched password")
	}
}

func TestHashPasswordRejectsEmpty(t *testing.T) {
	if _, err := HashPassword("   "); err == nil {
		t.Fatal("HashPassword() should reject empty password")
	}
}

func TestHashOpaqueToken(t *testing.T) {
	const token = "invite-token-123"

	first := HashOpaqueToken(token)
	second := HashOpaqueToken(token)

	if first == "" {
		t.Fatal("HashOpaqueToken() should return a hash")
	}
	if first != second {
		t.Fatalf("HashOpaqueToken() should be stable, got %q and %q", first, second)
	}
}

package focusedpage

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestServiceCreateIsIdempotentAndRedeemsUntilExactExpiry(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	service, err := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), clock)
	if err != nil {
		t.Fatal(err)
	}
	input := CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "turn-1", RecipientName: "Aina", Message: " Keep going. "}
	first, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.URL != second.URL || first.PublicID != second.PublicID {
		t.Fatal("retry changed focused page artifact")
	}
	parsed, _ := url.Parse(first.URL)
	if parsed.Fragment == "" {
		t.Fatal("capability fragment is empty")
	}
	page, err := service.Redeem(context.Background(), first.PublicID, parsed.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	if page.Message != "Keep going." || page.RecipientName != "Aina" {
		t.Fatal("redeemed page content did not match the created page")
	}
	if string(page.TokenHash) == parsed.Fragment {
		t.Fatal("store retained raw capability")
	}

	now = first.ExpiresAt
	if _, err := service.Redeem(context.Background(), first.PublicID, parsed.Fragment); !errors.Is(err, ErrExpired) {
		t.Fatalf("expiry error = %v", err)
	}
}

func TestMemoryStoreRejectsIdempotencyCollisionAcrossOwnerOrConversation(t *testing.T) {
	service, err := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	base := CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "turn-1", Message: "Report"}
	if _, err := service.Create(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	wrongOwner := base
	wrongOwner.OwnerUserID = "user-2"
	if _, err := service.Create(context.Background(), wrongOwner); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong owner error = %v", err)
	}
	wrongConversation := base
	wrongConversation.ConversationID = "conv-2"
	if _, err := service.Create(context.Background(), wrongConversation); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong conversation error = %v", err)
	}
}

func TestServiceWrongTokenRevocationAndOwnerIsolation(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	service, _ := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return now })
	artifact, err := service.Create(context.Background(), CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "turn-1", Message: "Report"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Redeem(context.Background(), artifact.PublicID, "wrong"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong token error = %v", err)
	}
	if err := service.Revoke(context.Background(), artifact.PublicID, "tenant-2", "user-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong tenant revoke = %v", err)
	}
	if err := service.Revoke(context.Background(), artifact.PublicID, "tenant-1", "user-2"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong owner revoke = %v", err)
	}
	if err := service.Revoke(context.Background(), artifact.PublicID, "tenant-1", "user-1"); err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(artifact.URL)
	if _, err := service.Redeem(context.Background(), artifact.PublicID, parsed.Fragment); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked error = %v", err)
	}
}

func TestServiceWrongTokenDoesNotRevealLifecycleState(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	service, _ := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return now })
	expired, _ := service.Create(context.Background(), CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "expired", Message: "Expired report"})
	revoked, _ := service.Create(context.Background(), CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "revoked", Message: "Revoked report"})
	if err := service.Revoke(context.Background(), revoked.PublicID, "tenant-1", "user-1"); err != nil {
		t.Fatal(err)
	}

	now = expired.ExpiresAt
	if _, err := service.Redeem(context.Background(), expired.PublicID, "wrong"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong token for expired page = %v", err)
	}
	if _, err := service.Redeem(context.Background(), revoked.PublicID, "wrong"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong token for revoked page = %v", err)
	}

	expiredURL, _ := url.Parse(expired.URL)
	if _, err := service.Redeem(context.Background(), expired.PublicID, expiredURL.Fragment); !errors.Is(err, ErrExpired) {
		t.Fatalf("correct token for expired page = %v", err)
	}
	revokedURL, _ := url.Parse(revoked.URL)
	if _, err := service.Redeem(context.Background(), revoked.PublicID, revokedURL.Fragment); !errors.Is(err, ErrRevoked) {
		t.Fatalf("correct token for revoked page = %v", err)
	}
}

func TestParseMessageRejectsEmptyAndOversizedContent(t *testing.T) {
	if _, err := ParseMessage(" \n "); err == nil {
		t.Fatal("empty message was accepted")
	}
	if _, err := ParseMessage(strings.Repeat("a", MaxMessageLength+1)); err == nil {
		t.Fatal("oversized message was accepted")
	}
}

func FuzzParseMessageNormalizationProperties(f *testing.F) {
	f.Add("")
	f.Add(" Keep going. ")
	f.Add("\t\u2003目标报告\u3000\n")
	f.Add(strings.Repeat("界", MaxMessageLength))
	f.Add(strings.Repeat("界", MaxMessageLength+1))

	f.Fuzz(func(t *testing.T, raw string) {
		parsed, err := ParseMessage(raw)
		padded, paddedErr := ParseMessage("\t\u2003" + raw + "\u3000\n")
		if (err == nil) != (paddedErr == nil) {
			t.Fatalf("adding surrounding whitespace changed validity: raw error = %v, padded error = %v", err, paddedErr)
		}
		if err != nil {
			return
		}
		if padded != parsed {
			t.Fatalf("adding surrounding whitespace changed message: got %q, want %q", padded, parsed)
		}
		if parsed == "" || len([]rune(parsed)) > MaxMessageLength {
			t.Fatalf("accepted message violates output invariant: rune count = %d", len([]rune(parsed)))
		}
		reparsed, err := ParseMessage(parsed)
		if err != nil || reparsed != parsed {
			t.Fatalf("parsing was not idempotent: reparsed = %q, error = %v", reparsed, err)
		}
	})
}

func TestURLForReconstructsTheOriginalCapabilityFromPersistedIdentity(t *testing.T) {
	service, err := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := service.Create(context.Background(), CreateInput{
		TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conversation-1",
		TurnID: "turn-1", RecipientName: "Aina", Message: "Goal report",
	})
	if err != nil {
		t.Fatal(err)
	}
	reconstructed, err := service.URLFor(artifact.TenantID, artifact.TurnID, artifact.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	if reconstructed != artifact.URL {
		t.Fatalf("reconstructed URL = %q, want %q", reconstructed, artifact.URL)
	}
	if _, err := service.URLFor("", artifact.TurnID, artifact.PublicID); err == nil {
		t.Fatal("incomplete persisted identity was accepted")
	}
}

func TestFocusedPageConfigurationRejectsInsecureOriginsAndSecrets(t *testing.T) {
	store := NewMemoryStore()
	validSecret := []byte("0123456789abcdef0123456789abcdef")
	tests := []struct {
		name    string
		baseURL string
		secret  []byte
	}{
		{name: "HTTP origin", baseURL: "http://pages.example", secret: validSecret},
		{name: "origin with credentials", baseURL: "https://user:pass@pages.example", secret: validSecret},
		{name: "origin with path", baseURL: "https://pages.example/private", secret: validSecret},
		{name: "origin with query", baseURL: "https://pages.example?token=value", secret: validSecret},
		{name: "short secret", baseURL: "https://pages.example", secret: []byte("too-short")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewService(store, test.baseURL, test.secret, time.Now); err == nil {
				t.Fatal("insecure focused-page configuration was accepted")
			}
		})
	}
}

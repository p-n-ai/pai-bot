package focusedpage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
		t.Fatalf("retry changed artifact: %#v %#v", first, second)
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
		t.Fatalf("page = %#v", page)
	}
	if string(page.TokenHash) == parsed.Fragment {
		t.Fatal("store retained raw capability")
	}

	now = first.ExpiresAt
	if _, err := service.Redeem(context.Background(), first.PublicID, parsed.Fragment); !errors.Is(err, ErrExpired) {
		t.Fatalf("expiry error = %v", err)
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

func TestParseMessageRejectsEmptyAndOversizedContent(t *testing.T) {
	if _, err := ParseMessage(" \n "); err == nil {
		t.Fatal("empty message was accepted")
	}
	if _, err := ParseMessage(strings.Repeat("a", MaxMessageLength+1)); err == nil {
		t.Fatal("oversized message was accepted")
	}
}

func TestHandlerKeepsContentOutOfShellAndUsesPrivateSecurityHeaders(t *testing.T) {
	service, _ := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC) })
	artifact, _ := service.Create(context.Background(), CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "turn-1", RecipientName: "Aina", Message: "Private report"})
	handler, err := NewHandler(service, "https://t.me/pandai_bot")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/a/"+artifact.PublicID, nil)
	req.SetPathValue("publicID", artifact.PublicID)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if strings.Contains(recorder.Body.String(), "Private report") || strings.Contains(recorder.Body.String(), "Aina") {
		t.Fatal("private page data leaked into shell")
	}
	if got := recorder.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'none'") || !strings.Contains(got, "frame-ancestors 'none'") {
		t.Fatalf("CSP = %q", got)
	}

	parsed, _ := url.Parse(artifact.URL)
	redeemReq := httptest.NewRequest(http.MethodPost, "/a/"+artifact.PublicID, strings.NewReader(`{"token":"`+parsed.Fragment+`"}`))
	redeemReq.SetPathValue("publicID", artifact.PublicID)
	redeemRecorder := httptest.NewRecorder()
	handler.ServeHTTP(redeemRecorder, redeemReq)
	if redeemRecorder.Code != http.StatusOK || !strings.Contains(redeemRecorder.Body.String(), "Private report") {
		t.Fatalf("redeem = %d %s", redeemRecorder.Code, redeemRecorder.Body.String())
	}
}

func TestHandlerRejectsWrongTokenExpiredAndRevokedWithoutPrivateContent(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	service, _ := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return now })
	wrongTokenPage, _ := service.Create(context.Background(), CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "wrong-token", RecipientName: "Aina", Message: "Wrong-token private message"})
	expiredPage, _ := service.Create(context.Background(), CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "expired", RecipientName: "Aina", Message: "Expired private message"})
	revokedPage, _ := service.Create(context.Background(), CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "revoked", RecipientName: "Aina", Message: "Revoked private message"})
	if err := service.Revoke(context.Background(), revokedPage.PublicID, "tenant-1", "user-1"); err != nil {
		t.Fatal(err)
	}
	handler, _ := NewHandler(service, "https://t.me/pandai_bot")

	assertRejectedRedeem(t, handler, wrongTokenPage.PublicID, "wrong", http.StatusNotFound, "This page is unavailable.", "Wrong-token private message", "Aina")
	now = expiredPage.ExpiresAt
	expiredURL, _ := url.Parse(expiredPage.URL)
	assertRejectedRedeem(t, handler, expiredPage.PublicID, expiredURL.Fragment, http.StatusGone, "This page has expired.", "Expired private message", "Aina")
	revokedURL, _ := url.Parse(revokedPage.URL)
	assertRejectedRedeem(t, handler, revokedPage.PublicID, revokedURL.Fragment, http.StatusGone, "This page is no longer available.", "Revoked private message", "Aina")
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
	service, _ := NewService(store, "https://pages.example", validSecret, time.Now)
	if _, err := NewHandler(service, "http://t.me/pandai_bot"); err == nil {
		t.Fatal("insecure CTA URL was accepted")
	}
}

func assertRejectedRedeem(t *testing.T, handler http.Handler, publicID, token string, wantStatus int, wantCopy string, privateValues ...string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/a/"+publicID, strings.NewReader(`{"token":"`+token+`"}`))
	req.SetPathValue("publicID", publicID)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != wantStatus || !strings.Contains(recorder.Body.String(), wantCopy) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
	for _, privateValue := range privateValues {
		if strings.Contains(recorder.Body.String(), privateValue) {
			t.Fatalf("private value %q leaked in response %s", privateValue, recorder.Body.String())
		}
	}
}

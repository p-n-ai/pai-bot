package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

func TestFocusedPageHandlerKeepsContentOutOfShellAndUsesPrivateSecurityHeaders(t *testing.T) {
	service, _ := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time {
		return time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	})
	artifact, _ := service.Create(context.Background(), focusedpage.CreateInput{
		TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1",
		TurnID: "turn-1", RecipientName: "Aina", Message: "Private report",
	})
	handler, err := NewFocusedPageHandler(service, "https://t.me/pandai_bot")
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

func TestFocusedPageHandlerRejectsWrongTokenExpiredAndRevokedWithoutPrivateContent(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	service, _ := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return now })
	wrongTokenPage, _ := service.Create(context.Background(), focusedpage.CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "wrong-token", RecipientName: "Aina", Message: "Wrong-token private message"})
	expiredPage, _ := service.Create(context.Background(), focusedpage.CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "expired", RecipientName: "Aina", Message: "Expired private message"})
	revokedPage, _ := service.Create(context.Background(), focusedpage.CreateInput{TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conv-1", TurnID: "revoked", RecipientName: "Aina", Message: "Revoked private message"})
	if err := service.Revoke(context.Background(), revokedPage.PublicID, "tenant-1", "user-1"); err != nil {
		t.Fatal(err)
	}
	handler, _ := NewFocusedPageHandler(service, "https://t.me/pandai_bot")

	assertRejectedFocusedPageRedeem(t, handler, wrongTokenPage.PublicID, "wrong", http.StatusNotFound, "This page is unavailable.", "Wrong-token private message", "Aina")
	now = expiredPage.ExpiresAt
	expiredURL, _ := url.Parse(expiredPage.URL)
	assertRejectedFocusedPageRedeem(t, handler, expiredPage.PublicID, expiredURL.Fragment, http.StatusGone, "This page has expired.", "Expired private message", "Aina")
	revokedURL, _ := url.Parse(revokedPage.URL)
	assertRejectedFocusedPageRedeem(t, handler, revokedPage.PublicID, revokedURL.Fragment, http.StatusGone, "This page is no longer available.", "Revoked private message", "Aina")
}

func TestFocusedPageHandlerRejectsInsecureCTAURL(t *testing.T) {
	service, _ := focusedpage.NewService(focusedpage.NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), time.Now)
	if _, err := NewFocusedPageHandler(service, "http://t.me/pandai_bot"); err == nil {
		t.Fatal("insecure CTA URL was accepted")
	}
}

func assertRejectedFocusedPageRedeem(t *testing.T, handler http.Handler, publicID, token string, wantStatus int, wantCopy string, privateValues ...string) {
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

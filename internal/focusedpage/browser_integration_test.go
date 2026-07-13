//go:build integration && browser

package focusedpage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFocusedPageCapabilityFlowInChromium(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	service, err := NewService(NewMemoryStore(), "https://pages.example", []byte("0123456789abcdef0123456789abcdef"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	valid := createBrowserTestPage(t, service, "valid", "Private goal report")
	now = now.Add(-2 * time.Hour)
	expired := createBrowserTestPage(t, service, "expired", "Expired private report")
	now = now.Add(2 * time.Hour)
	revoked := createBrowserTestPage(t, service, "revoked", "Revoked private report")
	if err := service.Revoke(context.Background(), revoked.PublicID, "tenant-1", "user-1"); err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(service, "https://t.me/pandai_bot")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/a/{publicID}", handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	adminDir, err := filepath.Abs(filepath.Join("..", "..", "admin"))
	if err != nil {
		t.Fatal(err)
	}
	commandCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(commandCtx, "pnpm", "exec", "playwright", "test", "e2e/focused-page.spec.ts", "--project=chromium", "--reporter=line")
	command.Dir = adminDir
	command.Env = append(os.Environ(),
		"PLAYWRIGHT_BASE_URL="+server.URL,
		"FOCUSED_PAGE_VALID_URL="+browserURL(t, server.URL, valid, ""),
		"FOCUSED_PAGE_WRONG_TOKEN_URL="+browserURL(t, server.URL, valid, "wrong-token"),
		"FOCUSED_PAGE_EXPIRED_URL="+browserURL(t, server.URL, expired, ""),
		"FOCUSED_PAGE_REVOKED_URL="+browserURL(t, server.URL, revoked, ""),
	)
	if _, err := os.Stat("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"); err == nil {
		command.Env = append(command.Env, "PLAYWRIGHT_USE_SYSTEM_CHROME=true")
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("focused page Playwright test: %v\n%s", err, output)
	}
}

func createBrowserTestPage(t *testing.T, service *Service, turnID, message string) Artifact {
	t.Helper()
	artifact, err := service.Create(context.Background(), CreateInput{
		TenantID: "tenant-1", OwnerUserID: "user-1", ConversationID: "conversation-1",
		TurnID: turnID, RecipientName: "Aina", Message: message,
	})
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func browserURL(t *testing.T, origin string, artifact Artifact, tokenOverride string) string {
	t.Helper()
	capabilityURL, err := url.Parse(artifact.URL)
	if err != nil {
		t.Fatal(err)
	}
	token := capabilityURL.Fragment
	if tokenOverride != "" {
		token = tokenOverride
	}
	return origin + "/a/" + url.PathEscape(artifact.PublicID) + "#" + token
}

package chat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramChannelSyncCommands(t *testing.T) {
	var gotPath string
	var gotCommands string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		gotCommands = r.Form.Get("commands")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	ch := &TelegramChannel{
		token:   "test-token",
		baseURL: server.URL,
		client:  server.Client(),
		stop:    make(chan struct{}),
	}

	if err := ch.syncCommands(); err != nil {
		t.Fatalf("syncCommands() error = %v", err)
	}
	if gotPath != "/setMyCommands" {
		t.Fatalf("path = %q, want /setMyCommands", gotPath)
	}
	if gotCommands == "" {
		t.Fatal("commands payload is empty")
	}
	if !(containsString(gotCommands, `"start"`) && containsString(gotCommands, `"clear"`)) {
		t.Fatalf("commands payload = %q, expected start and clear", gotCommands)
	}
}

func containsString(s, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(needle); i++ {
		if s[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

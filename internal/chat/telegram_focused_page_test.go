package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestTelegramSendsTutorTextWithFocusedPageURLButton(t *testing.T) {
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()
	channel, _ := NewTelegramChannel("token")
	channel.baseURL = server.URL
	pageURL := "https://pages.example/a/public#private-capability"
	keyboard := AppendFocusedPageButton([][]InlineButton{{{Text: "Existing", CallbackData: "existing"}}}, pageURL)
	if err := channel.SendMessage(context.Background(), "123", OutboundMessage{Text: "Tutor reply", InlineKeyboard: keyboard}); err != nil {
		t.Fatal(err)
	}
	if form.Get("text") != "Tutor reply" {
		t.Fatalf("text = %q", form.Get("text"))
	}
	var markup struct {
		InlineKeyboard [][]struct {
			Text         string `json:"text"`
			CallbackData string `json:"callback_data"`
			URL          string `json:"url"`
		} `json:"inline_keyboard"`
	}
	if err := json.Unmarshal([]byte(form.Get("reply_markup")), &markup); err != nil {
		t.Fatal(err)
	}
	if len(markup.InlineKeyboard) != 2 {
		t.Fatalf("rows = %#v", markup.InlineKeyboard)
	}
	if markup.InlineKeyboard[0][0].CallbackData != "existing" {
		t.Fatalf("first row = %#v", markup.InlineKeyboard[0])
	}
	if markup.InlineKeyboard[1][0].URL != pageURL {
		t.Fatalf("focused page row = %#v", markup.InlineKeyboard[1])
	}
}

package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWhatsAppChannel_SendMessage(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v21.0/phone-123/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Fatalf("expected Bearer test-token, got %q", auth)
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.xxx"}]}`))
	}))
	defer server.Close()

	ch := &WhatsAppChannel{
		accessToken: "test-token",
		phoneID:     "phone-123",
		baseURL:     server.URL,
		client:      http.DefaultClient,
	}

	err := ch.SendMessage(context.Background(), "6281234567890", OutboundMessage{
		Text: "Hello from P&AI!",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if gotBody["messaging_product"] != "whatsapp" {
		t.Fatalf("messaging_product = %v, want whatsapp", gotBody["messaging_product"])
	}
	if gotBody["to"] != "6281234567890" {
		t.Fatalf("to = %v, want 6281234567890", gotBody["to"])
	}
	textObj, ok := gotBody["text"].(map[string]any)
	if !ok {
		t.Fatalf("text field missing or not object")
	}
	if textObj["body"] != "Hello from P&AI!" {
		t.Fatalf("text.body = %v, want Hello from P&AI!", textObj["body"])
	}
}

func TestWhatsAppChannel_SendMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid token"}}`))
	}))
	defer server.Close()

	ch := &WhatsAppChannel{
		accessToken: "bad-token",
		phoneID:     "phone-123",
		baseURL:     server.URL,
		client:      http.DefaultClient,
	}

	err := ch.SendMessage(context.Background(), "123", OutboundMessage{Text: "hi"})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error should mention status code, got: %v", err)
	}
}

func TestWhatsAppChannel_SendTyping(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	ch := &WhatsAppChannel{
		accessToken: "test-token",
		phoneID:     "phone-123",
		baseURL:     server.URL,
		client:      http.DefaultClient,
	}

	err := ch.SendTyping(context.Background(), "6281234567890")
	if err != nil {
		t.Fatalf("SendTyping() error = %v", err)
	}
	// WhatsApp doesn't have a native typing indicator — this should be a no-op or
	// send a "typing" status. Either way, no error expected.
}

func TestWhatsAppWebhookVerification(t *testing.T) {
	ch := &WhatsAppChannel{
		verifyToken: "my-verify-token",
	}

	handler := ch.WebhookHandler(func(InboundMessage) {})

	req := httptest.NewRequest(http.MethodGet,
		"/webhook?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=challenge-123",
		nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != "challenge-123" {
		t.Fatalf("body = %q, want challenge-123", w.Body.String())
	}
}

func TestWhatsAppWebhookVerification_BadToken(t *testing.T) {
	ch := &WhatsAppChannel{
		verifyToken: "my-verify-token",
	}

	handler := ch.WebhookHandler(func(InboundMessage) {})

	req := httptest.NewRequest(http.MethodGet,
		"/webhook?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge-123",
		nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestWhatsAppWebhookInboundMessage(t *testing.T) {
	ch := &WhatsAppChannel{
		verifyToken: "tok",
		phoneID:     "phone-123",
	}

	var got InboundMessage
	handler := ch.WebhookHandler(func(msg InboundMessage) {
		got = msg
	})

	payload := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {"phone_number_id": "phone-123"},
					"contacts": [{"profile": {"name": "Alya"}, "wa_id": "60123456789"}],
					"messages": [{
						"from": "60123456789",
						"id": "wamid.abc",
						"timestamp": "1234567890",
						"type": "text",
						"text": {"body": "Tolong saya dengan algebra"}
					}]
				}
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got.Channel != "whatsapp" {
		t.Fatalf("Channel = %q, want whatsapp", got.Channel)
	}
	if got.UserID != "60123456789" {
		t.Fatalf("UserID = %q, want 60123456789", got.UserID)
	}
	if got.Text != "Tolong saya dengan algebra" {
		t.Fatalf("Text = %q, want Tolong saya dengan algebra", got.Text)
	}
	if got.FirstName != "Alya" {
		t.Fatalf("FirstName = %q, want Alya", got.FirstName)
	}
	if got.ExternalID != "wamid.abc" {
		t.Fatalf("ExternalID = %q, want wamid.abc", got.ExternalID)
	}
}

func TestWhatsAppWebhookIgnoresStatusUpdates(t *testing.T) {
	ch := &WhatsAppChannel{
		verifyToken: "tok",
		phoneID:     "phone-123",
	}

	called := false
	handler := ch.WebhookHandler(func(msg InboundMessage) {
		called = true
	})

	// Status update (delivery receipt), not a message.
	payload := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {"phone_number_id": "phone-123"},
					"statuses": [{"id": "wamid.abc", "status": "delivered"}]
				}
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if called {
		t.Fatal("handler should not be called for status updates")
	}
}

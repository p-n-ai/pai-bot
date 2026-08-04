package chat

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const whatsappTestAppSecret = "whatsapp-test-app-secret"

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

func TestWhatsAppChannel_SendMessagePreservesCallerContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ch := &WhatsAppChannel{
		accessToken: "test-token",
		phoneID:     "phone-123",
		baseURL:     server.URL,
		client:      http.DefaultClient,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ch.SendMessage(ctx, "123", OutboundMessage{Text: "hi"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendMessage() error = %v, want context canceled", err)
	}
}

func TestWhatsAppChannel_SendMessageBoundsAPIErrorBody(t *testing.T) {
	const responseBodySize = 32 << 10

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", responseBodySize)))
	}))
	defer server.Close()

	ch := &WhatsAppChannel{
		accessToken: "test-token",
		phoneID:     "phone-123",
		baseURL:     server.URL,
		client:      http.DefaultClient,
	}

	err := ch.SendMessage(context.Background(), "123", OutboundMessage{Text: "hi"})
	if err == nil {
		t.Fatal("SendMessage() error = nil, want API error")
	}
	if len(err.Error()) >= responseBodySize {
		t.Fatalf("SendMessage() error length = %d, want bounded response body", len(err.Error()))
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
		appSecret:   whatsappTestAppSecret,
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

	req := newSignedWhatsAppRequest(payload)
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
	if got.ExternalID != "60123456789" {
		t.Fatalf("ExternalID = %q, want stable WhatsApp sender", got.ExternalID)
	}
	if got.MessageID != "wamid.abc" || got.DeliveryID != "wamid.abc" {
		t.Fatalf("message identity = %q/%q, want WhatsApp delivery wamid.abc", got.MessageID, got.DeliveryID)
	}
}

func TestWhatsAppWebhookRejectsOversizedPayload(t *testing.T) {
	ch := &WhatsAppChannel{phoneID: "phone-123"}
	handler := ch.WebhookHandler(func(InboundMessage) {})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(strings.Repeat(" ", (1<<20)+1)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestWhatsAppWebhookRejectsUnsignedPayload(t *testing.T) {
	ch := &WhatsAppChannel{phoneID: "phone-123", appSecret: whatsappTestAppSecret}
	called := false
	handler := ch.WebhookHandler(func(InboundMessage) { called = true })

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"entry":[]}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("unsigned webhook dispatched an inbound message")
	}
}

func TestWhatsAppWebhookAllowsNilHandler(t *testing.T) {
	ch := &WhatsAppChannel{phoneID: "phone-123", appSecret: whatsappTestAppSecret}
	handler := ch.WebhookHandler(nil)
	payload := `{
		"entry": [{
			"changes": [{
				"value": {
					"metadata": {"phone_number_id": "phone-123"},
					"messages": [{
						"from": "60123456789",
						"id": "wamid.abc",
						"type": "text",
						"text": {"body": "hello"}
					}]
				}
			}]
		}]
	}`

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("WebhookHandler(nil) panicked: %v", recovered)
		}
	}()

	req := newSignedWhatsAppRequest(payload)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestWhatsAppWebhookIgnoresStatusUpdates(t *testing.T) {
	ch := &WhatsAppChannel{
		verifyToken: "tok",
		phoneID:     "phone-123",
		appSecret:   whatsappTestAppSecret,
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

	req := newSignedWhatsAppRequest(payload)
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

func newSignedWhatsAppRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	mac := hmac.New(sha256.New, []byte(whatsappTestAppSecret))
	_, _ = mac.Write([]byte(body))
	request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	return request
}

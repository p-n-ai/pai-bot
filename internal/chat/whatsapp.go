package chat

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const (
	defaultWhatsAppBaseURL             = "https://graph.facebook.com"
	maxWhatsAppWebhookBodyBytes  int64 = 1 << 20
	maxWhatsAppAPIErrorBodyBytes int64 = 4 << 10
)

// WhatsAppChannel implements the Channel interface for WhatsApp Cloud API.
type WhatsAppChannel struct {
	accessToken string
	phoneID     string
	verifyToken string
	appSecret   string
	baseURL     string
	client      *http.Client
}

// WhatsAppCloudConfig contains the credentials for a WhatsApp Cloud API channel.
type WhatsAppCloudConfig struct {
	AccessToken string
	PhoneID     string
	VerifyToken string
	AppSecret   string
}

// NewWhatsAppChannel creates a WhatsApp channel adapter.
func NewWhatsAppChannel(cfg WhatsAppCloudConfig) (*WhatsAppChannel, error) {
	if strings.TrimSpace(cfg.AccessToken) == "" {
		return nil, fmt.Errorf("whatsapp access token is required (LEARN_WHATSAPP_ACCESS_TOKEN)")
	}
	if strings.TrimSpace(cfg.PhoneID) == "" {
		return nil, fmt.Errorf("whatsapp phone number ID is required (LEARN_WHATSAPP_PHONE_ID)")
	}
	if strings.TrimSpace(cfg.VerifyToken) == "" {
		return nil, fmt.Errorf("whatsapp verify token is required (LEARN_WHATSAPP_VERIFY_TOKEN)")
	}
	if strings.TrimSpace(cfg.AppSecret) == "" {
		return nil, fmt.Errorf("whatsapp app secret is required (LEARN_WHATSAPP_APP_SECRET)")
	}
	return &WhatsAppChannel{
		accessToken: cfg.AccessToken,
		phoneID:     cfg.PhoneID,
		verifyToken: cfg.VerifyToken,
		appSecret:   cfg.AppSecret,
		baseURL:     defaultWhatsAppBaseURL,
		client:      &http.Client{},
	}, nil
}

// SendMessage sends a text message to a WhatsApp user.
func (w *WhatsAppChannel) SendMessage(ctx context.Context, userID string, msg OutboundMessage) error {
	body := whatsAppMessageRequest{
		MessagingProduct: "whatsapp",
		To:               userID,
		Type:             "text",
		Text:             whatsAppText{Body: msg.Text},
	}

	return w.postJSON(ctx, fmt.Sprintf("/v21.0/%s/messages", w.phoneID), body)
}

// SendTyping is a no-op for WhatsApp — the Cloud API does not support typing indicators.
func (w *WhatsAppChannel) SendTyping(_ context.Context, _ string) error {
	return nil
}

// Start is a no-op for WhatsApp — messages arrive via webhook, not polling.
// Use WebhookHandler() to mount the HTTP handler on the server mux.
func (w *WhatsAppChannel) Start(_ context.Context, _ InboundHandler) error {
	return nil
}

// Stop is a no-op for WhatsApp.
func (w *WhatsAppChannel) Stop() error {
	return nil
}

// WebhookHandler returns an http.Handler for the WhatsApp webhook endpoint.
// GET requests handle verification; POST requests handle inbound messages.
func (w *WhatsAppChannel) WebhookHandler(handler InboundHandler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.handleVerification(rw, r)
		case http.MethodPost:
			w.handleInbound(rw, r, handler)
		default:
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// handleVerification handles the WhatsApp webhook verification challenge.
func (w *WhatsAppChannel) handleVerification(rw http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == w.verifyToken {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(challenge))
		return
	}

	http.Error(rw, "forbidden", http.StatusForbidden)
}

// handleInbound parses an inbound WhatsApp webhook payload and dispatches messages.
func (w *WhatsAppChannel) handleInbound(rw http.ResponseWriter, r *http.Request, handler InboundHandler) {
	body, err := io.ReadAll(http.MaxBytesReader(rw, r.Body, maxWhatsAppWebhookBodyBytes))
	if err != nil {
		slog.Error("whatsapp webhook: read body failed", "error", err)
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(rw, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}
	if !w.validInboundSignature(r.Header.Get("X-Hub-Signature-256"), body) {
		http.Error(rw, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload waWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("whatsapp webhook: unmarshal failed", "error", err)
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			val := change.Value
			// Skip if this isn't for our phone number.
			if val.Metadata.PhoneNumberID != "" && val.Metadata.PhoneNumberID != w.phoneID {
				continue
			}
			// Build a contact lookup for profile names.
			contacts := make(map[string]waContact, len(val.Contacts))
			for _, c := range val.Contacts {
				contacts[c.WaID] = c
			}
			for _, msg := range val.Messages {
				if msg.Type != "text" {
					slog.Debug("whatsapp webhook: ignoring non-text message", "type", msg.Type, "from", msg.From)
					continue
				}
				inbound := InboundMessage{
					Channel:    "whatsapp",
					UserID:     msg.From,
					ExternalID: msg.From,
					ThreadID:   msg.From,
					MessageID:  msg.ID,
					DeliveryID: msg.ID,
					Text:       msg.Text.Body,
				}
				if contact, ok := contacts[msg.From]; ok {
					inbound.FirstName = contact.Profile.Name
				}
				if handler != nil {
					if err := handler(r.Context(), inbound); err != nil {
						http.Error(rw, "temporarily unavailable", http.StatusServiceUnavailable)
						return
					}
				}
			}
		}
	}
	rw.WriteHeader(http.StatusOK)
}

func (w *WhatsAppChannel) validInboundSignature(header string, body []byte) bool {
	const prefix = "sha256="
	if w.appSecret == "" || !strings.HasPrefix(header, prefix) {
		return false
	}
	signature, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(w.appSecret))
	_, _ = mac.Write(body)
	return hmac.Equal(signature, mac.Sum(nil))
}

type whatsAppText struct {
	Body string `json:"body"`
}

type whatsAppMessageRequest struct {
	MessagingProduct string       `json:"messaging_product"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Text             whatsAppText `json:"text"`
}

func (w *WhatsAppChannel) postJSON(ctx context.Context, path string, body whatsAppMessageRequest) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxWhatsAppAPIErrorBodyBytes))
		return fmt.Errorf("whatsapp api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ── WhatsApp Cloud API webhook payload types ─────────────────────────────

type waWebhookPayload struct {
	Object string    `json:"object"`
	Entry  []waEntry `json:"entry"`
}

type waEntry struct {
	ID      string     `json:"id"`
	Changes []waChange `json:"changes"`
}

type waChange struct {
	Value waValue `json:"value"`
	Field string  `json:"field"`
}

type waValue struct {
	MessagingProduct string      `json:"messaging_product"`
	Metadata         waMetadata  `json:"metadata"`
	Contacts         []waContact `json:"contacts"`
	Messages         []waMessage `json:"messages"`
}

type waMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type waContact struct {
	Profile waProfile `json:"profile"`
	WaID    string    `json:"wa_id"`
}

type waProfile struct {
	Name string `json:"name"`
}

type waMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      waText `json:"text"`
}

type waText struct {
	Body string `json:"body"`
}

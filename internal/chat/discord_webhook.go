// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
)

const maxDiscordWebhookBody = 1 << 20

// WebhookHandler verifies and handles Discord interactions.
func (d *DiscordChannel) WebhookHandler(_ InboundWebhookHandler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, maxDiscordWebhookBody))
		if err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}

		if !d.validInteractionSignature(request.Header, body) {
			http.Error(response, "invalid signature", http.StatusUnauthorized)
			return
		}
		d.handleInteraction(response, body)
	})
}

func (d *DiscordChannel) handleInteraction(response http.ResponseWriter, body []byte) {
	var interaction struct {
		Type int `json:"type"`
	}
	if err := json.Unmarshal(body, &interaction); err != nil {
		http.Error(response, "bad request", http.StatusBadRequest)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	if interaction.Type == 1 {
		_ = json.NewEncoder(response).Encode(struct {
			Type int `json:"type"`
		}{Type: 1})
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (d *DiscordChannel) validInteractionSignature(headers http.Header, body []byte) bool {
	timestamp := headers.Get("X-Signature-Timestamp")
	signature, err := hex.DecodeString(headers.Get("X-Signature-Ed25519"))
	if timestamp == "" || err != nil || len(signature) != ed25519.SignatureSize {
		return false
	}
	signed := make([]byte, 0, len(timestamp)+len(body))
	signed = append(signed, timestamp...)
	signed = append(signed, body...)
	return ed25519.Verify(d.publicKey, signed, signature)
}

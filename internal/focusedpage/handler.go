// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpage

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Handler struct {
	service *Service
	ctaURL  string
}

func NewHandler(service *Service, ctaURL string) (*Handler, error) {
	if service == nil {
		return nil, fmt.Errorf("focused page service is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(ctaURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("focused page CTA URL must be absolute HTTPS")
	}
	return &Handler{service: service, ctaURL: parsed.String()}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setPrivateHeaders(w.Header())
	publicID := r.PathValue("publicID")
	switch r.Method {
	case http.MethodGet:
		h.serveShell(w, publicID)
	case http.MethodPost:
		h.redeem(w, r, publicID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) serveShell(w http.ResponseWriter, publicID string) {
	if strings.TrimSpace(publicID) == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	nonce := randomNonce()
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+nonce+"'; style-src 'nonce-"+nonce+"'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = focusedPageTemplate.Execute(w, map[string]string{"Nonce": nonce, "PublicID": publicID, "CTAURL": h.ctaURL})
}

func (h *Handler) redeem(w http.ResponseWriter, r *http.Request, publicID string) {
	var input struct {
		Token string `json:"token"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeRedeemError(w, http.StatusBadRequest, "invalid request")
		return
	}
	page, err := h.service.Redeem(r.Context(), publicID, input.Token)
	if err != nil {
		switch {
		case errors.Is(err, ErrExpired):
			writeRedeemError(w, http.StatusGone, "This page has expired.")
		case errors.Is(err, ErrRevoked):
			writeRedeemError(w, http.StatusGone, "This page is no longer available.")
		default:
			writeRedeemError(w, http.StatusNotFound, "This page is unavailable.")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"recipient_name": page.RecipientName,
		"message":        page.Message,
		"expires_at":     page.ExpiresAt.Format(time.RFC3339),
	})
}

func setPrivateHeaders(header http.Header) {
	header.Set("Cache-Control", "private, no-store, max-age=0")
	header.Set("Pragma", "no-cache")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Cross-Origin-Opener-Policy", "same-origin")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	header.Set("X-Robots-Tag", "noindex, nofollow, noarchive")
}

func writeRedeemError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func randomNonce() string {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("generate CSP nonce: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

var focusedPageTemplate = template.Must(template.New("focused-page").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>A message from P&amp;AI</title>
<style nonce="{{.Nonce}}">:root{font-family:system-ui,sans-serif;color:#20201d;background:#f6f1e8}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:24px;box-sizing:border-box}main{width:min(680px,100%);background:#fff;border:1px solid #ded7ca;border-radius:24px;padding:clamp(24px,6vw,52px);box-shadow:0 18px 60px #493c2520}small{color:#756b5d}h1{font-size:clamp(24px,5vw,40px);margin:.3em 0 1em}p{font-size:clamp(18px,3vw,24px);line-height:1.55;white-space:pre-wrap}.cta{display:inline-block;margin-top:24px;padding:12px 18px;border-radius:999px;background:#20201d;color:#fff;text-decoration:none}[hidden]{display:none}</style></head>
<body><main><small>P&amp;AI · private message</small><h1 id="heading">Opening your message…</h1><p id="message"></p><small id="expiry"></small><a id="cta" class="cta" href="{{.CTAURL}}" rel="noreferrer" hidden>Continue with P&amp;AI</a></main>
<script nonce="{{.Nonce}}">(async()=>{const secret=location.hash.slice(1);history.replaceState(null,'',location.pathname);const heading=document.querySelector('#heading'),message=document.querySelector('#message'),expiry=document.querySelector('#expiry'),cta=document.querySelector('#cta');if(!secret){heading.textContent='This page is unavailable.';return}try{const response=await fetch('/a/{{.PublicID}}',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({token:secret}),cache:'no-store',referrerPolicy:'no-referrer'});const data=await response.json();if(!response.ok){heading.textContent=data.error||'This page is unavailable.';return}heading.textContent='A message for '+data.recipient_name;message.textContent=data.message;expiry.textContent='Available until '+new Date(data.expires_at).toLocaleString();cta.hidden=false}catch(_){heading.textContent='This page is unavailable.'}})()</script></body></html>`))

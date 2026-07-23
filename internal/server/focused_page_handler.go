// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

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

	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

type FocusedPageHandler struct {
	service *focusedpage.Service
	ctaURL  string
}

func NewFocusedPageHandler(service *focusedpage.Service, ctaURL string) (*FocusedPageHandler, error) {
	if service == nil {
		return nil, fmt.Errorf("focused page service is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(ctaURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("focused page CTA URL must be absolute HTTPS")
	}
	return &FocusedPageHandler{service: service, ctaURL: parsed.String()}, nil
}

func (h *FocusedPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setFocusedPagePrivateHeaders(w.Header())
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

func (h *FocusedPageHandler) serveShell(w http.ResponseWriter, publicID string) {
	if strings.TrimSpace(publicID) == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	nonce := focusedPageNonce()
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+nonce+"'; style-src 'nonce-"+nonce+"'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = focusedPageTemplate.Execute(w, map[string]string{"Nonce": nonce, "PublicID": publicID, "CTAURL": h.ctaURL})
}

func (h *FocusedPageHandler) redeem(w http.ResponseWriter, r *http.Request, publicID string) {
	var input struct {
		Token string `json:"token"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeFocusedPageError(w, http.StatusBadRequest, "invalid request")
		return
	}
	page, err := h.service.Redeem(r.Context(), publicID, input.Token)
	if err != nil {
		switch {
		case errors.Is(err, focusedpage.ErrExpired):
			writeFocusedPageError(w, http.StatusGone, "This page has expired.")
		case errors.Is(err, focusedpage.ErrRevoked):
			writeFocusedPageError(w, http.StatusGone, "This page is no longer available.")
		default:
			writeFocusedPageError(w, http.StatusNotFound, "This page is unavailable.")
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

func setFocusedPagePrivateHeaders(header http.Header) {
	header.Set("Cache-Control", "private, no-store, max-age=0")
	header.Set("Pragma", "no-cache")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Cross-Origin-Opener-Policy", "same-origin")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	header.Set("X-Robots-Tag", "noindex, nofollow, noarchive")
}

func writeFocusedPageError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func focusedPageNonce() string {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("generate CSP nonce: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

var focusedPageTemplate = template.Must(template.New("focused-page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>A message from P&amp;AI</title>
<style nonce="{{.Nonce}}">
  :root {
    color: #26352d;
    background: #e8eee9;
    font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    font-synthesis: none;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }
  * { box-sizing: border-box; }
  body {
    min-height: 100vh;
    margin: 0;
    display: grid;
    place-items: center;
    padding: clamp(18px, 5vw, 64px);
  }
  .page {
    width: min(1060px, 100%);
    min-height: min(620px, calc(100vh - 128px));
    display: grid;
    grid-template-columns: minmax(260px, 300px) 1fr;
    overflow: hidden;
    border-radius: 32px;
    background: #fbfcfa;
    box-shadow:
      0 0 0 1px rgb(12 35 24 / 7%),
      0 2px 5px rgb(12 35 24 / 5%),
      0 28px 70px rgb(12 35 24 / 12%);
  }
  .context {
    display: flex;
    flex-direction: column;
    padding: clamp(32px, 5vw, 48px);
    color: #f5faf6;
    background: #123d2c;
  }
  .brand, .eyebrow {
    margin: 0;
    font-size: 12px;
    font-weight: 700;
    line-height: 1.2;
    letter-spacing: .16em;
    text-transform: uppercase;
  }
  .identity { margin-block: auto; }
  .eyebrow { color: #b9cbbf; }
  h1 {
    margin: 14px 0 0;
    max-width: 12ch;
    font-size: clamp(30px, 4vw, 44px);
    line-height: 1.05;
    letter-spacing: -.035em;
    overflow-wrap: break-word;
    text-wrap: balance;
  }
  .privacy {
    max-width: 24ch;
    margin: 36px 0 0;
    color: #b9cbbf;
    font-size: 15px;
    line-height: 1.55;
    text-wrap: pretty;
  }
  .content {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: clamp(42px, 7vw, 72px);
    min-width: 0;
    padding: clamp(42px, 7vw, 84px);
  }
  .message {
    max-width: 24ch;
    margin: 0;
    font-size: clamp(32px, 5vw, 58px);
    font-weight: 620;
    line-height: 1.12;
    letter-spacing: -.04em;
    white-space: pre-wrap;
    overflow-wrap: break-word;
    text-wrap: pretty;
  }
  .footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
  }
  .expiry {
    color: #69756e;
    font-size: 13px;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .cta {
    min-height: 48px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding-inline: 18px 16px;
    border-radius: 14px;
    background: #123d2c;
    color: #fff;
    font-size: 15px;
    font-weight: 700;
    text-decoration: none;
    user-select: none;
    box-shadow: 0 1px 2px rgb(12 35 24 / 10%), 0 7px 18px rgb(12 35 24 / 14%);
    transition-property: scale, background-color, box-shadow;
    transition-duration: 150ms;
    transition-timing-function: ease-out;
  }
  .cta:hover { background: #0d3324; box-shadow: 0 2px 4px rgb(12 35 24 / 12%), 0 9px 22px rgb(12 35 24 / 17%); }
  .cta:active { scale: .96; }
  .cta:focus-visible { outline: 3px solid #92b6a0; outline-offset: 3px; }
  .arrow { translate: 0 -1px; }
  [hidden] { display: none; }
  ::selection { color: #10271d; background: #c8dfcf; }
  @media (max-width: 720px) {
    body { place-items: start center; padding: 18px; }
    .page { min-height: calc(100vh - 36px); grid-template-columns: 1fr; border-radius: 26px; }
    .context { min-height: 280px; padding: 32px 30px; }
    .identity { margin-block: 48px 0; }
    h1 { font-size: 34px; }
    .privacy { margin-top: 28px; }
    .content { justify-content: flex-start; gap: 40px; padding: 42px 28px 34px; }
    .message { font-size: clamp(31px, 9.5vw, 42px); }
    .footer { align-items: flex-start; flex-direction: column; }
    .cta { width: 100%; }
  }
  @media (prefers-reduced-motion: reduce) {
    .cta { transition-duration: 0ms; }
  }
</style>
</head>
<body>
<main class="page">
  <aside class="context">
    <p class="brand">P&amp;AI</p>
    <div class="identity">
      <p class="eyebrow">Private message for</p>
      <h1 id="heading">Opening your message…</h1>
    </div>
    <p class="privacy">Only someone with this private link can open the message.</p>
  </aside>
  <section class="content" aria-live="polite">
    <p id="message" class="message"></p>
    <footer id="footer" class="footer" hidden>
      <span id="expiry" class="expiry"></span>
      <a id="cta" class="cta" href="{{.CTAURL}}" rel="noreferrer">Continue with P&amp;AI <span class="arrow" aria-hidden="true">→</span></a>
    </footer>
  </section>
</main>
<script nonce="{{.Nonce}}">
(async()=>{
  const secret=location.hash.slice(1);
  history.replaceState(null,'',location.pathname);
  const heading=document.querySelector('#heading');
  const message=document.querySelector('#message');
  const expiry=document.querySelector('#expiry');
  const footer=document.querySelector('#footer');
  const showError=(copy)=>{heading.textContent=copy;heading.removeAttribute('aria-label');message.textContent='';footer.hidden=true};
  if(!secret){showError('This page is unavailable.');return}
  try{
    const response=await fetch('/a/{{.PublicID}}',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({token:secret}),cache:'no-store',referrerPolicy:'no-referrer'});
    const data=await response.json();
    if(!response.ok){showError(data.error||'This page is unavailable.');return}
    heading.textContent=data.recipient_name;
    heading.setAttribute('aria-label','A message for '+data.recipient_name);
    message.textContent=data.message;
    expiry.textContent='Available until '+new Date(data.expires_at).toLocaleString();
    footer.hidden=false;
  }catch(_){showError('This page is unavailable.')}
})()
</script>
</body>
</html>`))

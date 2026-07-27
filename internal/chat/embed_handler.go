// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	_ "embed"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

//go:embed embed/widget.js
var widgetJS []byte

//go:embed embed/chat.html
var chatHTML []byte

// HandleWidgetJS returns an HTTP handler that serves the embed loader script.
func HandleWidgetJS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(widgetJS) //nolint:errcheck
	}
}

// HandleChatPage returns an HTTP handler that serves the embed chat page.
// The store is used to look up allowed origins for CSP frame-ancestors.
func HandleChatPage(store EmbedConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := r.URL.Query().Get("tenant")
		if tenant == "" {
			http.Error(w, "missing tenant parameter", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		// Look up tenant's allowed origins for CSP frame-ancestors.
		if store != nil {
			cfg, err := store.GetByTenantSlug(r.Context(), tenant)
			origins := validFrameAncestors(cfg.AllowedOrigins)
			if err == nil && cfg.Enabled && len(origins) > 0 {
				csp := "frame-ancestors " + strings.Join(origins, " ")
				w.Header().Set("Content-Security-Policy", csp)
			} else if err != nil {
				slog.Debug("embed chat page: could not look up tenant config", "tenant", tenant, "error", err)
			}
		}

		// Remove X-Frame-Options since CSP frame-ancestors supersedes it.
		w.Header().Del("X-Frame-Options")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(chatHTML) //nolint:errcheck
	}
}

func validFrameAncestors(origins []string) []string {
	valid := make([]string, 0, len(origins))
	for _, origin := range origins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
			continue
		}
		valid = append(valid, parsed.Scheme+"://"+parsed.Host)
	}
	return valid
}

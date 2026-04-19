// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	_ "embed"
	"log/slog"
	"net/http"
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

		// Look up tenant's allowed origins for CSP frame-ancestors.
		if store != nil {
			cfg, err := store.GetByTenantSlug(r.Context(), tenant)
			if err == nil && len(cfg.AllowedOrigins) > 0 {
				csp := "frame-ancestors " + strings.Join(cfg.AllowedOrigins, " ")
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

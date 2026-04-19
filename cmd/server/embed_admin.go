// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

// handleEmbedGuestAuth issues a guest JWT for an embed widget connection.
// POST /api/embed/auth/guest
// Body: {"tenant": "slug"}
// No authentication required — public endpoint.
func handleEmbedGuestAuth(embedStore chat.EmbedConfigStore, guestSvc *auth.GuestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Tenant      string `json:"tenant"`
			Fingerprint string `json:"fingerprint"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Tenant) == "" {
			http.Error(w, "missing tenant", http.StatusBadRequest)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			// Also check Referer as fallback for same-origin iframe requests
			origin = r.Header.Get("Referer")
			if origin != "" {
				// Extract just the origin from the Referer URL
				if u, err := url.Parse(origin); err == nil {
					origin = u.Scheme + "://" + u.Host
				}
			}
		}
		if origin == "" {
			http.Error(w, "missing origin", http.StatusForbidden)
			return
		}

		// Validate tenant + origin combination.
		tenantID, err := embedStore.FindTenantBySlugAndOrigin(r.Context(), req.Tenant, origin)
		if err != nil {
			if errors.Is(err, chat.ErrEmbedNotConfigured) {
				http.Error(w, "embed not configured for this tenant/origin", http.StatusForbidden)
				return
			}
			slog.Error("embed guest auth: find tenant", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		token, userID, err := guestSvc.IssueGuestToken(r.Context(), tenantID, origin, req.Fingerprint)
		if err != nil {
			slog.Error("embed guest auth: issue token", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      token,
			"user_id":    userID,
			"expires_in": 3600,
		})
	}
}

// handleEmbedUpgradeGuest upgrades a guest user to a student account.
// POST /api/embed/auth/upgrade
// Requires a valid guest JWT in the Authorization: Bearer header.
func handleEmbedUpgradeGuest(guestSvc *auth.GuestService, tm *auth.TokenManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse JWT from Authorization header.
		authHeader := r.Header.Get("Authorization")
		rawToken := strings.TrimPrefix(authHeader, "Bearer ")
		rawToken = strings.TrimSpace(rawToken)
		if rawToken == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		claims, err := tm.Parse(rawToken, time.Now())
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		if claims.Role != auth.RoleGuest {
			http.Error(w, "token must be a guest token", http.StatusForbidden)
			return
		}

		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.Name) == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Email) == "" || !strings.Contains(req.Email, "@") {
			http.Error(w, "valid email is required", http.StatusBadRequest)
			return
		}
		if len(strings.TrimSpace(req.Password)) < 8 {
			http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		token, err := guestSvc.UpgradeGuest(r.Context(), claims.Subject, claims.TenantID, req.Name, req.Email, req.Password)
		if err != nil {
			if errors.Is(err, auth.ErrNotGuest) {
				http.Error(w, "user is not a guest", http.StatusForbidden)
				return
			}
			if errors.Is(err, auth.ErrEmailAlreadyUsed) {
				http.Error(w, "email already in use", http.StatusConflict)
				return
			}
			slog.Error("embed upgrade guest", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token": token,
			"role":  "student",
		})
	}
}

// handleEmbedMessages returns paginated message history for the authenticated user.
// GET /api/embed/messages?before=<cursor>&limit=20
// Requires a valid JWT (guest or student) in Authorization: Bearer header.
func handleEmbedMessages(pool *pgxpool.Pool, tm *auth.TokenManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse JWT from Authorization header.
		authHeader := r.Header.Get("Authorization")
		rawToken := strings.TrimPrefix(authHeader, "Bearer ")
		rawToken = strings.TrimSpace(rawToken)
		if rawToken == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		claims, err := tm.Parse(rawToken, time.Now())
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Parse query parameters.
		before := strings.TrimSpace(r.URL.Query().Get("before"))
		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		if limit > 50 {
			limit = 50
		}

		if pool == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages":  []any{},
				"has_more":  false,
			})
			return
		}

		// Fetch limit+1 to determine has_more.
		var beforePtr *string
		if before != "" {
			beforePtr = &before
		}

		query := `
			SELECT m.id, m.role, m.content, m.created_at
			FROM messages m
			JOIN conversations c ON c.id = m.conversation_id
			WHERE c.user_id = $1 AND c.tenant_id = $2
			  AND m.role IN ('user', 'assistant')
			  AND ($3::uuid IS NULL OR m.created_at < (SELECT created_at FROM messages WHERE id = $3))
			ORDER BY m.created_at DESC
			LIMIT $4
		`

		rows, err := pool.Query(r.Context(), query, claims.Subject, claims.TenantID, beforePtr, limit+1)
		if err != nil {
			slog.Error("embed messages: query", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type message struct {
			ID        string    `json:"id"`
			Role      string    `json:"role"`
			Content   string    `json:"content"`
			CreatedAt time.Time `json:"created_at"`
		}

		var msgs []message
		for rows.Next() {
			var m message
			if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
				slog.Error("embed messages: scan", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			msgs = append(msgs, m)
		}
		if err := rows.Err(); err != nil {
			slog.Error("embed messages: rows", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		hasMore := len(msgs) > limit
		if hasMore {
			msgs = msgs[:limit]
		}

		// Reverse to chronological order (oldest first).
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}

		resp := map[string]any{
			"messages": msgs,
			"has_more": hasMore,
		}
		if hasMore && len(msgs) > 0 {
			resp["next_cursor"] = msgs[0].ID
		}
		if msgs == nil {
			resp["messages"] = []any{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handleAdminGetEmbedConfig returns the embed config for the authenticated tenant.
// GET /api/admin/embed/config
func handleAdminGetEmbedConfig(store chat.EmbedConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}

		cfg, err := store.GetByTenantID(r.Context(), claims.TenantID)
		if err != nil {
			slog.Error("get embed config", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}

// handleAdminUpdateEmbedConfig updates enabled/theme settings.
// PUT /api/admin/embed/config
func handleAdminUpdateEmbedConfig(store chat.EmbedConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}

		var req struct {
			Enabled     *bool          `json:"enabled"`
			ThemeConfig map[string]any `json:"theme_config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Get current config to merge with updates.
		existing, err := store.GetByTenantID(r.Context(), claims.TenantID)
		if err != nil {
			slog.Error("get embed config for update", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if req.Enabled != nil {
			existing.Enabled = *req.Enabled
		}
		if req.ThemeConfig != nil {
			existing.ThemeConfig = req.ThemeConfig
		}
		existing.TenantID = claims.TenantID

		updated, err := store.Upsert(r.Context(), existing)
		if err != nil {
			slog.Error("update embed config", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)
	}
}

// handleAdminAddEmbedOrigin adds an allowed origin.
// POST /api/admin/embed/origins
func handleAdminAddEmbedOrigin(store chat.EmbedConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}

		var req struct {
			Origin string `json:"origin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		origin := strings.TrimSpace(req.Origin)
		if origin == "" {
			http.Error(w, "missing origin", http.StatusBadRequest)
			return
		}
		// Validate origin format: must start with http:// or https://
		if !strings.HasPrefix(origin, "https://") && !strings.HasPrefix(origin, "http://") {
			http.Error(w, "origin must start with http:// or https://", http.StatusBadRequest)
			return
		}

		if err := store.AddOrigin(r.Context(), claims.TenantID, origin); err != nil {
			slog.Error("add embed origin", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// handleAdminDeleteEmbedOrigin removes an allowed origin.
// DELETE /api/admin/embed/origins
func handleAdminDeleteEmbedOrigin(store chat.EmbedConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}

		var req struct {
			Origin string `json:"origin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Origin) == "" {
			http.Error(w, "missing origin", http.StatusBadRequest)
			return
		}

		if err := store.RemoveOrigin(r.Context(), claims.TenantID, req.Origin); err != nil {
			slog.Error("remove embed origin", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

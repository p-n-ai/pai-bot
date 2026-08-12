// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
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

const maxEmbedRequestBytes = 64 << 10

type EmbedGuestService interface {
	IssueGuestToken(ctx context.Context, tenantID, origin, fingerprint string) (string, string, error)
	UpgradeGuest(ctx context.Context, userID, tenantID, parentOrigin, name, email, password string) (string, error)
}

type EmbedMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type EmbedMessageStore interface {
	ListEmbedMessages(ctx context.Context, tenantID, userID, before string, limit int) ([]EmbedMessage, bool, error)
}

type EmbedIdentity struct {
	Channel    string
	ExternalID string
}

type EmbedIdentityResolver interface {
	ResolveEmbedIdentity(ctx context.Context, tenantID, userID string) (EmbedIdentity, error)
}

type EmbedPasswordAuthenticator interface {
	AuthenticatePassword(ctx context.Context, req auth.LoginRequest) (auth.UserSession, error)
}

type PostgresEmbedMessageStore struct {
	pool *pgxpool.Pool
}

func (s *PostgresEmbedMessageStore) ResolveEmbedIdentity(ctx context.Context, tenantID, userID string) (EmbedIdentity, error) {
	var identity EmbedIdentity
	err := s.pool.QueryRow(ctx,
		`SELECT channel, external_id
		 FROM users
		 WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		userID, tenantID,
	).Scan(&identity.Channel, &identity.ExternalID)
	if err != nil {
		return EmbedIdentity{}, err
	}
	return identity, nil
}

func NewPostgresEmbedMessageStore(pool *pgxpool.Pool) *PostgresEmbedMessageStore {
	return &PostgresEmbedMessageStore{pool: pool}
}

func (s *PostgresEmbedMessageStore) ListEmbedMessages(ctx context.Context, tenantID, userID, before string, limit int) ([]EmbedMessage, bool, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id::text, m.role, m.content, m.created_at
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE c.user_id = $1::uuid
		  AND c.tenant_id = $2::uuid
		  AND m.tenant_id = $2::uuid
		  AND m.role IN ('user', 'assistant')
		  AND ($3 = '' OR (m.created_at, m.id) < (
		    SELECT cursor_message.created_at, cursor_message.id
		    FROM messages cursor_message
		    JOIN conversations cursor_conversation ON cursor_conversation.id = cursor_message.conversation_id
		    WHERE cursor_message.id = $3::uuid
		      AND cursor_conversation.user_id = $1::uuid
		      AND cursor_conversation.tenant_id = $2::uuid
		      AND cursor_message.tenant_id = $2::uuid
		  ))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4`, userID, tenantID, before, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	messages := make([]EmbedMessage, 0, limit)
	for rows.Next() {
		var message EmbedMessage
		if err := rows.Scan(&message.ID, &message.Role, &message.Content, &message.CreatedAt); err != nil {
			return nil, false, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, hasMore, nil
}

func registerEmbedRoutes(mux *http.ServeMux, opts TopMuxOptions, manager *auth.TokenManager, tokenTTL time.Duration) {
	if opts.EmbedConfigStore == nil {
		return
	}
	expiresIn := max(int(tokenTTL.Seconds()), 0)

	if opts.EmbedGuestService != nil {
		mux.Handle("POST /api/embed/auth/guest", handleEmbedGuestAuth(opts.EmbedConfigStore, opts.EmbedGuestService, expiresIn))
		mux.Handle("POST /api/embed/auth/upgrade", handleEmbedUpgradeGuest(opts.EmbedConfigStore, opts.EmbedGuestService, manager))
	}
	mux.Handle("GET /api/embed/config", handlePublicEmbedConfig(opts.EmbedConfigStore))
	if opts.EmbedAuthenticator != nil {
		mux.Handle("POST /api/embed/auth/login", handleEmbedLogin(opts.EmbedConfigStore, opts.EmbedAuthenticator, opts.EmbedIdentityResolver, manager, expiresIn))
	}
	if opts.EmbedMessageStore != nil {
		mux.Handle("GET /api/embed/messages", handleEmbedMessages(opts.EmbedConfigStore, opts.EmbedMessageStore, manager))
	}
}

func handlePublicEmbedConfig(store chat.EmbedConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant"))
		if tenant == "" {
			http.Error(w, "missing tenant", http.StatusBadRequest)
			return
		}
		parentOrigin, err := validatedEmbedParentOrigin(r, r.URL.Query().Get("parent_origin"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		tenantID, err := store.FindTenantBySlugAndOrigin(r.Context(), tenant, parentOrigin)
		if err != nil {
			http.Error(w, "embed unavailable", http.StatusForbidden)
			return
		}
		config, err := store.GetByTenantSlug(r.Context(), tenant)
		if err != nil || !config.Enabled || config.TenantID != tenantID {
			http.Error(w, "embed unavailable", http.StatusForbidden)
			return
		}
		normalizeEmbedConfig(&config, tenantID)
		writeJSON(w, http.StatusOK, struct {
			Enabled     bool                  `json:"enabled"`
			ThemeConfig chat.EmbedThemeConfig `json:"theme_config"`
		}{Enabled: true, ThemeConfig: config.ThemeConfig})
	}
}

func handleEmbedGuestAuth(store chat.EmbedConfigStore, guests EmbedGuestService, expiresIn int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Tenant       string `json:"tenant"`
			ParentOrigin string `json:"parent_origin"`
			Fingerprint  string `json:"fingerprint"`
		}
		if !decodeEmbedJSON(w, r, &request) {
			return
		}
		if strings.TrimSpace(request.Tenant) == "" {
			http.Error(w, "missing tenant", http.StatusBadRequest)
			return
		}
		request.Fingerprint = strings.TrimSpace(request.Fingerprint)
		if len(request.Fingerprint) > 128 {
			http.Error(w, "fingerprint is too long", http.StatusBadRequest)
			return
		}
		parentOrigin, err := validatedEmbedParentOrigin(r, request.ParentOrigin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		tenantID, err := store.FindTenantBySlugAndOrigin(r.Context(), request.Tenant, parentOrigin)
		if err != nil {
			if errors.Is(err, chat.ErrEmbedNotConfigured) {
				http.Error(w, "embed not configured for this tenant/origin", http.StatusForbidden)
				return
			}
			slog.Error("embed guest auth tenant lookup failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		token, userID, err := guests.IssueGuestToken(r.Context(), tenantID, parentOrigin, request.Fingerprint)
		if err != nil {
			slog.Error("embed guest token issue failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Token     string `json:"token"`
			UserID    string `json:"user_id"`
			ExpiresIn int    `json:"expires_in"`
		}{Token: token, UserID: userID, ExpiresIn: expiresIn})
	}
}

func handleEmbedUpgradeGuest(store chat.EmbedConfigStore, guests EmbedGuestService, manager *auth.TokenManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := parseAuthorizedEmbedBearer(w, r, store, manager)
		if !ok {
			return
		}
		if claims.Role != auth.RoleGuest {
			http.Error(w, "token must be a guest token", http.StatusForbidden)
			return
		}
		if claims.ParentOrigin == "" {
			http.Error(w, "token is not bound to a parent origin", http.StatusForbidden)
			return
		}
		var request struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !decodeEmbedJSON(w, r, &request) {
			return
		}
		if strings.TrimSpace(request.Name) == "" || !strings.Contains(request.Email, "@") || auth.ValidatePassword(strings.TrimSpace(request.Password)) != nil {
			http.Error(w, "name, valid email, and password of at least 12 characters are required", http.StatusBadRequest)
			return
		}
		token, err := guests.UpgradeGuest(r.Context(), claims.Subject, claims.TenantID, claims.ParentOrigin, request.Name, request.Email, request.Password)
		switch {
		case errors.Is(err, auth.ErrNotGuest):
			http.Error(w, "user is not a guest", http.StatusForbidden)
			return
		case errors.Is(err, auth.ErrEmailAlreadyUsed):
			http.Error(w, "email already in use", http.StatusConflict)
			return
		case err != nil:
			slog.Error("embed guest upgrade failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Token string    `json:"token"`
			Role  auth.Role `json:"role"`
		}{Token: token, Role: auth.RoleStudent})
	}
}

func handleEmbedLogin(store chat.EmbedConfigStore, authSvc EmbedPasswordAuthenticator, identities EmbedIdentityResolver, manager *auth.TokenManager, expiresIn int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Tenant       string `json:"tenant"`
			ParentOrigin string `json:"parent_origin"`
			Email        string `json:"email"`
			Password     string `json:"password"`
		}
		if !decodeEmbedJSON(w, r, &request) {
			return
		}
		if strings.TrimSpace(request.Tenant) == "" || strings.TrimSpace(request.Email) == "" || strings.TrimSpace(request.Password) == "" {
			http.Error(w, "tenant, email, and password are required", http.StatusBadRequest)
			return
		}
		parentOrigin, err := validatedEmbedParentOrigin(r, request.ParentOrigin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		tenantID, err := store.FindTenantBySlugAndOrigin(r.Context(), request.Tenant, parentOrigin)
		if err != nil {
			if errors.Is(err, chat.ErrEmbedNotConfigured) {
				http.Error(w, "embed not configured for this tenant/origin", http.StatusForbidden)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		user, err := authSvc.AuthenticatePassword(r.Context(), auth.LoginRequest{
			TenantID: tenantID,
			Email:    request.Email,
			Password: request.Password,
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}
		if user.Role != auth.RoleStudent {
			http.Error(w, "embed login requires a student account", http.StatusForbidden)
			return
		}
		if identities == nil {
			http.Error(w, "embed identity resolution unavailable", http.StatusServiceUnavailable)
			return
		}
		identity, err := identities.ResolveEmbedIdentity(r.Context(), user.TenantID, user.UserID)
		if err != nil || strings.TrimSpace(identity.Channel) == "" || strings.TrimSpace(identity.ExternalID) == "" {
			slog.Error("embed login identity resolution failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		token, err := manager.Issue(auth.TokenClaims{
			Subject: user.UserID, TenantID: user.TenantID, Role: user.Role, ParentOrigin: parentOrigin,
			Channel: identity.Channel, ExternalID: identity.ExternalID,
		}, time.Now().UTC())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Token     string    `json:"token"`
			UserID    string    `json:"user_id"`
			Role      auth.Role `json:"role"`
			Name      string    `json:"name"`
			ExpiresIn int       `json:"expires_in"`
		}{Token: token, UserID: user.UserID, Role: user.Role, Name: user.Name, ExpiresIn: expiresIn})
	}
}

func handleEmbedMessages(configStore chat.EmbedConfigStore, store EmbedMessageStore, manager *auth.TokenManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := parseAuthorizedEmbedBearer(w, r, configStore, manager)
		if !ok {
			return
		}
		if claims.Role != auth.RoleGuest && claims.Role != auth.RoleStudent {
			http.Error(w, "embed user token required", http.StatusForbidden)
			return
		}
		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
			limit = min(parsed, 50)
		}
		messages, hasMore, err := store.ListEmbedMessages(r.Context(), claims.TenantID, claims.Subject, strings.TrimSpace(r.URL.Query().Get("before")), limit)
		if err != nil {
			slog.Error("embed message history failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		response := struct {
			Messages   []EmbedMessage `json:"messages"`
			HasMore    bool           `json:"has_more"`
			NextCursor string         `json:"next_cursor,omitempty"`
		}{Messages: messages, HasMore: hasMore}
		if hasMore && len(messages) > 0 {
			response.NextCursor = messages[0].ID
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func handleAdminGetEmbedConfig(store chat.EmbedConfigStore, publicBaseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := embedTenantFromClaims(w, r)
		if !ok {
			return
		}
		config, err := store.GetByTenantID(r.Context(), tenantID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		normalizeEmbedConfig(&config, tenantID)
		writeJSON(w, http.StatusOK, newEmbedConfigResponse(config, publicEmbedBaseURL(r, publicBaseURL)))
	}
}

func handleAdminUpdateEmbedConfig(store chat.EmbedConfigStore, publicBaseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := embedTenantFromClaims(w, r)
		if !ok {
			return
		}
		var request struct {
			Enabled     *bool                  `json:"enabled"`
			ThemeConfig *chat.EmbedThemeConfig `json:"theme_config"`
		}
		if !decodeEmbedJSON(w, r, &request) {
			return
		}
		config, err := store.GetByTenantID(r.Context(), tenantID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		config.TenantID = tenantID
		if request.Enabled != nil {
			config.Enabled = *request.Enabled
		}
		if request.ThemeConfig != nil {
			config.ThemeConfig = *request.ThemeConfig
		}
		config, err = store.Upsert(r.Context(), config)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		normalizeEmbedConfig(&config, tenantID)
		writeJSON(w, http.StatusOK, newEmbedConfigResponse(config, publicEmbedBaseURL(r, publicBaseURL)))
	}
}

type embedConfigResponse struct {
	chat.EmbedConfig
	PublicEmbedBaseURL string `json:"public_embed_base_url"`
}

func newEmbedConfigResponse(config chat.EmbedConfig, publicBaseURL string) embedConfigResponse {
	return embedConfigResponse{EmbedConfig: config, PublicEmbedBaseURL: publicBaseURL}
}

func publicEmbedBaseURL(r *http.Request, configured string) string {
	if normalized, err := normalizeWebOrigin(configured); err == nil {
		return normalized
	}
	return requestBaseURL(r)
}

func handleAdminAddEmbedOrigin(store chat.EmbedConfigStore) http.HandlerFunc {
	return handleAdminEmbedOrigin(store, true)
}

func handleAdminDeleteEmbedOrigin(store chat.EmbedConfigStore) http.HandlerFunc {
	return handleAdminEmbedOrigin(store, false)
}

func handleAdminEmbedOrigin(store chat.EmbedConfigStore, add bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := embedTenantFromClaims(w, r)
		if !ok {
			return
		}
		var request struct {
			Origin string `json:"origin"`
		}
		if !decodeEmbedJSON(w, r, &request) {
			return
		}
		origin, err := normalizeWebOrigin(request.Origin)
		if err != nil {
			http.Error(w, "invalid origin", http.StatusBadRequest)
			return
		}
		if add {
			err = store.AddOrigin(r.Context(), tenantID, origin)
		} else {
			err = store.RemoveOrigin(r.Context(), tenantID, origin)
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		status := http.StatusOK
		if add {
			status = http.StatusCreated
		}
		writeJSON(w, status, map[string]string{"origin": origin})
	}
}

func parseEmbedBearer(w http.ResponseWriter, r *http.Request, manager *auth.TokenManager) (auth.TokenClaims, bool) {
	token, err := bearerToken(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "missing authorization", http.StatusUnauthorized)
		return auth.TokenClaims{}, false
	}
	claims, err := manager.Parse(token, time.Now().UTC())
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return auth.TokenClaims{}, false
	}
	return claims, true
}

func decodeEmbedJSON[T any](w http.ResponseWriter, r *http.Request, destination *T) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxEmbedRequestBytes)
	if err := json.NewDecoder(r.Body).Decode(destination); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return false
		}
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

func parseAuthorizedEmbedBearer(
	w http.ResponseWriter,
	r *http.Request,
	store chat.EmbedConfigStore,
	manager *auth.TokenManager,
) (auth.TokenClaims, bool) {
	claims, ok := parseEmbedBearer(w, r, manager)
	if !ok {
		return auth.TokenClaims{}, false
	}
	if strings.TrimSpace(claims.ParentOrigin) == "" {
		http.Error(w, "token is not bound to a parent origin", http.StatusForbidden)
		return auth.TokenClaims{}, false
	}
	requestOrigin, err := normalizeOptionalRequestOrigin(r)
	if err != nil {
		http.Error(w, "valid origin or referer required", http.StatusForbidden)
		return auth.TokenClaims{}, false
	}
	// Same-origin browser GETs may omit both headers. A supplied browser origin
	// still has to match either the parent bound into the token or this server.
	if requestOrigin != "" && requestOrigin != claims.ParentOrigin && requestOrigin != embedServerOrigin(r) {
		http.Error(w, "origin does not match parent origin", http.StatusForbidden)
		return auth.TokenClaims{}, false
	}
	allowed, err := store.IsOriginAllowed(r.Context(), claims.TenantID, claims.ParentOrigin)
	if err != nil {
		slog.Error("embed bearer origin check failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return auth.TokenClaims{}, false
	}
	if !allowed {
		http.Error(w, "embed unavailable", http.StatusForbidden)
		return auth.TokenClaims{}, false
	}
	return claims, true
}

func validatedEmbedParentOrigin(r *http.Request, rawParentOrigin string) (string, error) {
	parentOrigin, err := normalizeWebOrigin(rawParentOrigin)
	if err != nil {
		return "", errors.New("invalid parent_origin")
	}
	requestOrigin, err := normalizeOptionalRequestOrigin(r)
	if err != nil {
		return "", errors.New("invalid origin")
	}
	if requestOrigin != "" && requestOrigin != parentOrigin && requestOrigin != embedServerOrigin(r) {
		return "", errors.New("origin does not match parent_origin")
	}
	return parentOrigin, nil
}

func embedServerOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto == "http" || proto == "https" {
		scheme = proto
	}
	return scheme + "://" + strings.ToLower(strings.TrimSpace(r.Host))
}

func normalizeOptionalRequestOrigin(r *http.Request) (string, error) {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return normalizeWebOrigin(origin)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		parsed, err := url.Parse(referer)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
			return "", errors.New("invalid referer")
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return "", errors.New("invalid referer")
		}
		return scheme + "://" + strings.ToLower(parsed.Host), nil
	}
	return "", nil
}

func normalizeWebOrigin(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("invalid origin")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("invalid origin")
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid origin")
	}
	return scheme + "://" + strings.ToLower(parsed.Host), nil
}

func embedTenantFromClaims(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || strings.TrimSpace(claims.TenantID) == "" {
		http.Error(w, "tenant-scoped authentication required", http.StatusForbidden)
		return "", false
	}
	return claims.TenantID, true
}

func normalizeEmbedConfig(config *chat.EmbedConfig, tenantID string) {
	config.TenantID = tenantID
	if config.AllowedOrigins == nil {
		config.AllowedOrigins = []string{}
	}
}

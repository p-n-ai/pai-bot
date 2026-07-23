// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/p-n-ai/pai-bot/internal/auth"
)

// wsInboundMsg is the JSON envelope clients send over the WebSocket.
type wsInboundMsg struct {
	Type   string `json:"type"`
	UserID string `json:"user_id,omitempty"`
	Text   string `json:"text,omitempty"`
}

// wsOutboundMsg is the JSON envelope the server sends over the WebSocket.
type wsOutboundMsg struct {
	Type        string         `json:"type"`
	Text        string         `json:"text,omitempty"`
	FocusedPage *wsFocusedPage `json:"focused_page,omitempty"`
}

type wsFocusedPage struct {
	URL string `json:"url"`
}

// WSChannel implements the Channel interface for WebSocket connections.
type WSChannel struct {
	mu               sync.RWMutex
	conns            map[string]*websocket.Conn // userID -> connection
	handler          func(InboundMessage)       // set by Start()
	stop             chan struct{}
	embedConfigStore EmbedConfigStore   // nil for non-embed (terminal-chat) use
	tokenManager     *auth.TokenManager // nil for non-embed use
	maxMessageSize   int64              // 0 means no limit
	rateLimiter      *EmbedRateLimiter  // nil for non-embed use
}

// NewWSChannel creates a new WebSocket channel.
func NewWSChannel() *WSChannel {
	return &WSChannel{
		conns: make(map[string]*websocket.Conn),
		stop:  make(chan struct{}),
	}
}

// NewEmbedWSChannel creates a WebSocket channel with embed security features.
func NewEmbedWSChannel(store EmbedConfigStore, tm *auth.TokenManager) *WSChannel {
	return &WSChannel{
		conns:            make(map[string]*websocket.Conn),
		stop:             make(chan struct{}),
		embedConfigStore: store,
		tokenManager:     tm,
		maxMessageSize:   8192, // 8KB default for embed
		rateLimiter:      NewEmbedRateLimiter(10, 30, time.Minute),
	}
}

// Handler returns the HTTP handler for WebSocket upgrades at GET /ws/chat.
func (ws *WSChannel) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var jwtToken string
		var acceptOpts websocket.AcceptOptions

		if ws.embedConfigStore != nil {
			// Embed mode: validate origin.
			origin := r.Header.Get("Origin")
			if origin == "" {
				http.Error(w, "missing origin", http.StatusForbidden)
				return
			}

			// Check for subprotocol auth: Sec-WebSocket-Protocol: pai-auth.<token>
			protocols := strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",")
			for _, p := range protocols {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "pai-auth.") {
					jwtToken = strings.TrimPrefix(p, "pai-auth.")
					acceptOpts.Subprotocols = []string{p}
					break
				}
			}

			// Embed mode requires JWT subprotocol auth — reject first-message auth.
			if jwtToken == "" {
				http.Error(w, "jwt auth required for embed", http.StatusUnauthorized)
				return
			}

			// Validate JWT and extract tenant ID for origin check.
			claims, err := ws.tokenManager.Parse(jwtToken, time.Now().UTC())
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Validate origin against the tenant's allowlist.
			allowed, err := ws.embedConfigStore.IsOriginAllowed(r.Context(), claims.TenantID, origin)
			if err != nil {
				slog.Error("websocket origin check failed", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}

			// IP-based handshake rate limiting.
			if ws.rateLimiter != nil {
				ip := extractClientIP(r)
				if !ws.rateLimiter.AllowHandshake(ip, time.Now()) {
					http.Error(w, "too many connections", http.StatusTooManyRequests)
					return
				}
			}

			// Set CORS headers for the validated origin.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")

			acceptOpts.InsecureSkipVerify = true
		} else {
			// Non-embed (terminal-chat): permissive.
			acceptOpts.InsecureSkipVerify = true
		}

		conn, err := websocket.Accept(w, r, &acceptOpts)
		if err != nil {
			slog.Warn("websocket accept failed", "error", err)
			return
		}

		// Set message size limit for embed connections.
		if ws.maxMessageSize > 0 {
			conn.SetReadLimit(ws.maxMessageSize)
		}

		ws.handleConn(r.Context(), conn, jwtToken)
	})
}

// extractClientIP extracts the client IP from the request, checking
// X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// RemoteAddr is "host:port".
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// handleConn manages a single WebSocket connection lifecycle.
func (ws *WSChannel) handleConn(ctx context.Context, conn *websocket.Conn, jwtToken string) {
	var userID string

	if jwtToken != "" && ws.tokenManager != nil {
		// Subprotocol JWT auth (embed mode — already validated in Handler,
		// but parse again to extract claims after upgrade).
		claims, err := ws.tokenManager.Parse(jwtToken, time.Now().UTC())
		if err != nil {
			slog.Warn("websocket jwt auth failed", "error", err)
			_ = conn.Close(websocket.StatusPolicyViolation, "invalid token")
			return
		}
		userID = claims.Subject
	} else if ws.embedConfigStore == nil {
		// Non-embed (terminal-chat): first message must be auth.
		var err error
		userID, err = ws.readAuth(ctx, conn)
		if err != nil {
			slog.Warn("websocket auth failed", "error", err)
			_ = conn.Close(websocket.StatusPolicyViolation, "auth required")
			return
		}
	} else {
		// Embed mode without JWT — should not reach here (Handler rejects).
		_ = conn.Close(websocket.StatusPolicyViolation, "jwt auth required")
		return
	}

	// Register the connection.
	ws.mu.Lock()
	ws.conns[userID] = conn
	ws.mu.Unlock()

	slog.Info("websocket client connected", "user_id", userID)

	// Send auth_ok.
	if err := ws.writeJSON(ctx, conn, wsOutboundMsg{Type: "auth_ok"}); err != nil {
		slog.Warn("websocket write auth_ok failed", "error", err, "user_id", userID)
		ws.removeConn(userID)
		return
	}

	// Start keepalive pinger.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ws.stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
				err := conn.Ping(pingCtx)
				pingCancel()
				if err != nil {
					slog.Debug("websocket ping failed, closing", "user_id", userID, "error", err)
					_ = conn.Close(websocket.StatusGoingAway, "ping timeout")
					return
				}
			}
		}
	}()

	// Read loop.
	ws.readLoop(ctx, conn, userID)

	// Cleanup on disconnect.
	ws.removeConn(userID)
	slog.Info("websocket client disconnected", "user_id", userID)
}

// readAuth reads and validates the first auth message.
func (ws *WSChannel) readAuth(ctx context.Context, conn *websocket.Conn) (string, error) {
	// Give the client 10 seconds to authenticate.
	authCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, data, err := conn.Read(authCtx)
	if err != nil {
		return "", fmt.Errorf("reading auth message: %w", err)
	}

	var msg wsInboundMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", fmt.Errorf("parsing auth message: %w", err)
	}

	if msg.Type != "auth" || msg.UserID == "" {
		return "", fmt.Errorf("expected auth message with user_id, got type=%q", msg.Type)
	}

	return msg.UserID, nil
}

// readLoop reads messages from the client and dispatches them to the handler.
func (ws *WSChannel) readLoop(ctx context.Context, conn *websocket.Conn, userID string) {
	for {
		select {
		case <-ws.stop:
			_ = conn.Close(websocket.StatusGoingAway, "server shutting down")
			return
		default:
		}

		_, data, err := conn.Read(ctx)
		if err != nil {
			// Connection closed or error — exit loop.
			return
		}

		var msg wsInboundMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			slog.Warn("websocket invalid message", "error", err, "user_id", userID)
			continue
		}

		if msg.Type != "message" {
			slog.Warn("websocket unexpected message type", "type", msg.Type, "user_id", userID)
			continue
		}

		// Content filtering for embed connections.
		if ws.embedConfigStore != nil && containsPromptInjection(msg.Text) {
			slog.Warn("embed content filter triggered", "user_id", userID)
			_ = ws.writeJSON(ctx, conn, wsOutboundMsg{
				Type: "error",
				Text: "Message blocked by content filter.",
			})
			continue
		}

		// Rate limit messages for embed connections.
		if ws.rateLimiter != nil && !ws.rateLimiter.AllowMessage(userID, time.Now()) {
			_ = ws.writeJSON(ctx, conn, wsOutboundMsg{
				Type: "error",
				Text: "Rate limit exceeded. Please slow down.",
			})
			continue
		}

		ws.mu.RLock()
		handler := ws.handler
		ws.mu.RUnlock()

		if handler != nil {
			handler(InboundMessage{
				Channel: "websocket",
				UserID:  userID,
				Text:    msg.Text,
			})
		}
	}
}

// SendMessage sends a response or notification to a connected user.
func (ws *WSChannel) SendMessage(ctx context.Context, userID string, msg OutboundMessage) error {
	ws.mu.RLock()
	conn, ok := ws.conns[userID]
	ws.mu.RUnlock()

	if !ok {
		return fmt.Errorf("websocket: user %q not connected", userID)
	}

	response := wsOutboundMsg{
		Type: "response",
		Text: msg.Text,
	}
	if pageURL := strings.TrimSpace(msg.FocusedPageURL); pageURL != "" {
		response.FocusedPage = &wsFocusedPage{URL: pageURL}
	}
	return ws.writeJSON(ctx, conn, response)
}

// SendTyping sends a typing indicator to the connected user.
// WebSocket clients can use this to show a "typing..." indicator.
func (ws *WSChannel) SendTyping(ctx context.Context, userID string) error {
	ws.mu.RLock()
	conn, ok := ws.conns[userID]
	ws.mu.RUnlock()

	if !ok {
		return fmt.Errorf("websocket: user %q not connected", userID)
	}

	return ws.writeJSON(ctx, conn, wsOutboundMsg{
		Type: "typing",
	})
}

// Start sets the inbound message handler. For WebSocket, actual connection
// handling happens via the HTTP handler — Start just stores the callback.
func (ws *WSChannel) Start(_ context.Context, handler func(InboundMessage)) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.handler = handler
	slog.Info("websocket channel started")
	return nil
}

// Stop closes all active connections and signals the channel to shut down.
func (ws *WSChannel) Stop() error {
	close(ws.stop)

	ws.mu.Lock()
	defer ws.mu.Unlock()

	for userID, conn := range ws.conns {
		_ = conn.Close(websocket.StatusGoingAway, "server shutting down")
		delete(ws.conns, userID)
	}

	slog.Info("websocket channel stopped")
	return nil
}

// SendNotification sends a notification-type message to a connected user.
// This is used for proactive messages (nudges, challenge updates, etc.).
func (ws *WSChannel) SendNotification(ctx context.Context, userID string, text string) error {
	ws.mu.RLock()
	conn, ok := ws.conns[userID]
	ws.mu.RUnlock()

	if !ok {
		return fmt.Errorf("websocket: user %q not connected", userID)
	}

	return ws.writeJSON(ctx, conn, wsOutboundMsg{
		Type: "notification",
		Text: text,
	})
}

// ConnectedUsers returns the list of currently connected user IDs.
func (ws *WSChannel) ConnectedUsers() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	users := make([]string, 0, len(ws.conns))
	for uid := range ws.conns {
		users = append(users, uid)
	}
	return users
}

// writeJSON marshals and writes a JSON message to the connection.
func (ws *WSChannel) writeJSON(ctx context.Context, conn *websocket.Conn, msg wsOutboundMsg) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshalling websocket message: %w", err)
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

// removeConn removes a user's connection from the map.
func (ws *WSChannel) removeConn(userID string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.conns, userID)
}

// containsPromptInjection checks if a message contains common prompt injection markers.
func containsPromptInjection(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"<|system|>",
		"<|im_start|>system",
		"<<sys>>",
		"[inst]",
		"ignore previous instructions",
		"ignore all previous instructions",
		"disregard all previous instructions",
		"forget all previous instructions",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// wsInboundMsg is the JSON envelope clients send over the WebSocket.
type wsInboundMsg struct {
	Type   string `json:"type"`
	UserID string `json:"user_id,omitempty"`
	Text   string `json:"text,omitempty"`
}

// wsOutboundMsg is the JSON envelope the server sends over the WebSocket.
type wsOutboundMsg struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// WSChannel implements the Channel interface for WebSocket connections.
type WSChannel struct {
	mu      sync.RWMutex
	conns   map[string]*websocket.Conn // userID -> connection
	handler func(InboundMessage)       // set by Start()
	stop    chan struct{}
}

// NewWSChannel creates a new WebSocket channel.
func NewWSChannel() *WSChannel {
	return &WSChannel{
		conns: make(map[string]*websocket.Conn),
		stop:  make(chan struct{}),
	}
}

// Handler returns the HTTP handler for WebSocket upgrades at GET /ws/chat.
func (ws *WSChannel) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// Allow any origin for dev use; tighten in production.
			InsecureSkipVerify: true,
		})
		if err != nil {
			slog.Warn("websocket accept failed", "error", err)
			return
		}

		ws.handleConn(r.Context(), conn)
	})
}

// handleConn manages a single WebSocket connection lifecycle.
func (ws *WSChannel) handleConn(ctx context.Context, conn *websocket.Conn) {
	// First message must be auth.
	userID, err := ws.readAuth(ctx, conn)
	if err != nil {
		slog.Warn("websocket auth failed", "error", err)
		_ = conn.Close(websocket.StatusPolicyViolation, "auth required")
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

	return ws.writeJSON(ctx, conn, wsOutboundMsg{
		Type: "response",
		Text: msg.Text,
	})
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
		conn.Close(websocket.StatusGoingAway, "server shutting down")
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

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// dialAndAuth connects to the test server and sends the auth handshake.
func dialAndAuth(t *testing.T, url, userID string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	auth, _ := json.Marshal(wsInboundMsg{Type: "auth", UserID: userID})
	if err := conn.Write(ctx, websocket.MessageText, auth); err != nil {
		t.Fatalf("write auth: %v", err)
	}

	// Read auth_ok.
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read auth_ok: %v", err)
	}
	var resp wsOutboundMsg
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal auth_ok: %v", err)
	}
	if resp.Type != "auth_ok" {
		t.Fatalf("expected auth_ok, got %q", resp.Type)
	}

	return conn
}

func TestWSChannel_ConnectAuthAndMessage(t *testing.T) {
	ws := NewWSChannel()

	var received []InboundMessage
	var mu sync.Mutex
	_ = ws.Start(context.Background(), func(msg InboundMessage) {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
	})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialAndAuth(t, wsURL, "test-user-1")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Send a message.
	ctx := context.Background()
	msg, _ := json.Marshal(wsInboundMsg{Type: "message", Text: "hello world"})
	if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
		t.Fatalf("write message: %v", err)
	}

	// Give the handler a moment to process.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 message, got %d", len(received))
	}
	if received[0].Channel != "websocket" {
		t.Errorf("expected channel websocket, got %q", received[0].Channel)
	}
	if received[0].UserID != "test-user-1" {
		t.Errorf("expected user test-user-1, got %q", received[0].UserID)
	}
	if received[0].Text != "hello world" {
		t.Errorf("expected text 'hello world', got %q", received[0].Text)
	}
}

func TestWSChannel_SendMessageToCorrectUser(t *testing.T) {
	ws := NewWSChannel()
	_ = ws.Start(context.Background(), func(msg InboundMessage) {})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn1 := dialAndAuth(t, wsURL, "user-a")
	defer func() { _ = conn1.Close(websocket.StatusNormalClosure, "") }()

	conn2 := dialAndAuth(t, wsURL, "user-b")
	defer func() { _ = conn2.Close(websocket.StatusNormalClosure, "") }()

	// Send a message to user-a.
	ctx := context.Background()
	err := ws.SendMessage(ctx, "user-a", OutboundMessage{Text: "for user-a"})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	// Read from conn1.
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, data, err := conn1.Read(readCtx)
	if err != nil {
		t.Fatalf("read from conn1: %v", err)
	}

	var resp wsOutboundMsg
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Type != "response" {
		t.Errorf("expected type response, got %q", resp.Type)
	}
	if resp.Text != "for user-a" {
		t.Errorf("expected text 'for user-a', got %q", resp.Text)
	}
}

func TestWSChannel_SendMessageUnknownUserReturnsError(t *testing.T) {
	ws := NewWSChannel()
	_ = ws.Start(context.Background(), func(msg InboundMessage) {})

	err := ws.SendMessage(context.Background(), "nonexistent", OutboundMessage{Text: "hi"})
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("expected 'not connected' in error, got: %v", err)
	}
}

func TestWSChannel_DisconnectRemovesUser(t *testing.T) {
	ws := NewWSChannel()
	_ = ws.Start(context.Background(), func(msg InboundMessage) {})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialAndAuth(t, wsURL, "ephemeral-user")

	// Verify user is connected.
	users := ws.ConnectedUsers()
	found := false
	for _, u := range users {
		if u == "ephemeral-user" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected ephemeral-user in connected users")
	}

	// Disconnect.
	_ = conn.Close(websocket.StatusNormalClosure, "bye")

	// Give the server time to process the disconnect.
	time.Sleep(200 * time.Millisecond)

	// Verify user is removed.
	users = ws.ConnectedUsers()
	for _, u := range users {
		if u == "ephemeral-user" {
			t.Fatal("expected ephemeral-user to be removed after disconnect")
		}
	}
}

func TestWSChannel_MultipleConcurrentConnections(t *testing.T) {
	ws := NewWSChannel()

	var received []InboundMessage
	var mu sync.Mutex
	_ = ws.Start(context.Background(), func(msg InboundMessage) {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
	})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	const numClients = 5
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		userID := "concurrent-user-" + strings.Repeat("x", i+1) // unique IDs
		conns[i] = dialAndAuth(t, wsURL, userID)
		defer func(c *websocket.Conn) { _ = c.Close(websocket.StatusNormalClosure, "") }(conns[i])
	}

	// Each client sends a message.
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg, _ := json.Marshal(wsInboundMsg{Type: "message", Text: "hello from client"})
			if err := conns[idx].Write(ctx, websocket.MessageText, msg); err != nil {
				t.Errorf("client %d write: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	// Give handler time to process.
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != numClients {
		t.Fatalf("expected %d messages, got %d", numClients, len(received))
	}

	// Verify all connected.
	users := ws.ConnectedUsers()
	if len(users) != numClients {
		t.Errorf("expected %d connected users, got %d", numClients, len(users))
	}
}

func TestWSChannel_SendTyping(t *testing.T) {
	ws := NewWSChannel()
	_ = ws.Start(context.Background(), func(msg InboundMessage) {})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialAndAuth(t, wsURL, "typing-user")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	ctx := context.Background()
	if err := ws.SendTyping(ctx, "typing-user"); err != nil {
		t.Fatalf("SendTyping: %v", err)
	}

	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read typing: %v", err)
	}

	var resp wsOutboundMsg
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Type != "typing" {
		t.Errorf("expected type typing, got %q", resp.Type)
	}
}

func TestWSChannel_SendNotification(t *testing.T) {
	ws := NewWSChannel()
	_ = ws.Start(context.Background(), func(msg InboundMessage) {})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialAndAuth(t, wsURL, "notif-user")
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	ctx := context.Background()
	if err := ws.SendNotification(ctx, "notif-user", "Someone joined your challenge!"); err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read notification: %v", err)
	}

	var resp wsOutboundMsg
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Type != "notification" {
		t.Errorf("expected type notification, got %q", resp.Type)
	}
	if resp.Text != "Someone joined your challenge!" {
		t.Errorf("expected notification text, got %q", resp.Text)
	}
}

func TestWSChannel_AuthFailure_NoUserID(t *testing.T) {
	ws := NewWSChannel()
	_ = ws.Start(context.Background(), func(msg InboundMessage) {})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Send auth without user_id.
	auth, _ := json.Marshal(wsInboundMsg{Type: "auth", UserID: ""})
	if err := conn.Write(ctx, websocket.MessageText, auth); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Server should close the connection.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error after invalid auth, got nil")
	}
}

func TestWSChannel_Stop(t *testing.T) {
	ws := NewWSChannel()
	_ = ws.Start(context.Background(), func(msg InboundMessage) {})

	srv := httptest.NewServer(ws.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialAndAuth(t, wsURL, "stop-user")

	// Stop the channel.
	if err := ws.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Connection should be closed.
	time.Sleep(100 * time.Millisecond)
	_, _, err := conn.Read(context.Background())
	if err == nil {
		t.Fatal("expected error after Stop, got nil")
	}

	// No connected users.
	if len(ws.ConnectedUsers()) != 0 {
		t.Error("expected 0 connected users after Stop")
	}
}

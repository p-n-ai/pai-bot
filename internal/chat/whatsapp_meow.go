package chat

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	qrcode "github.com/skip2/go-qrcode"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver for whatsmeow session store (no CGO).

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"google.golang.org/protobuf/proto"
)

// WhatsAppMeowChannel implements the Channel interface using whatsmeow (native Go).
type WhatsAppMeowChannel struct {
	client    *whatsmeow.Client
	container *sqlstore.Container

	handler func(InboundMessage)
	mu      sync.RWMutex

	// latestQR holds the current QR code string for the HTTP endpoint.
	latestQR string
	qrMu     sync.RWMutex
}

// NewWhatsAppMeowChannel creates a WhatsApp channel backed by whatsmeow.
// dbPath is the SQLite path for session persistence (e.g. "file:whatsmeow.db?_foreign_keys=on").
func NewWhatsAppMeowChannel(dbPath string) (*WhatsAppMeowChannel, error) {
	if dbPath == "" {
		dbPath = "file:whatsmeow.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	}

	logger := waLog.Stdout("whatsmeow", "WARN", true)
	// modernc.org/sqlite uses _pragma for PRAGMA settings instead of _foreign_keys.
	// WAL mode + busy_timeout prevents SQLITE_BUSY under concurrent access.
	container, err := sqlstore.New(context.Background(), "sqlite", dbPath, logger)
	if err != nil {
		return nil, fmt.Errorf("whatsmeow store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("whatsmeow device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, logger)

	ch := &WhatsAppMeowChannel{
		client:    client,
		container: container,
	}

	// Register event handler for incoming messages.
	client.AddEventHandler(ch.eventHandler)

	return ch, nil
}

// SendMessage sends a text message to a WhatsApp user via whatsmeow.
func (w *WhatsAppMeowChannel) SendMessage(_ context.Context, userID string, msg OutboundMessage) error {
	jid, err := parseJID(userID)
	if err != nil {
		return err
	}

	_, err = w.client.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(msg.Text),
	})
	if err != nil {
		return fmt.Errorf("whatsmeow send: %w", err)
	}
	return nil
}

// SendTyping sends a typing indicator (composing presence).
func (w *WhatsAppMeowChannel) SendTyping(_ context.Context, userID string) error {
	jid, err := parseJID(userID)
	if err != nil {
		return err
	}
	return w.client.SendChatPresence(context.Background(), jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
}

// Start connects to WhatsApp. If not yet authenticated, it generates a QR code
// available via the QRHandler HTTP endpoint.
func (w *WhatsAppMeowChannel) Start(_ context.Context, handler func(InboundMessage)) error {
	w.mu.Lock()
	w.handler = handler
	w.mu.Unlock()

	if w.client.Store.ID == nil {
		// Not yet authenticated — need QR code scan.
		qrChan, _ := w.client.GetQRChannel(context.Background())
		if err := w.client.Connect(); err != nil {
			return fmt.Errorf("whatsmeow connect: %w", err)
		}
		// Process QR events in background.
		go func() {
			for evt := range qrChan {
				switch evt.Event {
				case "code":
					w.qrMu.Lock()
					w.latestQR = evt.Code
					w.qrMu.Unlock()
					slog.Info("whatsapp QR code ready — open /whatsapp/qr in browser to scan")
				case "success":
					w.qrMu.Lock()
					w.latestQR = ""
					w.qrMu.Unlock()
					slog.Info("whatsapp authenticated successfully")
				case "timeout":
					w.qrMu.Lock()
					w.latestQR = ""
					w.qrMu.Unlock()
					slog.Error("whatsapp QR code timed out — restart to try again")
				}
			}
		}()
	} else {
		// Already authenticated — reconnect.
		if err := w.client.Connect(); err != nil {
			return fmt.Errorf("whatsmeow reconnect: %w", err)
		}
		slog.Info("whatsapp connected (existing session)")
	}

	return nil
}

// Stop disconnects from WhatsApp.
func (w *WhatsAppMeowChannel) Stop() error {
	w.client.Disconnect()
	return nil
}

// IsConnected returns true if the client is currently connected.
func (w *WhatsAppMeowChannel) IsConnected() bool {
	return w.client.IsConnected()
}

// QRHandler returns an HTTP handler that serves the QR code as a PNG image.
// Returns an HTML page with the QR code and auto-refresh, or a status page if already connected.
func (w *WhatsAppMeowChannel) QRHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// If already connected, show status.
		if w.client.Store.ID != nil && w.client.IsConnected() {
			rw.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(rw, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding:60px">
				<h2>&#9989; WhatsApp Connected</h2>
				<p>Session is active. Bot is ready to send and receive messages.</p>
			</body></html>`)
			return
		}

		w.qrMu.RLock()
		qr := w.latestQR
		w.qrMu.RUnlock()

		// Preserve query string for auto-refresh so auth tokens carry through.
		qs := r.URL.RawQuery

		if qr == "" {
			rw.Header().Set("Content-Type", "text/html")
			rw.Header().Set("Refresh", "3")
			_, _ = fmt.Fprint(rw, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding:60px">
				<h2>Waiting for QR code...</h2>
				<p>Page will auto-refresh. If this persists, restart the server.</p>
			</body></html>`)
			return
		}

		// Return QR as PNG if ?format=png
		if r.URL.Query().Get("format") == "png" {
			png, err := qrcode.Encode(qr, qrcode.Medium, 512)
			if err != nil {
				http.Error(rw, "failed to generate QR", http.StatusInternalServerError)
				return
			}
			rw.Header().Set("Content-Type", "image/png")
			_, _ = rw.Write(png)
			return
		}

		// Default: HTML page with embedded QR image and auto-refresh.
		rw.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(rw, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="5; url=/whatsapp/qr?%s"></head>
			<body style="font-family:sans-serif;text-align:center;padding:40px">
			<h2>Scan QR Code with WhatsApp</h2>
			<p>Open WhatsApp &rarr; Settings &rarr; Linked Devices &rarr; Link a Device</p>
			<img src="/whatsapp/qr?format=png&%s" style="margin:20px" />
			<p style="color:#888">Page auto-refreshes every 5 seconds</p>
		</body></html>`, qs, qs)
	})
}

// eventHandler processes whatsmeow events and dispatches inbound messages.
func (w *WhatsAppMeowChannel) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		w.handleMessage(v)
	}
}

func (w *WhatsAppMeowChannel) handleMessage(msg *events.Message) {
	// Ignore messages from self, groups, and status broadcasts.
	if msg.Info.IsFromMe || msg.Info.IsGroup || msg.Info.Chat.Server == "broadcast" {
		return
	}

	text := extractText(msg)
	if text == "" {
		slog.Debug("whatsmeow: ignoring non-text message", "from", msg.Info.Sender.User, "type", msg.Info.Type)
		return
	}

	w.mu.RLock()
	handler := w.handler
	w.mu.RUnlock()

	if handler == nil {
		return
	}

	// Use the full JID string (user@server) as the user ID so replies
	// route correctly for both phone-number JIDs and LID JIDs.
	inbound := InboundMessage{
		Channel:    "whatsapp",
		UserID:     msg.Info.Sender.ToNonAD().String(),
		ExternalID: msg.Info.ID,
		Text:       text,
		FirstName:  msg.Info.PushName,
	}

	handler(inbound)
}

// extractText gets the text content from a whatsmeow message.
func extractText(msg *events.Message) string {
	if msg.Message == nil {
		return ""
	}
	if msg.Message.Conversation != nil {
		return msg.Message.GetConversation()
	}
	if msg.Message.ExtendedTextMessage != nil {
		return msg.Message.ExtendedTextMessage.GetText()
	}
	return ""
}

// parseJID converts a user ID to a WhatsApp JID.
// Accepts full JID strings (e.g. "123@s.whatsapp.net", "123@lid") or bare phone numbers.
func parseJID(userID string) (types.JID, error) {
	if strings.Contains(userID, "@") {
		return types.ParseJID(userID)
	}
	return types.ParseJID(userID + "@s.whatsapp.net")
}

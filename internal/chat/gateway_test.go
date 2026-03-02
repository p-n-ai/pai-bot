package chat_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestNewGateway(t *testing.T) {
	gw := chat.NewGateway()
	if gw == nil {
		t.Fatal("NewGateway() returned nil")
	}
}

func TestGateway_RegisterChannel(t *testing.T) {
	gw := chat.NewGateway()
	mock := &chat.MockChannel{}

	gw.Register("telegram", mock)

	if !gw.HasChannel("telegram") {
		t.Error("HasChannel(telegram) should be true after Register")
	}
}

func TestGateway_HasChannel_NotRegistered(t *testing.T) {
	gw := chat.NewGateway()

	if gw.HasChannel("whatsapp") {
		t.Error("HasChannel(whatsapp) should be false when not registered")
	}
}

func TestGateway_SendMessage(t *testing.T) {
	gw := chat.NewGateway()
	mock := &chat.MockChannel{}
	gw.Register("telegram", mock)

	err := gw.Send(context.Background(), chat.OutboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "Hello!",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if len(mock.SentMessages) != 1 {
		t.Errorf("SentMessages = %d, want 1", len(mock.SentMessages))
	}
}

func TestGateway_SendMessage_UnknownChannel(t *testing.T) {
	gw := chat.NewGateway()

	err := gw.Send(context.Background(), chat.OutboundMessage{
		Channel: "unknown",
		UserID:  "123",
		Text:    "Hello!",
	})
	if err == nil {
		t.Error("Send() should error for unknown channel")
	}
}

func TestInboundMessage_Fields(t *testing.T) {
	msg := chat.InboundMessage{
		Channel:    "telegram",
		UserID:     "123456",
		ExternalID: "tg_123456",
		Text:       "Hello bot",
		Username:   "testuser",
	}
	if msg.Channel != "telegram" {
		t.Errorf("Channel = %q, want telegram", msg.Channel)
	}
	if msg.UserID != "123456" {
		t.Errorf("UserID = %q, want 123456", msg.UserID)
	}
}

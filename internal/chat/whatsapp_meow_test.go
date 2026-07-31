// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestWhatsAppMeowHandleMessageUsesSenderIdentityAndMessageDelivery(t *testing.T) {
	sender := types.NewADJID("60123456789", types.WhatsAppDomain, 2)
	var got InboundMessage
	channel := &WhatsAppMeowChannel{
		handler: func(message InboundMessage) { got = message },
	}

	channel.handleMessage(&events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   sender,
				Sender: sender,
			},
			ID:       types.MessageID("wamid.meow"),
			PushName: "Alya",
		},
		Message: &waE2E.Message{Conversation: proto.String("hello from whatsmeow")},
	})

	wantSender := sender.ToNonAD().String()
	if got.UserID != wantSender || got.ExternalID != wantSender || got.ThreadID != wantSender {
		t.Fatalf(
			"sender identity = user:%q external:%q thread:%q, want %q",
			got.UserID,
			got.ExternalID,
			got.ThreadID,
			wantSender,
		)
	}
	if got.MessageID != "wamid.meow" || got.DeliveryID != "wamid.meow" {
		t.Fatalf("message identity = %q/%q, want whatsmeow delivery wamid.meow", got.MessageID, got.DeliveryID)
	}
	if got.Text != "hello from whatsmeow" || got.FirstName != "Alya" {
		t.Fatalf("message = text:%q name:%q, want normalized whatsmeow content", got.Text, got.FirstName)
	}
}

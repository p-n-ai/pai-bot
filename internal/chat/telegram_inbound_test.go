package chat

import "testing"

func TestMapTelegramInbound_TextMessage(t *testing.T) {
	msg, ok := mapTelegramInbound(tgUpdate{
		UpdateID: 1,
		Message: &tgMessage{
			Text: "hello",
			Chat: tgChat{ID: 123},
			From: tgUser{ID: 456, Username: "u1"},
		},
	})
	if !ok {
		t.Fatal("expected text update to map")
	}
	if msg.Text != "hello" {
		t.Fatalf("Text = %q, want hello", msg.Text)
	}
	if msg.HasImage {
		t.Fatalf("HasImage = true, want false")
	}
}

func TestMapTelegramInbound_PhotoWithCaption(t *testing.T) {
	msg, ok := mapTelegramInbound(tgUpdate{
		UpdateID: 2,
		Message: &tgMessage{
			Caption: "solve this",
			Photo: []tgPhoto{
				{FileID: "small"},
				{FileID: "large"},
			},
			Chat: tgChat{ID: 123},
			From: tgUser{ID: 456},
		},
	})
	if !ok {
		t.Fatal("expected photo update to map")
	}
	if msg.Text != "solve this" {
		t.Fatalf("Text = %q, want solve this", msg.Text)
	}
	if !msg.HasImage {
		t.Fatalf("HasImage = false, want true")
	}
	if msg.ImageFileID != "large" {
		t.Fatalf("ImageFileID = %q, want large", msg.ImageFileID)
	}
}

func TestMapTelegramInbound_PhotoOnly(t *testing.T) {
	msg, ok := mapTelegramInbound(tgUpdate{
		UpdateID: 3,
		Message: &tgMessage{
			Photo: []tgPhoto{
				{FileID: "p1"},
			},
			Chat: tgChat{ID: 789},
			From: tgUser{ID: 111},
		},
	})
	if !ok {
		t.Fatal("expected photo-only update to map")
	}
	if msg.Text != "" {
		t.Fatalf("Text = %q, want empty", msg.Text)
	}
	if !msg.HasImage {
		t.Fatalf("HasImage = false, want true")
	}
	if msg.ImageFileID != "p1" {
		t.Fatalf("ImageFileID = %q, want p1", msg.ImageFileID)
	}
}

func TestMapTelegramInbound_EmptyMessage(t *testing.T) {
	_, ok := mapTelegramInbound(tgUpdate{
		UpdateID: 4,
		Message: &tgMessage{
			Chat: tgChat{ID: 1},
			From: tgUser{ID: 2},
		},
	})
	if ok {
		t.Fatal("expected empty message to be ignored")
	}
}

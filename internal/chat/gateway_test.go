// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

type recordingChannel struct {
	name       string
	starts     int
	stops      int
	typing     []string
	sentTo     []string
	startOrder *[]string
	onStart    func()
	startErr   error
	stopErr    error
}

type recordingWebhookChannel struct {
	recordingChannel
	handler http.Handler
}

func (c *recordingWebhookChannel) WebhookHandler(func(chat.InboundMessage)) http.Handler {
	return c.handler
}

func (c *recordingChannel) SendMessage(_ context.Context, destination string, _ chat.OutboundMessage) error {
	c.sentTo = append(c.sentTo, destination)
	return nil
}

func (c *recordingChannel) SendTyping(_ context.Context, userID string) error {
	c.typing = append(c.typing, userID)
	return nil
}

func (c *recordingChannel) Start(context.Context, func(chat.InboundMessage)) error {
	c.starts++
	if c.startOrder != nil {
		*c.startOrder = append(*c.startOrder, c.name)
	}
	if c.onStart != nil {
		c.onStart()
	}
	return c.startErr
}

func (c *recordingChannel) Stop() error {
	c.stops++
	return c.stopErr
}

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

func TestGateway_ChannelNamesAreStable(t *testing.T) {
	gw := chat.NewGateway()
	gw.Register("whatsapp", &recordingChannel{})
	gw.Register("telegram", &recordingChannel{})

	if got, want := gw.ChannelNames(), []string{"telegram", "whatsapp"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ChannelNames() = %v, want %v", got, want)
	}
}

func TestGateway_SendTyping(t *testing.T) {
	gw := chat.NewGateway()
	channel := &recordingChannel{}
	gw.Register("telegram", channel)

	if err := gw.SendTyping(context.Background(), "telegram", "student-1"); err != nil {
		t.Fatalf("SendTyping() error = %v", err)
	}
	if got, want := channel.typing, []string{"student-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("typing users = %v, want %v", got, want)
	}
}

func TestGateway_SendUsesThreadRoute(t *testing.T) {
	gw := chat.NewGateway()
	channel := &recordingChannel{}
	gw.Register("slack", channel)

	err := gw.Send(context.Background(), chat.OutboundMessage{
		Channel:  "slack",
		UserID:   "U123",
		ThreadID: "slack:C123:1725000000.000100",
		Text:     "Tutor reply",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got, want := channel.sentTo, []string{"slack:C123:1725000000.000100"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("destinations = %v, want %v", got, want)
	}
}

func TestGatewayDiscoversFutureWebhookAdapters(t *testing.T) {
	gw := chat.NewGateway()
	expected := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	gw.Register("future-chat", &recordingWebhookChannel{handler: expected})
	gw.Register("polling-only", &recordingChannel{})

	webhooks := gw.Webhooks(func(chat.InboundMessage) {})
	if len(webhooks) != 1 || webhooks["future-chat"] == nil {
		t.Fatalf("Webhooks() = %#v, want future-chat only", webhooks)
	}
}

func TestGateway_StartAllStartsEachChannel(t *testing.T) {
	gw := chat.NewGateway()
	var order []string
	telegram := &recordingChannel{name: "telegram", startOrder: &order}
	whatsapp := &recordingChannel{name: "whatsapp", startOrder: &order}
	gw.Register("telegram", telegram)
	gw.Register("whatsapp", whatsapp)

	if err := gw.StartAll(context.Background(), func(chat.InboundMessage) {}); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}
	if telegram.starts != 1 || whatsapp.starts != 1 {
		t.Fatalf("start calls = telegram:%d whatsapp:%d, want each once", telegram.starts, whatsapp.starts)
	}
	if want := []string{"telegram", "whatsapp"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("start order = %v, want %v", order, want)
	}
}

func TestGateway_StartAllDoesNotHoldLockWhileStartingChannel(t *testing.T) {
	gw := chat.NewGateway()
	late := &recordingChannel{}
	first := &recordingChannel{
		onStart: func() {
			gw.Register("late", late)
		},
	}
	gw.Register("first", first)

	done := make(chan error, 1)
	go func() {
		done <- gw.StartAll(context.Background(), func(chat.InboundMessage) {})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("StartAll() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("StartAll() blocked while a channel registered another channel")
	}
	if !gw.HasChannel("late") {
		t.Fatal("channel registered during StartAll() was not retained")
	}
	if late.starts != 0 {
		t.Fatalf("late channel starts = %d, want 0 for the current startup snapshot", late.starts)
	}
}

func TestGateway_StartAllRollsBackStartedChannelsAfterFailure(t *testing.T) {
	gw := chat.NewGateway()
	discord := &recordingChannel{}
	slack := &recordingChannel{startErr: errors.New("Slack startup failed")}
	teams := &recordingChannel{}
	gw.Register("discord", discord)
	gw.Register("slack", slack)
	gw.Register("teams", teams)

	err := gw.StartAll(context.Background(), func(chat.InboundMessage) {})
	if err == nil || !errors.Is(err, slack.startErr) {
		t.Fatalf("StartAll() error = %v, want Slack startup error", err)
	}
	if discord.starts != 1 || discord.stops != 1 {
		t.Fatalf("Discord lifecycle = starts:%d stops:%d, want startup rolled back", discord.starts, discord.stops)
	}
	if slack.starts != 1 || slack.stops != 0 {
		t.Fatalf("Slack lifecycle = starts:%d stops:%d, want failed adapter not stopped", slack.starts, slack.stops)
	}
	if teams.starts != 0 {
		t.Fatalf("Teams starts = %d, want not started after failure", teams.starts)
	}
}

func TestGateway_StopAllStopsEveryChannelWhenOneFails(t *testing.T) {
	gw := chat.NewGateway()
	telegram := &recordingChannel{stopErr: errors.New("telegram shutdown failed")}
	whatsapp := &recordingChannel{stopErr: errors.New("whatsapp shutdown failed")}
	gw.Register("telegram", telegram)
	gw.Register("whatsapp", whatsapp)

	err := gw.StopAll()
	if err == nil || !errors.Is(err, telegram.stopErr) || !errors.Is(err, whatsapp.stopErr) {
		t.Fatalf("StopAll() error = %v, want both shutdown errors", err)
	}
	if telegram.stops != 1 || whatsapp.stops != 1 {
		t.Fatalf("stop calls = telegram:%d whatsapp:%d, want each once", telegram.stops, whatsapp.stops)
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

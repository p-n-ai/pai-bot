// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

var (
	ErrInboundDeliveryLeaseLost = errors.New("inbound delivery lease lost")
	ErrInboundDeliveryConflict  = errors.New("inbound delivery identity conflict")
)

type inboundDeliveryStatus string

const (
	inboundDeliveryReceived        inboundDeliveryStatus = "received"
	inboundDeliveryProcessing      inboundDeliveryStatus = "processing"
	inboundDeliveryDeliveryPending inboundDeliveryStatus = "delivery_pending"
	inboundDeliveryDelivering      inboundDeliveryStatus = "delivering"
	inboundDeliveryDelivered       inboundDeliveryStatus = "delivered"
)

type inboundDelivery struct {
	ID                     string
	TenantID               string
	Channel                string
	DeliveryID             string
	LearnerKey             string
	DestinationKey         string
	AcceptedSequence       int64
	Message                chat.InboundMessage
	Result                 agent.TurnResult
	Status                 inboundDeliveryStatus
	ProcessingAttemptCount int
	DeliveryAttemptCount   int
	NextAttemptAt          time.Time
	LeaseToken             string
	LeaseExpiresAt         *time.Time
	DeliveredAt            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type inboundDeliveryAcceptInput struct {
	TenantID       string
	Channel        string
	DeliveryID     string
	LearnerKey     string
	DestinationKey string
	Message        chat.InboundMessage
}

type inboundDeliveryStore interface {
	Accept(context.Context, inboundDeliveryAcceptInput, time.Time) (inboundDelivery, bool, error)
	ClaimDue(context.Context, string, time.Time, time.Time) (inboundDelivery, bool, error)
	RenewLease(context.Context, string, string, inboundDeliveryStatus, time.Time, time.Time) error
	CompleteProcessing(context.Context, string, string, agent.TurnResult, time.Time) error
	ScheduleDeliveryRetry(context.Context, string, string, time.Time, time.Time) error
	MarkDelivered(context.Context, string, string, time.Time) error
	RecoverExpiredProcessing(context.Context, agent.TurnResult, time.Time, int) (int64, error)
	DeleteDeliveredBefore(context.Context, time.Time, int) (int64, error)
}

type inboundMessagePayload struct {
	Version           int    `json:"version"`
	Channel           string `json:"channel"`
	UserID            string `json:"user_id"`
	TenantID          string `json:"tenant_id"`
	InternalUserID    string `json:"internal_user_id,omitempty"`
	IdentityChannel   string `json:"identity_channel,omitempty"`
	ExternalID        string `json:"external_id,omitempty"`
	ThreadID          string `json:"thread_id,omitempty"`
	MessageID         string `json:"message_id,omitempty"`
	DeliveryID        string `json:"delivery_id"`
	Text              string `json:"text,omitempty"`
	Caption           string `json:"caption,omitempty"`
	HasImage          bool   `json:"has_image,omitempty"`
	ImageFileID       string `json:"image_file_id,omitempty"`
	ImageDataURL      string `json:"image_data_url,omitempty"`
	ReplyToText       string `json:"reply_to_text,omitempty"`
	Username          string `json:"username,omitempty"`
	FirstName         string `json:"first_name,omitempty"`
	LastName          string `json:"last_name,omitempty"`
	Language          string `json:"language,omitempty"`
	CallbackQueryID   string `json:"callback_query_id,omitempty"`
	CallbackMessageID int    `json:"callback_message_id,omitempty"`
}

type inboundTurnResultPayload struct {
	Version     int                         `json:"version"`
	Text        string                      `json:"text"`
	FocusedPage *inboundFocusedPageArtifact `json:"focused_page,omitempty"`
}

type inboundFocusedPageArtifact struct {
	PublicID  string    `json:"public_id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	TenantID  string    `json:"tenant_id"`
	TurnID    string    `json:"turn_id"`
}

func newInboundMessagePayload(message chat.InboundMessage) inboundMessagePayload {
	return inboundMessagePayload{
		Version: 1, Channel: message.Channel, UserID: message.UserID, TenantID: message.TenantID,
		InternalUserID: message.InternalUserID, IdentityChannel: message.IdentityChannel,
		ExternalID: message.ExternalID, ThreadID: message.ThreadID, MessageID: message.MessageID,
		DeliveryID: message.DeliveryID, Text: message.Text, Caption: message.Caption,
		HasImage: message.HasImage, ImageFileID: message.ImageFileID, ImageDataURL: message.ImageDataURL,
		ReplyToText: message.ReplyToText, Username: message.Username, FirstName: message.FirstName,
		LastName: message.LastName, Language: message.Language, CallbackQueryID: message.CallbackQueryID,
		CallbackMessageID: message.CallbackMessageID,
	}
}

func (p inboundMessagePayload) message() (chat.InboundMessage, error) {
	if p.Version != 1 {
		return chat.InboundMessage{}, fmt.Errorf("unsupported inbound message payload version %d", p.Version)
	}
	return chat.InboundMessage{
		Channel: p.Channel, UserID: p.UserID, TenantID: p.TenantID, InternalUserID: p.InternalUserID,
		IdentityChannel: p.IdentityChannel, ExternalID: p.ExternalID, ThreadID: p.ThreadID,
		MessageID: p.MessageID, DeliveryID: p.DeliveryID, Text: p.Text, Caption: p.Caption,
		HasImage: p.HasImage, ImageFileID: p.ImageFileID, ImageDataURL: p.ImageDataURL,
		ReplyToText: p.ReplyToText, Username: p.Username, FirstName: p.FirstName, LastName: p.LastName,
		Language: p.Language, CallbackQueryID: p.CallbackQueryID, CallbackMessageID: p.CallbackMessageID,
	}, nil
}

func newInboundTurnResultPayload(result agent.TurnResult) inboundTurnResultPayload {
	payload := inboundTurnResultPayload{Version: 1, Text: result.Text}
	if result.FocusedPage != nil {
		payload.FocusedPage = &inboundFocusedPageArtifact{
			PublicID: result.FocusedPage.PublicID, URL: result.FocusedPage.URL,
			ExpiresAt: result.FocusedPage.ExpiresAt, TenantID: result.FocusedPage.TenantID,
			TurnID: result.FocusedPage.TurnID,
		}
	}
	return payload
}

func (p inboundTurnResultPayload) result() (agent.TurnResult, error) {
	if p.Version != 1 {
		return agent.TurnResult{}, fmt.Errorf("unsupported inbound turn result payload version %d", p.Version)
	}
	result := agent.TurnResult{Text: p.Text}
	if p.FocusedPage != nil {
		result.FocusedPage = &focusedpage.Artifact{
			PublicID: p.FocusedPage.PublicID, URL: p.FocusedPage.URL,
			ExpiresAt: p.FocusedPage.ExpiresAt, TenantID: p.FocusedPage.TenantID,
			TurnID: p.FocusedPage.TurnID,
		}
	}
	return result, nil
}

func validateInboundDeliveryAcceptInput(input inboundDeliveryAcceptInput) error {
	for name, value := range map[string]string{
		"tenant": input.TenantID, "channel": input.Channel, "delivery ID": input.DeliveryID,
		"learner key": input.LearnerKey, "destination key": input.DestinationKey,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("inbound delivery %s is required", name)
		}
	}
	if input.Message.TenantID != input.TenantID ||
		input.Message.Channel != input.Channel ||
		input.Message.DeliveryID != input.DeliveryID {
		return errors.New("inbound delivery identity does not match its message")
	}
	return nil
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const telegramMaxMessageLen = 4096

// TelegramChannel implements the Channel interface for Telegram Bot API.
type TelegramChannel struct {
	token    string
	baseURL  string
	client   *http.Client
	offset   int
	stop     chan struct{}
	stopOnce sync.Once

	devMode bool
}

// NewTelegramChannel creates a Telegram channel adapter.
func NewTelegramChannel(token string) (*TelegramChannel, error) {
	if token == "" {
		return nil, fmt.Errorf("telegram bot token is required (LEARN_TELEGRAM_BOT_TOKEN)")
	}
	return &TelegramChannel{
		token:   token,
		baseURL: "https://api.telegram.org/bot" + token,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		stop: make(chan struct{}),
	}, nil
}

// SetDevMode enables dev commands in the Telegram command menu.
func (t *TelegramChannel) SetDevMode(enabled bool) {
	t.devMode = enabled
}

func (t *TelegramChannel) SendTyping(ctx context.Context, userID string) error {
	chatID, messageThreadID := parseTelegramThreadID(userID)
	params := url.Values{
		"chat_id": {chatID},
		"action":  {"typing"},
	}
	if messageThreadID != "" {
		params.Set("message_thread_id", messageThreadID)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		t.baseURL+"/sendChatAction",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return fmt.Errorf("creating typing indicator request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(request)
	if err != nil {
		return telegramTransportError(ctx, "sending typing indicator")
	}
	_ = resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("sending typing indicator: Telegram returned status %d", resp.StatusCode)
	}
	return nil
}

func (t *TelegramChannel) SendMessage(ctx context.Context, userID string, msg OutboundMessage) error {
	chatID, messageThreadID := parseTelegramThreadID(userID)
	parts := SplitMessage(msg.Text, telegramMaxMessageLen)

	for i, part := range parts {
		params := url.Values{
			"chat_id": {chatID},
			"text":    {part},
		}
		if messageThreadID != "" {
			params.Set("message_thread_id", messageThreadID)
		}
		if msg.ParseMode != "" {
			params.Set("parse_mode", msg.ParseMode)
		}
		if i == len(parts)-1 && len(msg.InlineKeyboard) > 0 {
			type tgInlineButton struct {
				Text         string `json:"text"`
				CallbackData string `json:"callback_data,omitempty"`
				URL          string `json:"url,omitempty"`
			}
			inlineKeyboard := make([][]tgInlineButton, 0, len(msg.InlineKeyboard))
			for _, row := range msg.InlineKeyboard {
				inlineRow := make([]tgInlineButton, 0, len(row))
				for _, btn := range row {
					inlineRow = append(inlineRow, tgInlineButton(btn))
				}
				inlineKeyboard = append(inlineKeyboard, inlineRow)
			}
			replyMarkup := map[string]any{
				"inline_keyboard": inlineKeyboard,
			}
			b, err := json.Marshal(replyMarkup)
			if err != nil {
				return fmt.Errorf("marshal telegram inline reply markup: %w", err)
			}
			params.Set("reply_markup", string(b))
		}
		if len(msg.ReplyKeyboard) > 0 {
			replyMarkup := map[string]any{
				"keyboard":        msg.ReplyKeyboard,
				"resize_keyboard": true,
				"is_persistent":   true,
			}
			b, err := json.Marshal(replyMarkup)
			if err != nil {
				return fmt.Errorf("marshal telegram reply markup: %w", err)
			}
			params.Set("reply_markup", string(b))
		}

		resp, err := t.postForm(ctx, "/sendMessage", params)
		if err != nil {
			return fmt.Errorf("sending Telegram message: %w", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// If Markdown parsing fails, retry without parse mode
			if msg.ParseMode != "" && resp.StatusCode == http.StatusBadRequest {
				slog.Warn("Telegram markdown parse failed, retrying plain")
				params.Del("parse_mode")
				retryResp, retryErr := t.postForm(ctx, "/sendMessage", params)
				if retryErr != nil {
					return fmt.Errorf("sending Telegram message (retry): %w", retryErr)
				}
				_ = retryResp.Body.Close()
				if retryResp.StatusCode != http.StatusOK {
					return fmt.Errorf("telegram API error %d on retry", retryResp.StatusCode)
				}
				continue
			}
			return fmt.Errorf("telegram API error %d", resp.StatusCode)
		}
	}

	return nil
}

func (t *TelegramChannel) Start(ctx context.Context, handler func(InboundMessage)) error {
	if err := t.syncCommands(); err != nil {
		slog.Warn("failed to sync Telegram commands", "error", err)
	}
	go t.pollLoop(ctx, handler)
	return nil
}

func (t *TelegramChannel) Stop() error {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	return nil
}

func (t *TelegramChannel) pollLoop(ctx context.Context, handler func(InboundMessage)) {
	slog.Info("Telegram long-polling started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stop:
			return
		default:
			updates, err := t.getUpdates(ctx)
			if err != nil {
				slog.Error("Telegram getUpdates error", "error", err)
				select {
				case <-ctx.Done():
					return
				case <-t.stop:
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}

			for _, u := range updates {
				t.offset = u.UpdateID + 1
				msg, ok := mapTelegramInbound(u)
				if !ok {
					continue
				}
				if msg.HasImage && msg.ImageFileID != "" {
					dataURL, err := t.getImageDataURL(ctx, msg.ImageFileID)
					if err != nil {
						slog.Warn("failed to fetch telegram image", "error", err)
					} else {
						msg.ImageDataURL = dataURL
					}
				}
				if msg.CallbackQueryID != "" {
					if err := t.answerCallbackQuery(ctx, msg.CallbackQueryID); err != nil {
						slog.Warn("failed to answer callback query", "error", err)
					}
				}

				go handler(msg)
			}
		}
	}
}

func (t *TelegramChannel) getUpdates(ctx context.Context) ([]tgUpdate, error) {
	params := url.Values{
		"offset":  {strconv.Itoa(t.offset)},
		"timeout": {"30"},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", t.baseURL+"/getUpdates?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, telegramTransportError(ctx, "Telegram getUpdates")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API returned ok=false")
	}

	return result.Result, nil
}

// Telegram API types (minimal)
type tgUpdate struct {
	UpdateID      int              `json:"update_id"`
	Message       *tgMessage       `json:"message"`
	CallbackQuery *tgCallbackQuery `json:"callback_query,omitempty"`
}

type tgMessage struct {
	MessageID       int         `json:"message_id"`
	MessageThreadID int         `json:"message_thread_id,omitempty"`
	Text            string      `json:"text"`
	Caption         string      `json:"caption"`
	Photo           []tgPhoto   `json:"photo,omitempty"`
	Document        *tgDocument `json:"document,omitempty"`
	Chat            tgChat      `json:"chat"`
	From            tgUser      `json:"from"`
	ReplyToMessage  *tgMessage  `json:"reply_to_message,omitempty"`
}

type tgPhoto struct {
	FileID string `json:"file_id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type tgDocument struct {
	FileID   string `json:"file_id"`
	MimeType string `json:"mime_type,omitempty"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	LanguageCode string `json:"language_code"`
}

type tgCallbackQuery struct {
	ID      string     `json:"id"`
	From    tgUser     `json:"from"`
	Data    string     `json:"data"`
	Message *tgMessage `json:"message,omitempty"`
}

// SplitMessage splits text into chunks that fit Telegram's max message length.
func SplitMessage(text string, maxLen int) []string {
	if text == "" {
		return nil
	}
	if len(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			parts = append(parts, text)
			break
		}
		// Find last newline or space within limit
		cutAt := maxLen
		if idx := strings.LastIndex(text[:maxLen], "\n"); idx > 0 {
			cutAt = idx + 1
		} else if idx := strings.LastIndex(text[:maxLen], " "); idx > 0 {
			cutAt = idx + 1
		}
		parts = append(parts, text[:cutAt])
		text = text[cutAt:]
	}
	return parts
}

func mapTelegramInbound(u tgUpdate) (InboundMessage, bool) {
	if u.CallbackQuery != nil {
		cb := u.CallbackQuery
		if cb.Message == nil {
			return InboundMessage{}, false
		}
		data := strings.TrimSpace(cb.Data)
		if data == "" {
			return InboundMessage{}, false
		}
		return InboundMessage{
			Channel:           "telegram",
			UserID:            strconv.FormatInt(cb.From.ID, 10),
			ExternalID:        strconv.FormatInt(cb.From.ID, 10),
			ThreadID:          telegramThreadID(cb.Message.Chat.ID, cb.Message.MessageThreadID),
			MessageID:         strconv.Itoa(cb.Message.MessageID),
			DeliveryID:        strconv.Itoa(u.UpdateID),
			Text:              data,
			Username:          cb.From.Username,
			FirstName:         cb.From.FirstName,
			LastName:          cb.From.LastName,
			Language:          cb.From.LanguageCode,
			CallbackQueryID:   cb.ID,
			CallbackMessageID: cb.Message.MessageID,
		}, true
	}

	if u.Message == nil {
		return InboundMessage{}, false
	}

	text := strings.TrimSpace(u.Message.Text)
	caption := strings.TrimSpace(u.Message.Caption)
	if text == "" && caption != "" {
		text = caption
	}

	imageFileID := pickImageFileID(u.Message)
	hasImage := imageFileID != ""
	if text == "" && !hasImage {
		return InboundMessage{}, false
	}

	msg := InboundMessage{
		Channel:    "telegram",
		UserID:     strconv.FormatInt(u.Message.From.ID, 10),
		ExternalID: strconv.FormatInt(u.Message.From.ID, 10),
		ThreadID:   telegramThreadID(u.Message.Chat.ID, u.Message.MessageThreadID),
		MessageID:  strconv.Itoa(u.Message.MessageID),
		DeliveryID: strconv.Itoa(u.UpdateID),
		Text:       text,
		Caption:    caption,
		HasImage:   hasImage,
		Username:   u.Message.From.Username,
		FirstName:  u.Message.From.FirstName,
		LastName:   u.Message.From.LastName,
		Language:   u.Message.From.LanguageCode,
	}
	if hasImage {
		msg.ImageFileID = imageFileID
	}
	if u.Message.ReplyToMessage != nil {
		if u.Message.ReplyToMessage.Text != "" {
			msg.ReplyToText = u.Message.ReplyToMessage.Text
		} else if u.Message.ReplyToMessage.Caption != "" {
			msg.ReplyToText = u.Message.ReplyToMessage.Caption
		}
		// If this message is a reply to media and the current message has no media,
		// carry the replied image forward so AI can answer follow-up questions.
		if msg.ImageFileID == "" {
			if replyImageFileID := pickImageFileID(u.Message.ReplyToMessage); replyImageFileID != "" {
				msg.HasImage = true
				msg.ImageFileID = replyImageFileID
			}
		}
	}

	return msg, true
}

func telegramThreadID(chatID int64, messageThreadID int) string {
	threadID := "telegram:" + strconv.FormatInt(chatID, 10)
	if messageThreadID != 0 {
		threadID += ":" + strconv.Itoa(messageThreadID)
	}
	return threadID
}

func parseTelegramThreadID(route string) (chatID, messageThreadID string) {
	encoded, ok := strings.CutPrefix(route, "telegram:")
	if !ok {
		return route, ""
	}
	chatID, messageThreadID, _ = strings.Cut(encoded, ":")
	return chatID, messageThreadID
}

func (t *TelegramChannel) answerCallbackQuery(ctx context.Context, callbackQueryID string) error {
	params := url.Values{
		"callback_query_id": {callbackQueryID},
	}
	resp, err := t.postForm(ctx, "/answerCallbackQuery", params)
	if err != nil {
		return fmt.Errorf("answer callback query: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram answerCallbackQuery error %d", resp.StatusCode)
	}
	return nil
}

func (t *TelegramChannel) postForm(ctx context.Context, endpoint string, params url.Values) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		t.baseURL+endpoint,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := t.client.Do(request)
	if err != nil {
		return nil, telegramTransportError(ctx, "Telegram API")
	}
	return response, nil
}

func pickImageFileID(m *tgMessage) string {
	if m == nil {
		return ""
	}
	if len(m.Photo) > 0 {
		// Telegram sends photos in ascending size order. Keep the largest (last).
		return m.Photo[len(m.Photo)-1].FileID
	}
	if m.Document != nil && strings.HasPrefix(strings.ToLower(m.Document.MimeType), "image/") {
		return m.Document.FileID
	}
	return ""
}

func (t *TelegramChannel) getImageDataURL(ctx context.Context, fileID string) (string, error) {
	filePath, err := t.getFilePath(ctx, fileID)
	if err != nil {
		return "", err
	}

	downloadURL := "https://api.telegram.org/file/bot" + t.token + "/" + filePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create file download request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", telegramTransportError(ctx, "Telegram file download")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("telegram file download error %d: %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read telegram file: %w", err)
	}
	if len(content) == 0 {
		return "", fmt.Errorf("telegram file is empty")
	}

	mimeType := detectTelegramMIME(content, filePath)
	encoded := base64.StdEncoding.EncodeToString(content)
	return "data:" + mimeType + ";base64," + encoded, nil
}

func (t *TelegramChannel) getFilePath(ctx context.Context, fileID string) (string, error) {
	params := url.Values{"file_id": {fileID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL+"/getFile?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("create getFile request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", telegramTransportError(ctx, "Telegram getFile")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read getFile response: %w", err)
	}

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse getFile response: %w", err)
	}
	if !result.OK || result.Result.FilePath == "" {
		return "", fmt.Errorf("telegram getFile failed")
	}
	return result.Result.FilePath, nil
}

func detectTelegramMIME(content []byte, filePath string) string {
	if detected := strings.ToLower(http.DetectContentType(content)); strings.HasPrefix(detected, "image/") {
		return detected
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		// Telegram photo payloads are JPEG by default.
		return "image/jpeg"
	}
}

// MapTelegramInboundForTest helps tests build update payloads without depending
// on unexported Telegram transport structs.
func MapTelegramInboundForTest(update map[string]any) (InboundMessage, bool) {
	b, err := json.Marshal(update)
	if err != nil {
		return InboundMessage{}, false
	}
	var u tgUpdate
	if err := json.Unmarshal(b, &u); err != nil {
		return InboundMessage{}, false
	}
	return mapTelegramInbound(u)
}

func (t *TelegramChannel) syncCommands() error {
	commandsBytes, err := json.Marshal(AllCommands(t.devMode))
	if err != nil {
		return fmt.Errorf("marshal commands: %w", err)
	}
	params := url.Values{
		"commands": {string(commandsBytes)},
	}

	resp, err := t.client.PostForm(t.baseURL+"/setMyCommands", params)
	if err != nil {
		return errors.New("Telegram setMyCommands request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("setMyCommands api error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func telegramTransportError(ctx context.Context, operation string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%s request failed", operation)
}

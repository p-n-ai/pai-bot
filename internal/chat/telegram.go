package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const telegramMaxMessageLen = 4096

// TelegramChannel implements the Channel interface for Telegram Bot API.
type TelegramChannel struct {
	token   string
	baseURL string
	client  *http.Client
	offset  int
	stop    chan struct{}
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

func (t *TelegramChannel) SendTyping(_ context.Context, userID string) error {
	params := url.Values{
		"chat_id": {userID},
		"action":  {"typing"},
	}
	resp, err := t.client.PostForm(t.baseURL+"/sendChatAction", params)
	if err != nil {
		return fmt.Errorf("sending typing indicator: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

func (t *TelegramChannel) SendMessage(ctx context.Context, userID string, msg OutboundMessage) error {
	parts := SplitMessage(msg.Text, telegramMaxMessageLen)

	for _, part := range parts {
		params := url.Values{
			"chat_id": {userID},
			"text":    {part},
		}
		if msg.ParseMode != "" {
			params.Set("parse_mode", msg.ParseMode)
		}

		resp, err := t.client.PostForm(t.baseURL+"/sendMessage", params)
		if err != nil {
			return fmt.Errorf("sending Telegram message: %w", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// If Markdown parsing fails, retry without parse mode
			if msg.ParseMode != "" && resp.StatusCode == http.StatusBadRequest {
				slog.Warn("Telegram markdown parse failed, retrying plain")
				params.Del("parse_mode")
				retryResp, retryErr := t.client.PostForm(t.baseURL+"/sendMessage", params)
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
	go t.pollLoop(ctx, handler)
	return nil
}

func (t *TelegramChannel) Stop() error {
	close(t.stop)
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
				time.Sleep(5 * time.Second)
				continue
			}

			for _, u := range updates {
				t.offset = u.UpdateID + 1
				if u.Message == nil || u.Message.Text == "" {
					continue
				}

				msg := InboundMessage{
					Channel:    "telegram",
					UserID:     strconv.FormatInt(u.Message.Chat.ID, 10),
					ExternalID: strconv.FormatInt(u.Message.From.ID, 10),
					Text:       u.Message.Text,
					Username:   u.Message.From.Username,
					FirstName:  u.Message.From.FirstName,
					LastName:   u.Message.From.LastName,
					Language:   u.Message.From.LanguageCode,
				}
				if u.Message.ReplyToMessage != nil && u.Message.ReplyToMessage.Text != "" {
					msg.ReplyToText = u.Message.ReplyToMessage.Text
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
		return nil, err
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
	UpdateID int        `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	Text           string     `json:"text"`
	Chat           tgChat     `json:"chat"`
	From           tgUser     `json:"from"`
	ReplyToMessage *tgMessage `json:"reply_to_message,omitempty"`
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

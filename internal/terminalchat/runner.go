// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalchat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

// Processor handles inbound messages and returns assistant responses.
type Processor interface {
	ProcessMessage(ctx context.Context, msg chat.InboundMessage) (string, error)
}

// Config controls the terminal chat session.
type Config struct {
	UserID  string
	Channel string
}

// Run starts a terminal chat session over stdin/stdout style streams.
func Run(ctx context.Context, in io.Reader, out io.Writer, processor Processor, cfg Config) error {
	if processor == nil {
		return fmt.Errorf("processor is required")
	}

	userID := strings.TrimSpace(cfg.UserID)
	if userID == "" {
		userID = "terminal-user"
	}

	channel := strings.TrimSpace(cfg.Channel)
	if channel == "" {
		channel = "terminal"
	}

	scanner := bufio.NewScanner(in)
	if _, err := fmt.Fprintln(out, "Terminal chat ready. Type /exit to quit."); err != nil {
		return err
	}

	for {
		if _, err := fmt.Fprint(out, "You> "); err != nil {
			return err
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(out, "\nSession ended.")
			return nil
		}

		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if text == "/exit" || text == "/quit" {
			_, _ = fmt.Fprintln(out, "Session ended.")
			return nil
		}

		resp, err := processor.ProcessMessage(ctx, chat.InboundMessage{
			Channel: channel,
			UserID:  userID,
			Text:    text,
		})
		if err != nil {
			if _, writeErr := fmt.Fprintf(out, "Error: %v\n", err); writeErr != nil {
				return writeErr
			}
			continue
		}

		if _, err := fmt.Fprintf(out, "P&AI> %s\n", strings.TrimSpace(resp)); err != nil {
			return err
		}
	}
}

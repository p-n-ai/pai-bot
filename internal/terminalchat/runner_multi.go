package terminalchat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

// MultiConfig controls a multi-user terminal chat session.
type MultiConfig struct {
	// UserCount is the number of simulated users (default 2).
	UserCount int
	// UserPrefix is the base name for user IDs (e.g., "user" → "user-1", "user-2").
	UserPrefix string
	Channel    string
}

// RunMulti starts a multi-user terminal chat session.
//
// Input format: each line is "N:message" where N is the 1-based user number.
// Lines without a prefix default to user 1.
//
// Example:
//
//	1:teach me linear equations
//	1:/challenge invite linear equations
//	2:/challenge ABCD12
//	1:3
//	2:5
func RunMulti(ctx context.Context, in io.Reader, out io.Writer, processor Processor, cfg MultiConfig) error {
	if processor == nil {
		return fmt.Errorf("processor is required")
	}

	userCount := cfg.UserCount
	if userCount < 2 {
		userCount = 2
	}

	prefix := strings.TrimSpace(cfg.UserPrefix)
	if prefix == "" {
		prefix = "terminal-user"
	}

	channel := strings.TrimSpace(cfg.Channel)
	if channel == "" {
		channel = "terminal"
	}

	// Build user ID list.
	userIDs := make([]string, userCount)
	for i := range userIDs {
		userIDs[i] = fmt.Sprintf("%s-%d", prefix, i+1)
	}

	scanner := bufio.NewScanner(in)

	var userList strings.Builder
	for i, id := range userIDs {
		if i > 0 {
			userList.WriteString(", ")
		}
		fmt.Fprintf(&userList, "%d=%s", i+1, id)
	}
	if _, err := fmt.Fprintf(out, "Multi-user chat ready (%s). Prefix lines with N: to switch users.\n", userList.String()); err != nil {
		return err
	}

	for {
		if _, err := fmt.Fprint(out, "> "); err != nil {
			return err
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(out, "\nSession ended.")
			return nil
		}

		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		if raw == "/exit" || raw == "/quit" {
			_, _ = fmt.Fprintln(out, "Session ended.")
			return nil
		}

		userIdx, text := parseMultiInput(raw, userCount)
		userID := userIDs[userIdx]

		resp, err := processor.ProcessMessage(ctx, chat.InboundMessage{
			Channel: channel,
			UserID:  userID,
			Text:    text,
		})
		if err != nil {
			if _, writeErr := fmt.Fprintf(out, "[%s] Error: %v\n", userID, err); writeErr != nil {
				return writeErr
			}
			continue
		}

		if _, err := fmt.Fprintf(out, "[%s] P&AI> %s\n", userID, strings.TrimSpace(resp)); err != nil {
			return err
		}
	}
}

// parseMultiInput splits "N:message" into (0-based user index, message).
// If no valid prefix, returns (0, raw).
func parseMultiInput(raw string, maxUsers int) (int, string) {
	colonIdx := strings.IndexByte(raw, ':')
	if colonIdx <= 0 || colonIdx > 3 {
		return 0, raw
	}

	numStr := raw[:colonIdx]
	n, err := strconv.Atoi(numStr)
	if err != nil || n < 1 || n > maxUsers {
		return 0, raw
	}

	text := strings.TrimSpace(raw[colonIdx+1:])
	if text == "" {
		return 0, raw
	}

	return n - 1, text
}

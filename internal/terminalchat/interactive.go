// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalchat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

const candidatePollInterval = 500 * time.Millisecond

// InteractiveStatus is the safe, operator-visible state of a live chat.
type InteractiveStatus struct {
	Provider      string
	Memory        bool
	CharacterID   string
	CandidateHash string
}

// InteractiveSession owns the swappable engine and candidate snapshot used by
// RunInteractive.
type InteractiveSession interface {
	Processor
	New(context.Context) (InteractiveStatus, error)
	Reload(context.Context) (InteractiveStatus, error)
	SelectCharacter(context.Context, string) (InteractiveStatus, error)
	Status() InteractiveStatus
	CandidateChanged() (bool, error)
}

// InteractiveConfig controls a queue-aware local terminal session.
type InteractiveConfig struct {
	UserID  string
	Channel string
}

type interactiveInput struct {
	text string
	err  error
	eof  bool
}

type interactiveResult struct {
	result agent.TurnResult
	err    error
}

type interactiveTurn struct {
	cancel      context.CancelFunc
	done        <-chan interactiveResult
	interrupted bool
}

type interactiveControl struct {
	name string
	arg  string
}

// RunInteractive accepts input while a model turn is running. It serializes
// turns, owns queue order, and drains canceled work before changing sessions.
func RunInteractive(
	ctx context.Context,
	in io.Reader,
	out io.Writer,
	session InteractiveSession,
	cfg InteractiveConfig,
) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if in == nil {
		return errors.New("input is required")
	}
	if out == nil {
		return errors.New("output is required")
	}
	if session == nil {
		return errors.New("interactive session is required")
	}

	userID := strings.TrimSpace(cfg.UserID)
	if userID == "" {
		userID = "terminal-user"
	}
	channel := strings.TrimSpace(cfg.Channel)
	if channel == "" {
		channel = "terminal"
	}

	scanCtx, stopScanner := context.WithCancel(ctx)
	defer stopScanner()
	inputs := make(chan interactiveInput, 1)
	go scanInteractiveInput(scanCtx, in, inputs)

	ticker := time.NewTicker(candidatePollInterval)
	defer ticker.Stop()
	if closer, ok := in.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}

	if _, err := fmt.Fprintln(out, "Interactive chat ready. Commands: /status, /new, /reload, /character <id>, /interrupt <message>, /exit."); err != nil {
		return err
	}
	if err := writeInteractiveStatus(out, session.Status(), false, 0); err != nil {
		return err
	}
	if _, err := fmt.Fprint(out, "You> "); err != nil {
		return err
	}

	var current *interactiveTurn
	var pending []string
	var control *interactiveControl
	var inputClosed bool
	var exiting bool
	var candidateNotified bool
	var candidateError string
	ctxDone := ctx.Done()
	sessionID := time.Now().UnixNano()
	turnNumber := 0

	startTurn := func(text string) {
		turnNumber++
		deliveryID := fmt.Sprintf("terminal:%d:%d", sessionID, turnNumber)
		turnCtx, cancel := context.WithCancel(ctx)
		done := make(chan interactiveResult, 1)
		go func() {
			result, err := session.ProcessTurn(turnCtx, chat.InboundMessage{
				Channel:    channel,
				UserID:     userID,
				DeliveryID: deliveryID,
				Text:       text,
			})
			done <- interactiveResult{result: result, err: err}
		}()
		current = &interactiveTurn{cancel: cancel, done: done}
	}

	runControl := func(command interactiveControl) error {
		var (
			status InteractiveStatus
			err    error
		)
		switch command.name {
		case "new":
			status, err = session.New(ctx)
			if err == nil {
				_, err = fmt.Fprintln(out, "Session reset.")
			}
		case "reload":
			status, err = session.Reload(ctx)
			if err == nil {
				candidateNotified = false
				candidateError = ""
				_, err = fmt.Fprintln(out, "Candidate reloaded into a fresh session.")
			}
		case "character":
			status, err = session.SelectCharacter(ctx, command.arg)
			if err == nil {
				_, err = fmt.Fprintf(out, "Character selected: %s\n", status.CharacterID)
			}
		default:
			return fmt.Errorf("unsupported control %q", command.name)
		}
		if err != nil {
			return err
		}
		return writeInteractiveStatus(out, status, false, 0)
	}

	startNext := func() error {
		if control != nil {
			nextControl := *control
			control = nil
			if err := runControl(nextControl); err != nil {
				if _, writeErr := fmt.Fprintf(out, "Error: %v\n", err); writeErr != nil {
					return writeErr
				}
			}
		}
		if len(pending) > 0 {
			next := pending[0]
			pending = pending[1:]
			startTurn(next)
			return nil
		}
		if exiting || inputClosed {
			_, err := fmt.Fprintln(out, "Session ended.")
			return err
		}
		_, err := fmt.Fprint(out, "You> ")
		return err
	}

	for {
		var turnDone <-chan interactiveResult
		if current != nil {
			turnDone = current.done
		}
		select {
		case <-ctxDone:
			ctxDone = nil
			exiting = true
			pending = nil
			control = nil
			if current == nil {
				_, _ = fmt.Fprintln(out, "\nSession ended.")
				return nil
			}
			current.interrupted = true
			current.cancel()

		case input := <-inputs:
			if input.err != nil {
				if current != nil {
					current.cancel()
				}
				return input.err
			}
			if input.eof {
				inputClosed = true
				if current == nil && len(pending) == 0 && control == nil {
					_, _ = fmt.Fprintln(out, "\nSession ended.")
					return nil
				}
				continue
			}

			text := strings.TrimSpace(input.text)
			if text == "" {
				continue
			}
			command, isCommand, commandErr := parseInteractiveCommand(text)
			if commandErr != nil {
				if _, err := fmt.Fprintf(out, "Error: %v\n", commandErr); err != nil {
					return err
				}
				if current == nil {
					if _, err := fmt.Fprint(out, "You> "); err != nil {
						return err
					}
				}
				continue
			}
			if isCommand {
				switch command.name {
				case "exit":
					exiting = true
					pending = nil
					control = nil
					if current == nil {
						_, _ = fmt.Fprintln(out, "Session ended.")
						return nil
					}
					current.interrupted = true
					current.cancel()
				case "status":
					if err := writeInteractiveStatus(out, session.Status(), current != nil, len(pending)); err != nil {
						return err
					}
					if current == nil {
						if _, err := fmt.Fprint(out, "You> "); err != nil {
							return err
						}
					}
				case "interrupt":
					pending = []string{command.arg}
					control = nil
					if current == nil {
						startTurn(command.arg)
						pending = nil
					} else {
						current.interrupted = true
						current.cancel()
					}
				case "new", "reload", "character":
					pending = nil
					control = &command
					if current == nil {
						if err := startNext(); err != nil {
							return err
						}
					} else {
						current.interrupted = true
						current.cancel()
					}
				}
				continue
			}

			if current == nil {
				startTurn(text)
				continue
			}
			pending = append(pending, text)
			if _, err := fmt.Fprintf(out, "[queued #%d]\n", len(pending)); err != nil {
				return err
			}

		case result := <-turnDone:
			finished := current
			current = nil
			finished.cancel()
			if finished.interrupted {
				if !exiting {
					if _, err := fmt.Fprintln(out, "[interrupted]"); err != nil {
						return err
					}
				}
			} else if result.err != nil {
				if _, err := fmt.Fprintf(out, "Error: %v\n", result.err); err != nil {
					return err
				}
			} else if err := writeTurn(out, "", result.result); err != nil {
				return err
			}
			if exiting {
				_, _ = fmt.Fprintln(out, "Session ended.")
				return nil
			}
			if err := startNext(); err != nil {
				return err
			}
			if current == nil && inputClosed {
				return nil
			}

		case <-ticker.C:
			changed, err := session.CandidateChanged()
			if err != nil {
				if message := err.Error(); message != candidateError {
					candidateError = message
					if _, writeErr := fmt.Fprintf(out, "Candidate invalid: %v\n", err); writeErr != nil {
						return writeErr
					}
				}
				continue
			}
			candidateError = ""
			if changed && !candidateNotified {
				candidateNotified = true
				if _, err := fmt.Fprintln(out, "Candidate changed; type /reload to apply."); err != nil {
					return err
				}
			}
			if !changed {
				candidateNotified = false
			}
		}
	}
}

func scanInteractiveInput(ctx context.Context, in io.Reader, inputs chan<- interactiveInput) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case inputs <- interactiveInput{text: scanner.Text()}:
		case <-ctx.Done():
			return
		}
	}
	if err := scanner.Err(); err != nil {
		select {
		case inputs <- interactiveInput{err: err}:
		case <-ctx.Done():
		}
		return
	}
	select {
	case inputs <- interactiveInput{eof: true}:
	case <-ctx.Done():
	}
}

func parseInteractiveCommand(text string) (interactiveControl, bool, error) {
	if !strings.HasPrefix(text, "/") {
		return interactiveControl{}, false, nil
	}
	name, arg, _ := strings.Cut(strings.TrimPrefix(text, "/"), " ")
	name = strings.ToLower(strings.TrimSpace(name))
	arg = strings.TrimSpace(arg)
	switch name {
	case "exit", "quit":
		return interactiveControl{name: "exit"}, true, nil
	case "status", "new", "reload":
		if arg != "" {
			return interactiveControl{}, true, fmt.Errorf("/%s does not accept arguments", name)
		}
		return interactiveControl{name: name}, true, nil
	case "character":
		if arg == "" {
			return interactiveControl{}, true, errors.New("usage: /character <id>")
		}
		return interactiveControl{name: name, arg: arg}, true, nil
	case "interrupt":
		if arg == "" {
			return interactiveControl{}, true, errors.New("usage: /interrupt <message>")
		}
		return interactiveControl{name: name, arg: arg}, true, nil
	default:
		return interactiveControl{}, false, nil
	}
}

func writeInteractiveStatus(out io.Writer, status InteractiveStatus, active bool, queued int) error {
	_, err := fmt.Fprintf(
		out,
		"Status: provider=%s memory=%t character=%s prompt=%s active=%t queued=%d\n",
		status.Provider,
		status.Memory,
		status.CharacterID,
		status.CandidateHash,
		active,
		queued,
	)
	return err
}

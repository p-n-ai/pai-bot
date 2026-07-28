// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"errors"
	"strings"
)

// CodexAppServerClient executes completions through a Codex app-server session.
// The implementation owns authentication; PaiBot never reads Codex credentials.
type CodexAppServerClient interface {
	Complete(context.Context, CompletionRequest) (CompletionResponse, error)
	Refresh(context.Context) error
}

type codexAppServerProvider struct {
	client CodexAppServerClient
	model  string
}

func NewCodexAppServerProvider(client CodexAppServerClient, model string) Provider {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultCodexModel
	}
	return &codexAppServerProvider{client: client, model: model}
}

func (p *codexAppServerProvider) Complete(
	ctx context.Context,
	req CompletionRequest,
) (CompletionResponse, error) {
	if p.client == nil {
		return CompletionResponse{}, errors.New("Codex app-server is unavailable")
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = p.model
	}
	return p.client.Complete(ctx, req)
}

func (p *codexAppServerProvider) StreamComplete(
	ctx context.Context,
	req CompletionRequest,
) (<-chan StreamChunk, error) {
	chunks := make(chan StreamChunk, 1)
	go func() {
		defer close(chunks)
		response, err := p.Complete(ctx, req)
		if err != nil {
			chunks <- StreamChunk{Error: err, Done: true}
			return
		}
		chunks <- StreamChunk{Content: response.Content, Done: true}
	}()
	return chunks, nil
}

func (p *codexAppServerProvider) Models() []ModelInfo {
	return []ModelInfo{{
		ID:          p.model,
		Name:        "Codex " + p.model,
		Description: "Codex via server-managed device login",
	}}
}

func (p *codexAppServerProvider) HealthCheck(ctx context.Context) error {
	if p.client == nil {
		return errors.New("Codex app-server is unavailable")
	}
	return p.client.Refresh(ctx)
}

var _ Provider = (*codexAppServerProvider)(nil)

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"testing"
)

type recordingCodexAppServer struct {
	request CompletionRequest
}

func (*recordingCodexAppServer) Available() bool { return true }

func (client *recordingCodexAppServer) Complete(
	_ context.Context,
	request CompletionRequest,
) (CompletionResponse, error) {
	client.request = request
	return CompletionResponse{
		Content:      "connected through app-server",
		Model:        request.Model,
		InputTokens:  12,
		OutputTokens: 4,
	}, nil
}

func (*recordingCodexAppServer) Refresh(context.Context) error { return nil }

func TestCodexAppServerProviderUsesConfiguredDefaultModel(t *testing.T) {
	client := &recordingCodexAppServer{}
	provider := NewCodexAppServerProvider(client, "gpt-test")

	response, err := provider.Complete(t.Context(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if client.request.Model != "gpt-test" {
		t.Fatalf("client request model = %q, want gpt-test", client.request.Model)
	}
	if response.Content != "connected through app-server" ||
		response.Model != "gpt-test" ||
		response.TotalTokens() != 16 {
		t.Fatalf("Complete() response = %#v", response)
	}
}

func TestCodexAppServerProviderPreservesExplicitModel(t *testing.T) {
	client := &recordingCodexAppServer{}
	provider := NewCodexAppServerProvider(client, "gpt-default")

	if _, err := provider.Complete(t.Context(), CompletionRequest{Model: "gpt-explicit"}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if client.request.Model != "gpt-explicit" {
		t.Fatalf("client request model = %q, want gpt-explicit", client.request.Model)
	}
}

func TestCodexAppServerProviderRejectsMissingClient(t *testing.T) {
	provider := NewCodexAppServerProvider(nil, "")

	_, err := provider.Complete(t.Context(), CompletionRequest{})

	if err == nil || err.Error() != "codex app-server is unavailable" {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestCodexAppServerProviderStreamsOneFinalChunk(t *testing.T) {
	client := &recordingCodexAppServer{}
	provider := NewCodexAppServerProvider(client, "gpt-test")

	chunks, err := provider.StreamComplete(t.Context(), CompletionRequest{})
	if err != nil {
		t.Fatalf("StreamComplete() error = %v", err)
	}
	chunk, ok := <-chunks
	if !ok || chunk.Content != "connected through app-server" || !chunk.Done || chunk.Error != nil {
		t.Fatalf("stream chunk = %#v, open = %v", chunk, ok)
	}
	if _, open := <-chunks; open {
		t.Fatal("stream returned more than one chunk")
	}
}

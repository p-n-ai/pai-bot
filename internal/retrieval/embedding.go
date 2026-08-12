// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultEmbeddingModel      = "text-embedding-3-small"
	DefaultEmbeddingDimensions = 1536
)

type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type OpenAICompatibleEmbedder struct {
	baseURL    string
	apiKey     string
	model      string
	dimensions int
	client     *http.Client
}

func NewOpenAICompatibleEmbedder(baseURL, apiKey, model string, dimensions int, client *http.Client) (*OpenAICompatibleEmbedder, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("embedding base URL is required")
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbeddingModel
	}
	if dimensions == 0 {
		dimensions = DefaultEmbeddingDimensions
	}
	if dimensions != DefaultEmbeddingDimensions {
		return nil, fmt.Errorf("embedding dimensions must be %d", DefaultEmbeddingDimensions)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &OpenAICompatibleEmbedder{
		baseURL: baseURL, apiKey: strings.TrimSpace(apiKey), model: model,
		dimensions: dimensions, client: client,
	}, nil
}

func (e *OpenAICompatibleEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(struct {
		Model      string   `json:"model"`
		Input      []string `json:"input"`
		Dimensions int      `json:"dimensions"`
	}{Model: e.model, Input: inputs, Dimensions: e.dimensions})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request embeddings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("embedding endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	var decoded struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode embeddings: %w", err)
	}
	if len(decoded.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding endpoint returned %d vectors for %d inputs", len(decoded.Data), len(inputs))
	}
	out := make([][]float32, len(inputs))
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= len(out) || len(item.Embedding) != e.dimensions {
			return nil, fmt.Errorf("embedding response has invalid index or dimensions")
		}
		out[item.Index] = item.Embedding
	}
	for _, vector := range out {
		if vector == nil {
			return nil, fmt.Errorf("embedding response omitted an input")
		}
	}
	return out, nil
}

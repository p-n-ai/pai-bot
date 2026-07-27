// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleEmbedder(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("request = %s, authorization = %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var request struct {
			Model      string   `json:"model"`
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != DefaultEmbeddingModel || request.Dimensions != DefaultEmbeddingDimensions {
			t.Errorf("request = %#v", request)
		}
		vector := make([]float32, DefaultEmbeddingDimensions)
		vector[0] = 0.5
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"index": 0, "embedding": vector}},
		})
	}))
	defer server.Close()

	client, err := NewOpenAICompatibleEmbedder(server.URL+"/v1", "secret", "", 0, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := client.Embed(context.Background(), []string{"factorisation"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 1 || len(vectors[0]) != DefaultEmbeddingDimensions || vectors[0][0] != 0.5 {
		t.Fatalf("vectors = %#v", vectors)
	}
}

func TestOpenAICompatibleEmbedderRejectsWrongDimensions(t *testing.T) {
	t.Parallel()
	if _, err := NewOpenAICompatibleEmbedder("http://example.test", "", "", 42, nil); err == nil {
		t.Fatal("expected invalid dimensions error")
	}
}

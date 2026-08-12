// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderCatalogDefinitions(t *testing.T) {
	want := map[string]string{"deepseek": "https://api.deepseek.com", "groq": "https://api.groq.com/openai/v1", "xai": "https://api.x.ai/v1", "mistral": "https://api.mistral.ai/v1", "cerebras": "https://api.cerebras.ai/v1"}
	catalog := ProviderCatalog()
	if len(catalog) != len(want) {
		t.Fatalf("len(ProviderCatalog()) = %d, want %d", len(catalog), len(want))
	}
	for _, definition := range catalog {
		if definition.Protocol != ProtocolOpenAICompatible {
			t.Errorf("%s protocol = %q", definition.ID, definition.Protocol)
		}
		if definition.BaseURL != want[definition.ID] {
			t.Errorf("%s base URL = %q, want %q", definition.ID, definition.BaseURL, want[definition.ID])
		}
		if len(definition.Models) == 0 {
			t.Errorf("%s has no models", definition.ID)
		}
		for _, model := range definition.Models {
			if !model.Capabilities.Chat {
				t.Errorf("%s/%s lacks chat support", definition.ID, model.ID)
			}
			if model.Capabilities.Streaming {
				t.Errorf("%s/%s advertises incremental streaming through the buffered adapter", definition.ID, model.ID)
			}
			if model.Capabilities.Tools {
				t.Errorf("%s/%s advertises tools without a native continuation adapter", definition.ID, model.ID)
			}
		}
	}
}

func TestProviderCatalogCapabilitiesAreModelScoped(t *testing.T) {
	groq, _ := LookupProviderDefinition("groq")
	llama, ok := groq.CapabilitiesForModel("llama-3.3-70b-versatile")
	if !ok || llama.StructuredOutput != StructuredOutputJSONObject || llama.Vision {
		t.Fatalf("Groq Llama capabilities = %#v, want text-only JSON object mode", llama)
	}
	gptOSS, ok := groq.CapabilitiesForModel("openai/gpt-oss-120b")
	if !ok || gptOSS.StructuredOutput != StructuredOutputJSONSchema || gptOSS.Vision {
		t.Fatalf("Groq GPT-OSS capabilities = %#v, want text-only JSON schema mode", gptOSS)
	}

	cerebras, _ := LookupProviderDefinition("cerebras")
	defaultModel, ok := cerebras.CapabilitiesForModel(cerebras.DefaultModel)
	if !ok || defaultModel.Vision {
		t.Fatalf("Cerebras default capabilities = %#v, want no vision", defaultModel)
	}
	gemma, ok := cerebras.CapabilitiesForModel("gemma-4-31b")
	if !ok || !gemma.Vision {
		t.Fatalf("Cerebras Gemma capabilities = %#v, want vision", gemma)
	}
	if _, ok := cerebras.CapabilitiesForModel("unknown-model"); ok {
		t.Fatal("unknown catalog model unexpectedly has capabilities")
	}
}

func TestProviderCatalogLookupIsImmutable(t *testing.T) {
	catalog := ProviderCatalog()
	catalog[0].ID = "changed"
	catalog[0].Models[0].ID = "changed"
	definition, ok := LookupProviderDefinition("deepseek")
	if !ok {
		t.Fatal("deepseek missing")
	}
	if definition.ID != "deepseek" || definition.Models[0].ID != "deepseek-v4-flash" {
		t.Fatalf("catalog mutation leaked: %+v", definition)
	}
	definition.Models[0].ID = "changed-again"
	again, _ := LookupProviderDefinition("deepseek")
	if again.Models[0].ID != "deepseek-v4-flash" {
		t.Fatal("lookup mutation leaked")
	}
	if _, ok := LookupProviderDefinition("missing"); ok {
		t.Fatal("missing provider found")
	}
}

func TestNewCatalogProviderRoutesThroughDefinitionEndpoint(t *testing.T) {
	var path string
	var request openaiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"model":"deepseek-chat"}`))
	}))
	defer server.Close()
	provider, err := NewCatalogProvider("deepseek", "secret", WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.(NativeProvider); ok {
		t.Fatal("catalog provider must stay outside the native continuation contract")
	}
	response, err := provider.Complete(context.Background(), CompletionRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/chat/completions" || request.Model != "deepseek-v4-flash" || response.Content != "ok" {
		t.Fatalf("path=%q request=%+v response=%+v", path, request, response)
	}
}

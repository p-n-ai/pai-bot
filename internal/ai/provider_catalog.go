// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"fmt"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

// ProviderProtocol identifies the wire protocol used by a provider.
type ProviderProtocol string

const ProtocolOpenAICompatible ProviderProtocol = "openai-compatible"

// ProviderCapabilities describes optional protocol features advertised by a provider.
type ProviderCapabilities struct {
	Chat             bool
	Streaming        bool
	Tools            bool
	StructuredOutput bool
	Vision           bool
}

// ProviderDefinition is the static, non-secret description of a provider.
type ProviderDefinition struct {
	ID           string
	Name         string
	Protocol     ProviderProtocol
	BaseURL      string
	DefaultModel string
	Capabilities ProviderCapabilities
	Models       []ModelInfo
}

var builtInProviderDefinitions = []ProviderDefinition{
	openAICompatibleDefinition("deepseek", "DeepSeek", "https://api.deepseek.com", "deepseek-v4-flash", false,
		"deepseek-v4-flash", "deepseek-v4-pro"),
	openAICompatibleDefinition("groq", "Groq", "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", true,
		"llama-3.1-8b-instant", "llama-3.3-70b-versatile", "openai/gpt-oss-120b", "openai/gpt-oss-20b", "openai/gpt-oss-safeguard-20b", "qwen/qwen3.6-27b"),
	openAICompatibleDefinition("xai", "xAI", "https://api.x.ai/v1", "grok-4.3", true,
		"grok-4.3", "grok-build-0.1", "grok-4.5"),
	openAICompatibleDefinition("mistral", "Mistral", "https://api.mistral.ai/v1", "mistral-large-latest", true,
		"codestral-latest", "devstral-latest", "magistral-medium-latest", "magistral-small", "mistral-large-latest", "mistral-medium-latest", "mistral-small-latest", "pixtral-large-latest"),
	openAICompatibleDefinition("cerebras", "Cerebras", "https://api.cerebras.ai/v1", "gpt-oss-120b", true,
		"gemma-4-31b", "gpt-oss-120b", "zai-glm-4.7"),
}

func openAICompatibleDefinition(id, name, baseURL, defaultModel string, vision bool, models ...string) ProviderDefinition {
	definition := ProviderDefinition{
		ID: id, Name: name, Protocol: ProtocolOpenAICompatible, BaseURL: baseURL, DefaultModel: defaultModel,
		Capabilities: ProviderCapabilities{Chat: true, Streaming: true, Tools: true, StructuredOutput: true, Vision: vision},
		Models:       make([]ModelInfo, len(models)),
	}
	for i, model := range models {
		definition.Models[i] = ModelInfo{ID: model, Name: model}
	}
	return definition
}

// ProviderCatalog returns a defensive copy of the built-in provider catalog.
func ProviderCatalog() []ProviderDefinition {
	catalog := make([]ProviderDefinition, len(builtInProviderDefinitions))
	for i := range builtInProviderDefinitions {
		catalog[i] = cloneProviderDefinition(builtInProviderDefinitions[i])
	}
	return catalog
}

// BuiltInProviderCatalog is an explicit alias for ProviderCatalog.
func BuiltInProviderCatalog() []ProviderDefinition { return ProviderCatalog() }

// LookupProviderDefinition returns a defensive copy of a built-in definition.
func LookupProviderDefinition(id string) (ProviderDefinition, bool) {
	for i := range builtInProviderDefinitions {
		if builtInProviderDefinitions[i].ID == id {
			return cloneProviderDefinition(builtInProviderDefinitions[i]), true
		}
	}
	return ProviderDefinition{}, false
}

func cloneProviderDefinition(definition ProviderDefinition) ProviderDefinition {
	definition.Models = append([]ModelInfo(nil), definition.Models...)
	return definition
}

type catalogOpenAIProvider struct {
	*OpenAIProvider
}

var _ NativeProvider = (*catalogOpenAIProvider)(nil)

func (p *catalogOpenAIProvider) CompleteNative(ctx context.Context, model string, c llm.Context, opts *llm.StreamOptions) (llm.AssistantMessage, error) {
	return (&directOpenAIProvider{OpenAIProvider: p.OpenAIProvider}).CompleteNative(ctx, model, c, opts)
}

// NewProviderFromDefinition constructs a provider using the definition's wire protocol.
func NewProviderFromDefinition(definition ProviderDefinition, apiKey string, opts ...OpenAIOption) (Provider, error) {
	switch definition.Protocol {
	case ProtocolOpenAICompatible:
		defaults := []OpenAIOption{WithBaseURL(definition.BaseURL), WithProviderName(definition.ID), WithDefaultModel(definition.DefaultModel), WithModels(append([]ModelInfo(nil), definition.Models...))}
		provider := newOpenAIProvider(apiKey, append(defaults, opts...)...)
		if definition.Capabilities.Tools {
			return &catalogOpenAIProvider{OpenAIProvider: provider}, nil
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("ai: unsupported provider protocol %q", definition.Protocol)
	}
}

// NewCatalogProvider looks up and constructs a built-in provider.
func NewCatalogProvider(id, apiKey string, opts ...OpenAIOption) (Provider, error) {
	definition, ok := LookupProviderDefinition(id)
	if !ok {
		return nil, fmt.Errorf("ai: unknown provider %q", id)
	}
	return NewProviderFromDefinition(definition, apiKey, opts...)
}

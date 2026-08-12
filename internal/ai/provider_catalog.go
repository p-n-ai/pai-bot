// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"fmt"
	"strings"
)

// ProviderProtocol identifies the wire protocol used by a provider.
type ProviderProtocol string

const ProtocolOpenAICompatible ProviderProtocol = "openai-compatible"

// StructuredOutputMode identifies the response-format contract supported by a model.
type StructuredOutputMode string

const (
	StructuredOutputNone       StructuredOutputMode = ""
	StructuredOutputJSONObject StructuredOutputMode = "json_object"
	StructuredOutputJSONSchema StructuredOutputMode = "json_schema"
)

// ProviderCapabilities describes optional protocol features advertised by one model.
type ProviderCapabilities struct {
	Chat             bool
	Streaming        bool
	Tools            bool
	StructuredOutput StructuredOutputMode
	Vision           bool
}

// SupportsStructuredOutput reports whether the model has a usable structured-output mode.
func (c ProviderCapabilities) SupportsStructuredOutput() bool {
	return c.StructuredOutput != StructuredOutputNone
}

// ProviderModelDefinition describes one curated model and its request capabilities.
type ProviderModelDefinition struct {
	ID           string
	Name         string
	Capabilities ProviderCapabilities
}

// ProviderDefinition is the static, non-secret description of a provider.
type ProviderDefinition struct {
	ID           string
	Name         string
	Protocol     ProviderProtocol
	BaseURL      string
	DefaultModel string
	Models       []ProviderModelDefinition
}

var builtInProviderDefinitions = []ProviderDefinition{
	openAICompatibleDefinition("deepseek", "DeepSeek", "https://api.deepseek.com", "deepseek-v4-flash",
		catalogModel("deepseek-v4-flash", StructuredOutputJSONObject, false),
		catalogModel("deepseek-v4-pro", StructuredOutputJSONObject, false)),
	openAICompatibleDefinition("groq", "Groq", "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile",
		catalogModel("llama-3.1-8b-instant", StructuredOutputJSONObject, false),
		catalogModel("llama-3.3-70b-versatile", StructuredOutputJSONObject, false),
		catalogModel("openai/gpt-oss-120b", StructuredOutputJSONSchema, false),
		catalogModel("openai/gpt-oss-20b", StructuredOutputJSONSchema, false),
		catalogModel("openai/gpt-oss-safeguard-20b", StructuredOutputJSONObject, false),
		catalogModel("qwen/qwen3.6-27b", StructuredOutputJSONObject, false)),
	openAICompatibleDefinition("xai", "xAI", "https://api.x.ai/v1", "grok-4.3",
		catalogModel("grok-4.3", StructuredOutputJSONSchema, true),
		catalogModel("grok-build-0.1", StructuredOutputJSONSchema, true),
		catalogModel("grok-4.5", StructuredOutputJSONSchema, true)),
	openAICompatibleDefinition("mistral", "Mistral", "https://api.mistral.ai/v1", "mistral-large-latest",
		catalogModel("codestral-latest", StructuredOutputJSONSchema, false),
		catalogModel("devstral-latest", StructuredOutputJSONSchema, false),
		catalogModel("magistral-medium-latest", StructuredOutputJSONSchema, false),
		catalogModel("magistral-small", StructuredOutputJSONSchema, false),
		catalogModel("mistral-large-latest", StructuredOutputJSONSchema, false),
		catalogModel("mistral-medium-latest", StructuredOutputJSONSchema, false),
		catalogModel("mistral-small-latest", StructuredOutputJSONSchema, false),
		catalogModel("pixtral-large-latest", StructuredOutputJSONSchema, true)),
	openAICompatibleDefinition("cerebras", "Cerebras", "https://api.cerebras.ai/v1", "gpt-oss-120b",
		catalogModel("gemma-4-31b", StructuredOutputJSONSchema, true),
		catalogModel("gpt-oss-120b", StructuredOutputJSONSchema, false),
		catalogModel("zai-glm-4.7", StructuredOutputJSONSchema, false)),
}

func openAICompatibleDefinition(id, name, baseURL, defaultModel string, models ...ProviderModelDefinition) ProviderDefinition {
	return ProviderDefinition{
		ID:           id,
		Name:         name,
		Protocol:     ProtocolOpenAICompatible,
		BaseURL:      baseURL,
		DefaultModel: defaultModel,
		Models:       append([]ProviderModelDefinition(nil), models...),
	}
}

func catalogModel(id string, structuredOutput StructuredOutputMode, vision bool) ProviderModelDefinition {
	return ProviderModelDefinition{
		ID:   id,
		Name: id,
		Capabilities: ProviderCapabilities{
			Chat:             true,
			StructuredOutput: structuredOutput,
			Vision:           vision,
		},
	}
}

// CapabilitiesForModel returns the exact curated capabilities for model.
func (d ProviderDefinition) CapabilitiesForModel(model string) (ProviderCapabilities, bool) {
	model = strings.TrimSpace(model)
	for _, candidate := range d.Models {
		if candidate.ID == model {
			return candidate.Capabilities, true
		}
	}
	return ProviderCapabilities{}, false
}

// ModelInfos projects the curated models into the provider-neutral model contract.
func (d ProviderDefinition) ModelInfos() []ModelInfo {
	models := make([]ModelInfo, len(d.Models))
	for i, model := range d.Models {
		models[i] = ModelInfo{ID: model.ID, Name: model.Name}
	}
	return models
}

// ProviderCatalog returns a defensive copy of the built-in provider catalog.
func ProviderCatalog() []ProviderDefinition {
	catalog := make([]ProviderDefinition, len(builtInProviderDefinitions))
	for i := range builtInProviderDefinitions {
		catalog[i] = cloneProviderDefinition(builtInProviderDefinitions[i])
	}
	return catalog
}

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
	definition.Models = append([]ProviderModelDefinition(nil), definition.Models...)
	return definition
}

// NewProviderFromDefinition constructs a provider using the definition's wire protocol.
func NewProviderFromDefinition(definition ProviderDefinition, apiKey string, opts ...OpenAIOption) (Provider, error) {
	switch definition.Protocol {
	case ProtocolOpenAICompatible:
		defaults := []OpenAIOption{
			WithBaseURL(definition.BaseURL),
			WithProviderName(definition.ID),
			WithDefaultModel(definition.DefaultModel),
			WithModels(definition.ModelInfos()),
		}
		return newOpenAIProvider(apiKey, append(defaults, opts...)...), nil
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

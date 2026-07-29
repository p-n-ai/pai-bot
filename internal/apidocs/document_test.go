// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package apidocs

import (
	"encoding/json"
	"testing"
)

func TestBuild_GeneratesExplicitSchemas(t *testing.T) {
	doc, err := Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("doc.OpenAPI = %q, want 3.1.0", doc.OpenAPI)
	}
	if doc.Info.Title != "P&AI Bot API" {
		t.Fatalf("doc.Info.Title = %q, want P&AI Bot API", doc.Info.Title)
	}
	if _, ok := doc.Paths["/api/auth/login"]; !ok {
		t.Fatal("missing /api/auth/login path")
	}
	if _, ok := doc.Paths["/api/admin/ai/usage"]; !ok {
		t.Fatal("missing /api/admin/ai/usage path")
	}
	aiSettings, ok := doc.Paths["/api/admin/ai/settings"]
	if !ok {
		t.Fatal("missing /api/admin/ai/settings path")
	}
	if aiSettings.Get == nil || aiSettings.Put == nil {
		t.Fatalf("/api/admin/ai/settings = %#v, want GET and PUT", aiSettings)
	}
	updateSchema := aiSettings.Put.RequestBody.Content["application/json"].Schema
	if len(updateSchema.Required) != 0 {
		t.Fatalf("AI settings update required = %v, want no required fields", updateSchema.Required)
	}
	provider := updateSchema.Properties["provider"]
	if provider == nil || len(provider.OneOf) != 3 ||
		provider.Discriminator == nil || provider.Discriminator.PropertyName != "type" {
		t.Fatalf("AI settings provider schema = %#v, want three discriminated variants", provider)
	}
	for _, variant := range provider.OneOf {
		if variant.AdditionalProperties != false {
			t.Fatalf("provider variant allows unknown fields: %#v", variant)
		}
	}
	selector := updateSchema.Properties["defaultProvider"]
	if selector == nil || len(selector.OneOf) != 4 {
		t.Fatalf("defaultProvider schema = %#v, want three selectors plus null", selector)
	}
	responseSchema := doc.Components.Schemas["aiSettingsResponseDoc"]
	providers := responseSchema.Properties["providers"]
	if providers == nil || providers.Items == nil || len(providers.Items.OneOf) != 3 {
		t.Fatalf("AI settings response providers = %#v, want closed three-variant projection", providers)
	}

	sessionSchema, ok := doc.Components.Schemas["Session"]
	if !ok {
		t.Fatal("missing Session schema")
	}
	if sessionSchema.Properties["user"].Ref != "#/components/schemas/UserSession" {
		t.Fatalf("Session.user ref = %q", sessionSchema.Properties["user"].Ref)
	}

	userSession, ok := doc.Components.Schemas["UserSession"]
	if !ok {
		t.Fatal("missing UserSession schema")
	}
	role := userSession.Properties["role"]
	if role == nil || len(role.Enum) != 5 {
		t.Fatalf("role enum = %#v, want 5 values", role)
	}

	classProgress, ok := doc.Components.Schemas["ClassProgress"]
	if !ok {
		t.Fatal("missing ClassProgress schema")
	}
	if classProgress.Properties["students"].Type != "array" {
		t.Fatalf("ClassProgress.students type = %#v", classProgress.Properties["students"])
	}
}

func TestJSON_ReturnsMarshalableDocument(t *testing.T) {
	data, err := JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("JSON() returned empty document")
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["openapi"] != "3.1.0" {
		t.Fatalf("openapi = %#v, want 3.1.0", payload["openapi"])
	}
}

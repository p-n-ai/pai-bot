// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package apicontract

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCollectUsesRuntimeMuxRegistrations(t *testing.T) {
	root := t.TempDir()
	source := `package sample
func routes(mux interface {
	Handle(string, any)
	HandleFunc(string, func())
}) {
	mux.Handle("GET /api/auth/session", nil)
	mux.Handle("POST /api/auth/session", nil)
	mux.Handle("GET /healthz", nil)
	mux.Handle(dynamicPattern, nil)
	mux.HandleFunc("DELETE /api/admin/users/{id}", nil)
}`
	if err := os.WriteFile(filepath.Join(root, "routes.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "routes_test.go"),
		[]byte(`package sample; const ignored = "GET /api/test-only"`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	got, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []Route{
		{Method: "DELETE", Path: "/api/admin/users/{id}"},
		{Method: "GET", Path: "/api/auth/session"},
		{Method: "POST", Path: "/api/auth/session"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("routes = %#v, want %#v", got, want)
	}
}

func TestRemovedReportsOnlyMissingBaseRoutes(t *testing.T) {
	kept := Route{Method: "GET", Path: "/api/auth/session"}
	removed := Route{Method: "POST", Path: "/api/auth/login"}
	added := Route{Method: "GET", Path: "/api/auth/identities"}

	got := Removed([]Route{kept, removed}, []Route{kept, added})
	if !slices.Equal(got, []Route{removed}) {
		t.Fatalf("removed routes = %#v, want %#v", got, []Route{removed})
	}
}

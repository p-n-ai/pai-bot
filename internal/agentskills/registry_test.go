// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agentskills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func TestLoadAndProgressivelyReadSkill(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "math-coach")
	writeFixture(t, filepath.Join(skillRoot, skillFilename), `---
name: math-coach
description: Coach learners through maths problems. Use for maths tutoring.
license: Apache-2.0
compatibility: Requires the curriculum lookup tool
metadata:
  author: pai
  version: "1.0"
allowed-tools: lookup_curriculum_topic
---
# Maths coach

Ask one question at a time.
`)
	writeFixture(t, filepath.Join(skillRoot, "references", "fractions.md"), "Use visual fractions.")

	registry, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if registry.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", registry.Len())
	}
	catalog := registry.CatalogPrompt()
	if !strings.Contains(catalog, "math-coach: Coach learners") || strings.Contains(catalog, "Ask one question") {
		t.Fatalf("CatalogPrompt() = %q", catalog)
	}

	tools := registry.Tools()
	loaded, err := tools[0].Execute(context.Background(), llm.ToolCall{Arguments: llm.ToolArgumentsFrom(map[string]any{"name": "math-coach"})})
	loadedText := textResult(t, loaded)
	if err != nil || loaded.IsError || !strings.Contains(loadedText, "name: math-coach") || !strings.Contains(loadedText, "Ask one question at a time.") {
		t.Fatalf("load skill = %#v, %v", loaded, err)
	}
	resource, err := tools[1].Execute(context.Background(), llm.ToolCall{Arguments: llm.ToolArgumentsFrom(map[string]any{
		"skill": "math-coach",
		"path":  "references/fractions.md",
	})})
	if err != nil || resource.IsError || textResult(t, resource) != "Use visual fractions." {
		t.Fatalf("read resource = %#v, %v", resource, err)
	}
}

func TestLoadRejectsInvalidSkills(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		contents  string
		want      string
	}{
		{name: "missing frontmatter", directory: "broken", contents: "# Broken", want: "must start with YAML frontmatter"},
		{name: "directory mismatch", directory: "math", contents: "---\nname: science\ndescription: Teach science.\n---\nBody", want: "must match parent directory"},
		{name: "unknown field", directory: "math", contents: "---\nname: math\ndescription: Teach math.\nextra: nope\n---\nBody", want: "field extra not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, filepath.Join(root, tt.directory, skillFilename), tt.contents)
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadAcceptsPortableFrontmatterDelimiters(t *testing.T) {
	tests := []struct {
		name     string
		contents string
	}{
		{name: "closing delimiter at EOF", contents: "---\nname: portable\ndescription: Portable skill.\n---"},
		{name: "CRLF line endings", contents: "---\r\nname: portable\r\ndescription: Portable skill.\r\n---\r\nInstructions.\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, filepath.Join(root, "portable", skillFilename), tt.contents)
			if _, err := Load(root); err != nil {
				t.Fatalf("Load() error = %v", err)
			}
		})
	}
}

func TestReadResourceRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "safe-skill")
	writeFixture(t, filepath.Join(skillRoot, skillFilename), "---\nname: safe-skill\ndescription: Read safe local resources.\n---\nDo it.")
	outside := filepath.Join(root, "secret.txt")
	writeFixture(t, outside, "secret")

	registry, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, path := range []string{"../secret.txt", outside} {
		if _, err := registry.readResource("safe-skill", path); err == nil {
			t.Fatalf("readResource(%q) unexpectedly succeeded", path)
		}
	}
	link := filepath.Join(skillRoot, "linked.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	if _, err := registry.readResource("safe-skill", "linked.txt"); err == nil {
		t.Fatal("readResource() followed a symlink outside the skill")
	}
}

func TestLoadOptionalDisablesSkillsAndRejectsBadConfiguredPath(t *testing.T) {
	registry, err := LoadOptional("  ")
	if err != nil || registry != nil {
		t.Fatalf("LoadOptional(empty) = %#v, %v", registry, err)
	}
	if _, err := LoadOptional(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("LoadOptional(missing) unexpectedly succeeded")
	}
}

func TestAlwaysActiveSkillIsInjectedInsteadOfExposedAsTool(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "teacher", skillFilename), `---
name: teacher
description: Teach as a capable and approachable learning buddy.
metadata:
  activation: always
---
Teach one useful move at a time.
`)
	registry, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := registry.AlwaysActivePrompt(); !strings.Contains(got, "ACTIVE SKILL: teacher") || !strings.Contains(got, "Teach one useful move") {
		t.Fatalf("AlwaysActivePrompt() = %q", got)
	}
	if catalog := registry.CatalogPrompt(); catalog != "" {
		t.Fatalf("CatalogPrompt() = %q, want empty", catalog)
	}
	if tools := registry.Tools(); len(tools) != 0 {
		t.Fatalf("Tools() count = %d, want 0", len(tools))
	}
}

func TestReadResourceRejectsUnsupportedContent(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "bounded-skill")
	writeFixture(t, filepath.Join(skillRoot, skillFilename), "---\nname: bounded-skill\ndescription: Read bounded text resources.\n---\nDo it.")
	registry, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	invalidUTF8 := filepath.Join(skillRoot, "invalid.txt")
	if err := os.WriteFile(invalidUTF8, []byte{0xff}, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := registry.readResource("bounded-skill", "invalid.txt"); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("read invalid UTF-8 error = %v", err)
	}

	oversized := filepath.Join(skillRoot, "oversized.txt")
	if err := os.WriteFile(oversized, make([]byte, maxSkillBytes+1), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := registry.readResource("bounded-skill", "oversized.txt"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("read oversized error = %v", err)
	}
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func textResult(t *testing.T, result llm.ToolResultMessage) string {
	t.Helper()
	content, ok := result.Content[0].(llm.TextContent)
	if !ok {
		t.Fatalf("tool content = %#v", result.Content)
	}
	return content.Text
}

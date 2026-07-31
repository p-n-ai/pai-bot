// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalchat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateSourceLoadsPromptAndCharacters(t *testing.T) {
	directory := t.TempDir()
	writeCandidateTestFile(t, directory, `prompt: Keep replies short.
default: aina
characters:
  - id: aina
    first_name: Aina
    username: aina_student
    language: EN
  - id: faris
    first_name: Faris
    language: ms
`)

	candidate, err := NewCandidateSource(directory).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if candidate.Prompt != "Keep replies short." {
		t.Fatalf("Prompt = %q, want trimmed prompt", candidate.Prompt)
	}
	if !strings.HasPrefix(candidate.Hash, "sha256:") || len(candidate.Hash) != len("sha256:")+64 {
		t.Fatalf("Hash = %q, want full SHA-256 identity", candidate.Hash)
	}
	if candidate.DefaultCharacterID != "aina" {
		t.Fatalf("DefaultCharacterID = %q, want aina", candidate.DefaultCharacterID)
	}
	aina, ok := candidate.Character("aina")
	if !ok {
		t.Fatal("Character(aina) missing")
	}
	if aina.FirstName != "Aina" || aina.Username != "aina_student" || aina.Language != "en" {
		t.Fatalf("Character(aina) = %#v, want normalized profile", aina)
	}
}

func TestCandidateSourceRejectsUnknownCharacterField(t *testing.T) {
	directory := t.TempDir()
	writeCandidateTestFile(t, directory, `characters:
  - id: aina
    langauge: en
`)

	_, err := NewCandidateSource(directory).Load()
	if err == nil || !strings.Contains(err.Error(), "field langauge not found") {
		t.Fatalf("Load() error = %v, want strict unknown-field failure", err)
	}
}

func TestCandidateSourceDetectsFileChanges(t *testing.T) {
	directory := t.TempDir()
	source := NewCandidateSource(directory)
	initial, err := source.Load()
	if err != nil {
		t.Fatalf("initial Load() error = %v", err)
	}
	changed, err := source.Changed(initial.Hash)
	if err != nil || changed {
		t.Fatalf("Changed(initial) = %t, %v; want false, nil", changed, err)
	}

	writeCandidateTestFile(t, directory, "prompt: Try one question at a time.\n")
	changed, err = source.Changed(initial.Hash)
	if err != nil || !changed {
		t.Fatalf("Changed(updated) = %t, %v; want true, nil", changed, err)
	}
}

func TestCandidateSourceRetriesChangingFile(t *testing.T) {
	reads := 0
	source := CandidateSource{
		directory: "/candidate",
		readFile: func(path string) ([]byte, error) {
			if filepath.Base(path) != candidateFilename {
				return nil, os.ErrNotExist
			}
			reads++
			if reads == 1 {
				return []byte("prompt: old prompt\n"), nil
			}
			return []byte("prompt: new prompt\n"), nil
		},
	}

	candidate, err := source.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if candidate.Prompt != "new prompt" {
		t.Fatalf("Prompt = %q, want stable new generation", candidate.Prompt)
	}
	if reads != 4 {
		t.Fatalf("reads = %d, want one rejected pair and one stable pair", reads)
	}
}

func TestCandidateSourceRejectsContinuouslyChangingFiles(t *testing.T) {
	reads := 0
	source := CandidateSource{
		directory: "/candidate",
		readFile: func(path string) ([]byte, error) {
			if filepath.Base(path) == candidateFilename {
				reads++
				return []byte(fmt.Sprintf("prompt: prompt-%d\n", reads)), nil
			}
			return nil, os.ErrNotExist
		},
	}

	_, err := source.Load()
	if err == nil || !strings.Contains(err.Error(), "changed while loading") {
		t.Fatalf("Load() error = %v, want unstable snapshot failure", err)
	}
}

func TestCandidateSourceRejectsInvalidCharacterContracts(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "duplicate id",
			content: `characters:
  - id: aina
  - id: aina
`,
			want: `duplicate character id "aina"`,
		},
		{
			name: "undefined default",
			content: `default: missing
characters:
  - id: aina
`,
			want: `default character "missing" is not defined`,
		},
		{
			name: "unsupported language",
			content: `characters:
  - id: aina
    language: klingon
`,
			want: `unsupported language "klingon"`,
		},
		{
			name: "multiple documents",
			content: `prompt: first
---
prompt: second
`,
			want: "multiple YAML documents are not supported",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			writeCandidateTestFile(t, directory, test.content)

			_, err := NewCandidateSource(directory).Load()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want %q", err, test.want)
			}
		})
	}
}

func writeCandidateTestFile(t *testing.T, directory, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, candidateFilename), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", candidateFilename, err)
	}
}

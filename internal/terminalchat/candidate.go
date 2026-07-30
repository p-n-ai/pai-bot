// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalchat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/p-n-ai/pai-bot/internal/i18n"
)

const (
	candidateFilename         = "candidate.yaml"
	candidateSnapshotAttempts = 3
)

// Character is one stable learner identity available to an interactive chat.
type Character struct {
	ID        string `yaml:"id"`
	FirstName string `yaml:"first_name"`
	Username  string `yaml:"username"`
	Language  string `yaml:"language"`
}

// Candidate is one immutable prompt and learner-character snapshot.
type Candidate struct {
	Prompt             string
	Hash               string
	DefaultCharacterID string
	Characters         []Character
}

// Character returns the named character from the snapshot.
func (c Candidate) Character(id string) (Character, bool) {
	id = strings.TrimSpace(id)
	for _, character := range c.Characters {
		if character.ID == id {
			return character, true
		}
	}
	return Character{}, false
}

// DefaultCharacter returns the snapshot's selected character.
func (c Candidate) DefaultCharacter() Character {
	character, ok := c.Character(c.DefaultCharacterID)
	if ok {
		return character
	}
	return Character{ID: "default"}
}

// CandidateSource loads candidate.yaml from one local directory.
type CandidateSource struct {
	directory string
	readFile  func(string) ([]byte, error)
}

// NewCandidateSource creates a local candidate source. A missing or empty
// directory uses the built-in prompt and default learner until files appear.
func NewCandidateSource(directory string) CandidateSource {
	return CandidateSource{
		directory: strings.TrimSpace(directory),
		readFile:  os.ReadFile,
	}
}

// Load parses one complete candidate. Invalid changes never produce a partial
// snapshot.
func (s CandidateSource) Load() (Candidate, error) {
	if s.directory == "" {
		return builtInCandidate(), nil
	}
	readFile := s.readFile
	if readFile == nil {
		readFile = os.ReadFile
	}
	for range candidateSnapshotAttempts {
		first, firstExists, err := readCandidateFile(readFile, filepath.Join(s.directory, candidateFilename))
		if err != nil {
			return Candidate{}, err
		}
		second, secondExists, err := readCandidateFile(readFile, filepath.Join(s.directory, candidateFilename))
		if err != nil {
			return Candidate{}, err
		}
		if firstExists == secondExists && bytes.Equal(first, second) {
			return candidateFromFile(second, secondExists)
		}
	}
	return Candidate{}, errors.New("candidate file changed while loading; try /reload again")
}

func candidateFromFile(data []byte, exists bool) (Candidate, error) {
	if !exists {
		return builtInCandidate(), nil
	}
	document, err := parseCandidate(data)
	if err != nil {
		return Candidate{}, fmt.Errorf("parse %s: %w", candidateFilename, err)
	}
	return Candidate{
		Prompt:             strings.TrimSpace(document.Prompt),
		Hash:               hashCandidate(data),
		DefaultCharacterID: document.Default,
		Characters:         document.Characters,
	}, nil
}

// Changed reports whether the files now resolve to another valid snapshot.
func (s CandidateSource) Changed(activeHash string) (bool, error) {
	candidate, err := s.Load()
	if err != nil {
		return false, err
	}
	return candidate.Hash != activeHash, nil
}

type candidateFile struct {
	Prompt     string      `yaml:"prompt"`
	Default    string      `yaml:"default"`
	Characters []Character `yaml:"characters"`
}

func parseCandidate(data []byte) (candidateFile, error) {
	var document candidateFile
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			document.Characters = []Character{{ID: "default"}}
			document.Default = "default"
			return document, nil
		}
		return candidateFile{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return candidateFile{}, errors.New("multiple YAML documents are not supported")
		}
		return candidateFile{}, err
	}

	seen := make(map[string]struct{}, len(document.Characters))
	characters := make([]Character, 0, len(document.Characters))
	for index, character := range document.Characters {
		character.ID = strings.TrimSpace(character.ID)
		character.FirstName = strings.TrimSpace(character.FirstName)
		character.Username = strings.TrimSpace(character.Username)
		character.Language = strings.TrimSpace(character.Language)
		if character.ID == "" {
			return candidateFile{}, fmt.Errorf("character %d: id is required", index+1)
		}
		if _, exists := seen[character.ID]; exists {
			return candidateFile{}, fmt.Errorf("duplicate character id %q", character.ID)
		}
		if character.Language != "" {
			normalized := i18n.NormalizeLocale(character.Language)
			if normalized == "" {
				return candidateFile{}, fmt.Errorf("character %q: unsupported language %q", character.ID, character.Language)
			}
			character.Language = normalized
		}
		seen[character.ID] = struct{}{}
		characters = append(characters, character)
	}
	if len(characters) == 0 {
		characters = append(characters, Character{ID: "default"})
	}

	document.Default = strings.TrimSpace(document.Default)
	if document.Default == "" {
		document.Default = characters[0].ID
	}
	if _, exists := seen[document.Default]; !exists && document.Default != "default" {
		return candidateFile{}, fmt.Errorf("default character %q is not defined", document.Default)
	}
	if document.Default == "default" && len(document.Characters) > 0 {
		if _, exists := seen[document.Default]; !exists {
			return candidateFile{}, fmt.Errorf("default character %q is not defined", document.Default)
		}
	}
	document.Characters = characters
	return document, nil
}

func readCandidateFile(readFile func(string) ([]byte, error), path string) ([]byte, bool, error) {
	data, err := readFile(path)
	if err == nil {
		return data, true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("read %s: %w", filepath.Base(path), err)
}

func builtInCandidate() Candidate {
	character := Character{ID: "default"}
	return Candidate{
		Hash:               hashCandidate(nil),
		DefaultCharacterID: character.ID,
		Characters:         []Character{character},
	}
}

func hashCandidate(data []byte) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(candidateFilename))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write(data)
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

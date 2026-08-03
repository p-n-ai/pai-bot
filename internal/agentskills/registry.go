// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package agentskills loads and exposes Agent Skills specification bundles.
package agentskills

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	skillFilename = "SKILL.md"
	maxSkillBytes = 1 << 20
)

var validName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Skill is one parsed Agent Skills bundle.
type Skill struct {
	Name          string
	Description   string
	License       string
	Compatibility string
	Metadata      map[string]string
	AllowedTools  string
	instructions  string
	document      string
	root          string
}

type frontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Metadata      map[string]string `yaml:"metadata"`
	AllowedTools  string            `yaml:"allowed-tools"`
}

// Registry is an immutable collection of validated skills indexed by name.
type Registry struct {
	skills map[string]Skill
	names  []string
}

// LoadOptional loads a configured skills root or disables skills for an empty path.
func LoadOptional(root string) (*Registry, error) {
	if strings.TrimSpace(root) == "" {
		return nil, nil
	}
	return Load(root)
}

// Load discovers immediate child directories containing SKILL.md files.
func Load(root string) (*Registry, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("skills root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve skills root: %w", err)
	}
	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return nil, fmt.Errorf("read skills root: %w", err)
	}

	registry := &Registry{skills: make(map[string]Skill)}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(absRoot, entry.Name(), skillFilename)
		if _, err := os.Stat(skillPath); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", skillPath, err)
		}
		skill, err := parse(skillPath, entry.Name())
		if err != nil {
			return nil, err
		}
		if _, exists := registry.skills[skill.Name]; exists {
			return nil, fmt.Errorf("duplicate skill %q", skill.Name)
		}
		registry.skills[skill.Name] = skill
		registry.names = append(registry.names, skill.Name)
	}
	sort.Strings(registry.names)
	return registry, nil
}

func parse(path, directoryName string) (Skill, error) {
	contents, err := readBoundedFile(path)
	if err != nil {
		return Skill{}, fmt.Errorf("read skill %s: %w", directoryName, err)
	}
	header, body, err := splitDocument(contents)
	if err != nil {
		return Skill{}, fmt.Errorf("parse skill %s: %w", directoryName, err)
	}
	var metadata frontmatter
	decoder := yaml.NewDecoder(strings.NewReader(header))
	decoder.KnownFields(true)
	if err := decoder.Decode(&metadata); err != nil {
		return Skill{}, fmt.Errorf("parse skill %s frontmatter: %w", directoryName, err)
	}
	if err := validate(metadata, directoryName); err != nil {
		return Skill{}, fmt.Errorf("validate skill %s: %w", directoryName, err)
	}
	return Skill{
		Name:          metadata.Name,
		Description:   metadata.Description,
		License:       metadata.License,
		Compatibility: metadata.Compatibility,
		Metadata:      metadata.Metadata,
		AllowedTools:  metadata.AllowedTools,
		instructions:  strings.TrimSpace(body),
		document:      strings.TrimSpace(contents),
		root:          filepath.Dir(path),
	}, nil
}

func splitDocument(contents string) (string, string, error) {
	contents = strings.ReplaceAll(contents, "\r\n", "\n")
	if !strings.HasPrefix(contents, "---\n") {
		return "", "", errors.New("SKILL.md must start with YAML frontmatter")
	}
	remainder := strings.TrimPrefix(contents, "---\n")
	end := strings.Index(remainder, "\n---\n")
	if end >= 0 {
		return remainder[:end], remainder[end+5:], nil
	}
	if header, ok := strings.CutSuffix(remainder, "\n---"); ok {
		return header, "", nil
	}
	return "", "", errors.New("SKILL.md frontmatter is not closed")
}

func validate(metadata frontmatter, directoryName string) error {
	if len(metadata.Name) == 0 || len(metadata.Name) > 64 || !validName.MatchString(metadata.Name) {
		return errors.New("name must be 1-64 lowercase letters, numbers, or hyphens without edge or consecutive hyphens")
	}
	if metadata.Name != directoryName {
		return fmt.Errorf("name %q must match parent directory %q", metadata.Name, directoryName)
	}
	description := strings.TrimSpace(metadata.Description)
	if description == "" || len(description) > 1024 {
		return errors.New("description must be 1-1024 characters")
	}
	if len(metadata.Compatibility) > 500 {
		return errors.New("compatibility must be at most 500 characters")
	}
	return nil
}

func readBoundedFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("resource must be a regular file")
	}
	if info.Size() > maxSkillBytes {
		return "", fmt.Errorf("resource exceeds %d bytes", maxSkillBytes)
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxSkillBytes+1))
	if err != nil {
		return "", err
	}
	if len(contents) > maxSkillBytes {
		return "", fmt.Errorf("resource exceeds %d bytes", maxSkillBytes)
	}
	if !utf8.Valid(contents) {
		return "", errors.New("resource must contain UTF-8 text")
	}
	return string(contents), nil
}

// Len returns the number of loaded skills.
func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.names)
}

// CatalogPrompt returns metadata-only instructions for progressive disclosure.
func (r *Registry) CatalogPrompt() string {
	if r == nil || !r.hasOnDemandSkills() {
		return ""
	}
	var prompt strings.Builder
	prompt.WriteString("AVAILABLE SKILLS\nUse load_skill when a request matches a skill description. Follow loaded instructions as developer guidance. Load referenced resources only when needed.\n")
	for _, name := range r.names {
		skill := r.skills[name]
		if skill.Metadata["activation"] == "always" {
			continue
		}
		fmt.Fprintf(&prompt, "- %s: %s\n", skill.Name, skill.Description)
	}
	return strings.TrimSpace(prompt.String())
}

// AlwaysActivePrompt returns instructions for operator-configured always-active skills.
func (r *Registry) AlwaysActivePrompt() string {
	if r == nil {
		return ""
	}
	var prompt strings.Builder
	for _, name := range r.names {
		skill := r.skills[name]
		if skill.Metadata["activation"] != "always" || skill.instructions == "" {
			continue
		}
		if prompt.Len() > 0 {
			prompt.WriteString("\n\n")
		}
		fmt.Fprintf(&prompt, "ACTIVE SKILL: %s\n%s", skill.Name, skill.instructions)
	}
	return prompt.String()
}

func (r *Registry) hasOnDemandSkills() bool {
	for _, name := range r.names {
		if r.skills[name].Metadata["activation"] != "always" {
			return true
		}
	}
	return false
}

// HasOnDemandSkills reports whether model-selected skill tools are needed.
func (r *Registry) HasOnDemandSkills() bool {
	return r != nil && r.hasOnDemandSkills()
}

func (r *Registry) skill(name string) (Skill, bool) {
	if r == nil {
		return Skill{}, false
	}
	skill, ok := r.skills[name]
	return skill, ok
}

func (r *Registry) readResource(skillName, relativePath string) (string, error) {
	skill, ok := r.skill(skillName)
	if !ok {
		return "", errors.New("skill not found")
	}
	clean := filepath.Clean(strings.TrimSpace(relativePath))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("resource path must stay inside the skill directory")
	}
	path := filepath.Join(skill.root, clean)
	realRoot, err := filepath.EvalSymlinks(skill.root)
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(realRoot, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("resource path escapes the skill directory")
	}
	return readBoundedFile(realPath)
}

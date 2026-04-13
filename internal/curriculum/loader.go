// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader loads and caches curriculum content from the filesystem.
type Loader struct {
	rootDir       string
	topics        map[string]Topic
	subjects      map[string]Subject
	syllabi       map[string]Syllabus
	assessments   map[string]Assessment
	teachingNotes map[string]string
	mu            sync.RWMutex
}

// NewLoader creates a new curriculum loader and loads all content.
func NewLoader(rootDir string) (*Loader, error) {
	l := &Loader{
		rootDir:       rootDir,
		topics:        make(map[string]Topic),
		subjects:      make(map[string]Subject),
		syllabi:       make(map[string]Syllabus),
		assessments:   make(map[string]Assessment),
		teachingNotes: make(map[string]string),
	}

	if err := l.loadAll(); err != nil {
		return nil, fmt.Errorf("loading curriculum: %w", err)
	}

	slog.Info("curriculum loaded", "topics", len(l.topics))
	return l, nil
}

// GetTopic returns a topic by ID.
func (l *Loader) GetTopic(id string) (Topic, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.topics[id]
	return t, ok
}

// GetSubject returns a subject by ID.
func (l *Loader) GetSubject(id string) (Subject, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	subject, ok := l.subjects[id]
	return subject, ok
}

// GetSyllabus returns a syllabus by ID.
func (l *Loader) GetSyllabus(id string) (Syllabus, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	syllabus, ok := l.syllabi[id]
	return syllabus, ok
}

// GetTeachingNotes returns teaching notes for a topic ID.
func (l *Loader) GetTeachingNotes(id string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n, ok := l.teachingNotes[id]
	return n, ok
}

// GetAssessment returns an assessment by topic ID.
func (l *Loader) GetAssessment(topicID string) (Assessment, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	assessment, ok := l.assessments[topicID]
	return assessment, ok
}

// AllTopics returns all loaded topics.
func (l *Loader) AllTopics() []Topic {
	l.mu.RLock()
	defer l.mu.RUnlock()
	topics := make([]Topic, 0, len(l.topics))
	for _, t := range l.topics {
		topics = append(topics, t)
	}
	return topics
}

func (l *Loader) loadAll() error {
	return filepath.Walk(l.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		switch {
		case strings.HasSuffix(path, ".teaching.md"):
			return l.loadTeachingNotes(path)
		case base == "subject.yaml" || base == "subject.yml":
			return l.loadSubject(path)
		case base == "syllabus.yaml" || base == "syllabus.yml":
			return l.loadSyllabus(path)
		case isAssessmentPath(path):
			return l.loadAssessment(path)
		case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"):
			if strings.HasSuffix(path, ".examples.yaml") {
				return nil // Skip non-topic YAML
			}
			return l.loadTopic(path)
		}
		return nil
	})
}

func isAssessmentPath(path string) bool {
	return strings.HasSuffix(path, ".assessments.yaml") || strings.HasSuffix(path, ".assessments.yml")
}

func (l *Loader) loadTopic(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var topic Topic
	if err := yaml.Unmarshal(data, &topic); err != nil {
		slog.Warn("skipping invalid topic YAML", "path", path, "error", err)
		return nil
	}

	if topic.ID == "" {
		return nil // Not a topic file
	}

	l.mu.Lock()
	l.topics[topic.ID] = topic
	l.mu.Unlock()

	return nil
}

func (l *Loader) loadSubject(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var subject Subject
	if err := yaml.Unmarshal(data, &subject); err != nil {
		slog.Warn("skipping invalid subject YAML", "path", path, "error", err)
		return nil
	}
	if subject.ID == "" {
		return nil
	}

	l.mu.Lock()
	l.subjects[subject.ID] = subject
	l.mu.Unlock()
	return nil
}

func (l *Loader) loadSyllabus(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var syllabus Syllabus
	if err := yaml.Unmarshal(data, &syllabus); err != nil {
		slog.Warn("skipping invalid syllabus YAML", "path", path, "error", err)
		return nil
	}
	if syllabus.ID == "" {
		return nil
	}

	l.mu.Lock()
	l.syllabi[syllabus.ID] = syllabus
	l.mu.Unlock()
	return nil
}

func (l *Loader) loadTeachingNotes(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Derive topic ID from matching YAML file
	yamlPath := strings.TrimSuffix(path, ".teaching.md") + ".yaml"
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil // No matching YAML, skip
	}

	var partial struct {
		ID string `yaml:"id"`
	}
	if err := yaml.Unmarshal(yamlData, &partial); err != nil || partial.ID == "" {
		return nil
	}

	l.mu.Lock()
	l.teachingNotes[partial.ID] = string(data)
	l.mu.Unlock()

	return nil
}

func (l *Loader) loadAssessment(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var assessment Assessment
	if err := yaml.Unmarshal(data, &assessment); err != nil {
		slog.Warn("skipping invalid assessment YAML", "path", path, "error", err)
		return nil
	}

	if assessment.TopicID == "" || len(assessment.Questions) == 0 {
		return nil
	}

	l.mu.Lock()
	l.assessments[assessment.TopicID] = assessment
	l.mu.Unlock()
	return nil
}

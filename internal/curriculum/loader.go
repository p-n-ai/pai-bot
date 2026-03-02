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
	teachingNotes map[string]string
	mu            sync.RWMutex
}

// NewLoader creates a new curriculum loader and loads all content.
func NewLoader(rootDir string) (*Loader, error) {
	l := &Loader{
		rootDir:       rootDir,
		topics:        make(map[string]Topic),
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

// GetTeachingNotes returns teaching notes for a topic ID.
func (l *Loader) GetTeachingNotes(id string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n, ok := l.teachingNotes[id]
	return n, ok
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

		switch {
		case strings.HasSuffix(path, ".teaching.md"):
			return l.loadTeachingNotes(path)
		case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"):
			if strings.HasSuffix(path, ".assessments.yaml") || strings.HasSuffix(path, ".examples.yaml") {
				return nil // Skip non-topic YAML
			}
			return l.loadTopic(path)
		}
		return nil
	})
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

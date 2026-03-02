package curriculum_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestLoader_LoadTopics(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topics := loader.AllTopics()
	if len(topics) == 0 {
		t.Error("AllTopics() returned empty")
	}
}

func TestLoader_GetTopic(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topic, found := loader.GetTopic("F1-01")
	if !found {
		t.Error("GetTopic(F1-01) not found")
	}
	if topic.Name == "" {
		t.Error("Topic.Name is empty")
	}
}

func TestLoader_GetTopic_NotFound(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	_, found := loader.GetTopic("NONEXISTENT")
	if found {
		t.Error("GetTopic(NONEXISTENT) should not be found")
	}
}

func TestLoader_GetTeachingNotes(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	notes, found := loader.GetTeachingNotes("F1-01")
	if !found {
		t.Error("GetTeachingNotes(F1-01) not found")
	}
	if notes == "" {
		t.Error("Teaching notes is empty")
	}
}

func TestLoader_SkipsNonTopicYAML(t *testing.T) {
	dir := setupTestCurriculum(t)

	// Add a non-topic YAML (assessments)
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	os.WriteFile(filepath.Join(topicsDir, "01-variables.assessments.yaml"), []byte(`
topic_id: F1-01
questions:
  - id: Q1
    text: "What is 3x when x=2?"
`), 0o644)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	// Should still only have 1 topic
	topics := loader.AllTopics()
	if len(topics) != 1 {
		t.Errorf("AllTopics() = %d topics, want 1 (assessment YAML should be skipped)", len(topics))
	}
}

func TestLoader_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topics := loader.AllTopics()
	if len(topics) != 0 {
		t.Errorf("AllTopics() = %d, want 0 for empty dir", len(topics))
	}
}

func TestLoader_TeachingNotesWithoutYAML(t *testing.T) {
	dir := t.TempDir()

	topicsDir := filepath.Join(dir, "topics")
	os.MkdirAll(topicsDir, 0o755)

	// Teaching notes with no matching YAML
	os.WriteFile(filepath.Join(topicsDir, "orphan.teaching.md"), []byte("# Orphan notes"), 0o644)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	_, found := loader.GetTeachingNotes("orphan")
	if found {
		t.Error("Should not find teaching notes without matching topic YAML")
	}
}

func setupTestCurriculum(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	os.MkdirAll(topicsDir, 0o755)

	// Topic YAML
	os.WriteFile(filepath.Join(topicsDir, "01-variables.yaml"), []byte(`
id: F1-01
name: "Variables & Algebraic Expressions"
subject_id: algebra
syllabus_id: malaysia-kssm-matematik-tingkatan1
difficulty: beginner
learning_objectives:
  - id: LO1
    text: "Use letters to represent unknown quantities"
    bloom: remember
  - id: LO2
    text: "Write algebraic expressions from word problems"
    bloom: apply
prerequisites:
  required: []
quality_level: 1
provenance: human
`), 0o644)

	// Teaching notes markdown
	os.WriteFile(filepath.Join(topicsDir, "01-variables.teaching.md"), []byte(`# Variables & Algebraic Expressions — Teaching Notes

## Overview
This topic introduces the concept of using letters to represent unknown values.

## Teaching Sequence
1. Start with a guessing game (15 min)
2. Introduce variables as "mystery numbers" (10 min)
3. Practice writing expressions (20 min)

## Common Misconceptions
| Misconception | Remediation |
|---|---|
| 3x means "3 and x" not "3 times x" | Use multiplication sign explicitly first |
`), 0o644)

	return dir
}

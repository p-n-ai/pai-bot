// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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

func TestLoader_GetAssessment(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	assessment, found := loader.GetAssessment("F1-01")
	if !found {
		t.Fatal("GetAssessment(F1-01) not found")
	}
	if assessment.TopicID != "F1-01" {
		t.Fatalf("assessment.TopicID = %q, want F1-01", assessment.TopicID)
	}
	if len(assessment.Questions) != 2 {
		t.Fatalf("len(assessment.Questions) = %d, want 2", len(assessment.Questions))
	}
	if assessment.Questions[0].Answer.Type != "exact" {
		t.Fatalf("assessment.Questions[0].Answer.Type = %q, want exact", assessment.Questions[0].Answer.Type)
	}
}

func TestLoader_LoadsSubjectMetadata(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	subject, found := loader.GetSubject("malaysia-kssm-matematik-tingkatan-1")
	if !found {
		t.Fatal("GetSubject() not found")
	}
	if subject.GradeID != "tingkatan-1" {
		t.Fatalf("subject.GradeID = %q, want tingkatan-1", subject.GradeID)
	}
}

func TestLoader_DoesNotTreatSubjectOrSyllabusAsTopics(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topics := loader.AllTopics()
	if len(topics) != 1 {
		t.Fatalf("AllTopics() = %d, want 1 real topic", len(topics))
	}
	if topics[0].ID != "F1-01" {
		t.Fatalf("topics[0].ID = %q, want F1-01", topics[0].ID)
	}
}

func TestLoader_SkipsNonTopicYAML(t *testing.T) {
	dir := setupTestCurriculum(t)

	// Add a non-topic YAML (assessments)
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	_ = os.WriteFile(filepath.Join(topicsDir, "01-variables.assessments.yaml"), []byte(`
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

	assessment, found := loader.GetAssessment("F1-01")
	if !found {
		t.Fatal("assessment should still be loaded")
	}
	if len(assessment.Questions) != 1 {
		t.Fatalf("len(assessment.Questions) = %d, want 1", len(assessment.Questions))
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
	_ = os.MkdirAll(topicsDir, 0o755)

	// Teaching notes with no matching YAML
	_ = os.WriteFile(filepath.Join(topicsDir, "orphan.teaching.md"), []byte("# Orphan notes"), 0o644)

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

	curriculumDir := filepath.Join(dir, "curricula", "malaysia", "kssm")
	topicsDir := filepath.Join(curriculumDir, "topics", "algebra")
	_ = os.MkdirAll(topicsDir, 0o755)

	_ = os.WriteFile(filepath.Join(curriculumDir, "syllabus.yaml"), []byte(`
id: malaysia-kssm
name: "Kurikulum Standard Sekolah Menengah"
country: malaysia
board: kssm
level: secondary
`), 0o644)

	_ = os.WriteFile(filepath.Join(curriculumDir, "subject.yaml"), []byte(`
id: malaysia-kssm-matematik-tingkatan-1
name: "Matematik Tingkatan 1"
name_en: "Mathematics Form 1"
syllabus_id: malaysia-kssm
grade_id: tingkatan-1
country_id: malaysia
language: ms
topics:
  - F1-01
`), 0o644)

	// Topic YAML
	_ = os.WriteFile(filepath.Join(topicsDir, "01-variables.yaml"), []byte(`
id: F1-01
name: "Variables & Algebraic Expressions"
subject_id: malaysia-kssm-matematik-tingkatan-1
syllabus_id: malaysia-kssm
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
	_ = os.WriteFile(filepath.Join(topicsDir, "01-variables.teaching.md"), []byte(`# Variables & Algebraic Expressions — Teaching Notes

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

	// Assessment YAML
	_ = os.WriteFile(filepath.Join(topicsDir, "01-variables.assessments.yaml"), []byte(`
topic_id: F1-01
provenance: human
questions:
  - id: Q1
    text: "Evaluate the expression 3x when x = 2. Reply with the final value only."
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "6"
      working: "Substitute x = 2, then multiply 3 by 2 to get 6."
    marks: 1
    hints:
      - level: 1
        text: "Replace x with 2 before you do the multiplication."
  - id: Q2
    text: "Is the value of x fixed or can it vary? Explain briefly in one sentence."
    difficulty: medium
    learning_objective: LO2
    answer:
      type: free_text
      value: "varied"
      working: "The value can change depending on the situation or problem being described."
    marks: 2
    rubric:
      - marks: 1
        criteria: "States that the value is varied."
      - marks: 1
        criteria: "Gives a brief valid explanation."
    hints:
      - level: 1
        text: "Think about whether a variable must keep one value forever."
`), 0o644)

	return dir
}

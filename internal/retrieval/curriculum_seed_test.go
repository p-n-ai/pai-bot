// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

func TestSeedCurriculumIndexesOnlyAITeachingReadyTopics(t *testing.T) {
	loader := newSeedCurriculumLoader(t)
	service := retrieval.NewService()

	if err := retrieval.SeedCurriculum(service, loader); err != nil {
		t.Fatalf("SeedCurriculum() error = %v", err)
	}
	if _, err := service.GetDocument("topic:F1-01"); err != nil {
		t.Fatalf("GetDocument(ready topic) error = %v", err)
	}
	if _, err := service.GetDocument("topic:F1-02"); !errors.Is(err, retrieval.ErrNotFound) {
		t.Fatalf("GetDocument(lower-quality topic) error = %v, want ErrNotFound", err)
	}
}

func newSeedCurriculumLoader(t *testing.T) *curriculum.Loader {
	t.Helper()
	root := t.TempDir()
	curriculumDir := filepath.Join(root, "curricula", "test")
	topicsDir := filepath.Join(curriculumDir, "grade", "topics")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(topics) error = %v", err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}
	write(filepath.Join(curriculumDir, "syllabus.yaml"), `
id: test-syllabus
name: Test Syllabus
subjects:
  - test-math
`)
	write(filepath.Join(curriculumDir, "subject.yaml"), `
id: test-math
name: Test Mathematics
syllabus_id: test-syllabus
`)
	write(filepath.Join(curriculumDir, "grade", "subject-grade.yaml"), `
id: test-math-form-1
name: Test Mathematics Form 1
subject_id: test-math
syllabus_id: test-syllabus
grade_id: form-1
topics:
  - F1-01
  - F1-02
`)
	write(filepath.Join(topicsDir, "F1-01.yaml"), `
id: F1-01
name: Ready Algebra
subject_grade_id: test-math-form-1
subject_id: test-math
syllabus_id: test-syllabus
content_standards:
  - id: CS1
    text: Algebra
learning_objectives:
  - id: LO1
    content_standard_id: CS1
    text: Solve an equation.
quality_level: 3
`)
	write(filepath.Join(topicsDir, "F1-01.teaching.md"), "# Ready algebra\n")
	write(filepath.Join(topicsDir, "F1-01.examples.yaml"), `
topic_id: F1-01
worked_examples:
  - id: WE-01
    topic: Solving equations
    difficulty: easy
    learning_objective: LO1
    misconception_alert: Equality applies to only one side.
    scenario: Solve x plus 1 equals 2.
    working: Subtract 1 from both sides.
`)
	write(filepath.Join(topicsDir, "F1-01.assessments.yaml"), `
topic_id: F1-01
questions:
  - id: Q1
    text: Solve x plus 1 equals 2.
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "1"
`)
	write(filepath.Join(topicsDir, "F1-02.yaml"), `
id: F1-02
name: Draft Geometry
subject_grade_id: test-math-form-1
subject_id: test-math
syllabus_id: test-syllabus
content_standards:
  - id: CS2
    text: Geometry
learning_objectives:
  - id: LO2
    content_standard_id: CS2
    text: Identify an angle.
quality_level: 2
`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

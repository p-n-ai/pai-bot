// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

func TestPendingUnlocks_AddAndDrain(t *testing.T) {
	pu := newPendingUnlocks()
	identity, err := NewLearnerIdentity("telegram", "user1")
	if err != nil {
		t.Fatal(err)
	}

	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "Linear Equations"},
	}

	pu.add(identity, topics)

	drained := pu.drain(identity)
	if len(drained) != 1 {
		t.Fatalf("expected 1 drained topic, got %d", len(drained))
	}
	if drained[0].ID != "F1-06" {
		t.Errorf("expected F1-06, got %s", drained[0].ID)
	}

	// Second drain should be empty.
	drained = pu.drain(identity)
	if len(drained) != 0 {
		t.Errorf("expected 0 drained topics after second drain, got %d", len(drained))
	}
}

func TestPendingUnlocks_AddEmpty(t *testing.T) {
	pu := newPendingUnlocks()
	identity, err := NewLearnerIdentity("telegram", "user1")
	if err != nil {
		t.Fatal(err)
	}
	pu.add(identity, nil)

	drained := pu.drain(identity)
	if len(drained) != 0 {
		t.Errorf("expected 0 drained topics, got %d", len(drained))
	}
}

func TestPendingUnlocks_IsolatesSameExternalIDAcrossChannels(t *testing.T) {
	pu := newPendingUnlocks()
	telegram, _ := NewLearnerIdentity("telegram", "shared")
	slack, _ := NewLearnerIdentity("slack", "shared")

	pu.add(telegram, []curriculum.Topic{{ID: "telegram-topic"}})
	pu.add(slack, []curriculum.Topic{{ID: "slack-topic"}})

	if got := pu.drain(telegram); len(got) != 1 || got[0].ID != "telegram-topic" {
		t.Fatalf("telegram drain = %#v, want telegram topic only", got)
	}
	if got := pu.drain(slack); len(got) != 1 || got[0].ID != "slack-topic" {
		t.Fatalf("slack drain = %#v, want slack topic only", got)
	}
}

func TestBuildPrereqGraphExcludesLowerQualityUnlockDestinations(t *testing.T) {
	loader := newUnlockCurriculumLoader(t)
	graph := buildPrereqGraph(loader)
	if graph == nil {
		t.Fatal("buildPrereqGraph() = nil, want graph for ready source topic")
	}

	unlocked := graph.UnlockableTopics("F1-01", map[string]float64{"F1-01": 1})
	if len(unlocked) != 0 {
		t.Fatalf("UnlockableTopics() = %#v, want lower-quality destination excluded", unlocked)
	}
}

func TestFormatUnlockNotification(t *testing.T) {
	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "Persamaan Linear"},
		{ID: "F1-07", Name: "Ketaksamaan Linear"},
	}

	msg := formatUnlockNotification("ms", topics)
	if !strings.Contains(msg, "Persamaan Linear") {
		t.Errorf("expected topic name in notification, got: %s", msg)
	}
	if !strings.Contains(msg, "Ketaksamaan Linear") {
		t.Errorf("expected second topic name in notification, got: %s", msg)
	}
	if !strings.Contains(msg, "/learn") {
		t.Errorf("expected /learn hint in notification, got: %s", msg)
	}
}

func TestFormatUnlockNotification_Empty(t *testing.T) {
	msg := formatUnlockNotification("ms", nil)
	if msg != "" {
		t.Errorf("expected empty notification for nil topics, got: %s", msg)
	}
}

func TestFormatUnlockNotification_English(t *testing.T) {
	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "Linear Equations"},
	}
	msg := formatUnlockNotification("en", topics)
	if !strings.Contains(msg, "Congratulations") {
		t.Errorf("expected English notification, got: %s", msg)
	}
}

func TestFormatUnlockNotification_Chinese(t *testing.T) {
	_ = i18n.S("zh", i18n.MsgTopicUnlocked, "test") // ensure key exists
	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "线性方程"},
	}
	msg := formatUnlockNotification("zh", topics)
	if !strings.Contains(msg, "线性方程") {
		t.Errorf("expected Chinese topic name, got: %s", msg)
	}
}

func newUnlockCurriculumLoader(t *testing.T) *curriculum.Loader {
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
topics:
  - F1-01
  - F1-02
`)
	write(filepath.Join(topicsDir, "F1-01.yaml"), `
id: F1-01
name: Ready Numbers
subject_grade_id: test-math-form-1
subject_id: test-math
syllabus_id: test-syllabus
content_standards:
  - id: CS1
    text: Numbers
learning_objectives:
  - id: LO1
    content_standard_id: CS1
    text: Compare numbers.
quality_level: 3
`)
	write(filepath.Join(topicsDir, "F1-01.teaching.md"), "# Ready numbers\n")
	write(filepath.Join(topicsDir, "F1-01.examples.yaml"), `
topic_id: F1-01
worked_examples:
  - id: WE-01
    topic: Comparing numbers
    difficulty: easy
    learning_objective: LO1
    misconception_alert: The smaller number is larger.
    scenario: Compare 2 and 1.
    working: Locate both on a number line.
`)
	write(filepath.Join(topicsDir, "F1-01.assessments.yaml"), `
topic_id: F1-01
questions:
  - id: Q1
    text: Which is larger, 2 or 1?
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "2"
`)
	write(filepath.Join(topicsDir, "F1-02.yaml"), `
id: F1-02
name: Draft Algebra
subject_grade_id: test-math-form-1
subject_id: test-math
syllabus_id: test-syllabus
content_standards:
  - id: CS2
    text: Algebra
learning_objectives:
  - id: LO2
    content_standard_id: CS2
    text: Solve equations.
prerequisites:
  required:
    - F1-01
quality_level: 2
`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

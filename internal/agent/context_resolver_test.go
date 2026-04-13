// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestCurriculumContextResolver_Resolve_MatchedTopicWithTeachingNotes(t *testing.T) {
	loader := createCurriculumLoaderForResolverTest(t, true)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.Resolve("Please teach me linear equations")
	if topic == nil {
		t.Fatal("expected matched topic, got nil")
	}
	if topic.ID != "F1-02" {
		t.Fatalf("topic.ID = %q, want F1-02", topic.ID)
	}
	if notes == "" {
		t.Fatal("expected teaching notes, got empty")
	}
	if !contains(notes, "subtract 5 on both sides") {
		t.Fatalf("teaching notes missing expected content, got: %s", notes)
	}
}

func TestCurriculumContextResolver_Resolve_NoMatch(t *testing.T) {
	loader := createCurriculumLoaderForResolverTest(t, true)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.Resolve("what is your favorite color")
	if topic != nil {
		t.Fatalf("expected no topic match, got %q", topic.ID)
	}
	if notes != "" {
		t.Fatalf("expected empty notes, got: %s", notes)
	}
}

func TestCurriculumContextResolver_Resolve_MatchWithoutTeachingNotes(t *testing.T) {
	loader := createCurriculumLoaderForResolverTest(t, false)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.Resolve("help me with linear equations")
	if topic == nil {
		t.Fatal("expected matched topic, got nil")
	}
	if topic.ID != "F1-02" {
		t.Fatalf("topic.ID = %q, want F1-02", topic.ID)
	}
	if notes != "" {
		t.Fatalf("expected empty notes when teaching notes are missing, got: %s", notes)
	}
}

func TestCurriculumContextResolver_ResolveQuery_PrefersUserForm(t *testing.T) {
	loader := createMultiTopicCurriculumLoaderForResolverTest(t, true)
	store := agent.NewMemoryStore()
	if err := store.SetUserForm("student-1", "2"); err != nil {
		t.Fatalf("SetUserForm() error = %v", err)
	}

	resolver := agent.NewCurriculumContextResolver(loader, agent.WithResolverStore(store))
	topic, _ := resolver.ResolveQuery(agent.ContextQuery{
		Text:   "Please teach me linear equations",
		UserID: "student-1",
	})
	if topic == nil {
		t.Fatal("expected matched topic, got nil")
	}
	if topic.ID != "F2-02" {
		t.Fatalf("topic.ID = %q, want F2-02", topic.ID)
	}
}

func TestCurriculumContextResolver_ResolveQuery_UsesActiveTopicForFollowUp(t *testing.T) {
	loader := createMultiTopicCurriculumLoaderForResolverTest(t, true)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.ResolveQuery(agent.ContextQuery{
		Text:                "why move it to the other side again?",
		ConversationTopicID: "F1-02",
	})
	if topic == nil {
		t.Fatal("expected matched topic, got nil")
	}
	if topic.ID != "F1-02" {
		t.Fatalf("topic.ID = %q, want F1-02", topic.ID)
	}
	if !contains(notes, "subtract 5 on both sides") {
		t.Fatalf("notes = %q, want retrieved teaching note content", notes)
	}
}

func TestCurriculumContextResolver_ResolveQuery_RetrievesRelevantNoteSection(t *testing.T) {
	loader := createMultiTopicCurriculumLoaderForResolverTest(t, true)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.ResolveQuery(agent.ContextQuery{
		Text: "I keep getting confused by double negatives in elimination",
	})
	if topic == nil {
		t.Fatal("expected matched topic, got nil")
	}
	if topic.ID != "F2-02" {
		t.Fatalf("topic.ID = %q, want F2-02", topic.ID)
	}
	if !contains(notes, "double negatives") {
		t.Fatalf("notes = %q, want misconception section", notes)
	}
}

func TestCurriculumContextResolver_ResolveQuery_ReturnsNoMatchWhenFormIsAmbiguous(t *testing.T) {
	loader := createMultiTopicCurriculumLoaderForResolverTest(t, true)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.ResolveQuery(agent.ContextQuery{
		Text: "Please teach me linear equations",
	})
	if topic != nil {
		t.Fatalf("expected no topic for ambiguous cross-form query, got %q", topic.ID)
	}
	if notes != "" {
		t.Fatalf("expected empty notes for ambiguous query, got %q", notes)
	}
}

func createCurriculumLoaderForResolverTest(t *testing.T, withNotes bool) *curriculum.Loader {
	t.Helper()

	dir := t.TempDir()
	curriculumDir := filepath.Join(dir, "curricula", "malaysia", "kssm")
	f1TopicsDir := filepath.Join(curriculumDir, "form-1", "topics")
	if err := os.MkdirAll(f1TopicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(form1) error = %v", err)
	}

	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	write(filepath.Join(curriculumDir, "syllabus.yaml"), `id: malaysia-kssm
name: "Kurikulum Standard Sekolah Menengah"
country: malaysia
board: kssm
level: secondary
`)

	write(filepath.Join(filepath.Join(curriculumDir, "form-1"), "subject.yaml"), `id: malaysia-kssm-matematik-tingkatan-1
name: "Matematik Tingkatan 1"
name_en: "Mathematics Form 1"
syllabus_id: malaysia-kssm
grade_id: tingkatan-1
topics:
  - F1-02
`)

	write(filepath.Join(f1TopicsDir, "01-linear-equations.yaml"), `id: F1-02
official_ref: "Bab 6"
name: "Persamaan Linear (Linear Equations)"
subject_id: malaysia-kssm-matematik-tingkatan-1
syllabus_id: malaysia-kssm
difficulty: beginner
learning_objectives:
  - id: LO1
    text: Solve linear equations in one variable
    bloom: apply
`)

	if withNotes {
		write(filepath.Join(f1TopicsDir, "01-linear-equations.teaching.md"), `# Linear Equations Teaching Notes

## Balance Method
Use balance method and subtract 5 on both sides.`)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

func createMultiTopicCurriculumLoaderForResolverTest(t *testing.T, withNotes bool) *curriculum.Loader {
	t.Helper()

	dir := t.TempDir()
	curriculumDir := filepath.Join(dir, "curricula", "malaysia", "kssm")
	f1TopicsDir := filepath.Join(curriculumDir, "form-1", "topics")
	f2TopicsDir := filepath.Join(curriculumDir, "form-2", "topics")
	if err := os.MkdirAll(f1TopicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(form1) error = %v", err)
	}
	if err := os.MkdirAll(f2TopicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(form2) error = %v", err)
	}

	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	write(filepath.Join(curriculumDir, "syllabus.yaml"), `id: malaysia-kssm
name: "Kurikulum Standard Sekolah Menengah"
country: malaysia
board: kssm
level: secondary
`)

	write(filepath.Join(filepath.Join(curriculumDir, "form-1"), "subject.yaml"), `id: malaysia-kssm-matematik-tingkatan-1
name: "Matematik Tingkatan 1"
name_en: "Mathematics Form 1"
syllabus_id: malaysia-kssm
grade_id: tingkatan-1
topics:
  - F1-02
`)
	write(filepath.Join(filepath.Join(curriculumDir, "form-2"), "subject.yaml"), `id: malaysia-kssm-matematik-tingkatan-2
name: "Matematik Tingkatan 2"
name_en: "Mathematics Form 2"
syllabus_id: malaysia-kssm
grade_id: tingkatan-2
topics:
  - F2-02
`)

	write(filepath.Join(f1TopicsDir, "01-linear-equations.yaml"), `id: F1-02
official_ref: "Bab 6"
name: "Persamaan Linear (Linear Equations)"
subject_id: malaysia-kssm-matematik-tingkatan-1
syllabus_id: malaysia-kssm
difficulty: beginner
learning_objectives:
  - id: LO1
    text: Solve linear equations in one variable
    bloom: apply
`)
	write(filepath.Join(f2TopicsDir, "02-linear-equations.yaml"), `id: F2-02
official_ref: "Bab 3"
name: "Persamaan Linear (Linear Equations)"
subject_id: malaysia-kssm-matematik-tingkatan-2
syllabus_id: malaysia-kssm
difficulty: intermediate
learning_objectives:
  - id: LO1
    text: Solve simultaneous linear equations using elimination
    bloom: apply
`)

	if withNotes {
		write(filepath.Join(f1TopicsDir, "01-linear-equations.teaching.md"), `# Linear Equations Teaching Notes

## Balance Method
Use balance method and subtract 5 on both sides.`)
		write(filepath.Join(f2TopicsDir, "02-linear-equations.teaching.md"), `# Simultaneous Linear Equations Teaching Notes

## Common Misconceptions
Students often get confused by double negatives during elimination.`)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

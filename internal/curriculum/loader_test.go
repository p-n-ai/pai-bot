// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestLoader_LoadsCanonicalTopicWithEnglishNameOnly(t *testing.T) {
	dir := setupTestCurriculum(t)
	topicPath := filepath.Join(
		dir,
		"curricula",
		"malaysia",
		"kssm",
		"topics",
		"algebra",
		"01-variables.yaml",
	)
	replaceFixtureText(
		t,
		topicPath,
		`name: "Variables & Algebraic Expressions"`,
		`name_en: "Variables & Algebraic Expressions"`,
	)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	topic, found := loader.GetTopic("F1-01")
	if !found || topic.NameEN != "Variables & Algebraic Expressions" {
		t.Fatalf("GetTopic(F1-01) = %#v, found = %v", topic, found)
	}
}

func TestLoader_SnapshotStats(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	want := curriculum.SnapshotStats{
		Syllabi:       1,
		Subjects:      1,
		SubjectGrades: 1,
		Topics:        1,
		TeachingNotes: 1,
		Assessments:   1,
	}
	if got := loader.SnapshotStats(); got != want {
		t.Fatalf("SnapshotStats() = %#v, want %#v", got, want)
	}
}

func TestLoader_LoadsAndCountsLevelZeroCanonicalStub(t *testing.T) {
	dir := setupTestCurriculum(t)
	replaceFixtureTopicWithStub(t, dir, 0)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topic, found := loader.GetTopic("F1-01")
	if !found {
		t.Fatal("GetTopic(F1-01) not found")
	}
	if topic.QualityLevel != 0 {
		t.Fatalf("topic.QualityLevel = %d, want 0", topic.QualityLevel)
	}
	if len(topic.ContentStandards) != 0 || len(topic.LearningObjectives) != 0 {
		t.Fatalf(
			"stub teaching identity = %d standards, %d objectives; want both empty",
			len(topic.ContentStandards),
			len(topic.LearningObjectives),
		)
	}
	if got := loader.SnapshotStats().Topics; got != 1 {
		t.Fatalf("SnapshotStats().Topics = %d, want 1", got)
	}
}

func TestLoader_RejectsCatalogStubWithMissingQualityClaim(t *testing.T) {
	dir := setupTestCurriculum(t)
	replaceFixtureTopicWithStub(t, dir, 0)
	topicPath := filepath.Join(
		dir,
		"curricula",
		"malaysia",
		"kssm",
		"topics",
		"algebra",
		"01-variables.yaml",
	)
	replaceFixtureText(t, topicPath, "quality_level: 0", "")

	_, err := curriculum.NewLoader(dir)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want missing quality claim rejected as a catalog stub")
	}
	if !strings.Contains(err.Error(), `topic "F1-01" requires content standards`) {
		t.Fatalf("NewLoader() error = %v, want missing teaching identity", err)
	}
}

func TestLoader_RejectsEmptyTeachingIdentityAboveLevelZero(t *testing.T) {
	dir := setupTestCurriculum(t)
	replaceFixtureTopicWithStub(t, dir, 1)

	_, err := curriculum.NewLoader(dir)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want empty teaching identity rejected")
	}
	if !strings.Contains(err.Error(), `topic "F1-01" requires content standards`) {
		t.Fatalf("NewLoader() error = %v, want missing content standards", err)
	}
}

func TestLoader_RejectsPartiallyAuthoredLevelZeroTopic(t *testing.T) {
	dir := setupTestCurriculum(t)
	replaceFixtureTopicWithStub(t, dir, 0)
	topicPath := filepath.Join(
		dir,
		"curricula",
		"malaysia",
		"kssm",
		"topics",
		"algebra",
		"01-variables.yaml",
	)
	replaceFixtureText(
		t,
		topicPath,
		"content_standards: []",
		"content_standards:\n  - id: \"1.1\"\n    text: Variables",
	)

	_, err := curriculum.NewLoader(dir)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want partial Level 0 topic rejected")
	}
	if !strings.Contains(err.Error(), `topic "F1-01" requires learning objectives`) {
		t.Fatalf("NewLoader() error = %v, want missing learning objectives", err)
	}
}

func TestLoader_SnapshotRevisionTracksCurriculumContent(t *testing.T) {
	dir := setupTestCurriculum(t)

	first, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("first NewLoader() error = %v", err)
	}
	firstRevision := first.SnapshotRevision()
	if !strings.HasPrefix(firstRevision, "sha256:") {
		t.Fatalf("SnapshotRevision() = %q, want sha256 digest", firstRevision)
	}
	firstAssessment, found := first.GetAssessment("F1-01")
	if !found || !strings.HasPrefix(firstAssessment.Source.Revision, "sha256:") {
		t.Fatalf("assessment source = %#v, found = %v", firstAssessment.Source, found)
	}
	firstTopic, found := first.GetTopic("F1-01")
	if !found || !strings.HasPrefix(firstTopic.Source.Revision, "sha256:") {
		t.Fatalf("topic source = %#v, found = %v", firstTopic.Source, found)
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("not curriculum"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	withReadme, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() after README error = %v", err)
	}
	if got := withReadme.SnapshotRevision(); got != firstRevision {
		t.Fatalf("revision changed for non-curriculum file: %q != %q", got, firstRevision)
	}
	readmeAssessment, _ := withReadme.GetAssessment("F1-01")
	if readmeAssessment.Source.Revision != firstAssessment.Source.Revision {
		t.Fatalf(
			"assessment source revision changed for README: %q != %q",
			readmeAssessment.Source.Revision,
			firstAssessment.Source.Revision,
		)
	}

	assessmentPath := filepath.Join(
		dir,
		"curricula",
		"malaysia",
		"kssm",
		"topics",
		"algebra",
		"01-variables.assessments.yaml",
	)
	assessment, err := os.ReadFile(assessmentPath)
	if err != nil {
		t.Fatalf("ReadFile(assessment) error = %v", err)
	}
	if err := os.WriteFile(assessmentPath, append(assessment, []byte("\n# revision change\n")...), 0o644); err != nil {
		t.Fatalf("WriteFile(assessment) error = %v", err)
	}

	changed, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() after assessment change error = %v", err)
	}
	if got := changed.SnapshotRevision(); got == firstRevision {
		t.Fatalf("SnapshotRevision() = %q after curriculum content changed", got)
	}
	loadedAssessment, found := changed.GetAssessment("F1-01")
	if !found || loadedAssessment.Source.Revision == firstAssessment.Source.Revision {
		t.Fatalf(
			"assessment source revision = %q after artifact changed, found = %v",
			loadedAssessment.Source.Revision,
			found,
		)
	}
	loadedTopic, found := changed.GetTopic("F1-01")
	if !found || loadedTopic.Source.Revision != firstTopic.Source.Revision {
		t.Fatalf(
			"topic source revision = %q after unrelated assessment changed, want %q",
			loadedTopic.Source.Revision,
			firstTopic.Source.Revision,
		)
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

func TestLoader_GetTeachingNotePreservesSource(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	note, found := loader.GetTeachingNote("F1-01")
	if !found || !strings.Contains(note.Content, "Variables & Algebraic Expressions") {
		t.Fatalf("GetTeachingNote(F1-01) = %#v, %v; want typed note content", note, found)
	}
	if note.TopicID != "F1-01" ||
		note.Source.Kind != "teaching_note" ||
		note.Source.Path == "" ||
		!strings.HasPrefix(note.Source.Revision, "sha256:") {
		t.Fatalf("note = %#v, want topic-scoped artifact source", note)
	}
	notes, found := loader.GetTeachingNotes("F1-01")
	if !found || notes != note.Content {
		t.Fatalf("GetTeachingNotes(F1-01) = %q, %v; want content projection", notes, found)
	}
}

func TestLoader_GetTeachingNotesVariantWithoutTopicVariant(t *testing.T) {
	dir := setupTestCurriculum(t)
	translationDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra", "translations", "en")
	if err := os.MkdirAll(translationDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(translationDir, "F1-01.teaching.md"),
		[]byte("# Teach variables in English"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	notes, found := loader.GetTeachingNotesVariant("F1-01", "en")
	if !found || notes == "" {
		t.Fatalf("GetTeachingNotesVariant(F1-01, en) = %q, %v; want notes", notes, found)
	}
	if got := loader.SnapshotStats().TeachingNoteVariants; got != 1 {
		t.Fatalf("SnapshotStats().TeachingNoteVariants = %d, want 1", got)
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
	if assessment.Questions[0].Type != "short_answer" {
		t.Fatalf("assessment.Questions[0].Type = %q, want short_answer", assessment.Questions[0].Type)
	}
}

func TestLoader_GetExamples(t *testing.T) {
	dir := setupTestCurriculum(t)
	examplesPath := filepath.Join(
		dir,
		"curricula",
		"malaysia",
		"kssm",
		"topics",
		"algebra",
		"01-variables.examples.yaml",
	)
	if err := os.WriteFile(examplesPath, []byte(`
topic_id: F1-01
provenance: human
description: Worked algebra examples.
worked_examples:
  - id: WE-01
    topic: Evaluating expressions
    difficulty: easy
    learning_objective: LO1
    real_world_analogy: Three bags with two apples each.
    misconception_alert: Addition and multiplication are interchangeable.
    scenario: Evaluate 3x when x is 2.
    working: Substitute 2 for x, then calculate 3 times 2.
    source_ref: Textbook page 12.
`), 0o644); err != nil {
		t.Fatalf("WriteFile(examples) error = %v", err)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	examples, found := loader.GetExamples("F1-01")
	if !found {
		t.Fatal("GetExamples(F1-01) not found")
	}
	if examples.TopicID != "F1-01" ||
		examples.Description != "Worked algebra examples." ||
		len(examples.WorkedExamples) != 1 ||
		examples.WorkedExamples[0].ID != "WE-01" {
		t.Fatalf("GetExamples(F1-01) = %#v, want typed worked example", examples)
	}
	if examples.Source.Kind != "examples" ||
		examples.Source.Path == "" ||
		!strings.HasPrefix(examples.Source.Revision, "sha256:") {
		t.Fatalf("examples.Source = %#v, want artifact source reference", examples.Source)
	}
}

func TestLoader_RejectsExamplesForMissingTopic(t *testing.T) {
	dir := setupTestCurriculum(t)
	examplesPath := filepath.Join(
		dir,
		"curricula",
		"malaysia",
		"kssm",
		"topics",
		"algebra",
		"missing.examples.yaml",
	)
	if err := os.WriteFile(examplesPath, []byte(`
topic_id: F1-99
worked_examples:
  - id: WE-01
    topic: Missing topic example
    difficulty: easy
    misconception_alert: Do not guess the topic.
    scenario: Explain the missing topic.
    working: Find the canonical topic first.
`), 0o644); err != nil {
		t.Fatalf("WriteFile(examples) error = %v", err)
	}

	_, err := curriculum.NewLoader(dir)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want missing example topic rejected")
	}
	if !strings.Contains(err.Error(), `examples reference missing topic "F1-99"`) {
		t.Fatalf("NewLoader() error = %v, want missing example topic", err)
	}
}

func TestLoader_RejectsInvalidWorkedExamples(t *testing.T) {
	const validExamples = `
topic_id: F1-01
worked_examples:
  - id: WE-01
    topic: Evaluating expressions
    difficulty: easy
    learning_objective: LO1
    misconception_alert: Addition and multiplication are interchangeable.
    scenario: Evaluate 3x when x is 2.
    working: Substitute 2 for x, then calculate 3 times 2.
`
	tests := []struct {
		name    string
		old     string
		new     string
		wantErr string
	}{
		{
			name:    "missing ID",
			old:     "id: WE-01",
			new:     `id: ""`,
			wantErr: "worked example requires id",
		},
		{
			name: "duplicate ID",
			old:  "    working: Substitute 2 for x, then calculate 3 times 2.",
			new: `    working: Substitute 2 for x, then calculate 3 times 2.
  - id: WE-01
    topic: Another evaluation
    difficulty: medium
    misconception_alert: Repeated IDs are different examples.
    scenario: Evaluate x plus 1.
    working: Substitute x, then add 1.`,
			wantErr: `repeats worked example "WE-01"`,
		},
		{
			name:    "missing topic",
			old:     "topic: Evaluating expressions",
			new:     `topic: ""`,
			wantErr: `worked example "WE-01" requires topic`,
		},
		{
			name:    "invalid difficulty",
			old:     "difficulty: easy",
			new:     "difficulty: impossible",
			wantErr: `worked example "WE-01" has invalid difficulty`,
		},
		{
			name:    "missing misconception",
			old:     "misconception_alert: Addition and multiplication are interchangeable.",
			new:     `misconception_alert: ""`,
			wantErr: `worked example "WE-01" requires misconception_alert`,
		},
		{
			name:    "missing scenario",
			old:     "scenario: Evaluate 3x when x is 2.",
			new:     `scenario: ""`,
			wantErr: `worked example "WE-01" requires scenario`,
		},
		{
			name:    "missing working",
			old:     "working: Substitute 2 for x, then calculate 3 times 2.",
			new:     `working: ""`,
			wantErr: `worked example "WE-01" requires working`,
		},
		{
			name:    "unknown objective",
			old:     "learning_objective: LO1",
			new:     "learning_objective: LO-UNKNOWN",
			wantErr: `worked example "WE-01" references missing objective "LO-UNKNOWN"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := setupTestCurriculum(t)
			examplesPath := filepath.Join(
				dir,
				"curricula",
				"malaysia",
				"kssm",
				"topics",
				"algebra",
				"01-variables.examples.yaml",
			)
			content := strings.Replace(validExamples, test.old, test.new, 1)
			if err := os.WriteFile(examplesPath, []byte(content), 0o644); err != nil {
				t.Fatalf("WriteFile(examples) error = %v", err)
			}

			_, err := curriculum.NewLoader(dir)
			if err == nil {
				t.Fatal("NewLoader() error = nil, want invalid worked example rejected")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("NewLoader() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestLoader_RejectsExampleVariantWithoutCanonicalExample(t *testing.T) {
	dir := setupTestCurriculum(t)
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	writeExamplesFixture(t, filepath.Join(topicsDir, "01-variables.examples.yaml"), "F1-01", "WE-01", "LO1", "Evaluate 3x.")
	translationDir := filepath.Join(topicsDir, "translations", "ms")
	if err := os.MkdirAll(translationDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(translation) error = %v", err)
	}
	writeExamplesFixture(t, filepath.Join(translationDir, "F1-01.examples.yaml"), "F1-01", "WE-02", "LO1", "Hitung 3x.")

	_, err := curriculum.NewLoader(dir)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want non-canonical localized example rejected")
	}
	if !strings.Contains(err.Error(), `worked example "WE-02" has no canonical worked example`) {
		t.Fatalf("NewLoader() error = %v, want missing canonical example", err)
	}
}

func TestLoader_GetExamplesVariant(t *testing.T) {
	dir := setupTestCurriculum(t)
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	writeExamplesFixture(t, filepath.Join(topicsDir, "01-variables.examples.yaml"), "F1-01", "WE-01", "LO1", "Evaluate 3x.")
	translationDir := filepath.Join(topicsDir, "translations", "ms")
	if err := os.MkdirAll(translationDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(translation) error = %v", err)
	}
	writeExamplesFixture(t, filepath.Join(translationDir, "F1-01.examples.yaml"), "F1-01", "WE-01", "LO1", "Hitung 3x.")

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	variant, found := loader.GetExamplesVariant("F1-01", "ms")
	if !found || len(variant.WorkedExamples) != 1 ||
		variant.WorkedExamples[0].Scenario != "Hitung 3x." {
		t.Fatalf("GetExamplesVariant(F1-01, ms) = %#v, %v; want Malay example", variant, found)
	}
	if variant.Source.Kind != "examples_variant" || variant.Source.Locale != "ms" {
		t.Fatalf("variant.Source = %#v, want cited Malay example artifact", variant.Source)
	}
	stats := loader.SnapshotStats()
	if stats.Examples != 1 || stats.ExampleVariants != 1 {
		t.Fatalf("SnapshotStats() = %#v, want one canonical and one variant example set", stats)
	}
}

func TestLoader_RejectsExampleVariantObjectiveChange(t *testing.T) {
	dir := setupTestCurriculum(t)
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	writeExamplesFixture(t, filepath.Join(topicsDir, "01-variables.examples.yaml"), "F1-01", "WE-01", "LO1", "Evaluate 3x.")
	translationDir := filepath.Join(topicsDir, "translations", "ms")
	if err := os.MkdirAll(translationDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(translation) error = %v", err)
	}
	writeExamplesFixture(t, filepath.Join(translationDir, "F1-01.examples.yaml"), "F1-01", "WE-01", "LO2", "Hitung 3x.")

	_, err := curriculum.NewLoader(dir)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want localized objective change rejected")
	}
	if !strings.Contains(err.Error(), `worked example "WE-01" changes canonical objective identity`) {
		t.Fatalf("NewLoader() error = %v, want canonical objective identity", err)
	}
}

func TestLoader_ExamplesAreImmutableSnapshots(t *testing.T) {
	dir := setupTestCurriculum(t)
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	writeExamplesFixture(t, filepath.Join(topicsDir, "01-variables.examples.yaml"), "F1-01", "WE-01", "LO1", "Evaluate 3x.")

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	first, found := loader.GetExamples("F1-01")
	if !found {
		t.Fatal("GetExamples(F1-01) not found")
	}
	first.WorkedExamples[0].Scenario = "mutated"

	again, found := loader.GetExamples("F1-01")
	if !found || again.WorkedExamples[0].Scenario != "Evaluate 3x." {
		t.Fatalf("GetExamples(F1-01) after mutation = %#v, %v; want immutable snapshot", again, found)
	}
}

func TestLoader_LoadsSubjectGradeMetadata(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	subject, found := loader.GetSubjectGrade("malaysia-kssm-matematik-tingkatan-1")
	if !found {
		t.Fatal("GetSubjectGrade() not found")
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
provenance: human
questions:
  - id: Q1
    text: "What is 3x when x=2?"
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "6"
    marks: 1
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

func TestLoader_RejectsTeachingNotesWithoutYAML(t *testing.T) {
	dir := t.TempDir()

	topicsDir := filepath.Join(dir, "topics")
	_ = os.MkdirAll(topicsDir, 0o755)

	_ = os.WriteFile(filepath.Join(topicsDir, "orphan.teaching.md"), []byte("# Orphan notes"), 0o644)

	if _, err := curriculum.NewLoader(dir); err == nil {
		t.Fatal("NewLoader() error = nil, want orphan teaching notes rejected")
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
subjects:
  - malaysia-kssm-matematik
`), 0o644)

	_ = os.WriteFile(filepath.Join(curriculumDir, "subject.yaml"), []byte(`
id: malaysia-kssm-matematik
name: "Matematik"
name_en: "Mathematics"
syllabus_id: malaysia-kssm
country_id: malaysia
language: ms
`), 0o644)

	_ = os.WriteFile(filepath.Join(curriculumDir, "subject-grade.yaml"), []byte(`
id: malaysia-kssm-matematik-tingkatan-1
name: "Matematik Tingkatan 1"
name_en: "Mathematics Form 1"
subject_id: malaysia-kssm-matematik
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
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
difficulty: beginner
content_standards:
  - id: "1.1"
    text: "Variables and algebraic expressions"
learning_objectives:
  - id: LO1
    content_standard_id: "1.1"
    text: "Use letters to represent unknown quantities"
    bloom: remember
  - id: LO2
    content_standard_id: "1.1"
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
    type: short_answer
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

func writeExamplesFixture(
	t *testing.T,
	path, topicID, exampleID, objectiveID, scenario string,
) {
	t.Helper()
	content := fmt.Sprintf(`
topic_id: %s
provenance: human
worked_examples:
  - id: %s
    topic: Evaluating expressions
    difficulty: easy
    learning_objective: %s
    misconception_alert: Addition and multiplication are interchangeable.
    scenario: %s
    working: Substitute 2 for x, then calculate 3 times 2.
`, topicID, exampleID, objectiveID, scenario)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func replaceFixtureTopicWithStub(t *testing.T, root string, qualityLevel int) {
	t.Helper()

	topicsDir := filepath.Join(root, "curricula", "malaysia", "kssm", "topics", "algebra")
	topic := []byte(fmt.Sprintf(`
id: F1-01
name: "Variables & Algebraic Expressions"
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
difficulty: beginner
content_standards: []
learning_objectives: []
quality_level: %d
provenance: human
`, qualityLevel))
	if err := os.WriteFile(filepath.Join(topicsDir, "01-variables.yaml"), topic, 0o644); err != nil {
		t.Fatalf("WriteFile(topic) error = %v", err)
	}
	if err := os.Remove(filepath.Join(topicsDir, "01-variables.assessments.yaml")); err != nil {
		t.Fatalf("Remove(assessment) error = %v", err)
	}
}

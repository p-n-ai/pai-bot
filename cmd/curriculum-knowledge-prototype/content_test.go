package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBuildCurriculumContentMapsOSSShape(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "syllabus.yaml", `
id: malaysia-kssm
name: KSSM
name_en: KSSM
country_id: malaysia
language: ms
subjects:
  - malaysia-kssm-matematik
`)
	writeFile(t, root, "malaysia-kssm-matematik/subject.yaml", `
id: malaysia-kssm-matematik
name: Matematik
name_en: Mathematics
syllabus_id: malaysia-kssm
country_id: malaysia
language: ms
`)
	writeFile(t, root, "malaysia-kssm-matematik/malaysia-kssm-matematik-tingkatan-2/subject-grade.yaml", `
id: malaysia-kssm-matematik-tingkatan-2
name: Matematik Tingkatan 2
name_en: Mathematics Form 2
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
grade_id: tingkatan-2
country_id: malaysia
language: ms
topics:
  - MT2-01
`)
	writeFile(t, root, "malaysia-kssm-matematik/malaysia-kssm-matematik-tingkatan-2/topics/MT2-01.yaml", `
id: MT2-01
official_ref: Bab 1
name: Pola dan Jujukan
name_en: Patterns and Sequences
subject_grade_id: malaysia-kssm-matematik-tingkatan-2
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
country_id: malaysia
language: en
difficulty: intermediate
tier: core
content_standards:
  - id: "1.1"
    text_en: Patterns
learning_objectives:
  - id: 1.1.1
    text_en: Identify patterns.
    bloom: understand
prerequisites:
  required:
    - id: MT1-00
      name_en: Prior Topic
quality_level: 3
provenance: ai-assisted
`)
	writeFile(t, root, "malaysia-kssm-matematik/malaysia-kssm-matematik-tingkatan-2/topics/MT2-01.teaching.md", `
#### Overview
Teach patterns with short examples.

#### High Alert Misconceptions
Students may assume the first jump is enough.
`)
	writeFile(t, root, "malaysia-kssm-matematik/malaysia-kssm-matematik-tingkatan-2/topics/MT2-01.examples.yaml", `
topic_id: MT2-01
provenance: ai-assisted
worked_examples:
  - id: WE-01
    topic: Identifying Patterns
    difficulty: easy
    misconception_alert: Check every jump.
    scenario: Extend 2, 4, 6.
    working: Add 2 each time.
`)
	writeFile(t, root, "malaysia-kssm-matematik/malaysia-kssm-matematik-tingkatan-2/topics/MT2-01.assessments.yaml", `
topic_id: MT2-01
provenance: ai-assisted
questions:
  - id: Q1
    text: What comes next after 2, 4, 6?
    difficulty: easy
    learning_objective: 1.1.1
    answer:
      type: exact
      value: "8"
    marks: 1
`)

	rows, err := buildCurriculumContent(root)
	if err != nil {
		t.Fatalf("build content: %v", err)
	}

	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.Kind)
		if row.Body == "" {
			t.Fatalf("row %q has empty body", row.Kind)
		}
		if row.SearchText == "" {
			t.Fatalf("row %q has empty search text", row.Kind)
		}
	}
	slices.Sort(got)

	want := []string{
		"assessment_item",
		"subject",
		"subject_grade",
		"syllabus",
		"teaching_note",
		"teaching_note",
		"topic_card",
		"worked_example",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("kinds mismatch\ngot:  %v\nwant: %v", got, want)
	}
}

func TestNewContentBuildsDeterministicSearchText(t *testing.T) {
	row := newContent("topic_card", "Patterns", "body", map[string]string{
		"topic_id":   "MT2-01",
		"country_id": "malaysia",
		"grade_id":   "tingkatan-2",
	})

	countryIndex := strings.Index(row.SearchText, "country_id: malaysia")
	gradeIndex := strings.Index(row.SearchText, "grade_id: tingkatan-2")
	topicIndex := strings.Index(row.SearchText, "topic_id: MT2-01")
	if countryIndex < 0 || gradeIndex < 0 || topicIndex < 0 {
		t.Fatalf("search text missing metadata:\n%s", row.SearchText)
	}
	if countryIndex >= gradeIndex || gradeIndex >= topicIndex {
		t.Fatalf("metadata should be sorted in search text:\n%s", row.SearchText)
	}
}

func TestPrerequisitesCanBeASequence(t *testing.T) {
	var got topicFile
	data := []byte(`
id: MT4-07
name_en: Graphs of Motion
prerequisites:
  - "Form 1: Basic linear equations."
  - "Form 2: Speed, distance, and time."
`)
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse topic: %v", err)
	}
	if len(got.Prerequisites.Required) != 2 {
		t.Fatalf("required prerequisites = %v", got.Prerequisites.Required)
	}
}

func writeFile(t *testing.T, root, name, body string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

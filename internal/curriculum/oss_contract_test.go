// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestLoaderPreservesOSSArtifactIdentityAndLocaleVariants(t *testing.T) {
	root := setupOSSContractFixture(t)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topics := loader.AllTopics()
	if len(topics) != 1 {
		t.Fatalf("AllTopics() = %d topics, want only the canonical topic", len(topics))
	}

	subjectGrade, found := loader.GetSubjectGrade("malaysia-kssm-matematik-tingkatan-1")
	if !found {
		t.Fatal("GetSubjectGrade() not found")
	}
	if len(subjectGrade.Topics) != 1 || subjectGrade.Topics[0] != "MT1-05" {
		t.Fatalf("subjectGrade.Topics = %v, want [MT1-05]", subjectGrade.Topics)
	}

	canonical, found := loader.GetTopic("MT1-05")
	if !found {
		t.Fatal("GetTopic(MT1-05) not found")
	}
	if canonical.Language != "en" {
		t.Fatalf("canonical.Language = %q, want en", canonical.Language)
	}
	if canonical.LearningObjectives[0].TextEN == "" {
		t.Fatal("canonical learning objective lost text_en")
	}

	malay, found := loader.GetTopicVariant("MT1-05", "ms")
	if !found {
		t.Fatal("GetTopicVariant(MT1-05, ms) not found")
	}
	if malay.Language != "ms" || malay.LearningObjectives[0].Text == "" {
		t.Fatalf("Malay variant = %#v, want Malay localized fields", malay)
	}

	canonicalAssessment, found := loader.GetAssessment("MT1-05")
	if !found || canonicalAssessment.Questions[0].Text != "Evaluate 3x when x = 2." {
		t.Fatalf("canonical assessment = %#v, found = %v", canonicalAssessment, found)
	}
	malayAssessment, found := loader.GetAssessmentVariant("MT1-05", "ms")
	if !found || malayAssessment.Questions[0].Text != "Hitung 3x apabila x = 2." {
		t.Fatalf("Malay assessment = %#v, found = %v", malayAssessment, found)
	}

	canonicalNotes, found := loader.GetTeachingNotes("MT1-05")
	if !found || canonicalNotes != "# Teach algebra\n" {
		t.Fatalf("canonical teaching notes = %q, found = %v", canonicalNotes, found)
	}
	malayNotes, found := loader.GetTeachingNotesVariant("MT1-05", "ms")
	if !found || malayNotes != "# Ajar algebra\n" {
		t.Fatalf("Malay teaching notes = %q, found = %v", malayNotes, found)
	}
}

func TestLoaderRejectsMalformedCanonicalAssessment(t *testing.T) {
	root := setupOSSContractFixture(t)
	assessmentPath := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
		"malaysia-kssm-matematik-tingkatan-1",
		"topics",
		"MT1-05.assessments.yaml",
	)
	writeFixtureFile(t, assessmentPath, `
topic_id: MT1-05
provenance: human
questions:
  - id: Q1
    text: "Invalid YAML escape: \circ"
`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want malformed assessment rejection")
	}
	if !strings.Contains(err.Error(), "MT1-05.assessments.yaml") {
		t.Fatalf("NewLoader() error = %v, want source path", err)
	}
}

func TestLoaderRejectsMissingPrerequisite(t *testing.T) {
	root := setupOSSContractFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `prerequisites:
  required:
    - MT1-99
quality_level: 2`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want missing prerequisite rejection")
	}
	if !strings.Contains(err.Error(), "MT1-99") || !strings.Contains(err.Error(), "MT1-05.yaml") {
		t.Fatalf("NewLoader() error = %v, want topic path and missing prerequisite", err)
	}
}

func TestLoaderRejectsSubjectOmittedFromSyllabus(t *testing.T) {
	root := setupOSSContractFixture(t)
	syllabusPath := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"syllabus.yaml",
	)
	replaceFixtureText(t, syllabusPath, `subjects:
  - malaysia-kssm-matematik`, `subjects: []`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want unlisted subject rejected")
	}
	if !strings.Contains(err.Error(), "malaysia-kssm-matematik") || !strings.Contains(err.Error(), "not listed") {
		t.Fatalf("NewLoader() error = %v, want reciprocal syllabus membership", err)
	}
}

func TestLoaderRejectsDuplicateSyllabusSubject(t *testing.T) {
	root := setupOSSContractFixture(t)
	syllabusPath := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"syllabus.yaml",
	)
	replaceFixtureText(t, syllabusPath, `  - malaysia-kssm-matematik`, `  - malaysia-kssm-matematik
  - malaysia-kssm-matematik`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want duplicate syllabus subject rejected")
	}
	if !strings.Contains(err.Error(), "repeats subject") {
		t.Fatalf("NewLoader() error = %v, want duplicate subject identity", err)
	}
}

func TestLoaderRejectsSubjectGradeOutsideSubjectSyllabus(t *testing.T) {
	root := setupOSSContractFixture(t)
	otherDir := filepath.Join(root, "curricula", "other")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", otherDir, err)
	}
	writeFixtureFile(t, filepath.Join(otherDir, "syllabus.yaml"), `
id: other-syllabus
name: Other
subjects: []
`)
	subjectGradePath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"..",
		"subject-grade.yaml",
	)
	replaceFixtureText(t, subjectGradePath, "syllabus_id: malaysia-kssm", "syllabus_id: other-syllabus")

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want cross-syllabus subject grade rejected")
	}
	if !strings.Contains(err.Error(), "does not match subject") {
		t.Fatalf("NewLoader() error = %v, want subject-grade scope mismatch", err)
	}
}

func TestLoaderRejectsSubjectListedByWrongSyllabus(t *testing.T) {
	root := setupOSSContractFixture(t)
	otherDir := filepath.Join(root, "curricula", "other")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", otherDir, err)
	}
	writeFixtureFile(t, filepath.Join(otherDir, "syllabus.yaml"), `
id: other-syllabus
name: Other
subjects:
  - malaysia-kssm-matematik
`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want cross-syllabus subject listing rejected")
	}
	if !strings.Contains(err.Error(), "owned by syllabus") {
		t.Fatalf("NewLoader() error = %v, want subject ownership mismatch", err)
	}
}

func TestLoaderPreservesRichPrerequisiteAndBackgroundKnowledge(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, `  required:
    - MT1-01`, `  required:
    - id: MT1-01
      name: Rational Numbers
      reason: Variables can contain rational values.
background_knowledge:
  - Know how to compare integers.`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	topic, _ := loader.GetTopic("MT1-05")
	if len(topic.Prerequisites.Required) != 1 ||
		topic.Prerequisites.Required[0].TopicID != "MT1-01" ||
		topic.Prerequisites.Required[0].Reason != "Variables can contain rational values." {
		t.Fatalf("required prerequisites = %#v, want rich MT1-01 edge", topic.Prerequisites.Required)
	}
	if len(topic.BackgroundKnowledge) != 1 ||
		topic.BackgroundKnowledge[0] != "Know how to compare integers." {
		t.Fatalf("background knowledge = %#v", topic.BackgroundKnowledge)
	}
}

func TestLoaderNormalizesRootEngagementHooks(t *testing.T) {
	root := setupOSSContractFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `engagement_hooks:
  - Find the mystery number.
quality_level: 2`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	topic, _ := loader.GetTopic("MT1-05")
	if len(topic.Teaching.EngagementHooks) != 1 ||
		topic.Teaching.EngagementHooks[0] != "Find the mystery number." {
		t.Fatalf("engagement hooks = %#v, want normalized root hook", topic.Teaching.EngagementHooks)
	}
}

func TestLoaderPreservesValidatedCrossCurriculumEdge(t *testing.T) {
	root := setupValidatedCrossCurriculumFixture(t)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	topic, _ := loader.GetTopic("MT1-05")
	if len(topic.CrossCurriculum) != 1 || topic.CrossCurriculum[0].ConceptID != "variables" {
		t.Fatalf("cross-curriculum edges = %#v", topic.CrossCurriculum)
	}
	concept, found := loader.GetConcept("variables")
	if !found || len(concept.Curricula) != 2 || concept.Curricula[1].TopicID != "OM1-01" {
		t.Fatalf("concept = %#v, found = %v", concept, found)
	}
}

func TestLoaderRejectsCrossCurriculumSelfLink(t *testing.T) {
	root := setupOSSContractFixture(t)
	conceptDir := filepath.Join(root, "concepts", "mathematics")
	if err := os.MkdirAll(conceptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", conceptDir, err)
	}
	writeFixtureFile(t, filepath.Join(conceptDir, "variables.yaml"), `
id: variables
name: Variables
domain: mathematics
subdomain: algebra
definition: A symbol whose value may change.
curricula:
  - syllabus: malaysia-kssm
    topic: MT1-05
    scope: introduction
`)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `cross_curriculum:
  - concept_id: variables
    syllabus_id: malaysia-kssm
    topic_id: MT1-05
quality_level: 2`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want cross-curriculum self-link rejected")
	}
	if !strings.Contains(err.Error(), "other syllabus") {
		t.Fatalf("NewLoader() error = %v, want distinct-syllabus contract", err)
	}
}

func TestLoaderRejectsDanglingCrossCurriculumConcept(t *testing.T) {
	root := setupOSSContractFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `cross_curriculum:
  - concept_id: missing-concept
    syllabus_id: other-syllabus
    topic_id: OM1-01
quality_level: 2`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want dangling cross-curriculum concept rejected")
	}
	if !strings.Contains(err.Error(), "missing-concept") {
		t.Fatalf("NewLoader() error = %v, want missing concept identity", err)
	}
}

func TestLoaderRejectsConflictingEngagementHookPlacements(t *testing.T) {
	root := setupOSSContractFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `teaching:
  engagement_hooks:
    - Nested hook.
engagement_hooks:
  - Root hook.
quality_level: 2`)

	if _, err := curriculum.NewLoader(root); err == nil {
		t.Fatal("NewLoader() error = nil, want conflicting engagement hooks rejected")
	}
}

func TestLoaderRejectsAssessmentObjectiveDrift(t *testing.T) {
	root := setupOSSContractFixture(t)
	assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	replaceFixtureText(t, assessmentPath, `learning_objective: "5.1.1"`, `learning_objective: "5.1.99"`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want orphan assessment objective rejection")
	}
	if !strings.Contains(err.Error(), "Q1") || !strings.Contains(err.Error(), "5.1.99") {
		t.Fatalf("NewLoader() error = %v, want question and orphan objective", err)
	}
}

func TestLoaderRejectsTopicWithoutPlanFields(t *testing.T) {
	tests := []struct {
		name        string
		old         string
		replacement string
	}{
		{
			name:        "name",
			old:         "name: Ungkapan Algebra\nname_en: Algebraic Expressions",
			replacement: "name: \"\"\nname_en: \"\"",
		},
		{
			name: "learning objectives",
			old: `learning_objectives:
  - id: "5.1.1"
    content_standard_id: "5.1"
    text_en: Use letters to represent unknown quantities.
    bloom: understand`,
			replacement: "learning_objectives: []",
		},
		{
			name:        "objective text",
			old:         "text_en: Use letters to represent unknown quantities.",
			replacement: `text_en: ""`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := setupOSSContractFixture(t)
			topicPath := fixtureTopicPath(root, "MT1-05.yaml")
			replaceFixtureText(t, topicPath, test.old, test.replacement)

			if _, err := curriculum.NewLoader(root); err == nil {
				t.Fatalf("NewLoader() error = nil, want missing topic %s rejected", test.name)
			}
		})
	}
}

func TestLoaderPreservesSourceCitationsWithoutOptionalProvenanceMetadata(t *testing.T) {
	root := setupOSSContractFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	variantTopicPath := filepath.Join(
		filepath.Dir(topicPath),
		"translations",
		"ms",
		"MT1-05.yaml",
	)
	assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	variantAssessmentPath := filepath.Join(
		filepath.Dir(topicPath),
		"translations",
		"ms",
		"MT1-05.assessments.yaml",
	)
	for _, fixture := range []struct {
		path       string
		provenance string
	}{
		{path: topicPath, provenance: "ai-assisted"},
		{path: variantTopicPath, provenance: "ai-assisted"},
		{path: assessmentPath, provenance: "human"},
		{path: variantAssessmentPath, provenance: "human"},
	} {
		replaceFixtureText(
			t,
			fixture.path,
			"provenance: "+fixture.provenance,
			"# provenance intentionally absent",
		)
	}

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	type sourcedArtifact struct {
		name       string
		provenance string
		source     curriculum.SourceRef
	}
	topic, topicFound := loader.GetTopic("MT1-05")
	variantTopic, variantTopicFound := loader.GetTopicVariant("MT1-05", "ms")
	assessment, assessmentFound := loader.GetAssessment("MT1-05")
	variantAssessment, variantAssessmentFound := loader.GetAssessmentVariant("MT1-05", "ms")
	if !topicFound || !variantTopicFound || !assessmentFound || !variantAssessmentFound {
		t.Fatalf(
			"artifacts found = topic:%v variant-topic:%v assessment:%v variant-assessment:%v",
			topicFound,
			variantTopicFound,
			assessmentFound,
			variantAssessmentFound,
		)
	}
	artifacts := []sourcedArtifact{
		{name: "topic", provenance: topic.Provenance, source: topic.Source},
		{name: "topic variant", provenance: variantTopic.Provenance, source: variantTopic.Source},
		{name: "assessment", provenance: assessment.Provenance, source: assessment.Source},
		{name: "assessment variant", provenance: variantAssessment.Provenance, source: variantAssessment.Source},
	}
	for _, artifact := range artifacts {
		if artifact.provenance != "" ||
			artifact.source.Path == "" ||
			!strings.HasPrefix(artifact.source.Revision, "sha256:") {
			t.Fatalf(
				"%s source = %#v, provenance = %q",
				artifact.name,
				artifact.source,
				artifact.provenance,
			)
		}
	}
}

func TestLoaderRejectsAssessmentWithoutPlanFields(t *testing.T) {
	tests := []struct {
		name        string
		old         string
		replacement string
	}{
		{name: "question text", old: "text: Evaluate 3x when x = 2.", replacement: `text: ""`},
		{name: "difficulty", old: "difficulty: easy", replacement: "difficulty: mystery"},
		{name: "marks", old: "marks: 1", replacement: "marks: 0"},
		{name: "answer type", old: "type: multiple_choice", replacement: "type: mystery"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := setupOSSContractFixture(t)
			assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
			replaceFixtureText(t, assessmentPath, test.old, test.replacement)

			if _, err := curriculum.NewLoader(root); err == nil {
				t.Fatalf("NewLoader() error = nil, want invalid assessment %s rejected", test.name)
			}
		})
	}
}

func TestLoaderAcceptsAssessmentWithoutOptionalMarks(t *testing.T) {
	root := setupOSSContractFixture(t)
	canonicalPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	variantPath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"translations",
		"ms",
		"MT1-05.assessments.yaml",
	)
	for _, path := range []string{canonicalPath, variantPath} {
		replaceFixtureText(t, path, "    marks: 1\n", "")
	}

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	assessment, found := loader.GetAssessment("MT1-05")
	if !found || len(assessment.Questions) != 1 || assessment.Questions[0].Marks != 0 {
		t.Fatalf("GetAssessment(MT1-05) = %#v, found = %v", assessment, found)
	}
}

func TestLoaderRejectsLocalizedAssessmentContractDrift(t *testing.T) {
	root := setupOSSContractFixture(t)
	variantPath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"translations",
		"ms",
		"MT1-05.assessments.yaml",
	)
	replaceFixtureText(t, variantPath, "type: multiple_choice", "type: exact")

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want localized grading-contract drift rejection")
	}
	if !strings.Contains(err.Error(), "assessment variant") || !strings.Contains(err.Error(), "Q1") {
		t.Fatalf("NewLoader() error = %v, want localized question contract", err)
	}
}

func TestLoaderRejectsTopicVariantLocalePathMismatch(t *testing.T) {
	root := setupOSSContractFixture(t)
	variantPath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"translations",
		"ms",
		"MT1-05.yaml",
	)
	replaceFixtureText(t, variantPath, "language: ms", "language: en")

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want topic variant locale mismatch rejected")
	}
	if !strings.Contains(err.Error(), "language") || !strings.Contains(err.Error(), "locale") {
		t.Fatalf("NewLoader() error = %v, want language/path locale mismatch", err)
	}
}

func TestLoaderRejectsDuplicateLocalizedObjectiveID(t *testing.T) {
	root := setupOSSContractFixture(t)
	variantPath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"translations",
		"ms",
		"MT1-05.yaml",
	)
	replaceFixtureText(t, variantPath, "quality_level: 2", `  - id: "5.1.1"
    content_standard_id: "5.1"
    text: Objektif pendua.
    bloom: understand
quality_level: 2`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want duplicate localized objective rejected")
	}
	if !strings.Contains(err.Error(), "repeats objective") {
		t.Fatalf("NewLoader() error = %v, want duplicate localized objective identity", err)
	}
}

func TestLoaderRejectsDuplicateLocalizedQuestionID(t *testing.T) {
	root := setupOSSContractFixture(t)
	variantPath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"translations",
		"ms",
		"MT1-05.assessments.yaml",
	)
	replaceFixtureText(t, variantPath, "    marks: 1\n", `    marks: 1
  - id: Q1
    text: Soalan pendua.
    difficulty: easy
    learning_objective: "5.1.1"
    answer:
      type: exact
      value: "6"
    marks: 1
`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want duplicate localized question rejected")
	}
	if !strings.Contains(err.Error(), "repeats question") {
		t.Fatalf("NewLoader() error = %v, want duplicate localized question identity", err)
	}
}

func TestLoaderPreservesLocalizedExactAnswer(t *testing.T) {
	root := setupOSSContractFixture(t)
	canonicalPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	variantPath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"translations",
		"ms",
		"MT1-05.assessments.yaml",
	)
	replaceFixtureText(t, canonicalPath, "type: multiple_choice", "type: exact")
	replaceFixtureText(t, variantPath, "type: multiple_choice", "type: exact")
	replaceFixtureText(t, variantPath, `value: "6"`, `value: "enam"`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	variant, found := loader.GetAssessmentVariant("MT1-05", "ms")
	if !found ||
		len(variant.Questions) != 1 ||
		variant.Questions[0].Answer.Value != "enam" {
		t.Fatalf("localized assessment = %#v, found = %v", variant, found)
	}
}

func TestLoaderRejectsPrerequisiteCycle(t *testing.T) {
	root := setupOSSContractFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `prerequisites:
  required:
    - MT1-05
quality_level: 2`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want prerequisite cycle rejection")
	}
	if !strings.Contains(err.Error(), "cycle") || !strings.Contains(err.Error(), "MT1-05") {
		t.Fatalf("NewLoader() error = %v, want prerequisite cycle", err)
	}
}

func TestLoaderRejectsExplicitZeroAssessmentCount(t *testing.T) {
	root := setupOSSContractFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `mastery:
  assessment_count: 0
quality_level: 2`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want explicit zero assessment count rejected")
	}
	if !strings.Contains(err.Error(), "mastery policy") {
		t.Fatalf("NewLoader() error = %v, want invalid mastery policy", err)
	}
}

func TestLoaderReturnsImmutableCurriculumSnapshots(t *testing.T) {
	root := setupOSSContractFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topic, _ := loader.GetTopic("MT1-05")
	topic.LearningObjectives[0].TextEN = "mutated"
	assessment, _ := loader.GetAssessment("MT1-05")
	assessment.Questions[0].Answer.Options["B"] = "mutated"
	subjectGrade, _ := loader.GetSubjectGrade("malaysia-kssm-matematik-tingkatan-1")
	subjectGrade.Topics[0] = "mutated"

	topic, _ = loader.GetTopic("MT1-05")
	if topic.LearningObjectives[0].TextEN == "mutated" {
		t.Fatal("GetTopic() exposed mutable loader state")
	}
	assessment, _ = loader.GetAssessment("MT1-05")
	if assessment.Questions[0].Answer.Options["B"] == "mutated" {
		t.Fatal("GetAssessment() exposed mutable loader state")
	}
	subjectGrade, _ = loader.GetSubjectGrade("malaysia-kssm-matematik-tingkatan-1")
	if subjectGrade.Topics[0] == "mutated" {
		t.Fatal("GetSubjectGrade() exposed mutable loader state")
	}
}

func TestLoaderNormalizesArrayAssessmentOptions(t *testing.T) {
	root := setupOSSContractFixture(t)
	assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	replaceFixtureText(t, assessmentPath, `options:
        A: "5"
        B: "6"`, `options:
        - "5"
        - "6"`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	assessment, found := loader.GetAssessment("MT1-05")
	if !found {
		t.Fatal("GetAssessment(MT1-05) not found")
	}
	options := assessment.Questions[0].Answer.Options
	if len(options) != 2 || options["A"] != "5" || options["B"] != "6" {
		t.Fatalf("normalized options = %#v, want A/B IDs in source order", options)
	}
}

func TestLoaderPreservesMatchingPairsAsNonScalarAnswer(t *testing.T) {
	root := setupOSSContractFixture(t)
	assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	replaceFixtureText(t, assessmentPath, `      type: multiple_choice
      value: "6"
      options:
        A: "5"
        B: "6"`, `      type: matching
      pairs:
        - left: Variable
          right: A symbol whose value may change
        - left: Constant
          right: A fixed value`)
	variantPath := filepath.Join(filepath.Dir(assessmentPath), "translations", "ms", "MT1-05.assessments.yaml")
	if err := os.Remove(variantPath); err != nil {
		t.Fatalf("Remove(%s) error = %v", variantPath, err)
	}

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	assessment, _ := loader.GetAssessment("MT1-05")
	answer := assessment.Questions[0].Answer
	if answer.Value != "" || len(answer.Pairs) != 2 ||
		answer.Pairs[0].Left != "Variable" ||
		answer.Pairs[1].Right != "A fixed value" {
		t.Fatalf("matching answer = %#v, want preserved ordered pairs", answer)
	}
}

func TestLoaderRejectsAmbiguousMultipleChoiceAnswer(t *testing.T) {
	root := setupOSSContractFixture(t)
	assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	replaceFixtureText(t, assessmentPath, `        A: "5"
        B: "6"`, `        A: "6"
        B: "6"`)

	_, err := curriculum.NewLoader(root)
	if err == nil {
		t.Fatal("NewLoader() error = nil, want ambiguous multiple-choice answer rejected")
	}
	if !strings.Contains(err.Error(), "Q1") || !strings.Contains(err.Error(), "correct option") {
		t.Fatalf("NewLoader() error = %v, want ambiguous question identity", err)
	}
}

func TestLoaderRejectsAssessmentShapesOutsideSchema(t *testing.T) {
	tests := []struct {
		name        string
		old         string
		replacement string
	}{
		{
			name: "structured answer value",
			old:  `      value: "6"`,
			replacement: `      value:
        x: "2"`,
		},
		{
			name: "scalar hint",
			old:  "    marks: 1\n",
			replacement: `    marks: 1
    hints:
      - Substitute the value first.
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := setupOSSContractFixture(t)
			assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
			replaceFixtureText(t, assessmentPath, test.old, test.replacement)

			if _, err := curriculum.NewLoader(root); err == nil {
				t.Fatal("NewLoader() error = nil, want schema-invalid assessment rejected")
			}
		})
	}
}

func setupOSSContractFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	curriculumDir := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
	)
	gradeDir := filepath.Join(curriculumDir, "malaysia-kssm-matematik-tingkatan-1")
	topicsDir := filepath.Join(gradeDir, "topics")
	translationDir := filepath.Join(topicsDir, "translations", "ms")
	if err := os.MkdirAll(translationDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	writeFixtureFile(t, filepath.Join(curriculumDir, "..", "syllabus.yaml"), `
id: malaysia-kssm
name: KSSM
subjects:
  - malaysia-kssm-matematik
`)
	writeFixtureFile(t, filepath.Join(curriculumDir, "subject.yaml"), `
id: malaysia-kssm-matematik
name: Matematik
name_en: Mathematics
syllabus_id: malaysia-kssm
`)
	writeFixtureFile(t, filepath.Join(gradeDir, "subject-grade.yaml"), `
id: malaysia-kssm-matematik-tingkatan-1
name: Matematik Tingkatan 1
name_en: Mathematics Form 1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
grade_id: tingkatan-1
topics:
  - MT1-05
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-05.yaml"), `
id: MT1-05
name: Ungkapan Algebra
name_en: Algebraic Expressions
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
language: en
difficulty: beginner
content_standards:
  - id: "5.1"
    text_en: Variables and algebraic expressions
learning_objectives:
  - id: "5.1.1"
    content_standard_id: "5.1"
    text_en: Use letters to represent unknown quantities.
    bloom: understand
quality_level: 2
provenance: ai-assisted
`)
	writeFixtureFile(t, filepath.Join(translationDir, "MT1-05.yaml"), `
id: MT1-05
name: Ungkapan Algebra
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
language: ms
difficulty: beginner
content_standards:
  - id: "5.1"
    text: Pemboleh ubah dan ungkapan algebra
learning_objectives:
  - id: "5.1.1"
    content_standard_id: "5.1"
    text: Menggunakan huruf untuk mewakili kuantiti yang tidak diketahui.
    bloom: understand
quality_level: 2
provenance: ai-assisted
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-05.assessments.yaml"), `
topic_id: MT1-05
provenance: human
questions:
  - id: Q1
    text: Evaluate 3x when x = 2.
    difficulty: easy
    learning_objective: "5.1.1"
    answer:
      type: multiple_choice
      value: "6"
      options:
        A: "5"
        B: "6"
    marks: 1
`)
	writeFixtureFile(t, filepath.Join(translationDir, "MT1-05.assessments.yaml"), `
topic_id: MT1-05
provenance: human
questions:
  - id: Q1
    text: Hitung 3x apabila x = 2.
    difficulty: easy
    learning_objective: "5.1.1"
    answer:
      type: multiple_choice
      value: "6"
      options:
        A: "5"
        B: "6"
    marks: 1
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-05.teaching.md"), "# Teach algebra\n")
	writeFixtureFile(t, filepath.Join(translationDir, "MT1-05.teaching.md"), "# Ajar algebra\n")

	return root
}

func setupValidatedCrossCurriculumFixture(t *testing.T) string {
	t.Helper()

	root := setupOSSContractFixture(t)
	otherDir := filepath.Join(root, "curricula", "other")
	topicsDir := filepath.Join(otherDir, "topics")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", topicsDir, err)
	}
	writeFixtureFile(t, filepath.Join(otherDir, "syllabus.yaml"), `
id: other-syllabus
name: Other
subjects:
  - other-math
`)
	writeFixtureFile(t, filepath.Join(otherDir, "subject.yaml"), `
id: other-math
name: Other Mathematics
syllabus_id: other-syllabus
`)
	writeFixtureFile(t, filepath.Join(otherDir, "subject-grade.yaml"), `
id: other-math-form-1
name: Other Mathematics Form 1
subject_id: other-math
syllabus_id: other-syllabus
grade_id: form-1
topics:
  - OM1-01
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "OM1-01.yaml"), `
id: OM1-01
name: Variables in Another Curriculum
subject_grade_id: other-math-form-1
subject_id: other-math
syllabus_id: other-syllabus
language: en
difficulty: beginner
content_standards:
  - id: "1.1"
    text_en: Variables
learning_objectives:
  - id: "1.1.1"
    content_standard_id: "1.1"
    text_en: Represent an unknown with a symbol.
    bloom: understand
quality_level: 2
provenance: human
`)
	conceptDir := filepath.Join(root, "concepts", "mathematics")
	if err := os.MkdirAll(conceptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", conceptDir, err)
	}
	writeFixtureFile(t, filepath.Join(conceptDir, "variables.yaml"), `
id: variables
name: Variables
domain: mathematics
subdomain: algebra
definition: A symbol whose value may change.
curricula:
  - syllabus: malaysia-kssm
    topic: MT1-05
    scope: introduction
  - syllabus: other-syllabus
    topic: OM1-01
    scope: introduction
`)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, "quality_level: 2", `cross_curriculum:
  - concept_id: variables
    syllabus_id: other-syllabus
    topic_id: OM1-01
quality_level: 2`)
	return root
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func fixtureTopicPath(root, name string) string {
	return filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
		"malaysia-kssm-matematik-tingkatan-1",
		"topics",
		name,
	)
}

func replaceFixtureText(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	content := string(data)
	if !strings.Contains(content, old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	writeFixtureFile(t, path, strings.Replace(content, old, replacement, 1))
}

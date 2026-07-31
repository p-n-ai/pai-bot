// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build ossintegration

package curriculum_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestLoaderLoadsPinnedOSSCorpus(t *testing.T) {
	root := filepath.Join("..", "..", "oss")
	if _, err := os.Stat(filepath.Join(root, "curricula")); err != nil {
		t.Skip("OSS submodule is not initialized")
	}

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader(pinned OSS) error = %v", err)
	}

	wantStats := curriculum.SnapshotStats{
		Syllabi:               1,
		Subjects:              3,
		SubjectGrades:         15,
		Topics:                152,
		TopicVariants:         53,
		Concepts:              2,
		Examples:              114,
		ExampleVariants:       53,
		TeachingNotes:         114,
		TeachingNoteVariants:  56,
		AITeachingReadyTopics: 37,
		Assessments:           114,
		AssessmentVariants:    53,
	}
	if got := loader.SnapshotStats(); got != wantStats {
		t.Fatalf("SnapshotStats() = %#v, want %#v", got, wantStats)
	}
	if _, found := loader.GetSubjectGrade("malaysia-kssm-matematik-tingkatan-1"); !found {
		t.Fatal("pinned OSS subject-grade not loaded")
	}

	trigonometry, found := loader.GetTopic("MT3-05")
	if !found {
		t.Fatal("pinned OSS canonical topic MT3-05 not loaded")
	}
	if trigonometry.Mastery.AssessmentCount != 3 {
		t.Fatalf(
			"MT3-05 mastery assessment count = %d, want authored 3",
			trigonometry.Mastery.AssessmentCount,
		)
	}
	if len(trigonometry.LearningObjectives) != 7 {
		t.Fatalf(
			"MT3-05 learning objectives = %d, want all 7 authored objectives",
			len(trigonometry.LearningObjectives),
		)
	}
	if len(trigonometry.Teaching.Sequence) != 5 {
		t.Fatalf(
			"MT3-05 teaching sequence = %d steps, want all 5 authored steps",
			len(trigonometry.Teaching.Sequence),
		)
	}
	if len(trigonometry.Teaching.CommonMisconceptions) != 3 ||
		len(trigonometry.Teaching.EngagementHooks) != 3 {
		t.Fatalf(
			"MT3-05 teaching guidance = %d misconceptions, %d hooks; want 3 and 3",
			len(trigonometry.Teaching.CommonMisconceptions),
			len(trigonometry.Teaching.EngagementHooks),
		)
	}

	taxAssessment, found := loader.GetAssessment("MT5-04")
	if !found {
		t.Fatal("pinned OSS canonical assessment MT5-04 not loaded")
	}
	var taxChoice curriculum.AssessmentQuestion
	for _, question := range taxAssessment.Questions {
		if question.ID == "Q2" {
			taxChoice = question
			break
		}
	}
	if len(taxChoice.Answer.Options) != 4 ||
		taxChoice.Answer.Value != "a" ||
		taxChoice.Answer.Options["a"] != "Property Assessment Tax" {
		t.Fatalf(
			"MT5-04/Q2 answer = %#v, want canonical four-option grading contract",
			taxChoice.Answer,
		)
	}

	canonical, found := loader.GetTopic("MT1-05")
	if !found {
		t.Fatal("pinned OSS canonical topic MT1-05 not loaded")
	}
	if canonical.Language != "en" || canonical.NameEN != "Algebraic Expressions" {
		t.Fatalf("canonical MT1-05 = %#v, want root English variant", canonical)
	}
	concepts := loader.ConceptsForTopic("malaysia-kssm", "MT1-05")
	if len(concepts) != 1 ||
		concepts[0].ID != "algebraic-expression" ||
		len(concepts[0].Curricula) != 5 {
		t.Fatalf("MT1-05 concepts = %#v, want source-authored algebraic-expression graph", concepts)
	}

	malay, found := loader.GetTopicVariant("MT1-05", "ms")
	if !found || malay.Language != "ms" {
		t.Fatalf("Malay MT1-05 variant = %#v, found = %v", malay, found)
	}
}

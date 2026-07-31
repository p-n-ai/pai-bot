// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestEnginePlanTurnRepairsUnmetPrerequisiteWithoutMutatingProgress(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	if err := tracker.UpdateMasteryForLearner(learnerID, "malaysia-kssm", "MT1-01", 0.2); err != nil {
		t.Fatalf("seed prerequisite mastery: %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{
		Loader:   loader,
		Progress: tracker,
	})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID:   learnerID,
		TopicID:     "MT1-05",
		ObjectiveID: "5.1.1",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}

	if plan.Action != curriculum.TeachingActionRepairPrerequisite {
		t.Fatalf("plan.Action = %q, want %q", plan.Action, curriculum.TeachingActionRepairPrerequisite)
	}
	if plan.Target.TopicID != "MT1-01" {
		t.Fatalf("plan.Target.TopicID = %q, want MT1-01", plan.Target.TopicID)
	}
	if plan.Target.TopicName != "Rational Numbers" {
		t.Fatalf("plan.Target.TopicName = %q, want English canonical name", plan.Target.TopicName)
	}
	if !plan.Target.MasteryKnown {
		t.Fatal("plan.Target.MasteryKnown = false, want seeded mastery evidence")
	}
	if plan.Check == nil || plan.Check.QuestionID != "Q1" {
		t.Fatalf("plan.Check = %#v, want source-backed MT1-01/Q1", plan.Check)
	}
	if len(plan.Evidence) == 0 || plan.Evidence[0].SourcePath == "" {
		t.Fatalf("plan.Evidence = %#v, want source provenance", plan.Evidence)
	}
	requires := requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationRequires,
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "MT1-05"},
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "MT1-01"},
	)
	requestedTopic, _ := loader.GetTopic("MT1-05")
	if requires.Evidence.SourceKind != "topic" ||
		requires.Evidence.Revision != requestedTopic.Source.Revision {
		t.Fatalf("requires relation evidence = %#v, want cited prerequisite edge artifact", requires.Evidence)
	}

	mastery, err := tracker.GetMasteryForLearner(learnerID, "malaysia-kssm", "MT1-01")
	if err != nil {
		t.Fatalf("GetMasteryForLearner() error = %v", err)
	}
	if mastery != 0.2 {
		t.Fatalf("mastery after PlanTurn = %v, want unchanged 0.2", mastery)
	}
}

func TestEnginePlanTurnDiagnosesPrerequisiteWhenMasteryIsUnknown(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID:   learnerID,
		TopicID:     "MT1-05",
		ObjectiveID: "5.1.1",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}

	if plan.Action != curriculum.TeachingActionDiagnosePrerequisite {
		t.Fatalf("plan.Action = %q, want %q", plan.Action, curriculum.TeachingActionDiagnosePrerequisite)
	}
	if plan.Target.TopicID != "MT1-01" || plan.Target.MasteryKnown {
		t.Fatalf("plan.Target = %#v, want unknown MT1-01 prerequisite", plan.Target)
	}
}

func TestEnginePlanTurnUsesOneCoherentMasterySnapshot(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	store := &fixedSnapshotStore{snapshot: []progress.MasterySnapshotItem{
		{
			SyllabusID:    "malaysia-kssm",
			TopicID:       "MT1-01",
			MasteryScore:  0.9,
			MasteryKnown:  true,
			EvidenceCount: 1,
		},
		{
			SyllabusID:    "malaysia-kssm",
			TopicID:       "MT1-05",
			MasteryScore:  0.4,
			MasteryKnown:  true,
			EvidenceCount: 2,
		},
	}}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: store})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID:   learnerID,
		TopicID:     "MT1-05",
		ObjectiveID: "5.1.1",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}

	if store.reads != 1 {
		t.Fatalf("mastery snapshot reads = %d, want exactly 1", store.reads)
	}
	if plan.Action != curriculum.TeachingActionTeachAndCheck {
		t.Fatalf("plan.Action = %q, want %q", plan.Action, curriculum.TeachingActionTeachAndCheck)
	}
	if plan.Target.MasteryScore != 0.4 ||
		!plan.Target.MasteryKnown ||
		plan.Target.EvidenceCount != 2 {
		t.Fatalf("plan.Target = %#v, want coherent target mastery 0.4 with 2 evidence records", plan.Target)
	}
}

func TestEnginePlanTurnRejectsLevelZeroStubWithoutProgressMutation(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	subjectGradePath := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
		"malaysia-kssm-matematik-tingkatan-1",
		"subject-grade.yaml",
	)
	replaceFixtureText(
		t,
		subjectGradePath,
		"topics:\n  - MT1-01\n  - MT1-05",
		"topics:\n  - MT1-00\n  - MT1-01\n  - MT1-05",
	)
	writeFixtureFile(t, filepath.Join(filepath.Dir(subjectGradePath), "topics", "MT1-00.yaml"), `
id: MT1-00
name: Catalog Stub
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
language: en
difficulty: beginner
content_standards: []
learning_objectives: []
quality_level: 0
provenance: human
`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	store := &fixedSnapshotStore{}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: store})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	_, err = engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   "MT1-00",
	})
	if err == nil {
		t.Fatal("PlanTurn() error = nil, want catalog stub rejected")
	}
	if !errors.Is(err, curriculum.ErrTopicNotAITeachingReady) {
		t.Fatalf("PlanTurn() error = %v, want ErrTopicNotAITeachingReady", err)
	}
	if store.writes != 0 {
		t.Fatalf("progress writes = %d, want 0", store.writes)
	}
}

func TestEnginePlanTurnRequiresExplicitAITeachingReadyQuality(t *testing.T) {
	tests := []struct {
		name         string
		qualityLine  string
		wantReady    bool
		wantPlanTurn bool
	}{
		{name: "missing", qualityLine: "", wantReady: false, wantPlanTurn: false},
		{name: "zero", qualityLine: "quality_level: 0", wantReady: false, wantPlanTurn: false},
		{name: "one", qualityLine: "quality_level: 1", wantReady: false, wantPlanTurn: false},
		{name: "two", qualityLine: "quality_level: 2", wantReady: false, wantPlanTurn: false},
		{name: "three", qualityLine: "quality_level: 3", wantReady: true, wantPlanTurn: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := setupCurriculumEngineFixture(t)
			topicPath := filepath.Join(
				root,
				"curricula",
				"malaysia",
				"malaysia-kssm",
				"malaysia-kssm-matematik",
				"malaysia-kssm-matematik-tingkatan-1",
				"topics",
				"MT1-05.yaml",
			)
			replaceFixtureText(t, topicPath, "quality_level: 3", test.qualityLine)

			loader, err := curriculum.NewLoader(root)
			if err != nil {
				t.Fatalf("NewLoader() error = %v", err)
			}
			topic, found := loader.GetTopic("MT1-05")
			if !found {
				t.Fatal("GetTopic(MT1-05) not found")
			}
			if got := topic.IsAITeachingReady(); got != test.wantReady {
				t.Fatalf("topic.IsAITeachingReady() = %v, want %v", got, test.wantReady)
			}
			wantReadyTopics := 1
			if test.wantReady {
				wantReadyTopics++
			}
			if got := loader.SnapshotStats().AITeachingReadyTopics; got != wantReadyTopics {
				t.Fatalf("SnapshotStats().AITeachingReadyTopics = %d, want %d", got, wantReadyTopics)
			}

			learnerID, err := progress.NewLearnerID("learner-1")
			if err != nil {
				t.Fatalf("NewLearnerID() error = %v", err)
			}
			store := &fixedSnapshotStore{snapshot: []progress.MasterySnapshotItem{{
				SyllabusID:    "malaysia-kssm",
				TopicID:       "MT1-01",
				MasteryScore:  1,
				MasteryKnown:  true,
				EvidenceCount: 1,
			}}}
			engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: store})
			if err != nil {
				t.Fatalf("NewEngine() error = %v", err)
			}

			_, err = engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
				LearnerID: learnerID,
				TopicID:   "MT1-05",
			})
			if test.wantPlanTurn {
				if err != nil {
					t.Fatalf("PlanTurn() error = %v, want accepted", err)
				}
				return
			}
			if !errors.Is(err, curriculum.ErrTopicNotAITeachingReady) {
				t.Fatalf("PlanTurn() error = %v, want ErrTopicNotAITeachingReady", err)
			}
			if store.reads != 0 || store.writes != 0 {
				t.Fatalf("progress calls = %d reads, %d writes; want none", store.reads, store.writes)
			}
		})
	}
}

func TestEngineRecordAttemptRejectsTopicBelowAITeachingQuality(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	topicPath := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
		"malaysia-kssm-matematik-tingkatan-1",
		"topics",
		"MT1-05.yaml",
	)
	replaceFixtureText(t, topicPath, "quality_level: 3", "quality_level: 2")
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	store := &fixedSnapshotStore{}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: store})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	_, err = engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "quality-bypass",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     "6",
	})
	if !errors.Is(err, curriculum.ErrTopicNotAITeachingReady) {
		t.Fatalf("RecordAttempt() error = %v, want ErrTopicNotAITeachingReady", err)
	}
	if store.reads != 0 || store.writes != 0 {
		t.Fatalf("progress calls = %d reads, %d writes; want none", store.reads, store.writes)
	}
}

func TestLoaderRequiresCanonicalTeachingArtifactsAtAITeachingQuality(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string
	}{
		{
			name:    "teaching note",
			file:    "MT1-05.teaching.md",
			wantErr: `AI-teaching-ready topic "MT1-05" requires canonical teaching notes`,
		},
		{
			name:    "examples",
			file:    "MT1-05.examples.yaml",
			wantErr: `AI-teaching-ready topic "MT1-05" requires canonical examples`,
		},
		{
			name:    "assessment",
			file:    "MT1-05.assessments.yaml",
			wantErr: `AI-teaching-ready topic "MT1-05" requires canonical assessment`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := setupCurriculumEngineFixture(t)
			path := filepath.Join(
				root,
				"curricula",
				"malaysia",
				"malaysia-kssm",
				"malaysia-kssm-matematik",
				"malaysia-kssm-matematik-tingkatan-1",
				"topics",
				test.file,
			)
			if err := os.Remove(path); err != nil {
				t.Fatalf("Remove(%s) error = %v", path, err)
			}

			_, err := curriculum.NewLoader(root)
			if err == nil {
				t.Fatal("NewLoader() error = nil, want missing AI teaching artifact rejected")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("NewLoader() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestEnginePlanTurnUsesLocalizedObjectiveAndSourceAssessment(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	recordFixtureEvidence(t, tracker, learnerID, "prerequisite-session-1", "MT1-01", "Q1", 0.8)
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{
		Loader:   loader,
		Progress: tracker,
	})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID:   learnerID,
		TopicID:     "MT1-05",
		ObjectiveID: "5.1.1",
		Locale:      "ms",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}

	if plan.Action != curriculum.TeachingActionTeachAndCheck {
		t.Fatalf("plan.Action = %q, want %q", plan.Action, curriculum.TeachingActionTeachAndCheck)
	}
	if plan.Target.TopicID != "MT1-05" {
		t.Fatalf("plan.Target.TopicID = %q, want MT1-05", plan.Target.TopicID)
	}
	if plan.Target.Objective != "Menggunakan huruf untuk mewakili kuantiti yang tidak diketahui." {
		t.Fatalf("plan.Target.Objective = %q, want Malay objective", plan.Target.Objective)
	}
	if plan.Target.TopicName != "Ungkapan Algebra" {
		t.Fatalf("plan.Target.TopicName = %q, want Malay localized name", plan.Target.TopicName)
	}
	if len(plan.TeachingSequence) != 1 || plan.TeachingSequence[0] != "Introduce a variable through a mystery-number example." {
		t.Fatalf("plan.TeachingSequence = %v, want canonical teaching policy", plan.TeachingSequence)
	}
	if plan.Check == nil || plan.Check.QuestionID != "Q1" || plan.Check.SourcePath == "" {
		t.Fatalf("plan.Check = %#v, want source-backed MT1-05/Q1", plan.Check)
	}
	if plan.Check.Question != "Hitung 3x apabila x = 2." {
		t.Fatalf("plan.Check.Question = %q, want Malay assessment text", plan.Check.Question)
	}
	if len(plan.Check.Options) != 2 || plan.Check.Options[1] != (curriculum.CheckOption{ID: "B", Text: "6"}) {
		t.Fatalf("plan.Check.Options = %#v, want ordered localized A/B options", plan.Check.Options)
	}

	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationContains,
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindSyllabus, ID: "malaysia-kssm"},
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindSubject, ID: "malaysia-kssm-matematik"},
	)
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationContains,
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindSubject, ID: "malaysia-kssm-matematik"},
		curriculum.CurriculumNodeRef{
			Kind: curriculum.CurriculumNodeKindSubjectGrade,
			ID:   "malaysia-kssm-matematik-tingkatan-1",
		},
	)
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationContains,
		curriculum.CurriculumNodeRef{
			Kind: curriculum.CurriculumNodeKindSubjectGrade,
			ID:   "malaysia-kssm-matematik-tingkatan-1",
		},
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "MT1-05"},
	)
	canonicalObjective := curriculum.CurriculumNodeRef{
		Kind:    curriculum.CurriculumNodeKindObjective,
		ID:      "5.1.1",
		TopicID: "MT1-05",
	}
	canonicalQuestion := curriculum.CurriculumNodeRef{
		Kind:    curriculum.CurriculumNodeKindQuestion,
		ID:      "Q1",
		TopicID: "MT1-05",
	}
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationContainsObjective,
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "MT1-05"},
		canonicalObjective,
	)
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationAssessedBy,
		canonicalObjective,
		canonicalQuestion,
	)
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationLocalizedBy,
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "MT1-05"},
		curriculum.CurriculumNodeRef{
			Kind:   curriculum.CurriculumNodeKindTopic,
			ID:     "MT1-05",
			Locale: "ms",
		},
	)
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationLocalizedBy,
		canonicalObjective,
		curriculum.CurriculumNodeRef{
			Kind:    curriculum.CurriculumNodeKindObjective,
			ID:      "5.1.1",
			TopicID: "MT1-05",
			Locale:  "ms",
		},
	)
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationLocalizedBy,
		canonicalQuestion,
		curriculum.CurriculumNodeRef{
			Kind:    curriculum.CurriculumNodeKindQuestion,
			ID:      "Q1",
			TopicID: "MT1-05",
			Locale:  "ms",
		},
	)
	for _, relation := range plan.Relations {
		if relation.Evidence.SourcePath == "" ||
			!strings.HasPrefix(relation.Evidence.Revision, "sha256:") {
			t.Fatalf("relation evidence = %#v, want exact artifact citation", relation)
		}
	}

	repeated, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID:   learnerID,
		TopicID:     "MT1-05",
		ObjectiveID: "5.1.1",
		Locale:      "ms",
	})
	if err != nil {
		t.Fatalf("repeated PlanTurn() error = %v", err)
	}
	if !reflect.DeepEqual(repeated.Relations, plan.Relations) {
		t.Fatalf("repeated relations = %#v, want deterministic %#v", repeated.Relations, plan.Relations)
	}
}

func TestEnginePlanTurnPreservesSourceCitedGuidanceAndMaterials(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	topicPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, topicPath, `prerequisites:
  required:
    - MT1-01
mastery:`, `prerequisites:
  required:
    - MT1-01
background_knowledge:
  - Substitution and multiplication
mastery:`)
	replaceFixtureText(t, topicPath, `teaching:
  sequence:
    - Introduce a variable through a mystery-number example.
quality_level: 3`, `teaching:
  sequence:
    - Introduce a variable through a mystery-number example.
  common_misconceptions:
    - misconception: The letter x always means multiplication.
      remediation: Use a multiplication dot before omitting it.
  engagement_hooks:
    - Use mystery bags to represent unknown values.
quality_level: 3`)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	recordFixtureEvidence(t, tracker, learnerID, "prerequisite-session-1", "MT1-01", "Q1", 1)
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID:   learnerID,
		TopicID:     "MT1-05",
		ObjectiveID: "5.1.1",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}
	if !reflect.DeepEqual(plan.Guidance.BackgroundKnowledge, []string{"Substitution and multiplication"}) ||
		len(plan.Guidance.CommonMisconceptions) != 1 ||
		plan.Guidance.CommonMisconceptions[0].Remediation != "Use a multiplication dot before omitting it." ||
		!reflect.DeepEqual(plan.Guidance.EngagementHooks, []string{"Use mystery bags to represent unknown values."}) {
		t.Fatalf("plan.Guidance = %#v, want complete topic guidance", plan.Guidance)
	}
	topic, _ := loader.GetTopic("MT1-05")
	if plan.Guidance.Evidence.SourcePath != topic.Source.Path ||
		plan.Guidance.Evidence.Revision != topic.Source.Revision {
		t.Fatalf("plan.Guidance.Evidence = %#v, want topic artifact citation", plan.Guidance.Evidence)
	}

	note, _ := loader.GetTeachingNote("MT1-05")
	if plan.Materials.TeachingNote == nil ||
		plan.Materials.TeachingNote.Content != "# Teach algebraic expressions\n" ||
		plan.Materials.TeachingNote.Evidence.Revision != note.Source.Revision {
		t.Fatalf("plan.Materials.TeachingNote = %#v, want cited note", plan.Materials.TeachingNote)
	}
	examples, _ := loader.GetExamples("MT1-05")
	if len(plan.Materials.WorkedExamples) != 1 ||
		plan.Materials.WorkedExamples[0].Example.Scenario != "Evaluate 3x when x = 2." ||
		plan.Materials.WorkedExamples[0].Evidence.Revision != examples.Source.Revision {
		t.Fatalf("plan.Materials.WorkedExamples = %#v, want cited typed example", plan.Materials.WorkedExamples)
	}
	constraints := strings.Join(plan.Constraints, "\n")
	for _, rawProse := range []string{
		"Substitution and multiplication",
		"The letter x always means multiplication.",
		"Use mystery bags to represent unknown values.",
		"# Teach algebraic expressions",
		"Evaluate 3x when x = 2.",
	} {
		if strings.Contains(constraints, rawProse) {
			t.Fatalf("plan.Constraints = %q, contains raw source prose %q", constraints, rawProse)
		}
	}

	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationGuidedBy,
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "MT1-05"},
		curriculum.CurriculumNodeRef{
			Kind:    curriculum.CurriculumNodeKindTeachingNote,
			ID:      "MT1-05",
			TopicID: "MT1-05",
		},
	)
	requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationIllustratedBy,
		curriculum.CurriculumNodeRef{
			Kind:    curriculum.CurriculumNodeKindObjective,
			ID:      "5.1.1",
			TopicID: "MT1-05",
		},
		curriculum.CurriculumNodeRef{
			Kind:    curriculum.CurriculumNodeKindWorkedExample,
			ID:      "WE-01",
			TopicID: "MT1-05",
		},
	)
}

func TestEnginePlanTurnUsesLocalizedTopicProvenance(t *testing.T) {
	root := setupOSSContractFixture(t)
	canonicalPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, canonicalPath, "quality_level: 2", "quality_level: 3")
	writeExamplesFixture(
		t,
		strings.TrimSuffix(canonicalPath, ".yaml")+".examples.yaml",
		"MT1-05",
		"WE-01",
		"5.1.1",
		"Evaluate 3x.",
	)
	variantPath := filepath.Join(
		filepath.Dir(canonicalPath),
		"translations",
		"ms",
		"MT1-05.yaml",
	)
	replaceFixtureText(t, variantPath, "provenance: ai-assisted", "provenance: human")
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   "MT1-05",
		Locale:    "ms",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}
	var localized curriculum.EvidenceRef
	for _, evidence := range plan.Evidence {
		if evidence.SourceKind == "topic_variant" {
			localized = evidence
			break
		}
	}
	if localized.Provenance != "human" {
		t.Fatalf("localized evidence = %#v, want variant provenance", localized)
	}
}

func TestEnginePlanTurnProjectsAuthoredConceptMappingsWithoutTopicDuplicates(t *testing.T) {
	root := setupValidatedCrossCurriculumFixture(t)
	replaceFixtureText(
		t,
		fixtureTopicPath(root, "MT1-05.yaml"),
		`cross_curriculum:
  - concept_id: variables
    syllabus_id: other-syllabus
    topic_id: OM1-01
quality_level: 2`,
		"quality_level: 2",
	)
	canonicalPath := fixtureTopicPath(root, "MT1-05.yaml")
	replaceFixtureText(t, canonicalPath, "quality_level: 2", "quality_level: 3")
	writeExamplesFixture(
		t,
		strings.TrimSuffix(canonicalPath, ".yaml")+".examples.yaml",
		"MT1-05",
		"WE-01",
		"5.1.1",
		"Evaluate 3x.",
	)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   "MT1-05",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}

	relation := requirePlanRelation(
		t,
		plan,
		curriculum.CurriculumRelationSharesConcept,
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "MT1-05"},
		curriculum.CurriculumNodeRef{Kind: curriculum.CurriculumNodeKindTopic, ID: "OM1-01"},
	)
	wantConcept := curriculum.CurriculumNodeRef{
		Kind: curriculum.CurriculumNodeKindConcept,
		ID:   "variables",
	}
	if relation.Via != wantConcept {
		t.Fatalf("shared-concept relation Via = %#v, want %#v", relation.Via, wantConcept)
	}
	concept, _ := loader.GetConcept("variables")
	if relation.Evidence.TopicID != "MT1-05" ||
		relation.Evidence.SourceKind != "concept" ||
		relation.Evidence.Revision != concept.Source.Revision {
		t.Fatalf("shared-concept evidence = %#v, want authored concept citation", relation.Evidence)
	}
}

func TestEnginePlanTurnDiagnosesPrerequisiteWithoutRequiredAssessmentSessions(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	if err := tracker.UpdateMasteryForLearner(learnerID, "malaysia-kssm", "MT1-01", 0.9); err != nil {
		t.Fatalf("seed legacy prerequisite mastery: %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   "MT1-05",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}

	if plan.Action != curriculum.TeachingActionDiagnosePrerequisite {
		t.Fatalf("plan.Action = %q, want %q", plan.Action, curriculum.TeachingActionDiagnosePrerequisite)
	}
	if plan.Target.EvidenceCount != 0 || plan.Target.EvidenceRequired != 1 {
		t.Fatalf("assessment sessions = %d/%d, want 0/1", plan.Target.EvidenceCount, plan.Target.EvidenceRequired)
	}
}

func TestEnginePlanTurnPreservesExplicitZeroMasteryThreshold(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	prerequisitePath := fixtureTopicPath(root, "MT1-01.yaml")
	replaceFixtureText(t, prerequisitePath, "minimum_score: 0.75", "minimum_score: 0")
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	recordFixtureEvidence(t, tracker, learnerID, "zero-threshold-session", "MT1-01", "Q1", 0)
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   "MT1-05",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}
	if plan.Action != curriculum.TeachingActionTeachAndCheck || plan.Target.TopicID != "MT1-05" {
		t.Fatalf("plan = %#v, explicit zero threshold must not become default threshold", plan)
	}
}

func TestEnginePlanTurnDiagnosesDeepestUnmetPrerequisite(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	subjectGradePath := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
		"malaysia-kssm-matematik-tingkatan-1",
		"subject-grade.yaml",
	)
	replaceFixtureText(t, subjectGradePath, "topics:\n  - MT1-01", "topics:\n  - MT1-00\n  - MT1-01")
	topicsDir := filepath.Dir(fixtureTopicPath(root, "MT1-01.yaml"))
	replaceFixtureText(t, fixtureTopicPath(root, "MT1-01.yaml"), "mastery:", `prerequisites:
  required:
    - MT1-00
mastery:`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-00.yaml"), `
id: MT1-00
name: Whole Numbers
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
language: en
difficulty: beginner
content_standards:
  - id: "0.1"
    text_en: Whole numbers
learning_objectives:
  - id: "0.1.1"
    content_standard_id: "0.1"
    text_en: Compare whole numbers.
    bloom: understand
mastery:
  minimum_score: 0.75
  assessment_count: 1
quality_level: 3
provenance: human
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-00.assessments.yaml"), `
topic_id: MT1-00
provenance: human
questions:
  - id: Q1
    text: Which is larger, 2 or 1?
    difficulty: easy
    learning_objective: "0.1.1"
    answer:
      type: exact
      value: "2"
    marks: 1
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-00.examples.yaml"), `
topic_id: MT1-00
provenance: human
worked_examples:
  - id: WE-01
    topic: Comparing whole numbers
    difficulty: easy
    learning_objective: "0.1.1"
    misconception_alert: More digits always means a larger number.
    scenario: Compare 2 and 1.
    working: Place both on a number line; 2 is farther right.
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-00.teaching.md"), "# Teach whole numbers\n")

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	recordFixtureEvidence(t, tracker, learnerID, "prerequisite-session-1", "MT1-01", "Q1", 1)
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   "MT1-05",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}
	if plan.Action != curriculum.TeachingActionDiagnosePrerequisite || plan.Target.TopicID != "MT1-00" {
		t.Fatalf("plan = %#v, want deepest unknown prerequisite MT1-00", plan)
	}
}

func TestEngineRecordAttemptGradesSourceQuestionAndUpdatesMastery(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{
		Loader:   loader,
		Progress: tracker,
	})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	result, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "attempt-correct-1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     " 6 ",
	})
	if err != nil {
		t.Fatalf("RecordAttempt() error = %v", err)
	}

	if !result.Correct || result.Score != 1 {
		t.Fatalf("result = %#v, want correct score 1", result)
	}
	if result.MasteryBefore != 0 || result.MasteryAfter != 1 {
		t.Fatalf("mastery transition = %v -> %v, want 0 -> 1", result.MasteryBefore, result.MasteryAfter)
	}
	assessment, _ := loader.GetAssessment("MT1-05")
	if result.Evidence.QuestionID != "Q1" ||
		result.Evidence.SourcePath == "" ||
		result.Evidence.Revision != assessment.Source.Revision {
		t.Fatalf("result.Evidence = %#v, want source-backed Q1 at exact artifact revision", result.Evidence)
	}
}

func TestEngineRecordAttemptComparesExactNumericAnswersByValue(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	assessmentPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	replaceFixtureText(t, assessmentPath, `value: "10"`, `value: "249.00"`)

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("numeric-answer-learner")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	equivalent, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "numeric-equivalent",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q4",
		Answer:     "249",
	})
	if err != nil {
		t.Fatalf("RecordAttempt(equivalent) error = %v", err)
	}
	if !equivalent.Correct {
		t.Fatalf("RecordAttempt(249) = %#v, want equivalent to 249.00", equivalent)
	}

	different, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "numeric-different",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q4",
		Answer:     "249.01",
	})
	if err != nil {
		t.Fatalf("RecordAttempt(different) error = %v", err)
	}
	if different.Correct {
		t.Fatalf("RecordAttempt(249.01) = %#v, want different from 249.00", different)
	}
}

func TestEngineRecordAttemptTreatsWrongAnswerAsNegativeEvidence(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	if err := tracker.UpdateMasteryForLearner(learnerID, "malaysia-kssm", "MT1-05", 0.8); err != nil {
		t.Fatalf("seed topic mastery: %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	result, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "attempt-wrong-1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     "7",
	})
	if err != nil {
		t.Fatalf("RecordAttempt() error = %v", err)
	}

	if result.Correct || result.Score != 0 {
		t.Fatalf("result = %#v, want incorrect score 0", result)
	}
	if result.MasteryAfter >= result.MasteryBefore {
		t.Fatalf("mastery transition = %v -> %v, wrong answer must not increase mastery", result.MasteryBefore, result.MasteryAfter)
	}
}

func TestEngineRecordAttemptRejectsFreeTextWithoutMutatingMastery(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	if err := tracker.UpdateMasteryForLearner(learnerID, "malaysia-kssm", "MT1-05", 0.8); err != nil {
		t.Fatalf("seed topic mastery: %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	_, err = engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "attempt-free-text-1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q2",
		Answer:     "because it changes",
	})
	if !errors.Is(err, curriculum.ErrAssessmentNotGradeable) {
		t.Fatalf("RecordAttempt() error = %v, want ErrAssessmentNotGradeable", err)
	}

	mastery, err := tracker.GetMasteryForLearner(learnerID, "malaysia-kssm", "MT1-05")
	if err != nil {
		t.Fatalf("GetMasteryForLearner() error = %v", err)
	}
	if mastery != 0.8 {
		t.Fatalf("mastery after rejected free text = %v, want unchanged 0.8", mastery)
	}
}

func TestEngineRecordAttemptReplayDoesNotApplyEvidenceTwice(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	if err := tracker.UpdateMasteryForLearner(learnerID, "malaysia-kssm", "MT1-05", 0.8); err != nil {
		t.Fatalf("seed topic mastery: %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	input := curriculum.AttemptInput{
		AttemptID:  "message-42/Q1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     "7",
	}

	first, err := engine.RecordAttempt(context.Background(), input)
	if err != nil {
		t.Fatalf("first RecordAttempt() error = %v", err)
	}
	replay, err := engine.RecordAttempt(context.Background(), input)
	if err != nil {
		t.Fatalf("replayed RecordAttempt() error = %v", err)
	}

	if !first.Applied || replay.Applied {
		t.Fatalf("applied flags = %v, %v, want true then false", first.Applied, replay.Applied)
	}
	if replay.MasteryBefore != first.MasteryBefore || replay.MasteryAfter != first.MasteryAfter {
		t.Fatalf("replay transition = %v -> %v, want original %v -> %v", replay.MasteryBefore, replay.MasteryAfter, first.MasteryBefore, first.MasteryAfter)
	}
	mastery, err := tracker.GetMasteryForLearner(learnerID, "malaysia-kssm", "MT1-05")
	if err != nil {
		t.Fatalf("GetMasteryForLearner() error = %v", err)
	}
	if mastery != first.MasteryAfter {
		t.Fatalf("mastery after replay = %v, want unchanged %v", mastery, first.MasteryAfter)
	}
}

func TestEngineRecordAttemptReplaySurvivesUnrelatedLoaderChange(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	firstLoader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("first NewLoader() error = %v", err)
	}
	firstAssessment, _ := firstLoader.GetAssessment("MT1-05")
	firstSnapshotRevision := firstLoader.SnapshotRevision()
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	firstEngine, err := curriculum.NewEngine(curriculum.EngineConfig{
		Loader:   firstLoader,
		Progress: tracker,
	})
	if err != nil {
		t.Fatalf("first NewEngine() error = %v", err)
	}
	input := curriculum.AttemptInput{
		AttemptID:  "stable-artifact-replay/Q1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     "6",
	}
	first, err := firstEngine.RecordAttempt(context.Background(), input)
	if err != nil || !first.Applied {
		t.Fatalf("first RecordAttempt() = %#v, %v; want applied", first, err)
	}

	unrelatedPath := fixtureTopicPath(root, "MT1-01.teaching.md")
	writeFixtureFile(t, unrelatedPath, "# Revised unrelated rational-number notes\n")
	secondLoader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("second NewLoader() error = %v", err)
	}
	secondAssessment, _ := secondLoader.GetAssessment("MT1-05")
	if secondLoader.SnapshotRevision() == firstSnapshotRevision {
		t.Fatal("SnapshotRevision() unchanged after unrelated curriculum artifact changed")
	}
	if secondAssessment.Source.Revision != firstAssessment.Source.Revision {
		t.Fatalf(
			"assessment revision = %q, want stable %q",
			secondAssessment.Source.Revision,
			firstAssessment.Source.Revision,
		)
	}
	secondEngine, err := curriculum.NewEngine(curriculum.EngineConfig{
		Loader:   secondLoader,
		Progress: tracker,
	})
	if err != nil {
		t.Fatalf("second NewEngine() error = %v", err)
	}

	replay, err := secondEngine.RecordAttempt(context.Background(), input)
	if err != nil {
		t.Fatalf("replayed RecordAttempt() error = %v", err)
	}
	if replay.Applied {
		t.Fatalf("replay = %#v, want Applied false", replay)
	}
}

func TestEngineRecordAttemptAcceptsLocalizedMultipleChoiceLabel(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	result, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "attempt-ms-choice-1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     "B",
		Locale:     "ms",
	})
	if err != nil {
		t.Fatalf("RecordAttempt() error = %v", err)
	}
	if !result.Correct {
		t.Fatalf("result = %#v, want localized correct option B accepted", result)
	}
}

func TestEngineRecordAttemptAcceptsLocalizedExactAnswer(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
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
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	result, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "attempt-ms-exact-1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     "enam",
		Locale:     "ms",
	})
	if err != nil {
		t.Fatalf("RecordAttempt() error = %v", err)
	}
	if !result.Correct {
		t.Fatalf("result = %#v, want localized exact answer accepted", result)
	}
}

func TestEngineGradesMultipleChoiceWithExactValueWhenOptionsAreNotAuthored(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	canonicalPath := fixtureTopicPath(root, "MT1-05.assessments.yaml")
	variantPath := filepath.Join(
		filepath.Dir(fixtureTopicPath(root, "MT1-05.yaml")),
		"translations",
		"ms",
		"MT1-05.assessments.yaml",
	)
	for _, path := range []string{canonicalPath, variantPath} {
		replaceFixtureText(
			t,
			path,
			`      options:
        A: "5"
        B: "6"
`,
			"",
		)
	}

	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	plan, err := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   "MT1-05",
	})
	if err != nil {
		t.Fatalf("PlanTurn() error = %v", err)
	}
	if plan.Check == nil || len(plan.Check.Options) != 0 {
		t.Fatalf("plan.Check = %#v, want exact-value check without structured options", plan.Check)
	}

	result, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "attempt-optionless-choice-1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q1",
		Answer:     "6",
	})
	if err != nil {
		t.Fatalf("RecordAttempt() error = %v", err)
	}
	if !result.Correct || result.Score != 1 {
		t.Fatalf("RecordAttempt() = %#v, want correct exact-value grade", result)
	}
}

func TestEngineRecordAttemptPreservesExactAnswerOperators(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatalf("NewLearnerID() error = %v", err)
	}
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	result, err := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
		AttemptID:  "attempt-equation-1",
		LearnerID:  learnerID,
		TopicID:    "MT1-05",
		QuestionID: "Q3",
		Answer:     "x2",
	})
	if err != nil {
		t.Fatalf("RecordAttempt() error = %v", err)
	}
	if result.Correct {
		t.Fatalf("result = %#v, x2 must not equal x=2", result)
	}
}

func TestEnginePlanTurnAdvancesAcrossGradeableObjectives(t *testing.T) {
	root := setupCurriculumEngineFixture(t)
	loader, err := curriculum.NewLoader(root)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	tracker := progress.NewMemoryTracker()
	learnerID, err := progress.NewLearnerID("learner-1")
	if err != nil {
		t.Fatal(err)
	}
	recordFixtureEvidence(t, tracker, learnerID, "prerequisite/Q1", "MT1-01", "Q1", 1)
	engine, err := curriculum.NewEngine(curriculum.EngineConfig{Loader: loader, Progress: tracker})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	assertPlan := func(wantObjective, wantQuestion string) {
		t.Helper()
		plan, planErr := engine.PlanTurn(context.Background(), curriculum.PlanTurnInput{
			LearnerID: learnerID,
			TopicID:   "MT1-05",
		})
		if planErr != nil {
			t.Fatalf("PlanTurn() error = %v", planErr)
		}
		if plan.Target.ObjectiveID != wantObjective || plan.Check == nil || plan.Check.QuestionID != wantQuestion {
			t.Fatalf("plan target/check = %#v / %#v, want %s/%s", plan.Target, plan.Check, wantObjective, wantQuestion)
		}
	}
	record := func(attemptID, questionID, answer string) {
		t.Helper()
		if _, recordErr := engine.RecordAttempt(context.Background(), curriculum.AttemptInput{
			AttemptID:  attemptID,
			LearnerID:  learnerID,
			TopicID:    "MT1-05",
			QuestionID: questionID,
			Answer:     answer,
		}); recordErr != nil {
			t.Fatalf("RecordAttempt(%s) error = %v", questionID, recordErr)
		}
	}

	assertPlan("5.1.1", "Q1")
	record("turn-1/Q1", "Q1", "6")
	assertPlan("5.1.1", "Q3")
	record("turn-2/Q3", "Q3", "x = 2")
	assertPlan("5.1.2", "Q4")
	record("turn-3/Q4", "Q4", "11")
	assertPlan("5.1.2", "Q4")
}

func setupCurriculumEngineFixture(t *testing.T) string {
	t.Helper()

	root := setupOSSContractFixture(t)
	subjectGradePath := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
		"malaysia-kssm-matematik-tingkatan-1",
		"subject-grade.yaml",
	)
	replaceFixtureText(t, subjectGradePath, "topics:\n  - MT1-05", "topics:\n  - MT1-01\n  - MT1-05")
	topicsDir := filepath.Join(
		root,
		"curricula",
		"malaysia",
		"malaysia-kssm",
		"malaysia-kssm-matematik",
		"malaysia-kssm-matematik-tingkatan-1",
		"topics",
	)

	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-01.yaml"), `
id: MT1-01
name: Nombor Nisbah
name_en: Rational Numbers
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
language: en
difficulty: beginner
content_standards:
  - id: "1.1"
    text_en: Integers
learning_objectives:
  - id: "1.1.1"
    content_standard_id: "1.1"
    text_en: Compare integers.
    bloom: apply
mastery:
  minimum_score: 0.75
  assessment_count: 1
quality_level: 3
provenance: human
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-01.assessments.yaml"), `
topic_id: MT1-01
provenance: human
questions:
  - id: Q1
    text: Which integer is larger, -2 or -5?
    difficulty: easy
    learning_objective: "1.1.1"
    answer:
      type: exact
      value: "-2"
      working: -2 is to the right of -5 on a number line.
    marks: 1
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-01.examples.yaml"), `
topic_id: MT1-01
provenance: human
worked_examples:
  - id: WE-01
    topic: Comparing integers
    difficulty: easy
    learning_objective: "1.1.1"
    misconception_alert: A larger magnitude is always the larger integer.
    scenario: Compare -2 and -5.
    working: Locate both integers on a number line; -2 is farther right.
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-01.teaching.md"), "# Teach rational numbers\n")

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
  - id: "5.1.0"
    content_standard_id: "5.1"
    text_en: Recognize variables in expressions.
    bloom: understand
  - id: "5.1.1"
    content_standard_id: "5.1"
    text_en: Use letters to represent unknown quantities.
    bloom: understand
  - id: "5.1.2"
    content_standard_id: "5.1"
    text_en: Evaluate expressions after substitution.
    bloom: apply
prerequisites:
  required:
    - MT1-01
mastery:
  minimum_score: 0.75
  assessment_count: 1
teaching:
  sequence:
    - Introduce a variable through a mystery-number example.
quality_level: 3
provenance: human
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
      working: Substitute x = 2, then multiply.
    marks: 1
  - id: Q2
    text: Explain whether x can change.
    difficulty: medium
    learning_objective: "5.1.1"
    answer:
      type: free_text
      value: It can vary.
      working: A variable may represent different values.
    marks: 1
  - id: Q3
    text: Write the equation saying x equals 2.
    difficulty: easy
    learning_objective: "5.1.1"
    answer:
      type: exact
      value: "x = 2"
    marks: 1
  - id: Q4
    text: Evaluate 2y when y = 5.
    difficulty: easy
    learning_objective: "5.1.2"
    answer:
      type: exact
      value: "10"
    marks: 1
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-05.examples.yaml"), `
topic_id: MT1-05
provenance: human
worked_examples:
  - id: WE-01
    topic: Evaluating algebraic expressions
    difficulty: easy
    learning_objective: "5.1.1"
    real_world_analogy: Three bags with two apples each.
    misconception_alert: The letter x always means multiplication.
    scenario: Evaluate 3x when x = 2.
    working: Substitute 2 for x, then calculate 3 times 2.
`)
	writeFixtureFile(t, filepath.Join(topicsDir, "MT1-05.teaching.md"), "# Teach algebraic expressions\n")

	return root
}

func recordFixtureEvidence(
	t *testing.T,
	tracker *progress.MemoryTracker,
	learnerID progress.LearnerID,
	attemptID, topicID, questionID string,
	score float64,
) {
	t.Helper()
	_, err := tracker.RecordMasteryEvidence(context.Background(), progress.MasteryEvidence{
		AttemptID:      attemptID,
		LearnerID:      learnerID,
		SyllabusID:     "malaysia-kssm",
		TopicID:        topicID,
		SourceKind:     "assessment",
		SourceID:       questionID,
		SourceRevision: "sha256:fixture",
		Score:          score,
		PayloadHash:    sha256.Sum256([]byte(attemptID)),
	})
	if err != nil {
		t.Fatalf("RecordMasteryEvidence() error = %v", err)
	}
}

func requirePlanRelation(
	t *testing.T,
	plan curriculum.TeachingPlan,
	relationType curriculum.CurriculumRelationType,
	from curriculum.CurriculumNodeRef,
	to curriculum.CurriculumNodeRef,
) curriculum.CurriculumRelation {
	t.Helper()
	for _, relation := range plan.Relations {
		if relation.Type == relationType && relation.From == from && relation.To == to {
			return relation
		}
	}
	t.Fatalf(
		"plan.Relations = %#v, want %q relation from %#v to %#v",
		plan.Relations,
		relationType,
		from,
		to,
	)
	return curriculum.CurriculumRelation{}
}

type fixedSnapshotStore struct {
	snapshot []progress.MasterySnapshotItem
	reads    int
	writes   int
}

func (s *fixedSnapshotStore) GetMasterySnapshot(
	ctx context.Context,
	_ progress.LearnerID,
) ([]progress.MasterySnapshotItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.reads++
	return append([]progress.MasterySnapshotItem(nil), s.snapshot...), nil
}

func (s *fixedSnapshotStore) RecordMasteryEvidence(
	context.Context,
	progress.MasteryEvidence,
) (progress.MasteryEvidenceResult, error) {
	s.writes++
	return progress.MasteryEvidenceResult{}, errors.New("unexpected mastery evidence write")
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

// TeachingAction is a deterministic pedagogical decision, not learner-facing copy.
type TeachingAction string

const (
	// TeachingActionTeachAndCheck teaches the selected objective and offers a
	// source-backed deterministic check when one is available.
	TeachingActionTeachAndCheck TeachingAction = "teach_and_check"
	// TeachingActionRepairPrerequisite redirects the turn to a required topic
	// whose demonstrated mastery is below its OSS-authored threshold.
	TeachingActionRepairPrerequisite TeachingAction = "repair_prerequisite"
	// TeachingActionDiagnosePrerequisite checks a required topic before teaching
	// when the learner has no recorded evidence for it.
	TeachingActionDiagnosePrerequisite TeachingAction = "diagnose_prerequisite"
)

// ProgressStore is the curriculum engine's single mastery read/write boundary.
type ProgressStore interface {
	GetMasterySnapshot(context.Context, progress.LearnerID) ([]progress.MasterySnapshotItem, error)
	RecordMasteryEvidence(context.Context, progress.MasteryEvidence) (progress.MasteryEvidenceResult, error)
}

// EngineConfig contains the curriculum engine's authoritative dependencies.
type EngineConfig struct {
	Loader   *Loader
	Progress ProgressStore
}

// Engine plans teaching turns from OSS and records gradeable learner evidence.
type Engine struct {
	loader   *Loader
	progress ProgressStore
}

// ErrAssessmentNotGradeable means OSS does not provide a deterministic answer
// contract for the requested question.
var ErrAssessmentNotGradeable = errors.New("assessment question is not deterministically gradeable")

// ErrTopicNotAITeachingReady means OSS does not explicitly claim that a topic
// meets the minimum quality contract for AI teaching.
var ErrTopicNotAITeachingReady = errors.New("topic is not ready for AI teaching")

// NewEngine constructs a curriculum engine over one validated OSS snapshot.
func NewEngine(config EngineConfig) (*Engine, error) {
	if config.Loader == nil {
		return nil, fmt.Errorf("curriculum loader is required")
	}
	if config.Progress == nil {
		return nil, fmt.Errorf("curriculum progress store is required")
	}
	return &Engine{
		loader:   config.Loader,
		progress: config.Progress,
	}, nil
}

// PlanTurnInput identifies the learner and curriculum target for one read-only
// planning decision. Topic resolution from free-form chat belongs to the harness.
type PlanTurnInput struct {
	LearnerID   progress.LearnerID
	TopicID     string
	ObjectiveID string
	Locale      string
}

// TeachingTarget is the exact topic and objective selected for this turn.
type TeachingTarget struct {
	TopicID          string
	TopicName        string
	SyllabusID       string
	ObjectiveID      string
	Objective        string
	MasteryScore     float64
	MasteryKnown     bool
	EvidenceCount    int
	EvidenceRequired int
}

// EvidenceRef traces one planning fact to an OSS artifact.
type EvidenceRef struct {
	SourcePath  string
	SourceKind  string
	Revision    string
	TopicID     string
	ObjectiveID string
	QuestionID  string
	Provenance  string
}

// CurriculumNodeKind identifies one validated curriculum artifact or a
// topic-scoped teaching identity.
type CurriculumNodeKind string

const (
	CurriculumNodeKindSyllabus      CurriculumNodeKind = "syllabus"
	CurriculumNodeKindSubject       CurriculumNodeKind = "subject"
	CurriculumNodeKindSubjectGrade  CurriculumNodeKind = "subject_grade"
	CurriculumNodeKindTopic         CurriculumNodeKind = "topic"
	CurriculumNodeKindObjective     CurriculumNodeKind = "objective"
	CurriculumNodeKindQuestion      CurriculumNodeKind = "question"
	CurriculumNodeKindConcept       CurriculumNodeKind = "concept"
	CurriculumNodeKindTeachingNote  CurriculumNodeKind = "teaching_note"
	CurriculumNodeKindWorkedExample CurriculumNodeKind = "worked_example"
)

// CurriculumRelationType identifies a source-backed edge projected from the
// validated curriculum snapshot.
type CurriculumRelationType string

const (
	CurriculumRelationContains          CurriculumRelationType = "contains"
	CurriculumRelationContainsObjective CurriculumRelationType = "contains_objective"
	CurriculumRelationRequires          CurriculumRelationType = "requires"
	CurriculumRelationAssessedBy        CurriculumRelationType = "assessed_by"
	CurriculumRelationLocalizedBy       CurriculumRelationType = "localized_by"
	CurriculumRelationSharesConcept     CurriculumRelationType = "shares_concept"
	CurriculumRelationGuidedBy          CurriculumRelationType = "guided_by"
	CurriculumRelationIllustratedBy     CurriculumRelationType = "illustrated_by"
)

// CurriculumNodeRef identifies one endpoint of a plan relation. TopicID scopes
// objective and question IDs, which are only unique within a topic.
type CurriculumNodeRef struct {
	Kind    CurriculumNodeKind
	ID      string
	TopicID string
	Locale  string
}

// CurriculumRelation is a deterministic, cited projection of the validated
// OSS snapshot. It is evidence for a plan, not a mutable graph store.
type CurriculumRelation struct {
	Type     CurriculumRelationType
	From     CurriculumNodeRef
	Via      CurriculumNodeRef
	To       CurriculumNodeRef
	Evidence EvidenceRef
}

// PlannedCheck is a source-backed question that RecordAttempt may grade.
type PlannedCheck struct {
	TopicID     string
	ObjectiveID string
	QuestionID  string
	Question    string
	Options     []CheckOption
	AnswerType  string
	Difficulty  string
	SourcePath  string
}

// CheckOption is one ordered learner-visible multiple-choice option.
type CheckOption struct {
	ID   string
	Text string
}

// SourcedTeachingGuidance carries source-authored guidance that is useful to a
// tutor but is not trusted policy.
type SourcedTeachingGuidance struct {
	BackgroundKnowledge  []string
	CommonMisconceptions []CommonMisconception
	EngagementHooks      []string
	Evidence             EvidenceRef
}

// TeachingNoteMaterial is one source-cited Markdown note.
type TeachingNoteMaterial struct {
	Content  string
	Evidence EvidenceRef
}

// WorkedExampleMaterial is one typed example cited to its backing artifact.
type WorkedExampleMaterial struct {
	Example  WorkedExample
	Evidence EvidenceRef
}

// TeachingMaterials contains source prose available to the tutor.
type TeachingMaterials struct {
	TeachingNote   *TeachingNoteMaterial
	WorkedExamples []WorkedExampleMaterial
}

// TeachingPlan is structured policy and evidence for a tutor. It deliberately
// contains no final learner-facing response.
type TeachingPlan struct {
	Action           TeachingAction
	Target           TeachingTarget
	TeachingSequence []string
	Guidance         SourcedTeachingGuidance
	Materials        TeachingMaterials
	Check            *PlannedCheck
	Evidence         []EvidenceRef
	Relations        []CurriculumRelation
	Constraints      []string
	Rationale        []string
}

// AttemptInput contains learner-authored evidence for one exact OSS question.
// It intentionally has no tutor-response field.
type AttemptInput struct {
	AttemptID  string
	LearnerID  progress.LearnerID
	TopicID    string
	QuestionID string
	Answer     string
	Locale     string
}

// AttemptResult is the deterministic grade and resulting mastery transition.
type AttemptResult struct {
	Applied       bool
	Correct       bool
	Score         float64
	MasteryBefore float64
	MasteryAfter  float64
	Evidence      EvidenceRef
}

// PlanTurn returns a deterministic, read-only teaching plan.
func (e *Engine) PlanTurn(ctx context.Context, input PlanTurnInput) (TeachingPlan, error) {
	if err := ctx.Err(); err != nil {
		return TeachingPlan{}, err
	}
	topicID := strings.TrimSpace(input.TopicID)
	if input.LearnerID.String() == "" {
		return TeachingPlan{}, fmt.Errorf("learner ID is required")
	}
	if topicID == "" {
		return TeachingPlan{}, fmt.Errorf("topic ID is required")
	}
	topic, found := e.loader.GetTopic(topicID)
	if !found {
		return TeachingPlan{}, fmt.Errorf("topic %q not found", topicID)
	}
	if err := requireAITeachingReady(topic); err != nil {
		return TeachingPlan{}, err
	}
	mastery, err := e.masteryObservations(ctx, input.LearnerID)
	if err != nil {
		return TeachingPlan{}, fmt.Errorf("read learner mastery: %w", err)
	}

	block, err := e.firstPrerequisiteBlock(topic, mastery)
	if err != nil {
		return TeachingPlan{}, err
	}
	if block != nil {
		plan, err := e.planForTopic(block.topic, "", input.Locale, block.observation)
		if err != nil {
			return TeachingPlan{}, err
		}
		plan.Action = block.action
		plan.Relations = append(e.prerequisitePathRelations(block.path), plan.Relations...)
		plan.Rationale = append(plan.Rationale, block.rationale)
		return plan, nil
	}

	observation := mastery[masteryKey(topic.SyllabusID, topic.ID)]
	return e.planForTopic(topic, input.ObjectiveID, input.Locale, observation)
}

type prerequisiteBlock struct {
	topic       Topic
	observation masteryObservation
	action      TeachingAction
	rationale   string
	path        []prerequisiteStep
}

type prerequisiteStep struct {
	dependent    Topic
	prerequisite Topic
}

func (e *Engine) firstPrerequisiteBlock(
	topic Topic,
	mastery map[string]masteryObservation,
) (*prerequisiteBlock, error) {
	for _, prerequisiteRef := range topic.Prerequisites.Required {
		prerequisiteID := prerequisiteRef.TopicID
		prerequisite, found := e.loader.GetTopic(prerequisiteID)
		if !found {
			return nil, fmt.Errorf("topic %q requires missing prerequisite %q", topic.ID, prerequisiteID)
		}
		nested, err := e.firstPrerequisiteBlock(prerequisite, mastery)
		if err != nil {
			return nil, err
		}
		step := prerequisiteStep{dependent: topic, prerequisite: prerequisite}
		if nested != nil {
			nested.path = append([]prerequisiteStep{step}, nested.path...)
			return nested, nil
		}

		observation := mastery[masteryKey(prerequisite.SyllabusID, prerequisite.ID)]
		switch {
		case !observation.known:
			return &prerequisiteBlock{
				topic:       prerequisite,
				observation: observation,
				action:      TeachingActionDiagnosePrerequisite,
				rationale:   fmt.Sprintf("required prerequisite %s has no recorded learner evidence", prerequisite.ID),
				path:        []prerequisiteStep{step},
			}, nil
		case observation.score < masteryThreshold(prerequisite):
			return &prerequisiteBlock{
				topic:       prerequisite,
				observation: observation,
				action:      TeachingActionRepairPrerequisite,
				rationale:   fmt.Sprintf("required prerequisite %s is below its mastery threshold", prerequisite.ID),
				path:        []prerequisiteStep{step},
			}, nil
		case observation.evidenceCount < minimumAssessmentCount(prerequisite):
			return &prerequisiteBlock{
				topic:       prerequisite,
				observation: observation,
				action:      TeachingActionDiagnosePrerequisite,
				rationale: fmt.Sprintf(
					"required prerequisite %s has %d of %d required assessment sessions",
					prerequisite.ID,
					observation.evidenceCount,
					minimumAssessmentCount(prerequisite),
				),
				path: []prerequisiteStep{step},
			}, nil
		}
	}
	return nil, nil
}

// RecordAttempt grades one source-backed deterministic question and records the
// resulting learner evidence. Ordinary conversation cannot enter this path.
func (e *Engine) RecordAttempt(ctx context.Context, input AttemptInput) (AttemptResult, error) {
	if err := ctx.Err(); err != nil {
		return AttemptResult{}, err
	}
	attemptID := strings.TrimSpace(input.AttemptID)
	topicID := strings.TrimSpace(input.TopicID)
	questionID := strings.TrimSpace(input.QuestionID)
	if input.LearnerID.String() == "" {
		return AttemptResult{}, fmt.Errorf("learner ID is required")
	}
	if attemptID == "" || topicID == "" || questionID == "" {
		return AttemptResult{}, fmt.Errorf("attempt ID, topic ID, and question ID are required")
	}
	topic, found := e.loader.GetTopic(topicID)
	if !found {
		return AttemptResult{}, fmt.Errorf("topic %q not found", topicID)
	}
	if err := requireAITeachingReady(topic); err != nil {
		return AttemptResult{}, err
	}
	assessment, found := e.loader.GetAssessment(topicID)
	if !found {
		return AttemptResult{}, fmt.Errorf("assessment for topic %q not found", topicID)
	}
	question, found := assessmentQuestion(assessment, questionID)
	if !found {
		return AttemptResult{}, fmt.Errorf("assessment question %q not found for topic %q", questionID, topicID)
	}
	if !isDeterministicallyGradeable(question.Answer.Type) {
		return AttemptResult{}, fmt.Errorf("%w: %s/%s uses %q", ErrAssessmentNotGradeable, topicID, questionID, question.Answer.Type)
	}

	presentationQuestion, _, _ := localizedAssessmentQuestion(e.loader, topic.ID, input.Locale, question)
	correct := gradeDeterministicAnswer(question, presentationQuestion, input.Answer)
	score := 0.0
	if correct {
		score = 1
	}
	transition, err := e.progress.RecordMasteryEvidence(ctx, progress.MasteryEvidence{
		AttemptID:      attemptID,
		LearnerID:      input.LearnerID,
		SyllabusID:     topic.SyllabusID,
		TopicID:        topic.ID,
		SourceKind:     assessment.Source.Kind,
		SourceID:       question.ID,
		SourceRevision: assessment.Source.Revision,
		Score:          score,
		PayloadHash:    attemptPayloadHash(topic, question, input.Answer),
	})
	if err != nil {
		return AttemptResult{}, fmt.Errorf("record mastery evidence: %w", err)
	}

	return AttemptResult{
		Applied:       transition.Applied,
		Correct:       correct,
		Score:         score,
		MasteryBefore: transition.MasteryBefore,
		MasteryAfter:  transition.MasteryAfter,
		Evidence: EvidenceRef{
			SourcePath:  assessment.Source.Path,
			SourceKind:  assessment.Source.Kind,
			Revision:    assessment.Source.Revision,
			TopicID:     topic.ID,
			ObjectiveID: question.LearningObjective,
			QuestionID:  question.ID,
			Provenance:  assessment.Provenance,
		},
	}, nil
}

func attemptPayloadHash(topic Topic, question AssessmentQuestion, answer string) [32]byte {
	payload := strings.Join([]string{
		"curriculum-attempt/v1",
		topic.SyllabusID,
		topic.ID,
		question.ID,
		question.Answer.Type,
		normalizeDeterministicAnswer(answer),
	}, "\x00")
	return sha256.Sum256([]byte(payload))
}

type masteryObservation struct {
	score         float64
	known         bool
	evidenceCount int
}

func (e *Engine) masteryObservations(ctx context.Context, learnerID progress.LearnerID) (map[string]masteryObservation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	snapshot, err := e.progress.GetMasterySnapshot(ctx, learnerID)
	if err != nil {
		return nil, err
	}
	observations := make(map[string]masteryObservation, len(snapshot))
	for _, item := range snapshot {
		observations[masteryKey(item.SyllabusID, item.TopicID)] = masteryObservation{
			score:         item.MasteryScore,
			known:         item.MasteryKnown,
			evidenceCount: item.EvidenceCount,
		}
	}
	return observations, nil
}

func masteryKey(syllabusID, topicID string) string {
	return syllabusID + "\x00" + topicID
}

func (e *Engine) planForTopic(topic Topic, objectiveID, locale string, mastery masteryObservation) (TeachingPlan, error) {
	if err := requireAITeachingReady(topic); err != nil {
		return TeachingPlan{}, err
	}
	selected := localizedTopic(e.loader, topic, locale)
	objective, err := selectObjective(selected, objectiveID)
	if err != nil {
		return TeachingPlan{}, err
	}

	plan := TeachingPlan{
		Action: TeachingActionTeachAndCheck,
		Target: TeachingTarget{
			TopicID:          selected.ID,
			TopicName:        localizedTopicName(selected),
			SyllabusID:       selected.SyllabusID,
			ObjectiveID:      objective.ID,
			Objective:        localizedObjectiveText(selected.Language, objective),
			MasteryScore:     mastery.score,
			MasteryKnown:     mastery.known,
			EvidenceCount:    mastery.evidenceCount,
			EvidenceRequired: minimumAssessmentCount(topic),
		},
		TeachingSequence: append([]string(nil), selected.Teaching.Sequence...),
		Guidance: SourcedTeachingGuidance{
			BackgroundKnowledge:  append([]string(nil), topic.BackgroundKnowledge...),
			CommonMisconceptions: append([]CommonMisconception(nil), topic.Teaching.CommonMisconceptions...),
			EngagementHooks:      append([]string(nil), topic.Teaching.EngagementHooks...),
			Evidence:             sourceEvidence(topic.Source, topic.ID, "", "", topic.Provenance),
		},
		Evidence: []EvidenceRef{{
			SourcePath:  topic.Source.Path,
			SourceKind:  topic.Source.Kind,
			Revision:    topic.Source.Revision,
			TopicID:     topic.ID,
			ObjectiveID: objective.ID,
			Provenance:  topic.Provenance,
		}},
		Constraints: []string{
			"do not update mastery from tutor prose or ordinary conversation",
			"update mastery only through RecordAttempt with a source-backed deterministic question",
		},
		Rationale: []string{
			"target selected from the validated OSS topic and objective graph",
		},
	}
	plan.Relations, err = e.topicRelations(topic, selected, objective)
	if err != nil {
		return TeachingPlan{}, err
	}
	if selected.Source.Path != topic.Source.Path {
		plan.Evidence = append(plan.Evidence, EvidenceRef{
			SourcePath:  selected.Source.Path,
			SourceKind:  selected.Source.Kind,
			Revision:    selected.Source.Revision,
			TopicID:     selected.ID,
			ObjectiveID: objective.ID,
			Provenance:  selected.Provenance,
		})
	}
	e.addTeachingMaterials(&plan, topic, locale)

	assessment, found := e.loader.GetAssessment(topic.ID)
	if !found {
		plan.Constraints = append(plan.Constraints, "no source assessment is available for this topic")
		return plan, nil
	}
	question, found := selectGradeableQuestion(assessment, objective.ID)
	if !found {
		plan.Constraints = append(plan.Constraints, "no deterministic source assessment is available for this objective")
		return plan, nil
	}
	presentationQuestion, presentationSource, localized := localizedAssessmentQuestion(
		e.loader,
		topic.ID,
		locale,
		question,
	)
	plan.Check = &PlannedCheck{
		TopicID:     topic.ID,
		ObjectiveID: question.LearningObjective,
		QuestionID:  question.ID,
		Question:    presentationQuestion.Text,
		Options:     orderedCheckOptions(presentationQuestion.Answer.Options),
		AnswerType:  question.Answer.Type,
		Difficulty:  question.Difficulty,
		SourcePath:  presentationSource.Path,
	}
	plan.Evidence = append(plan.Evidence, EvidenceRef{
		SourcePath:  assessment.Source.Path,
		SourceKind:  assessment.Source.Kind,
		Revision:    assessment.Source.Revision,
		TopicID:     topic.ID,
		ObjectiveID: question.LearningObjective,
		QuestionID:  question.ID,
		Provenance:  assessment.Provenance,
	})
	plan.Relations = append(plan.Relations, newCurriculumRelation(
		CurriculumRelationAssessedBy,
		objectiveNode(topic.ID, question.LearningObjective, ""),
		questionNode(topic.ID, question.ID, ""),
		sourceEvidence(
			assessment.Source,
			topic.ID,
			question.LearningObjective,
			question.ID,
			assessment.Provenance,
		),
	))
	if localized {
		variant, _ := e.loader.GetAssessmentVariant(topic.ID, strings.TrimSpace(locale))
		plan.Evidence = append(plan.Evidence, EvidenceRef{
			SourcePath:  presentationSource.Path,
			SourceKind:  presentationSource.Kind,
			Revision:    presentationSource.Revision,
			TopicID:     topic.ID,
			ObjectiveID: presentationQuestion.LearningObjective,
			QuestionID:  presentationQuestion.ID,
			Provenance:  variant.Provenance,
		})
		plan.Relations = append(plan.Relations, newCurriculumRelation(
			CurriculumRelationLocalizedBy,
			questionNode(topic.ID, question.ID, ""),
			questionNode(topic.ID, presentationQuestion.ID, strings.TrimSpace(locale)),
			sourceEvidence(
				presentationSource,
				topic.ID,
				presentationQuestion.LearningObjective,
				presentationQuestion.ID,
				variant.Provenance,
			),
		))
	}
	return plan, nil
}

func (e *Engine) addTeachingMaterials(plan *TeachingPlan, topic Topic, locale string) {
	note, found := e.loader.GetTeachingNote(topic.ID)
	locale = strings.TrimSpace(locale)
	if locale != "" {
		if variant, variantFound := e.loader.GetTeachingNoteVariant(topic.ID, locale); variantFound {
			note = variant
			found = true
		}
	}
	if found {
		evidence := sourceEvidence(note.Source, topic.ID, "", "", "")
		plan.Materials.TeachingNote = &TeachingNoteMaterial{
			Content:  note.Content,
			Evidence: evidence,
		}
		plan.Evidence = append(plan.Evidence, evidence)
		plan.Relations = append(plan.Relations, newCurriculumRelation(
			CurriculumRelationGuidedBy,
			curriculumNode(CurriculumNodeKindTopic, topic.ID),
			teachingNoteNode(topic.ID, note.Source.Locale),
			evidence,
		))
	}

	examples, found := e.loader.GetExamples(topic.ID)
	if locale != "" {
		if variant, variantFound := e.loader.GetExamplesVariant(topic.ID, locale); variantFound {
			examples = variant
			found = true
		}
	}
	if !found {
		return
	}
	artifactEvidence := sourceEvidence(
		examples.Source,
		topic.ID,
		"",
		"",
		examples.Provenance,
	)
	plan.Evidence = append(plan.Evidence, artifactEvidence)
	for _, example := range examples.WorkedExamples {
		evidence := sourceEvidence(
			examples.Source,
			topic.ID,
			example.LearningObjective,
			"",
			examples.Provenance,
		)
		plan.Materials.WorkedExamples = append(
			plan.Materials.WorkedExamples,
			WorkedExampleMaterial{Example: example, Evidence: evidence},
		)
		from := curriculumNode(CurriculumNodeKindTopic, topic.ID)
		if example.LearningObjective != "" {
			from = objectiveNode(topic.ID, example.LearningObjective, "")
		}
		plan.Relations = append(plan.Relations, newCurriculumRelation(
			CurriculumRelationIllustratedBy,
			from,
			workedExampleNode(topic.ID, example.ID, examples.Source.Locale),
			evidence,
		))
	}
}

func requireAITeachingReady(topic Topic) error {
	if topic.IsAITeachingReady() {
		return nil
	}
	return fmt.Errorf(
		"%w: topic %q has no explicit source quality level of 3 or higher",
		ErrTopicNotAITeachingReady,
		topic.ID,
	)
}

func (e *Engine) topicRelations(
	topic Topic,
	selected Topic,
	objective LearningObjective,
) ([]CurriculumRelation, error) {
	subjectGrade, found := e.loader.GetSubjectGrade(topic.SubjectGradeID)
	if !found {
		return nil, fmt.Errorf("topic %q references missing subject grade %q", topic.ID, topic.SubjectGradeID)
	}
	subject, found := e.loader.GetSubject(topic.SubjectID)
	if !found {
		return nil, fmt.Errorf("topic %q references missing subject %q", topic.ID, topic.SubjectID)
	}
	syllabus, found := e.loader.GetSyllabus(topic.SyllabusID)
	if !found {
		return nil, fmt.Errorf("topic %q references missing syllabus %q", topic.ID, topic.SyllabusID)
	}

	relations := []CurriculumRelation{
		newCurriculumRelation(
			CurriculumRelationContains,
			curriculumNode(CurriculumNodeKindSyllabus, syllabus.ID),
			curriculumNode(CurriculumNodeKindSubject, subject.ID),
			sourceEvidence(syllabus.Source, "", "", "", ""),
		),
		newCurriculumRelation(
			CurriculumRelationContains,
			curriculumNode(CurriculumNodeKindSubject, subject.ID),
			curriculumNode(CurriculumNodeKindSubjectGrade, subjectGrade.ID),
			sourceEvidence(subjectGrade.Source, "", "", "", subjectGrade.Provenance),
		),
		newCurriculumRelation(
			CurriculumRelationContains,
			curriculumNode(CurriculumNodeKindSubjectGrade, subjectGrade.ID),
			curriculumNode(CurriculumNodeKindTopic, topic.ID),
			sourceEvidence(subjectGrade.Source, topic.ID, "", "", subjectGrade.Provenance),
		),
		newCurriculumRelation(
			CurriculumRelationContainsObjective,
			curriculumNode(CurriculumNodeKindTopic, topic.ID),
			objectiveNode(topic.ID, objective.ID, ""),
			sourceEvidence(topic.Source, topic.ID, objective.ID, "", topic.Provenance),
		),
	}
	for _, prerequisite := range topic.Prerequisites.Required {
		relations = append(relations, newCurriculumRelation(
			CurriculumRelationRequires,
			curriculumNode(CurriculumNodeKindTopic, topic.ID),
			curriculumNode(CurriculumNodeKindTopic, prerequisite.TopicID),
			sourceEvidence(topic.Source, topic.ID, "", "", topic.Provenance),
		))
	}
	for _, concept := range e.loader.ConceptsForTopic(topic.SyllabusID, topic.ID) {
		for _, occurrence := range concept.Curricula {
			if occurrence.SyllabusID == topic.SyllabusID && occurrence.TopicID == topic.ID {
				continue
			}
			relations = append(relations, CurriculumRelation{
				Type: CurriculumRelationSharesConcept,
				From: curriculumNode(CurriculumNodeKindTopic, topic.ID),
				Via:  curriculumNode(CurriculumNodeKindConcept, concept.ID),
				To:   curriculumNode(CurriculumNodeKindTopic, occurrence.TopicID),
				Evidence: sourceEvidence(
					concept.Source,
					topic.ID,
					"",
					"",
					"",
				),
			})
		}
	}
	if selected.Source.Path != topic.Source.Path {
		locale := selected.Source.Locale
		relations = append(
			relations,
			newCurriculumRelation(
				CurriculumRelationLocalizedBy,
				curriculumNode(CurriculumNodeKindTopic, topic.ID),
				curriculumNodeWithLocale(CurriculumNodeKindTopic, topic.ID, locale),
				sourceEvidence(selected.Source, topic.ID, "", "", selected.Provenance),
			),
			newCurriculumRelation(
				CurriculumRelationLocalizedBy,
				objectiveNode(topic.ID, objective.ID, ""),
				objectiveNode(topic.ID, objective.ID, locale),
				sourceEvidence(selected.Source, topic.ID, objective.ID, "", selected.Provenance),
			),
		)
	}
	return relations, nil
}

func (e *Engine) prerequisitePathRelations(path []prerequisiteStep) []CurriculumRelation {
	relations := make([]CurriculumRelation, 0, len(path))
	for _, step := range path {
		relations = append(relations, newCurriculumRelation(
			CurriculumRelationRequires,
			curriculumNode(CurriculumNodeKindTopic, step.dependent.ID),
			curriculumNode(CurriculumNodeKindTopic, step.prerequisite.ID),
			sourceEvidence(
				step.dependent.Source,
				step.dependent.ID,
				"",
				"",
				step.dependent.Provenance,
			),
		))
	}
	return relations
}

func newCurriculumRelation(
	relationType CurriculumRelationType,
	from CurriculumNodeRef,
	to CurriculumNodeRef,
	evidence EvidenceRef,
) CurriculumRelation {
	return CurriculumRelation{
		Type:     relationType,
		From:     from,
		To:       to,
		Evidence: evidence,
	}
}

func sourceEvidence(
	source SourceRef,
	topicID, objectiveID, questionID, provenance string,
) EvidenceRef {
	return EvidenceRef{
		SourcePath:  source.Path,
		SourceKind:  source.Kind,
		Revision:    source.Revision,
		TopicID:     topicID,
		ObjectiveID: objectiveID,
		QuestionID:  questionID,
		Provenance:  provenance,
	}
}

func curriculumNode(kind CurriculumNodeKind, id string) CurriculumNodeRef {
	return CurriculumNodeRef{Kind: kind, ID: id}
}

func curriculumNodeWithLocale(kind CurriculumNodeKind, id, locale string) CurriculumNodeRef {
	return CurriculumNodeRef{Kind: kind, ID: id, Locale: locale}
}

func objectiveNode(topicID, objectiveID, locale string) CurriculumNodeRef {
	return CurriculumNodeRef{
		Kind:    CurriculumNodeKindObjective,
		ID:      objectiveID,
		TopicID: topicID,
		Locale:  locale,
	}
}

func questionNode(topicID, questionID, locale string) CurriculumNodeRef {
	return CurriculumNodeRef{
		Kind:    CurriculumNodeKindQuestion,
		ID:      questionID,
		TopicID: topicID,
		Locale:  locale,
	}
}

func teachingNoteNode(topicID, locale string) CurriculumNodeRef {
	return CurriculumNodeRef{
		Kind:    CurriculumNodeKindTeachingNote,
		ID:      topicID,
		TopicID: topicID,
		Locale:  locale,
	}
}

func workedExampleNode(topicID, exampleID, locale string) CurriculumNodeRef {
	return CurriculumNodeRef{
		Kind:    CurriculumNodeKindWorkedExample,
		ID:      exampleID,
		TopicID: topicID,
		Locale:  locale,
	}
}

func localizedAssessmentQuestion(
	loader *Loader,
	topicID, locale string,
	canonical AssessmentQuestion,
) (AssessmentQuestion, SourceRef, bool) {
	assessment, _ := loader.GetAssessment(topicID)
	source := assessment.Source
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return canonical, source, false
	}
	variant, found := loader.GetAssessmentVariant(topicID, locale)
	if !found {
		return canonical, source, false
	}
	question, found := assessmentQuestion(variant, canonical.ID)
	if !found {
		return canonical, source, false
	}
	return question, variant.Source, true
}

func orderedCheckOptions(options AssessmentOptions) []CheckOption {
	if len(options) == 0 {
		return nil
	}
	ids := make([]string, 0, len(options))
	for id := range options {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]CheckOption, 0, len(ids))
	for _, id := range ids {
		result = append(result, CheckOption{ID: id, Text: options[id]})
	}
	return result
}

func masteryThreshold(topic Topic) float64 {
	if topic.Mastery.minimumScoreSet || topic.Mastery.MinimumScore != 0 {
		return topic.Mastery.MinimumScore
	}
	return progress.MasteryThreshold
}

func minimumAssessmentCount(topic Topic) int {
	if topic.Mastery.assessmentSet || topic.Mastery.AssessmentCount != 0 {
		return topic.Mastery.AssessmentCount
	}
	return 1
}

func localizedTopic(loader *Loader, canonical Topic, locale string) Topic {
	locale = strings.TrimSpace(locale)
	if locale == "" || locale == canonical.Language {
		return canonical
	}
	variant, found := loader.GetTopicVariant(canonical.ID, locale)
	if !found {
		return canonical
	}

	localized := canonical
	localized.Language = variant.Language
	localized.Source = variant.Source
	localized.Provenance = variant.Provenance
	if variant.Name != "" {
		localized.Name = variant.Name
	}
	if variant.NameEN != "" {
		localized.NameEN = variant.NameEN
	}
	localizedObjectives := append([]LearningObjective(nil), canonical.LearningObjectives...)
	variantObjectives := make(map[string]LearningObjective, len(variant.LearningObjectives))
	for _, objective := range variant.LearningObjectives {
		variantObjectives[objective.ID] = objective
	}
	for index, objective := range localizedObjectives {
		variantObjective, found := variantObjectives[objective.ID]
		if !found {
			continue
		}
		if variantObjective.Text != "" {
			objective.Text = variantObjective.Text
		}
		if variantObjective.TextEN != "" {
			objective.TextEN = variantObjective.TextEN
		}
		localizedObjectives[index] = objective
	}
	localized.LearningObjectives = localizedObjectives
	return localized
}

func localizedTopicName(topic Topic) string {
	if topic.Language == "en" && strings.TrimSpace(topic.NameEN) != "" {
		return topic.NameEN
	}
	if strings.TrimSpace(topic.Name) != "" {
		return topic.Name
	}
	return topic.NameEN
}

func localizedObjectiveText(language string, objective LearningObjective) string {
	if language == "en" && strings.TrimSpace(objective.TextEN) != "" {
		return objective.TextEN
	}
	if strings.TrimSpace(objective.Text) != "" {
		return objective.Text
	}
	return objective.TextEN
}

func selectObjective(topic Topic, objectiveID string) (LearningObjective, error) {
	if len(topic.LearningObjectives) == 0 {
		return LearningObjective{}, fmt.Errorf("topic %q has no learning objectives", topic.ID)
	}
	objectiveID = strings.TrimSpace(objectiveID)
	if objectiveID == "" {
		return topic.LearningObjectives[0], nil
	}
	for _, objective := range topic.LearningObjectives {
		if objective.ID == objectiveID {
			return objective, nil
		}
	}
	return LearningObjective{}, fmt.Errorf("topic %q has no objective %q", topic.ID, objectiveID)
}

func selectGradeableQuestion(assessment Assessment, objectiveID string) (AssessmentQuestion, bool) {
	for _, question := range assessment.Questions {
		if question.LearningObjective == objectiveID && isDeterministicallyGradeable(question.Answer.Type) {
			return question, true
		}
	}
	return AssessmentQuestion{}, false
}

func assessmentQuestion(assessment Assessment, questionID string) (AssessmentQuestion, bool) {
	for _, question := range assessment.Questions {
		if question.ID == questionID {
			return question, true
		}
	}
	return AssessmentQuestion{}, false
}

func isDeterministicallyGradeable(answerType string) bool {
	switch answerType {
	case "exact", "multiple_choice":
		return true
	default:
		return false
	}
}

func gradeDeterministicAnswer(canonical, presentation AssessmentQuestion, answer string) bool {
	actual := normalizeDeterministicAnswer(answer)
	if actual == "" {
		return false
	}
	expected := normalizeDeterministicAnswer(canonical.Answer.Value)
	if expected != "" && actual == expected {
		return true
	}
	localizedExpected := normalizeDeterministicAnswer(presentation.Answer.Value)
	if localizedExpected != "" && actual == localizedExpected {
		return true
	}
	if canonical.Answer.Type != "multiple_choice" {
		return false
	}
	for _, question := range []AssessmentQuestion{canonical, presentation} {
		optionID := correctOptionID(question)
		if optionID != "" && actual == normalizeDeterministicAnswer(optionID) {
			return true
		}
	}
	return false
}

func correctOptionID(question AssessmentQuestion) string {
	expected := normalizeDeterministicAnswer(question.Answer.Value)
	var matchedKey string
	for id := range question.Answer.Options {
		if normalizeDeterministicAnswer(id) != expected {
			continue
		}
		if matchedKey != "" {
			return ""
		}
		matchedKey = id
	}
	if matchedKey != "" {
		return matchedKey
	}

	var matchedValue string
	for id, value := range question.Answer.Options {
		if normalizeDeterministicAnswer(value) != expected {
			continue
		}
		if matchedValue != "" {
			return ""
		}
		matchedValue = id
	}
	return matchedValue
}

func normalizeDeterministicAnswer(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.NewReplacer(
		" ", "",
		"\n", "",
		"\t", "",
		"−", "-",
		"–", "-",
	).Replace(value)
}

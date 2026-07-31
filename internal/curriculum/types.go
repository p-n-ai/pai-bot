// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"
)

// Topic represents a curriculum topic loaded from YAML.
type Topic struct {
	ID                   string                `yaml:"id"`
	OfficialRef          string                `yaml:"official_ref"`
	Name                 string                `yaml:"name"`
	NameEN               string                `yaml:"name_en"`
	SubjectGradeID       string                `yaml:"subject_grade_id"`
	SubjectID            string                `yaml:"subject_id"`
	SyllabusID           string                `yaml:"syllabus_id"`
	CountryID            string                `yaml:"country_id"`
	Language             string                `yaml:"language"`
	Difficulty           string                `yaml:"difficulty"`
	Tier                 string                `yaml:"tier"`
	ContentStandards     []ContentStandard     `yaml:"content_standards"`
	LearningObjectives   []LearningObjective   `yaml:"learning_objectives"`
	PerformanceStandards []PerformanceStandard `yaml:"performance_standards"`
	Prerequisites        Prerequisites         `yaml:"prerequisites"`
	BackgroundKnowledge  []string              `yaml:"background_knowledge"`
	CrossCurriculum      []CrossCurriculumLink `yaml:"cross_curriculum"`
	Teaching             TeachingGuidance      `yaml:"teaching"`
	Mastery              MasteryCriteria       `yaml:"mastery"`
	TeachingNotesFile    string                `yaml:"ai_teaching_notes"`
	ExamplesFile         string                `yaml:"examples_file"`
	AssessmentsFile      string                `yaml:"assessments_file"`
	QualityLevel         int                   `yaml:"quality_level"`
	Provenance           string                `yaml:"provenance"`
	Source               SourceRef             `yaml:"-"`
	qualityLevelSet      bool
}

// UnmarshalYAML normalizes the schema's root and nested engagement-hook
// placements into one teaching field.
func (topic *Topic) UnmarshalYAML(node *yaml.Node) error {
	type rawTopic Topic
	var decoded struct {
		rawTopic        `yaml:",inline"`
		EngagementHooks []string `yaml:"engagement_hooks"`
	}
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	nested := decoded.Teaching.EngagementHooks
	if len(decoded.EngagementHooks) > 0 && len(nested) > 0 &&
		!slices.Equal(decoded.EngagementHooks, nested) {
		return fmt.Errorf("root and teaching engagement_hooks conflict")
	}
	if len(decoded.EngagementHooks) > 0 {
		decoded.Teaching.EngagementHooks = slices.Clone(decoded.EngagementHooks)
	}
	*topic = Topic(decoded.rawTopic)
	var presence struct {
		QualityLevel *int `yaml:"quality_level"`
	}
	if err := node.Decode(&presence); err != nil {
		return err
	}
	topic.qualityLevelSet = presence.QualityLevel != nil
	return nil
}

// IsAITeachingReady reports whether OSS explicitly claims the topic meets the
// minimum quality level for AI teaching.
func (topic Topic) IsAITeachingReady() bool {
	return topic.qualityLevelSet && topic.QualityLevel >= 3
}

// SourceRef identifies the OSS artifact that supplied a curriculum value.
type SourceRef struct {
	Path     string
	Kind     string
	Locale   string
	Revision string
}

// CrossCurriculumLink is an explicit OSS-authored graph edge through a shared
// concept.
type CrossCurriculumLink struct {
	ConceptID  string `yaml:"concept_id"`
	SyllabusID string `yaml:"syllabus_id"`
	TopicID    string `yaml:"topic_id"`
}

// Concept is a curriculum-neutral identity shared by explicit topic edges.
type Concept struct {
	ID         string                 `yaml:"id"`
	Name       string                 `yaml:"name"`
	Domain     string                 `yaml:"domain"`
	Subdomain  string                 `yaml:"subdomain"`
	Definition string                 `yaml:"definition"`
	Curricula  []ConceptCurriculumRef `yaml:"curricula"`
	Source     SourceRef              `yaml:"-"`
}

// ConceptCurriculumRef identifies one syllabus-scoped concept occurrence.
type ConceptCurriculumRef struct {
	SyllabusID string `yaml:"syllabus"`
	TopicID    string `yaml:"topic"`
	Scope      string `yaml:"scope"`
}

// ContentStandard is an official curriculum content standard within a topic.
type ContentStandard struct {
	ID     string `yaml:"id"`
	Text   string `yaml:"text"`
	TextEN string `yaml:"text_en"`
}

// LearningObjective represents a learning objective within a topic.
type LearningObjective struct {
	ID                string `yaml:"id"`
	ContentStandardID string `yaml:"content_standard_id"`
	Text              string `yaml:"text"`
	TextEN            string `yaml:"text_en"`
	Bloom             string `yaml:"bloom"`
}

// PerformanceStandard describes one source-defined level of demonstrated mastery.
type PerformanceStandard struct {
	Level        int    `yaml:"level"`
	ID           string `yaml:"id"`
	Text         string `yaml:"text"`
	TextEN       string `yaml:"text_en"`
	Descriptor   string `yaml:"descriptor"`
	DescriptorEN string `yaml:"descriptor_en"`
}

// TeachingGuidance contains structured, source-authored teaching guidance.
type TeachingGuidance struct {
	Sequence             []string              `yaml:"sequence"`
	CommonMisconceptions []CommonMisconception `yaml:"common_misconceptions"`
	EngagementHooks      []string              `yaml:"engagement_hooks"`
}

// CommonMisconception pairs a known error with its source-authored remediation.
type CommonMisconception struct {
	Misconception string `yaml:"misconception"`
	Remediation   string `yaml:"remediation"`
}

// MasteryCriteria is the OSS-authored mastery policy for a topic.
type MasteryCriteria struct {
	MinimumScore     float64          `yaml:"minimum_score"`
	AssessmentCount  int              `yaml:"assessment_count"`
	SpacedRepetition SpacedRepetition `yaml:"spaced_repetition"`
	minimumScoreSet  bool
	assessmentSet    bool
}

// UnmarshalYAML preserves the difference between an absent policy and an
// explicitly authored zero value.
func (criteria *MasteryCriteria) UnmarshalYAML(node *yaml.Node) error {
	var raw struct {
		MinimumScore     *float64         `yaml:"minimum_score"`
		AssessmentCount  *int             `yaml:"assessment_count"`
		SpacedRepetition SpacedRepetition `yaml:"spaced_repetition"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}

	*criteria = MasteryCriteria{SpacedRepetition: raw.SpacedRepetition}
	if raw.MinimumScore != nil {
		criteria.MinimumScore = *raw.MinimumScore
		criteria.minimumScoreSet = true
	}
	if raw.AssessmentCount != nil {
		criteria.AssessmentCount = *raw.AssessmentCount
		criteria.assessmentSet = true
	}
	return nil
}

// SpacedRepetition is the OSS-authored review schedule for a topic.
type SpacedRepetition struct {
	InitialIntervalDays int     `yaml:"initial_interval_days"`
	Multiplier          float64 `yaml:"multiplier"`
}

// Prerequisites holds required and recommended prerequisites.
type Prerequisites struct {
	Required    []PrerequisiteRef `yaml:"required"`
	Recommended []PrerequisiteRef `yaml:"recommended"`
}

// PrerequisiteRef preserves source context while TopicID remains the only
// graph-bearing identity.
type PrerequisiteRef struct {
	TopicID string `yaml:"id"`
	Name    string `yaml:"name"`
	NameEN  string `yaml:"name_en"`
	Reason  string `yaml:"reason"`
}

// UnmarshalYAML accepts the compact topic-ID form and the schema's rich edge.
func (ref *PrerequisiteRef) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode && node.Tag != "!!null" {
		ref.TopicID = node.Value
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("prerequisite must be a topic ID or object")
	}
	type rawRef PrerequisiteRef
	return node.Decode((*rawRef)(ref))
}

// RequiredTopicIDs returns graph-bearing required topic identities.
func (prerequisites Prerequisites) RequiredTopicIDs() []string {
	ids := make([]string, len(prerequisites.Required))
	for index, ref := range prerequisites.Required {
		ids[index] = ref.TopicID
	}
	return ids
}

// Syllabus represents a top-level syllabus (e.g., KSSM Matematik Tingkatan 1).
type Syllabus struct {
	ID          string    `yaml:"id"`
	Name        string    `yaml:"name"`
	NameEN      string    `yaml:"name_en"`
	Country     string    `yaml:"country"`
	CountryID   string    `yaml:"country_id"`
	Board       string    `yaml:"board"`
	Level       string    `yaml:"level"`
	Language    string    `yaml:"language"`
	Description string    `yaml:"description"`
	Subjects    []string  `yaml:"subjects"`
	Source      SourceRef `yaml:"-"`
}

// Subject represents a subject within a syllabus (e.g., Algebra).
type Subject struct {
	ID          string    `yaml:"id"`
	Name        string    `yaml:"name"`
	NameEN      string    `yaml:"name_en"`
	SyllabusID  string    `yaml:"syllabus_id"`
	GradeID     string    `yaml:"grade_id"`
	CountryID   string    `yaml:"country_id"`
	Language    string    `yaml:"language"`
	Description string    `yaml:"description"`
	Topics      []string  `yaml:"topics"`
	Source      SourceRef `yaml:"-"`
}

// SubjectGrade binds one subject and grade to its source-declared topic order.
type SubjectGrade struct {
	ID          string    `yaml:"id"`
	Name        string    `yaml:"name"`
	NameEN      string    `yaml:"name_en"`
	SubjectID   string    `yaml:"subject_id"`
	SyllabusID  string    `yaml:"syllabus_id"`
	GradeID     string    `yaml:"grade_id"`
	CountryID   string    `yaml:"country_id"`
	Language    string    `yaml:"language"`
	Description string    `yaml:"description"`
	Topics      []string  `yaml:"topics"`
	Provenance  string    `yaml:"provenance"`
	Source      SourceRef `yaml:"-"`
}

// ExampleSet groups source-authored worked examples for one topic.
type ExampleSet struct {
	ID             string          `yaml:"id"`
	TopicID        string          `yaml:"topic_id"`
	Provenance     string          `yaml:"provenance"`
	Description    string          `yaml:"description"`
	WorkedExamples []WorkedExample `yaml:"worked_examples"`
	Source         SourceRef       `yaml:"-"`
}

// WorkedExample is one source-authored teaching example.
type WorkedExample struct {
	ID                   string `yaml:"id"`
	Topic                string `yaml:"topic"`
	Difficulty           string `yaml:"difficulty"`
	TPLevel              int    `yaml:"tp_level"`
	KBAT                 bool   `yaml:"kbat"`
	LearningObjective    string `yaml:"learning_objective"`
	RealWorldAnalogy     string `yaml:"real_world_analogy"`
	MisconceptionAlert   string `yaml:"misconception_alert"`
	Scenario             string `yaml:"scenario"`
	Working              string `yaml:"working"`
	SourceRefDescription string `yaml:"source_ref"`
}

// TeachingNote preserves one Markdown teaching artifact with its source.
type TeachingNote struct {
	TopicID string
	Content string
	Source  SourceRef
}

// Assessment groups quiz questions for a topic.
type Assessment struct {
	TopicID    string               `yaml:"topic_id"`
	Questions  []AssessmentQuestion `yaml:"questions"`
	Provenance string               `yaml:"provenance"`
	Source     SourceRef            `yaml:"-"`
}

// AssessmentQuestion represents a single assessment item from OSS.
type AssessmentQuestion struct {
	ID                string                 `yaml:"id"`
	Type              string                 `yaml:"type"`
	Text              string                 `yaml:"text"`
	Difficulty        string                 `yaml:"difficulty"`
	LearningObjective string                 `yaml:"learning_objective"`
	TPLevel           int                    `yaml:"tp_level"`
	KBAT              bool                   `yaml:"kbat"`
	Answer            AssessmentAnswer       `yaml:"answer"`
	Marks             int                    `yaml:"marks"`
	Rubric            []AssessmentRubricItem `yaml:"rubric"`
	Hints             []AssessmentHint       `yaml:"hints"`
	Distractors       []AssessmentDistractor `yaml:"distractors"`
	marksSet          bool                   `yaml:"-"`
}

// UnmarshalYAML preserves the distinction between omitted optional marks and
// an explicitly invalid zero value.
func (question *AssessmentQuestion) UnmarshalYAML(node *yaml.Node) error {
	type rawAssessmentQuestion AssessmentQuestion
	var decoded rawAssessmentQuestion
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	var presence struct {
		Marks *int `yaml:"marks"`
	}
	if err := node.Decode(&presence); err != nil {
		return err
	}
	*question = AssessmentQuestion(decoded)
	question.marksSet = presence.Marks != nil
	return nil
}

// AssessmentAnswer describes the expected answer format.
type AssessmentAnswer struct {
	Type        string                 `yaml:"type"`
	Value       string                 `yaml:"value"`
	Pairs       []AssessmentPair       `yaml:"pairs"`
	Working     string                 `yaml:"working"`
	Options     AssessmentOptions      `yaml:"options"`
	Distractors []AssessmentDistractor `yaml:"distractors"`
}

// AssessmentPair is one ordered left/right item in a non-auto-gradeable
// matching assessment.
type AssessmentPair struct {
	Left  string `yaml:"left"`
	Right string `yaml:"right"`
}

// AssessmentOptions normalizes schema-valid option arrays and keyed maps to
// stable learner-visible IDs.
type AssessmentOptions map[string]string

// UnmarshalYAML accepts both option shapes allowed by the OSS schema.
func (options *AssessmentOptions) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		var values []string
		if err := node.Decode(&values); err != nil {
			return err
		}
		normalized := make(AssessmentOptions, len(values))
		for index, value := range values {
			normalized[assessmentOptionID(index)] = value
		}
		*options = normalized
		return nil
	case yaml.MappingNode:
		var values map[string]string
		if err := node.Decode(&values); err != nil {
			return err
		}
		*options = AssessmentOptions(values)
		return nil
	case yaml.ScalarNode:
		if node.Tag == "!!null" {
			*options = nil
			return nil
		}
	}
	return fmt.Errorf("assessment options must be an array or object")
}

func assessmentOptionID(index int) string {
	var reversed []byte
	for {
		reversed = append(reversed, byte('A'+index%26))
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	result := make([]byte, len(reversed))
	for index := range reversed {
		result[len(reversed)-1-index] = reversed[index]
	}
	return string(result)
}

// AssessmentRubricItem describes one rubric line.
type AssessmentRubricItem struct {
	Marks    int    `yaml:"marks"`
	Criteria string `yaml:"criteria"`
}

// AssessmentHint is a progressive hint.
type AssessmentHint struct {
	Level int    `yaml:"level"`
	Text  string `yaml:"text"`
}

// AssessmentDistractor is an incorrect option with targeted feedback.
type AssessmentDistractor struct {
	Value    string `yaml:"value"`
	Feedback string `yaml:"feedback"`
}

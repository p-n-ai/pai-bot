// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum

import (
	"fmt"
	"sort"
	"strings"
)

func (l *Loader) validate() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, id := range sortedKeys(l.syllabi) {
		syllabus := l.syllabi[id]
		seen := make(map[string]struct{}, len(syllabus.Subjects))
		for _, subjectID := range syllabus.Subjects {
			if _, duplicate := seen[subjectID]; duplicate {
				return fmt.Errorf("%s: syllabus %q repeats subject %q", sourceName(syllabus.Source, id), id, subjectID)
			}
			seen[subjectID] = struct{}{}
			subject, found := l.subjects[subjectID]
			if !found {
				return fmt.Errorf("%s: syllabus %q references missing subject %q", sourceName(syllabus.Source, id), id, subjectID)
			}
			if subject.SyllabusID != id {
				return fmt.Errorf(
					"%s: syllabus %q lists subject %q owned by syllabus %q",
					sourceName(syllabus.Source, id),
					id,
					subjectID,
					subject.SyllabusID,
				)
			}
		}
	}
	for _, id := range sortedKeys(l.subjects) {
		subject := l.subjects[id]
		if _, found := l.syllabi[subject.SyllabusID]; !found {
			return fmt.Errorf("%s: subject %q references missing syllabus %q", sourceName(subject.Source, id), id, subject.SyllabusID)
		}
		if !contains(l.syllabi[subject.SyllabusID].Subjects, id) {
			return fmt.Errorf("%s: subject %q is not listed by syllabus %q", sourceName(subject.Source, id), id, subject.SyllabusID)
		}
	}
	for _, id := range sortedKeys(l.subjectGrades) {
		subjectGrade := l.subjectGrades[id]
		if _, found := l.syllabi[subjectGrade.SyllabusID]; !found {
			return fmt.Errorf("%s: subject grade %q references missing syllabus %q", sourceName(subjectGrade.Source, id), id, subjectGrade.SyllabusID)
		}
		subject, found := l.subjects[subjectGrade.SubjectID]
		if !found {
			return fmt.Errorf("%s: subject grade %q references missing subject %q", sourceName(subjectGrade.Source, id), id, subjectGrade.SubjectID)
		}
		if subject.SyllabusID != subjectGrade.SyllabusID {
			return fmt.Errorf(
				"%s: subject grade %q syllabus %q does not match subject %q syllabus %q",
				sourceName(subjectGrade.Source, id),
				id,
				subjectGrade.SyllabusID,
				subjectGrade.SubjectID,
				subject.SyllabusID,
			)
		}
		seen := make(map[string]struct{}, len(subjectGrade.Topics))
		for _, topicID := range subjectGrade.Topics {
			if _, duplicate := seen[topicID]; duplicate {
				return fmt.Errorf("%s: subject grade %q repeats topic %q", sourceName(subjectGrade.Source, id), id, topicID)
			}
			seen[topicID] = struct{}{}
			topic, found := l.topics[topicID]
			if !found {
				return fmt.Errorf("%s: subject grade %q references missing topic %q", sourceName(subjectGrade.Source, id), id, topicID)
			}
			if topic.SubjectGradeID != id {
				return fmt.Errorf("%s: topic %q declares subject grade %q, want %q", sourceName(topic.Source, topic.ID), topic.ID, topic.SubjectGradeID, id)
			}
		}
	}
	for _, id := range sortedKeys(l.concepts) {
		if err := l.validateConcept(l.concepts[id]); err != nil {
			return err
		}
	}
	for _, id := range sortedKeys(l.topics) {
		if err := l.validateTopic(l.topics[id]); err != nil {
			return err
		}
	}
	if err := l.validateRequiredPrerequisiteGraph(); err != nil {
		return err
	}
	for _, topicID := range sortedKeys(l.assessments) {
		if err := l.validateAssessment(l.assessments[topicID]); err != nil {
			return err
		}
	}
	for _, topicID := range sortedKeys(l.examples) {
		if err := l.validateExamples(l.examples[topicID]); err != nil {
			return err
		}
	}
	for _, topicID := range sortedKeys(l.exampleVariants) {
		canonical, found := l.examples[topicID]
		if !found {
			variants := l.exampleVariants[topicID]
			for _, locale := range sortedKeys(variants) {
				return fmt.Errorf(
					"%s: example variant references missing canonical examples for topic %q",
					sourceName(variants[locale].Source, topicID),
					topicID,
				)
			}
		}
		for _, locale := range sortedKeys(l.exampleVariants[topicID]) {
			variant := l.exampleVariants[topicID][locale]
			if err := l.validateExamples(variant); err != nil {
				return err
			}
			if err := validateExamplesVariant(canonical, locale, variant); err != nil {
				return err
			}
		}
	}
	for _, topicID := range sortedKeys(l.topicVariants) {
		canonical, found := l.topics[topicID]
		if !found {
			variants := l.topicVariants[topicID]
			for _, locale := range sortedKeys(variants) {
				return fmt.Errorf("%s: topic variant references missing canonical topic %q", sourceName(variants[locale].Source, topicID), topicID)
			}
		}
		for _, locale := range sortedKeys(l.topicVariants[topicID]) {
			if err := validateTopicVariant(canonical, locale, l.topicVariants[topicID][locale]); err != nil {
				return err
			}
		}
	}
	for _, topicID := range sortedKeys(l.assessmentVariants) {
		canonical, found := l.assessments[topicID]
		if !found {
			variants := l.assessmentVariants[topicID]
			for _, locale := range sortedKeys(variants) {
				return fmt.Errorf("%s: assessment variant references missing canonical assessment for topic %q", sourceName(variants[locale].Source, topicID), topicID)
			}
		}
		for _, locale := range sortedKeys(l.assessmentVariants[topicID]) {
			if err := validateAssessmentVariant(canonical, locale, l.assessmentVariants[topicID][locale]); err != nil {
				return err
			}
		}
	}
	for topicID := range l.teachingNotes {
		if _, found := l.topics[topicID]; !found {
			return fmt.Errorf("teaching notes reference missing topic %q", topicID)
		}
	}
	for topicID := range l.teachingNoteVariants {
		if _, found := l.topics[topicID]; !found {
			return fmt.Errorf("teaching note variants reference missing topic %q", topicID)
		}
	}
	return nil
}

func validateExamplesVariant(canonical ExampleSet, locale string, variant ExampleSet) error {
	source := sourceName(variant.Source, variant.TopicID)
	if variant.Source.Locale != locale {
		return fmt.Errorf("%s: example variant locale does not match %q", source, locale)
	}
	canonicalByID := make(map[string]WorkedExample, len(canonical.WorkedExamples))
	for _, example := range canonical.WorkedExamples {
		canonicalByID[example.ID] = example
	}
	for _, example := range variant.WorkedExamples {
		canonicalExample, found := canonicalByID[example.ID]
		if !found {
			return fmt.Errorf(
				"%s: worked example %q has no canonical worked example",
				source,
				example.ID,
			)
		}
		if example.LearningObjective != canonicalExample.LearningObjective {
			return fmt.Errorf(
				"%s: worked example %q changes canonical objective identity",
				source,
				example.ID,
			)
		}
	}
	if len(variant.WorkedExamples) != len(canonical.WorkedExamples) {
		return fmt.Errorf(
			"%s: example variant has %d worked examples, want canonical %d",
			source,
			len(variant.WorkedExamples),
			len(canonical.WorkedExamples),
		)
	}
	return nil
}

func (l *Loader) validateExamples(examples ExampleSet) error {
	source := sourceName(examples.Source, examples.TopicID)
	topic, found := l.topics[examples.TopicID]
	if !found {
		return fmt.Errorf("%s: examples reference missing topic %q", source, examples.TopicID)
	}
	objectives := make(map[string]struct{}, len(topic.LearningObjectives))
	for _, objective := range topic.LearningObjectives {
		objectives[objective.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(examples.WorkedExamples))
	for _, example := range examples.WorkedExamples {
		if strings.TrimSpace(example.ID) == "" {
			return fmt.Errorf("%s: worked example requires id", source)
		}
		if _, duplicate := seen[example.ID]; duplicate {
			return fmt.Errorf("%s: example set repeats worked example %q", source, example.ID)
		}
		seen[example.ID] = struct{}{}
		if strings.TrimSpace(example.Topic) == "" {
			return fmt.Errorf("%s: worked example %q requires topic", source, example.ID)
		}
		if !isWorkedExampleDifficulty(example.Difficulty) {
			return fmt.Errorf("%s: worked example %q has invalid difficulty %q", source, example.ID, example.Difficulty)
		}
		if example.TPLevel < 0 || example.TPLevel > 6 {
			return fmt.Errorf("%s: worked example %q has invalid tp_level %d", source, example.ID, example.TPLevel)
		}
		for field, value := range map[string]string{
			"misconception_alert": example.MisconceptionAlert,
			"scenario":            example.Scenario,
			"working":             example.Working,
		} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s: worked example %q requires %s", source, example.ID, field)
			}
		}
		if example.LearningObjective != "" {
			if _, found := objectives[example.LearningObjective]; !found {
				return fmt.Errorf(
					"%s: worked example %q references missing objective %q",
					source,
					example.ID,
					example.LearningObjective,
				)
			}
		}
	}
	return nil
}

func (l *Loader) validateTopic(topic Topic) error {
	source := sourceName(topic.Source, topic.ID)
	if strings.TrimSpace(topic.Name) == "" && strings.TrimSpace(topic.NameEN) == "" {
		return fmt.Errorf("%s: topic %q requires name or name_en", source, topic.ID)
	}
	if topic.SyllabusID == "" || topic.SubjectID == "" || topic.SubjectGradeID == "" {
		return fmt.Errorf("%s: topic %q requires syllabus_id, subject_id, and subject_grade_id", source, topic.ID)
	}
	if _, found := l.syllabi[topic.SyllabusID]; !found {
		return fmt.Errorf("%s: topic %q references missing syllabus %q", source, topic.ID, topic.SyllabusID)
	}
	if _, found := l.subjects[topic.SubjectID]; !found {
		return fmt.Errorf("%s: topic %q references missing subject %q", source, topic.ID, topic.SubjectID)
	}
	subjectGrade, found := l.subjectGrades[topic.SubjectGradeID]
	if !found {
		return fmt.Errorf("%s: topic %q references missing subject grade %q", source, topic.ID, topic.SubjectGradeID)
	}
	if subjectGrade.SyllabusID != topic.SyllabusID || subjectGrade.SubjectID != topic.SubjectID {
		return fmt.Errorf("%s: topic %q does not match subject grade %q scope", source, topic.ID, topic.SubjectGradeID)
	}
	if !contains(subjectGrade.Topics, topic.ID) {
		return fmt.Errorf("%s: topic %q is not listed by subject grade %q", source, topic.ID, topic.SubjectGradeID)
	}
	if topic.Mastery.MinimumScore < 0 || topic.Mastery.MinimumScore > 1 ||
		topic.Mastery.AssessmentCount < 0 ||
		(topic.Mastery.assessmentSet && topic.Mastery.AssessmentCount == 0) {
		return fmt.Errorf("%s: topic %q has invalid mastery policy", source, topic.ID)
	}
	catalogStub := topic.qualityLevelSet &&
		topic.QualityLevel == 0 &&
		len(topic.ContentStandards) == 0 &&
		len(topic.LearningObjectives) == 0
	if len(topic.ContentStandards) == 0 && !catalogStub {
		return fmt.Errorf("%s: topic %q requires content standards", source, topic.ID)
	}
	if len(topic.LearningObjectives) == 0 && !catalogStub {
		return fmt.Errorf("%s: topic %q requires learning objectives", source, topic.ID)
	}

	standards := make(map[string]struct{}, len(topic.ContentStandards))
	for _, standard := range topic.ContentStandards {
		if standard.ID == "" {
			return fmt.Errorf("%s: topic %q has content standard without ID", source, topic.ID)
		}
		if _, duplicate := standards[standard.ID]; duplicate {
			return fmt.Errorf("%s: topic %q repeats content standard %q", source, topic.ID, standard.ID)
		}
		standards[standard.ID] = struct{}{}
	}
	objectives := make(map[string]struct{}, len(topic.LearningObjectives))
	for _, objective := range topic.LearningObjectives {
		if objective.ID == "" {
			return fmt.Errorf("%s: topic %q has learning objective without ID", source, topic.ID)
		}
		if _, duplicate := objectives[objective.ID]; duplicate {
			return fmt.Errorf("%s: topic %q repeats learning objective %q", source, topic.ID, objective.ID)
		}
		objectives[objective.ID] = struct{}{}
		if strings.TrimSpace(objective.Text) == "" && strings.TrimSpace(objective.TextEN) == "" {
			return fmt.Errorf("%s: objective %q requires text or text_en", source, objective.ID)
		}
		if objective.ContentStandardID != "" {
			if _, found := standards[objective.ContentStandardID]; !found {
				return fmt.Errorf("%s: objective %q references missing content standard %q", source, objective.ID, objective.ContentStandardID)
			}
		}
	}
	for _, prerequisiteRefs := range [][]PrerequisiteRef{topic.Prerequisites.Required, topic.Prerequisites.Recommended} {
		for _, prerequisiteRef := range prerequisiteRefs {
			if prerequisiteRef.TopicID == "" {
				return fmt.Errorf("%s: topic %q has prerequisite without topic ID", source, topic.ID)
			}
			if _, found := l.topics[prerequisiteRef.TopicID]; !found {
				return fmt.Errorf("%s: topic %q references missing prerequisite %q", source, topic.ID, prerequisiteRef.TopicID)
			}
		}
	}
	seenCrossCurriculum := make(map[string]struct{}, len(topic.CrossCurriculum))
	for _, link := range topic.CrossCurriculum {
		key := link.ConceptID + "\x00" + link.SyllabusID + "\x00" + link.TopicID
		if _, duplicate := seenCrossCurriculum[key]; duplicate {
			return fmt.Errorf("%s: topic %q repeats cross-curriculum edge to %q", source, topic.ID, link.TopicID)
		}
		seenCrossCurriculum[key] = struct{}{}
		if link.SyllabusID == topic.SyllabusID {
			return fmt.Errorf("%s: topic %q cross-curriculum edge must target another syllabus", source, topic.ID)
		}
		concept, found := l.concepts[link.ConceptID]
		if !found {
			return fmt.Errorf("%s: topic %q references missing concept %q", source, topic.ID, link.ConceptID)
		}
		linkedTopic, found := l.topics[link.TopicID]
		if !found {
			return fmt.Errorf("%s: topic %q cross-curriculum edge references missing topic %q", source, topic.ID, link.TopicID)
		}
		if linkedTopic.SyllabusID != link.SyllabusID {
			return fmt.Errorf("%s: cross-curriculum topic %q does not belong to syllabus %q", source, link.TopicID, link.SyllabusID)
		}
		if !conceptIncludes(concept, topic.SyllabusID, topic.ID) {
			return fmt.Errorf("%s: concept %q does not declare source topic %q", source, link.ConceptID, topic.ID)
		}
		if !conceptIncludes(concept, link.SyllabusID, link.TopicID) {
			return fmt.Errorf("%s: concept %q does not declare topic %q in syllabus %q", source, link.ConceptID, link.TopicID, link.SyllabusID)
		}
	}
	if topic.IsAITeachingReady() {
		if _, found := l.teachingNotes[topic.ID]; !found {
			return fmt.Errorf("%s: AI-teaching-ready topic %q requires canonical teaching notes", source, topic.ID)
		}
		if _, found := l.examples[topic.ID]; !found {
			return fmt.Errorf("%s: AI-teaching-ready topic %q requires canonical examples", source, topic.ID)
		}
		if _, found := l.assessments[topic.ID]; !found {
			return fmt.Errorf("%s: AI-teaching-ready topic %q requires canonical assessment", source, topic.ID)
		}
	}
	return nil
}

func (l *Loader) validateConcept(concept Concept) error {
	source := sourceName(concept.Source, concept.ID)
	if strings.TrimSpace(concept.Name) == "" ||
		strings.TrimSpace(concept.Domain) == "" ||
		strings.TrimSpace(concept.Subdomain) == "" ||
		strings.TrimSpace(concept.Definition) == "" ||
		len(concept.Curricula) == 0 {
		return fmt.Errorf("%s: concept %q is incomplete", source, concept.ID)
	}
	seen := make(map[string]struct{}, len(concept.Curricula))
	for _, occurrence := range concept.Curricula {
		if strings.TrimSpace(occurrence.SyllabusID) == "" ||
			strings.TrimSpace(occurrence.TopicID) == "" ||
			strings.TrimSpace(occurrence.Scope) == "" {
			return fmt.Errorf("%s: concept %q has incomplete curriculum occurrence", source, concept.ID)
		}
		key := occurrence.SyllabusID + "\x00" + occurrence.TopicID
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%s: concept %q repeats curriculum topic %q", source, concept.ID, occurrence.TopicID)
		}
		seen[key] = struct{}{}
		topic, found := l.topics[occurrence.TopicID]
		if !found {
			return fmt.Errorf("%s: concept %q references missing topic %q", source, concept.ID, occurrence.TopicID)
		}
		if topic.SyllabusID != occurrence.SyllabusID {
			return fmt.Errorf(
				"%s: concept %q topic %q does not belong to syllabus %q",
				source,
				concept.ID,
				occurrence.TopicID,
				occurrence.SyllabusID,
			)
		}
	}
	return nil
}

func (l *Loader) validateAssessment(assessment Assessment) error {
	source := sourceName(assessment.Source, assessment.TopicID)
	topic, found := l.topics[assessment.TopicID]
	if !found {
		return fmt.Errorf("%s: assessment references missing topic %q", source, assessment.TopicID)
	}
	objectives := make(map[string]struct{}, len(topic.LearningObjectives))
	for _, objective := range topic.LearningObjectives {
		objectives[objective.ID] = struct{}{}
	}
	questions := make(map[string]struct{}, len(assessment.Questions))
	for _, question := range assessment.Questions {
		if question.ID == "" {
			return fmt.Errorf("%s: assessment has question without ID", source)
		}
		if _, duplicate := questions[question.ID]; duplicate {
			return fmt.Errorf("%s: assessment repeats question %q", source, question.ID)
		}
		questions[question.ID] = struct{}{}
		if strings.TrimSpace(question.Text) == "" {
			return fmt.Errorf("%s: question %q requires text", source, question.ID)
		}
		if !isAssessmentDifficulty(question.Difficulty) {
			return fmt.Errorf("%s: question %q has invalid difficulty %q", source, question.ID, question.Difficulty)
		}
		if question.marksSet && question.Marks < 1 {
			return fmt.Errorf("%s: question %q requires positive marks", source, question.ID)
		}
		if !isAssessmentAnswerType(question.Answer.Type) {
			return fmt.Errorf("%s: question %q has unsupported answer type %q", source, question.ID, question.Answer.Type)
		}
		if _, found := objectives[question.LearningObjective]; !found {
			return fmt.Errorf("%s: question %q references missing objective %q", source, question.ID, question.LearningObjective)
		}
		if question.Answer.Type == "exact" && strings.TrimSpace(question.Answer.Value) == "" {
			return fmt.Errorf("%s: exact question %q has no answer value", source, question.ID)
		}
		if question.Answer.Type == "multiple_choice" {
			if strings.TrimSpace(question.Answer.Value) == "" {
				return fmt.Errorf("%s: multiple-choice question %q has no answer value", source, question.ID)
			}
			if len(question.Answer.Options) > 0 &&
				(len(question.Answer.Options) < 2 || correctOptionID(question) == "") {
				return fmt.Errorf("%s: multiple-choice question %q has no valid correct option", source, question.ID)
			}
		}
		if question.Answer.Type == "matching" {
			if len(question.Answer.Pairs) == 0 {
				return fmt.Errorf("%s: matching question %q has no answer pairs", source, question.ID)
			}
			for _, pair := range question.Answer.Pairs {
				if strings.TrimSpace(pair.Left) == "" || strings.TrimSpace(pair.Right) == "" {
					return fmt.Errorf("%s: matching question %q has incomplete answer pair", source, question.ID)
				}
			}
		}
	}
	return nil
}

func validateTopicVariant(canonical Topic, locale string, variant Topic) error {
	source := sourceName(variant.Source, variant.ID)
	if variant.Language != locale || variant.Source.Locale != locale {
		return fmt.Errorf("%s: topic variant language %q does not match locale %q", source, variant.Language, locale)
	}
	for field, values := range map[string][2]string{
		"syllabus":      {canonical.SyllabusID, variant.SyllabusID},
		"subject":       {canonical.SubjectID, variant.SubjectID},
		"subject grade": {canonical.SubjectGradeID, variant.SubjectGradeID},
	} {
		if values[1] != "" && values[0] != values[1] {
			return fmt.Errorf("%s: topic variant %s %q does not match canonical %q", source, field, values[1], values[0])
		}
	}
	canonicalObjectives := make(map[string]struct{}, len(canonical.LearningObjectives))
	for _, objective := range canonical.LearningObjectives {
		canonicalObjectives[objective.ID] = struct{}{}
	}
	seenObjectives := make(map[string]struct{}, len(variant.LearningObjectives))
	for _, objective := range variant.LearningObjectives {
		if _, duplicate := seenObjectives[objective.ID]; duplicate {
			return fmt.Errorf("%s: topic variant repeats objective %q", source, objective.ID)
		}
		seenObjectives[objective.ID] = struct{}{}
		if strings.TrimSpace(objective.Text) == "" && strings.TrimSpace(objective.TextEN) == "" {
			return fmt.Errorf("%s: topic variant objective %q requires localized text", source, objective.ID)
		}
		if _, found := canonicalObjectives[objective.ID]; !found {
			return fmt.Errorf("%s: topic variant references missing canonical objective %q", source, objective.ID)
		}
	}
	return nil
}

func validateAssessmentVariant(canonical Assessment, locale string, variant Assessment) error {
	source := sourceName(variant.Source, variant.TopicID)
	if variant.Source.Locale != locale {
		return fmt.Errorf("%s: assessment variant locale does not match %q", source, locale)
	}
	seenQuestions := make(map[string]struct{}, len(variant.Questions))
	for _, question := range variant.Questions {
		if _, duplicate := seenQuestions[question.ID]; duplicate {
			return fmt.Errorf("%s: assessment variant repeats question %q", source, question.ID)
		}
		seenQuestions[question.ID] = struct{}{}
		if strings.TrimSpace(question.Text) == "" ||
			!isAssessmentDifficulty(question.Difficulty) ||
			(question.marksSet && question.Marks < 1) ||
			!isAssessmentAnswerType(question.Answer.Type) {
			return fmt.Errorf("%s: assessment variant question %q has invalid plan fields", source, question.ID)
		}
		canonicalQuestion, found := assessmentQuestion(canonical, question.ID)
		if !found {
			return fmt.Errorf("%s: assessment variant question %q has no canonical question", source, question.ID)
		}
		if question.LearningObjective != canonicalQuestion.LearningObjective || question.Answer.Type != canonicalQuestion.Answer.Type {
			return fmt.Errorf("%s: assessment variant question %q changes the canonical grading contract", source, question.ID)
		}
		if question.Answer.Type == "exact" && strings.TrimSpace(question.Answer.Value) == "" {
			return fmt.Errorf("%s: assessment variant exact question %q has no answer value", source, question.ID)
		}
		if question.Answer.Type == "multiple_choice" {
			if !sameKeys(question.Answer.Options, canonicalQuestion.Answer.Options) ||
				correctOptionID(question) != correctOptionID(canonicalQuestion) {
				return fmt.Errorf("%s: assessment variant question %q changes canonical option identity", source, question.ID)
			}
		}
	}
	return nil
}

func (l *Loader) validateRequiredPrerequisiteGraph() error {
	state := make(map[string]uint8, len(l.topics))
	var visit func(string, []string) error
	visit = func(topicID string, path []string) error {
		switch state[topicID] {
		case 1:
			cycle := append(path, topicID)
			return fmt.Errorf("%s: required prerequisite cycle: %s", sourceName(l.topics[topicID].Source, topicID), strings.Join(cycle, " -> "))
		case 2:
			return nil
		}
		state[topicID] = 1
		for _, prerequisiteRef := range l.topics[topicID].Prerequisites.Required {
			if err := visit(prerequisiteRef.TopicID, append(path, topicID)); err != nil {
				return err
			}
		}
		state[topicID] = 2
		return nil
	}
	for _, topicID := range sortedKeys(l.topics) {
		if err := visit(topicID, nil); err != nil {
			return err
		}
	}
	return nil
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sourceName(source SourceRef, fallback string) string {
	if source.Path != "" {
		return source.Path
	}
	return fallback
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sameKeys(left, right AssessmentOptions) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, found := right[key]; !found {
			return false
		}
	}
	return true
}

func conceptIncludes(concept Concept, syllabusID, topicID string) bool {
	for _, occurrence := range concept.Curricula {
		if occurrence.SyllabusID == syllabusID && occurrence.TopicID == topicID {
			return true
		}
	}
	return false
}

func isAssessmentDifficulty(difficulty string) bool {
	switch difficulty {
	case "easy", "medium", "hard":
		return true
	default:
		return false
	}
}

func isWorkedExampleDifficulty(difficulty string) bool {
	switch difficulty {
	case "easy", "medium", "hard":
		return true
	default:
		return false
	}
}

func isAssessmentAnswerType(answerType string) bool {
	switch answerType {
	case "exact",
		"range",
		"multiple_choice",
		"true_false",
		"matching",
		"free_text",
		"structured",
		"structured_response",
		"subjective",
		"opb":
		return true
	default:
		return false
	}
}

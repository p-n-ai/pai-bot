// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum

import (
	"maps"
	"slices"
)

func cloneTopic(topic Topic) Topic {
	topic.ContentStandards = slices.Clone(topic.ContentStandards)
	topic.LearningObjectives = slices.Clone(topic.LearningObjectives)
	topic.PerformanceStandards = slices.Clone(topic.PerformanceStandards)
	topic.Prerequisites.Required = slices.Clone(topic.Prerequisites.Required)
	topic.Prerequisites.Recommended = slices.Clone(topic.Prerequisites.Recommended)
	topic.BackgroundKnowledge = slices.Clone(topic.BackgroundKnowledge)
	topic.CrossCurriculum = slices.Clone(topic.CrossCurriculum)
	topic.Teaching.Sequence = slices.Clone(topic.Teaching.Sequence)
	topic.Teaching.CommonMisconceptions = slices.Clone(topic.Teaching.CommonMisconceptions)
	topic.Teaching.EngagementHooks = slices.Clone(topic.Teaching.EngagementHooks)
	return topic
}

func cloneConcept(concept Concept) Concept {
	concept.Curricula = slices.Clone(concept.Curricula)
	return concept
}

func cloneExampleSet(examples ExampleSet) ExampleSet {
	examples.WorkedExamples = slices.Clone(examples.WorkedExamples)
	return examples
}

func cloneAssessment(assessment Assessment) Assessment {
	assessment.Questions = slices.Clone(assessment.Questions)
	for index := range assessment.Questions {
		question := &assessment.Questions[index]
		question.Rubric = slices.Clone(question.Rubric)
		question.Hints = slices.Clone(question.Hints)
		question.Distractors = slices.Clone(question.Distractors)
		question.Answer.Options = maps.Clone(question.Answer.Options)
		question.Answer.Pairs = slices.Clone(question.Answer.Pairs)
		question.Answer.Distractors = slices.Clone(question.Answer.Distractors)
	}
	return assessment
}

func cloneSyllabus(syllabus Syllabus) Syllabus {
	syllabus.Subjects = slices.Clone(syllabus.Subjects)
	return syllabus
}

func cloneSubject(subject Subject) Subject {
	subject.Topics = slices.Clone(subject.Topics)
	return subject
}

func cloneSubjectGrade(subjectGrade SubjectGrade) SubjectGrade {
	subjectGrade.Topics = slices.Clone(subjectGrade.Topics)
	return subjectGrade
}

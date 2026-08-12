// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader loads and caches curriculum content from the filesystem.
type Loader struct {
	rootDir              string
	topics               map[string]Topic
	topicVariants        map[string]map[string]Topic
	subjects             map[string]Subject
	subjectGrades        map[string]SubjectGrade
	syllabi              map[string]Syllabus
	concepts             map[string]Concept
	examples             map[string]ExampleSet
	exampleVariants      map[string]map[string]ExampleSet
	assessments          map[string]Assessment
	assessmentVariants   map[string]map[string]Assessment
	teachingNotes        map[string]TeachingNote
	teachingNoteVariants map[string]map[string]TeachingNote
	snapshotRevision     string
	mu                   sync.RWMutex
}

// SnapshotStats describes the validated artifact classes consumed by the
// curriculum engine.
type SnapshotStats struct {
	Syllabi               int
	Subjects              int
	SubjectGrades         int
	Topics                int
	TopicVariants         int
	Concepts              int
	Examples              int
	ExampleVariants       int
	TeachingNotes         int
	TeachingNoteVariants  int
	AITeachingReadyTopics int
	Assessments           int
	AssessmentVariants    int
}

// NewLoader creates a new curriculum loader and loads all content.
func NewLoader(rootDir string) (*Loader, error) {
	revision, err := curriculumSnapshotRevision(rootDir)
	if err != nil {
		return nil, fmt.Errorf("hashing curriculum snapshot: %w", err)
	}
	l := &Loader{
		rootDir:              rootDir,
		topics:               make(map[string]Topic),
		topicVariants:        make(map[string]map[string]Topic),
		subjects:             make(map[string]Subject),
		subjectGrades:        make(map[string]SubjectGrade),
		syllabi:              make(map[string]Syllabus),
		concepts:             make(map[string]Concept),
		examples:             make(map[string]ExampleSet),
		exampleVariants:      make(map[string]map[string]ExampleSet),
		assessments:          make(map[string]Assessment),
		assessmentVariants:   make(map[string]map[string]Assessment),
		teachingNotes:        make(map[string]TeachingNote),
		teachingNoteVariants: make(map[string]map[string]TeachingNote),
		snapshotRevision:     revision,
	}

	if err := l.loadAll(); err != nil {
		return nil, fmt.Errorf("loading curriculum: %w", err)
	}
	if err := l.validate(); err != nil {
		return nil, fmt.Errorf("validating curriculum: %w", err)
	}

	return l, nil
}

// GetTopic returns a topic by ID.
func (l *Loader) GetTopic(id string) (Topic, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.topics[id]
	return cloneTopic(t), ok
}

// GetSubject returns a subject by ID.
func (l *Loader) GetSubject(id string) (Subject, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	subject, ok := l.subjects[id]
	return cloneSubject(subject), ok
}

// GetSubjectGrade returns a subject-grade and its source-declared topic order.
func (l *Loader) GetSubjectGrade(id string) (SubjectGrade, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	subjectGrade, ok := l.subjectGrades[id]
	return cloneSubjectGrade(subjectGrade), ok
}

// GetSyllabus returns a syllabus by ID.
func (l *Loader) GetSyllabus(id string) (Syllabus, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	syllabus, ok := l.syllabi[id]
	return cloneSyllabus(syllabus), ok
}

// GetConcept returns one explicit cross-curriculum concept.
func (l *Loader) GetConcept(id string) (Concept, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	concept, ok := l.concepts[id]
	return cloneConcept(concept), ok
}

// ConceptsForTopic returns explicit concept mappings containing one
// syllabus-scoped topic, ordered by concept ID.
func (l *Loader) ConceptsForTopic(syllabusID, topicID string) []Concept {
	l.mu.RLock()
	defer l.mu.RUnlock()

	concepts := make([]Concept, 0)
	for _, concept := range l.concepts {
		if conceptIncludes(concept, syllabusID, topicID) {
			concepts = append(concepts, cloneConcept(concept))
		}
	}
	sort.Slice(concepts, func(i, j int) bool {
		return concepts[i].ID < concepts[j].ID
	})
	return concepts
}

// GetTeachingNotes returns teaching notes for a topic ID.
func (l *Loader) GetTeachingNotes(id string) (string, bool) {
	note, ok := l.GetTeachingNote(id)
	return note.Content, ok
}

// GetTeachingNote returns a source-backed canonical teaching note.
func (l *Loader) GetTeachingNote(id string) (TeachingNote, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n, ok := l.teachingNotes[id]
	return n, ok
}

// GetTeachingNotesVariant returns locale-specific notes without replacing the
// canonical notes returned by GetTeachingNotes.
func (l *Loader) GetTeachingNotesVariant(topicID, locale string) (string, bool) {
	note, ok := l.GetTeachingNoteVariant(topicID, locale)
	return note.Content, ok
}

// GetTeachingNoteVariant returns one source-backed locale teaching note.
func (l *Loader) GetTeachingNoteVariant(topicID, locale string) (TeachingNote, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	variants := l.teachingNoteVariants[topicID]
	notes, ok := variants[locale]
	return notes, ok
}

// GetExamples returns the canonical worked examples for a topic.
func (l *Loader) GetExamples(topicID string) (ExampleSet, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	examples, ok := l.examples[topicID]
	return cloneExampleSet(examples), ok
}

// GetExamplesVariant returns locale-specific examples without replacing the
// canonical examples returned by GetExamples.
func (l *Loader) GetExamplesVariant(topicID, locale string) (ExampleSet, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	variants := l.exampleVariants[topicID]
	examples, ok := variants[locale]
	return cloneExampleSet(examples), ok
}

// GetAssessment returns an assessment by topic ID.
func (l *Loader) GetAssessment(topicID string) (Assessment, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	assessment, ok := l.assessments[topicID]
	return cloneAssessment(assessment), ok
}

// GetAssessmentVariant returns a locale-specific assessment without replacing
// the canonical grading contract returned by GetAssessment.
func (l *Loader) GetAssessmentVariant(topicID, locale string) (Assessment, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	variants := l.assessmentVariants[topicID]
	assessment, ok := variants[locale]
	return cloneAssessment(assessment), ok
}

// GetTopicVariant returns a locale-specific variant without replacing the
// canonical topic returned by GetTopic.
func (l *Loader) GetTopicVariant(topicID, locale string) (Topic, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	variants := l.topicVariants[topicID]
	topic, ok := variants[locale]
	return cloneTopic(topic), ok
}

// AllTopics returns all loaded topics.
func (l *Loader) AllTopics() []Topic {
	l.mu.RLock()
	defer l.mu.RUnlock()
	topics := make([]Topic, 0, len(l.topics))
	for _, t := range l.topics {
		topics = append(topics, cloneTopic(t))
	}
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].ID < topics[j].ID
	})
	return topics
}

// SnapshotRevision identifies the exact curriculum artifact bytes loaded by
// this instance.
func (l *Loader) SnapshotRevision() string {
	return l.snapshotRevision
}

// SnapshotStats returns exact counts for activation and deployment checks.
func (l *Loader) SnapshotStats() SnapshotStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := SnapshotStats{
		Syllabi:       len(l.syllabi),
		Subjects:      len(l.subjects),
		SubjectGrades: len(l.subjectGrades),
		Topics:        len(l.topics),
		Concepts:      len(l.concepts),
		Examples:      len(l.examples),
		TeachingNotes: len(l.teachingNotes),
		Assessments:   len(l.assessments),
	}
	for _, variants := range l.topicVariants {
		stats.TopicVariants += len(variants)
	}
	for _, variants := range l.assessmentVariants {
		stats.AssessmentVariants += len(variants)
	}
	for _, variants := range l.exampleVariants {
		stats.ExampleVariants += len(variants)
	}
	for _, variants := range l.teachingNoteVariants {
		stats.TeachingNoteVariants += len(variants)
	}
	for _, topic := range l.topics {
		if topic.IsAITeachingReady() {
			stats.AITeachingReadyTopics++
		}
	}
	return stats
}

func (l *Loader) loadAll() error {
	return filepath.Walk(l.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		switch {
		case strings.HasSuffix(path, ".teaching.md"):
			return l.loadTeachingNotes(path)
		case base == "subject.yaml" || base == "subject.yml":
			return l.loadSubject(path)
		case base == "subject-grade.yaml" || base == "subject-grade.yml":
			return l.loadSubjectGrade(path)
		case base == "syllabus.yaml" || base == "syllabus.yml":
			return l.loadSyllabus(path)
		case isConceptPath(path):
			return l.loadConcept(path)
		case isExamplesPath(path):
			if isTranslationPath(path) {
				return l.loadExamplesVariant(path)
			}
			return l.loadExamples(path)
		case isAssessmentPath(path):
			if isTranslationPath(path) {
				return l.loadAssessmentVariant(path)
			}
			return l.loadAssessment(path)
		case isTopicPath(path):
			if isTranslationPath(path) {
				return l.loadTopicVariant(path)
			}
			return l.loadTopic(path)
		}
		return nil
	})
}

func isConceptPath(path string) bool {
	slashPath := filepath.ToSlash(path)
	if !strings.Contains(slashPath, "/concepts/") {
		return false
	}
	return strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
}

func isAssessmentPath(path string) bool {
	return strings.HasSuffix(path, ".assessments.yaml") || strings.HasSuffix(path, ".assessments.yml")
}

func isExamplesPath(path string) bool {
	return strings.HasSuffix(path, ".examples.yaml") || strings.HasSuffix(path, ".examples.yml")
}

func isTopicPath(path string) bool {
	slashPath := filepath.ToSlash(path)
	if !strings.Contains(slashPath, "/topics/") {
		return false
	}
	if isAssessmentPath(path) || strings.HasSuffix(path, ".examples.yaml") || strings.HasSuffix(path, ".examples.yml") {
		return false
	}
	return strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
}

func isTranslationPath(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/translations/")
}

func (l *Loader) loadTopic(path string) error {
	var topic Topic
	if err := decodeYAMLFile(path, &topic); err != nil {
		return fmt.Errorf("decode topic %s: %w", path, err)
	}
	if topic.ID == "" {
		return fmt.Errorf("topic %s: id is required", path)
	}
	source, err := l.sourceRef(path, "topic", topic.Language)
	if err != nil {
		return fmt.Errorf("source topic %s: %w", path, err)
	}
	topic.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.topics[topic.ID]; exists {
		return fmt.Errorf("topic %s: duplicate canonical ID %q", path, topic.ID)
	}
	l.topics[topic.ID] = topic
	return nil
}

func (l *Loader) loadTopicVariant(path string) error {
	var topic Topic
	if err := decodeYAMLFile(path, &topic); err != nil {
		return fmt.Errorf("decode topic variant %s: %w", path, err)
	}
	locale := translationLocale(path)
	if topic.ID == "" || topic.Language == "" || locale == "" {
		return fmt.Errorf("topic variant %s: id, language, and translation locale are required", path)
	}
	if topic.Language != locale {
		return fmt.Errorf("topic variant %s: language %q does not match translation locale %q", path, topic.Language, locale)
	}
	source, err := l.sourceRef(path, "topic_variant", locale)
	if err != nil {
		return fmt.Errorf("source topic variant %s: %w", path, err)
	}
	topic.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.topicVariants[topic.ID] == nil {
		l.topicVariants[topic.ID] = make(map[string]Topic)
	}
	if _, exists := l.topicVariants[topic.ID][locale]; exists {
		return fmt.Errorf("topic variant %s: duplicate locale %q for topic %q", path, locale, topic.ID)
	}
	l.topicVariants[topic.ID][locale] = topic
	return nil
}

func (l *Loader) loadSubject(path string) error {
	var subject Subject
	if err := decodeYAMLFile(path, &subject); err != nil {
		return fmt.Errorf("decode subject %s: %w", path, err)
	}
	if subject.ID == "" {
		return fmt.Errorf("subject %s: id is required", path)
	}
	source, err := l.sourceRef(path, "subject", subject.Language)
	if err != nil {
		return fmt.Errorf("source subject %s: %w", path, err)
	}
	subject.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.subjects[subject.ID]; exists {
		return fmt.Errorf("subject %s: duplicate ID %q", path, subject.ID)
	}
	l.subjects[subject.ID] = subject
	return nil
}

func (l *Loader) loadSubjectGrade(path string) error {
	var subjectGrade SubjectGrade
	if err := decodeYAMLFile(path, &subjectGrade); err != nil {
		return fmt.Errorf("decode subject grade %s: %w", path, err)
	}
	if subjectGrade.ID == "" {
		return fmt.Errorf("subject grade %s: id is required", path)
	}
	source, err := l.sourceRef(path, "subject_grade", subjectGrade.Language)
	if err != nil {
		return fmt.Errorf("source subject grade %s: %w", path, err)
	}
	subjectGrade.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.subjectGrades[subjectGrade.ID]; exists {
		return fmt.Errorf("subject grade %s: duplicate ID %q", path, subjectGrade.ID)
	}
	l.subjectGrades[subjectGrade.ID] = subjectGrade
	return nil
}

func (l *Loader) loadSyllabus(path string) error {
	var syllabus Syllabus
	if err := decodeYAMLFile(path, &syllabus); err != nil {
		return fmt.Errorf("decode syllabus %s: %w", path, err)
	}
	if syllabus.ID == "" {
		return fmt.Errorf("syllabus %s: id is required", path)
	}
	source, err := l.sourceRef(path, "syllabus", syllabus.Language)
	if err != nil {
		return fmt.Errorf("source syllabus %s: %w", path, err)
	}
	syllabus.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.syllabi[syllabus.ID]; exists {
		return fmt.Errorf("syllabus %s: duplicate ID %q", path, syllabus.ID)
	}
	l.syllabi[syllabus.ID] = syllabus
	return nil
}

func (l *Loader) loadConcept(path string) error {
	var concept Concept
	if err := decodeYAMLFile(path, &concept); err != nil {
		return fmt.Errorf("decode concept %s: %w", path, err)
	}
	if concept.ID == "" {
		return fmt.Errorf("concept %s: id is required", path)
	}
	source, err := l.sourceRef(path, "concept", "")
	if err != nil {
		return fmt.Errorf("source concept %s: %w", path, err)
	}
	concept.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.concepts[concept.ID]; exists {
		return fmt.Errorf("concept %s: duplicate ID %q", path, concept.ID)
	}
	l.concepts[concept.ID] = concept
	return nil
}

func (l *Loader) loadTeachingNotes(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	yamlPath := strings.TrimSuffix(path, ".teaching.md") + ".yaml"
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("teaching notes %s: read topic metadata %s: %w", path, yamlPath, err)
	}

	topicID := strings.TrimSuffix(filepath.Base(path), ".teaching.md")
	if err == nil {
		var partial struct {
			ID string `yaml:"id"`
		}
		if err := yaml.Unmarshal(yamlData, &partial); err != nil {
			return fmt.Errorf("teaching notes %s: decode topic metadata %s: %w", path, yamlPath, err)
		}
		topicID = partial.ID
	}
	if topicID == "" {
		return fmt.Errorf("teaching notes %s: topic metadata %s has no ID", path, yamlPath)
	}

	locale := ""
	kind := "teaching_note"
	if isTranslationPath(path) {
		locale = translationLocale(path)
		if locale == "" {
			return fmt.Errorf("teaching notes %s: translation locale is required", path)
		}
		kind = "teaching_note_variant"
	}
	source, err := l.sourceRef(path, kind, locale)
	if err != nil {
		return fmt.Errorf("source teaching notes %s: %w", path, err)
	}
	note := TeachingNote{
		TopicID: topicID,
		Content: string(data),
		Source:  source,
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if isTranslationPath(path) {
		if l.teachingNoteVariants[topicID] == nil {
			l.teachingNoteVariants[topicID] = make(map[string]TeachingNote)
		}
		if _, exists := l.teachingNoteVariants[topicID][locale]; exists {
			return fmt.Errorf("teaching notes %s: duplicate locale %q for topic %q", path, locale, topicID)
		}
		l.teachingNoteVariants[topicID][locale] = note
		return nil
	}
	if _, exists := l.teachingNotes[topicID]; exists {
		return fmt.Errorf("teaching notes %s: duplicate canonical topic ID %q", path, topicID)
	}
	l.teachingNotes[topicID] = note
	return nil
}

func (l *Loader) loadExamples(path string) error {
	var examples ExampleSet
	if err := decodeYAMLFile(path, &examples); err != nil {
		return fmt.Errorf("decode examples %s: %w", path, err)
	}
	if examples.TopicID == "" || len(examples.WorkedExamples) == 0 {
		return fmt.Errorf("examples %s: topic_id and worked_examples are required", path)
	}
	source, err := l.sourceRef(path, "examples", "")
	if err != nil {
		return fmt.Errorf("source examples %s: %w", path, err)
	}
	examples.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.examples[examples.TopicID]; exists {
		return fmt.Errorf("examples %s: duplicate canonical topic ID %q", path, examples.TopicID)
	}
	l.examples[examples.TopicID] = examples
	return nil
}

func (l *Loader) loadExamplesVariant(path string) error {
	var examples ExampleSet
	if err := decodeYAMLFile(path, &examples); err != nil {
		return fmt.Errorf("decode examples variant %s: %w", path, err)
	}
	locale := translationLocale(path)
	if examples.TopicID == "" || len(examples.WorkedExamples) == 0 || locale == "" {
		return fmt.Errorf("examples variant %s: topic_id, worked_examples, and locale are required", path)
	}
	source, err := l.sourceRef(path, "examples_variant", locale)
	if err != nil {
		return fmt.Errorf("source examples variant %s: %w", path, err)
	}
	examples.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.exampleVariants[examples.TopicID] == nil {
		l.exampleVariants[examples.TopicID] = make(map[string]ExampleSet)
	}
	if _, exists := l.exampleVariants[examples.TopicID][locale]; exists {
		return fmt.Errorf("examples variant %s: duplicate locale %q for topic %q", path, locale, examples.TopicID)
	}
	l.exampleVariants[examples.TopicID][locale] = examples
	return nil
}

func (l *Loader) loadAssessmentVariant(path string) error {
	var assessment Assessment
	if err := decodeYAMLFile(path, &assessment); err != nil {
		return fmt.Errorf("decode assessment variant %s: %w", path, err)
	}
	locale := translationLocale(path)
	if assessment.TopicID == "" || len(assessment.Questions) == 0 || locale == "" {
		return fmt.Errorf("assessment variant %s: topic_id, questions, and locale are required", path)
	}
	source, err := l.sourceRef(path, "assessment_variant", locale)
	if err != nil {
		return fmt.Errorf("source assessment variant %s: %w", path, err)
	}
	assessment.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.assessmentVariants[assessment.TopicID] == nil {
		l.assessmentVariants[assessment.TopicID] = make(map[string]Assessment)
	}
	if _, exists := l.assessmentVariants[assessment.TopicID][locale]; exists {
		return fmt.Errorf("assessment variant %s: duplicate locale %q for topic %q", path, locale, assessment.TopicID)
	}
	l.assessmentVariants[assessment.TopicID][locale] = assessment
	return nil
}

func (l *Loader) loadAssessment(path string) error {
	var assessment Assessment
	if err := decodeYAMLFile(path, &assessment); err != nil {
		return fmt.Errorf("decode assessment %s: %w", path, err)
	}

	if assessment.TopicID == "" || len(assessment.Questions) == 0 {
		return fmt.Errorf("assessment %s: topic_id and questions are required", path)
	}
	source, err := l.sourceRef(path, "assessment", "")
	if err != nil {
		return fmt.Errorf("source assessment %s: %w", path, err)
	}
	assessment.Source = source

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.assessments[assessment.TopicID]; exists {
		return fmt.Errorf("assessment %s: duplicate canonical topic ID %q", path, assessment.TopicID)
	}
	l.assessments[assessment.TopicID] = assessment
	return nil
}

func decodeYAMLFile[T any](path string, target *T) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}

func curriculumSnapshotRevision(rootDir string) (string, error) {
	var paths []string
	if err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		if isSnapshotArtifact(relative) {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)

	digest := sha256.New()
	_, _ = digest.Write([]byte("pai-curriculum-snapshot/v1"))
	for _, path := range paths {
		relative, err := filepath.Rel(rootDir, path)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		writeSnapshotDigestPart(digest, []byte(filepath.ToSlash(relative)))
		writeSnapshotDigestPart(digest, data)
	}
	return fmt.Sprintf("sha256:%x", digest.Sum(nil)), nil
}

func isSnapshotArtifact(relativePath string) bool {
	path := filepath.ToSlash(relativePath)
	if !strings.HasPrefix(path, "curricula/") && !strings.HasPrefix(path, "concepts/") {
		return false
	}
	return strings.HasSuffix(path, ".yaml") ||
		strings.HasSuffix(path, ".yml") ||
		strings.HasSuffix(path, ".teaching.md")
}

func writeSnapshotDigestPart(digest interface{ Write([]byte) (int, error) }, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write(value)
}

func (l *Loader) sourceRef(path, kind, locale string) (SourceRef, error) {
	relative, err := filepath.Rel(l.rootDir, path)
	if err != nil {
		relative = path
	}
	revision, err := curriculumArtifactRevision(path)
	if err != nil {
		return SourceRef{}, err
	}
	return SourceRef{
		Path:     filepath.ToSlash(relative),
		Kind:     kind,
		Locale:   locale,
		Revision: revision,
	}, nil
}

func curriculumArtifactRevision(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("pai-curriculum-artifact/v1\x00"))
	_, _ = digest.Write(data)
	return fmt.Sprintf("sha256:%x", digest.Sum(nil)), nil
}

func translationLocale(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for index, part := range parts {
		if part == "translations" && index+1 < len(parts) {
			return strings.TrimSpace(parts[index+1])
		}
	}
	return ""
}

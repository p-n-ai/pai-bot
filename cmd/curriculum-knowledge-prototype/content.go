package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type CurriculumContent struct {
	ID               int
	CurriculumSource string
	Kind             string
	Title            string
	Body             string
	Metadata         map[string]string
	SearchText       string
}

type syllabusFile struct {
	ID        string   `yaml:"id"`
	Name      string   `yaml:"name"`
	NameEN    string   `yaml:"name_en"`
	CountryID string   `yaml:"country_id"`
	Language  string   `yaml:"language"`
	Subjects  []string `yaml:"subjects"`
}

type subjectFile struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	NameEN     string `yaml:"name_en"`
	SyllabusID string `yaml:"syllabus_id"`
	CountryID  string `yaml:"country_id"`
	Language   string `yaml:"language"`
}

type subjectGradeFile struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	NameEN     string   `yaml:"name_en"`
	SubjectID  string   `yaml:"subject_id"`
	SyllabusID string   `yaml:"syllabus_id"`
	GradeID    string   `yaml:"grade_id"`
	CountryID  string   `yaml:"country_id"`
	Language   string   `yaml:"language"`
	Topics     []string `yaml:"topics"`
}

type topicFile struct {
	ID                 string           `yaml:"id"`
	OfficialRef        string           `yaml:"official_ref"`
	Name               string           `yaml:"name"`
	NameEN             string           `yaml:"name_en"`
	SubjectGradeID     string           `yaml:"subject_grade_id"`
	SubjectID          string           `yaml:"subject_id"`
	SyllabusID         string           `yaml:"syllabus_id"`
	CountryID          string           `yaml:"country_id"`
	Language           string           `yaml:"language"`
	Difficulty         string           `yaml:"difficulty"`
	Tier               string           `yaml:"tier"`
	ContentStandards   []standardFile   `yaml:"content_standards"`
	LearningObjectives []objectiveFile  `yaml:"learning_objectives"`
	Prerequisites      prerequisiteFile `yaml:"prerequisites"`
	QualityLevel       int              `yaml:"quality_level"`
	Provenance         string           `yaml:"provenance"`
	Teaching           teachingFile     `yaml:"teaching"`
}

type standardFile struct {
	ID     string `yaml:"id"`
	Text   string `yaml:"text"`
	TextEN string `yaml:"text_en"`
}

type objectiveFile struct {
	ID                string `yaml:"id"`
	Text              string `yaml:"text"`
	TextEN            string `yaml:"text_en"`
	Bloom             string `yaml:"bloom"`
	ContentStandardID string `yaml:"content_standard_id"`
}

type prerequisiteFile struct {
	Required    []prerequisiteRef `yaml:"required"`
	Recommended []prerequisiteRef `yaml:"recommended"`
}

func (p *prerequisiteFile) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		var required []prerequisiteRef
		if err := value.Decode(&required); err != nil {
			return err
		}
		p.Required = required
		return nil
	}

	type raw prerequisiteFile
	var decoded raw
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*p = prerequisiteFile(decoded)
	return nil
}

type prerequisiteRef struct {
	ID     string `yaml:"id"`
	NameEN string `yaml:"name_en"`
}

func (r *prerequisiteRef) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		r.ID = strings.TrimSpace(value.Value)
		return nil
	}

	type raw prerequisiteRef
	var decoded raw
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*r = prerequisiteRef(decoded)
	return nil
}

type teachingFile struct {
	Sequence             []string            `yaml:"sequence"`
	CommonMisconceptions []misconceptionFile `yaml:"common_misconceptions"`
	EngagementHooks      []string            `yaml:"engagement_hooks"`
}

type misconceptionFile struct {
	Misconception string `yaml:"misconception"`
	Remediation   string `yaml:"remediation"`
}

type examplesFile struct {
	TopicID        string          `yaml:"topic_id"`
	Provenance     string          `yaml:"provenance"`
	WorkedExamples []workedExample `yaml:"worked_examples"`
}

type workedExample struct {
	ID                 string `yaml:"id"`
	Topic              string `yaml:"topic"`
	Difficulty         string `yaml:"difficulty"`
	TPLevel            int    `yaml:"tp_level"`
	KBAT               bool   `yaml:"kbat"`
	LearningObjective  string `yaml:"learning_objective"`
	RealWorldAnalogy   string `yaml:"real_world_analogy"`
	MisconceptionAlert string `yaml:"misconception_alert"`
	Scenario           string `yaml:"scenario"`
	Working            string `yaml:"working"`
}

func buildCurriculumContent(root string) ([]CurriculumContent, error) {
	return buildCurriculumContentFromSource(root, "")
}

func buildCurriculumContentFromSource(root string, curriculumSource string) ([]CurriculumContent, error) {
	rows := []CurriculumContent{}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "translations" {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)
		switch {
		case base == "syllabus.yaml" || base == "syllabus.yml":
			return appendYAMLContent(path, &rows, buildSyllabusContent)
		case base == "subject.yaml" || base == "subject.yml":
			return appendYAMLContent(path, &rows, buildSubjectContent)
		case base == "subject-grade.yaml" || base == "subject-grade.yml":
			return appendYAMLContent(path, &rows, buildSubjectGradeContent)
		case strings.HasSuffix(base, ".teaching.md"):
			return appendTeachingContent(path, &rows)
		case strings.HasSuffix(base, ".examples.yaml") || strings.HasSuffix(base, ".examples.yml"):
			return appendOptionalYAMLContent(path, &rows, buildExamplesContent)
		case strings.HasSuffix(base, ".assessments.yaml") || strings.HasSuffix(base, ".assessments.yml"):
			return appendOptionalYAMLContent(path, &rows, buildAssessmentContent)
		case strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml"):
			return appendYAMLContent(path, &rows, buildTopicContent)
		default:
			return nil
		}
	})
	if err != nil {
		return nil, fmt.Errorf("walk curriculum: %w", err)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kind == rows[j].Kind {
			return rows[i].Title < rows[j].Title
		}
		return rows[i].Kind < rows[j].Kind
	})
	for i := range rows {
		rows[i].ID = i + 1
		rows[i].CurriculumSource = curriculumSource
	}
	return rows, nil
}

func appendYAMLContent[T any](
	path string,
	rows *[]CurriculumContent,
	build func(T) []CurriculumContent,
) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read yaml %s: %w", path, err)
	}

	var value T
	if err := yaml.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("parse yaml %s: %w", path, err)
	}

	for _, row := range build(value) {
		if row.Body == "" {
			continue
		}
		*rows = append(*rows, row)
	}
	return nil
}

func appendOptionalYAMLContent[T any](
	path string,
	rows *[]CurriculumContent,
	build func(T) []CurriculumContent,
) error {
	if err := appendYAMLContent(path, rows, build); err != nil {
		return nil
	}
	return nil
}

func buildSyllabusContent(value syllabusFile) []CurriculumContent {
	if value.ID == "" {
		return nil
	}

	title := firstNonEmpty(value.NameEN, value.Name, value.ID)
	body := strings.Join([]string{
		"syllabus: " + title,
		"subjects: " + strings.Join(value.Subjects, ", "),
	}, "\n")
	return []CurriculumContent{newContent("syllabus", title, body, map[string]string{
		"syllabus_id": value.ID,
		"country_id":  value.CountryID,
		"language":    value.Language,
	})}
}

func buildSubjectContent(value subjectFile) []CurriculumContent {
	if value.ID == "" {
		return nil
	}

	title := firstNonEmpty(value.NameEN, value.Name, value.ID)
	body := strings.Join([]string{
		"subject: " + title,
		"syllabus: " + value.SyllabusID,
	}, "\n")
	return []CurriculumContent{newContent("subject", title, body, map[string]string{
		"subject_id":  value.ID,
		"syllabus_id": value.SyllabusID,
		"country_id":  value.CountryID,
		"language":    value.Language,
	})}
}

func buildSubjectGradeContent(value subjectGradeFile) []CurriculumContent {
	if value.ID == "" {
		return nil
	}

	title := firstNonEmpty(value.NameEN, value.Name, value.ID)
	body := strings.Join([]string{
		"subject grade: " + title,
		"topics: " + strings.Join(value.Topics, ", "),
	}, "\n")
	return []CurriculumContent{newContent("subject_grade", title, body, map[string]string{
		"subject_grade_id": value.ID,
		"subject_id":       value.SubjectID,
		"syllabus_id":      value.SyllabusID,
		"grade_id":         value.GradeID,
		"country_id":       value.CountryID,
		"language":         value.Language,
	})}
}

func buildTopicContent(value topicFile) []CurriculumContent {
	if value.ID == "" {
		return nil
	}

	title := firstNonEmpty(value.NameEN, value.Name, value.ID)
	parts := []string{
		"topic: " + title,
		"official ref: " + value.OfficialRef,
		"difficulty: " + value.Difficulty,
		"tier: " + value.Tier,
	}
	for _, standard := range value.ContentStandards {
		text := firstNonEmpty(standard.TextEN, standard.Text)
		if text != "" {
			parts = append(parts, "standard "+standard.ID+": "+text)
		}
	}
	for _, objective := range value.LearningObjectives {
		text := firstNonEmpty(objective.TextEN, objective.Text)
		if text != "" {
			parts = append(parts, "objective "+objective.ID+": "+text)
		}
	}
	if len(value.Prerequisites.Required) > 0 {
		parts = append(parts, "required prerequisites: "+joinPrerequisites(value.Prerequisites.Required))
	}
	for _, step := range value.Teaching.Sequence {
		parts = append(parts, "teaching step: "+step)
	}
	for _, item := range value.Teaching.CommonMisconceptions {
		if item.Misconception != "" {
			parts = append(parts, "misconception: "+item.Misconception)
		}
		if item.Remediation != "" {
			parts = append(parts, "remediation: "+item.Remediation)
		}
	}
	for _, hook := range value.Teaching.EngagementHooks {
		parts = append(parts, "engagement hook: "+hook)
	}

	return []CurriculumContent{newContent("topic_card", title, strings.Join(parts, "\n"), map[string]string{
		"topic_id":         value.ID,
		"subject_grade_id": value.SubjectGradeID,
		"subject_id":       value.SubjectID,
		"syllabus_id":      value.SyllabusID,
		"country_id":       value.CountryID,
		"language":         value.Language,
		"difficulty":       value.Difficulty,
		"tier":             value.Tier,
		"provenance":       value.Provenance,
	})}
}

func appendTeachingContent(path string, rows *[]CurriculumContent) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read teaching notes: %w", err)
	}

	topicID := strings.TrimSuffix(filepath.Base(path), ".teaching.md")
	for _, section := range splitMarkdownSections(string(data)) {
		body := strings.TrimSpace(section.text)
		if body == "" {
			continue
		}
		*rows = append(*rows, newContent("teaching_note", section.title, body, map[string]string{
			"topic_id": topicID,
			"heading":  section.title,
		}))
	}
	return nil
}

func buildExamplesContent(value examplesFile) []CurriculumContent {
	if value.TopicID == "" {
		return nil
	}

	rows := make([]CurriculumContent, 0, len(value.WorkedExamples))
	for _, example := range value.WorkedExamples {
		title := firstNonEmpty(example.Topic, example.Scenario, example.ID)
		body := strings.Join(nonEmptyStrings(
			"scenario: "+example.Scenario,
			"analogy: "+example.RealWorldAnalogy,
			"misconception: "+example.MisconceptionAlert,
			"working: "+example.Working,
		), "\n")
		rows = append(rows, newContent("worked_example", title, body, map[string]string{
			"topic_id":           value.TopicID,
			"example_id":         example.ID,
			"difficulty":         example.Difficulty,
			"learning_objective": example.LearningObjective,
			"provenance":         value.Provenance,
			"tp_level":           fmt.Sprint(example.TPLevel),
			"kbat":               fmt.Sprint(example.KBAT),
		}))
	}
	return rows
}

func buildAssessmentContent(value curriculumAssessmentFile) []CurriculumContent {
	if value.TopicID == "" {
		return nil
	}

	rows := make([]CurriculumContent, 0, len(value.Questions))
	for _, question := range value.Questions {
		title := firstNonEmpty(question.Text, question.ID)
		body := strings.Join(nonEmptyStrings(
			"question: "+question.Text,
			"answer: "+question.Answer.Value,
			"working: "+question.Answer.Working,
			"hints: "+joinHints(question.Hints),
		), "\n")
		rows = append(rows, newContent("assessment_item", title, body, map[string]string{
			"topic_id":           value.TopicID,
			"question_id":        question.ID,
			"difficulty":         question.Difficulty,
			"learning_objective": question.LearningObjective,
			"answer_type":        question.Answer.Type,
			"marks":              fmt.Sprint(question.Marks),
			"provenance":         value.Provenance,
		}))
	}
	return rows
}

type curriculumAssessmentFile struct {
	TopicID    string                     `yaml:"topic_id"`
	Provenance string                     `yaml:"provenance"`
	Questions  []curriculumAssessmentItem `yaml:"questions"`
}

type curriculumAssessmentItem struct {
	ID                string           `yaml:"id"`
	Text              string           `yaml:"text"`
	Difficulty        string           `yaml:"difficulty"`
	LearningObjective string           `yaml:"learning_objective"`
	Answer            curriculumAnswer `yaml:"answer"`
	Marks             int              `yaml:"marks"`
	Hints             []curriculumHint `yaml:"hints"`
}

type curriculumAnswer struct {
	Type    string `yaml:"type"`
	Value   string `yaml:"value"`
	Working string `yaml:"working"`
}

type curriculumHint struct {
	Level int    `yaml:"level"`
	Text  string `yaml:"text"`
}

func newContent(kind string, title string, body string, metadata map[string]string) CurriculumContent {
	searchTextParts := []string{
		"kind: " + kind,
		"title: " + title,
		"body: " + body,
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := metadata[key]
		if value != "" {
			searchTextParts = append(searchTextParts, key+": "+value)
		}
	}

	return CurriculumContent{
		Kind:       kind,
		Title:      title,
		Body:       body,
		Metadata:   metadata,
		SearchText: strings.Join(searchTextParts, "\n"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmptyStrings(values ...string) []string {
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func joinHints(hints []curriculumHint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint.Text != "" {
			parts = append(parts, fmt.Sprintf("%d: %s", hint.Level, hint.Text))
		}
	}
	return strings.Join(parts, "; ")
}

func joinPrerequisites(prerequisites []prerequisiteRef) string {
	parts := make([]string, 0, len(prerequisites))
	for _, prerequisite := range prerequisites {
		label := firstNonEmpty(prerequisite.ID, prerequisite.NameEN)
		if prerequisite.ID != "" && prerequisite.NameEN != "" {
			label = prerequisite.ID + " " + prerequisite.NameEN
		}
		if label != "" {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, ", ")
}

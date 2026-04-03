package curriculum

// Topic represents a curriculum topic loaded from YAML.
type Topic struct {
	ID                 string              `yaml:"id"`
	OfficialRef        string              `yaml:"official_ref"`
	Name               string              `yaml:"name"`
	SubjectID          string              `yaml:"subject_id"`
	SyllabusID         string              `yaml:"syllabus_id"`
	Difficulty         string              `yaml:"difficulty"`
	Tier               string              `yaml:"tier"`
	LearningObjectives []LearningObjective `yaml:"learning_objectives"`
	Prerequisites      Prerequisites       `yaml:"prerequisites"`
	QualityLevel       int                 `yaml:"quality_level"`
	Provenance         string              `yaml:"provenance"`
}

// LearningObjective represents a learning objective within a topic.
type LearningObjective struct {
	ID    string `yaml:"id"`
	Text  string `yaml:"text"`
	Bloom string `yaml:"bloom"`
}

// Prerequisites holds required and recommended prerequisites.
type Prerequisites struct {
	Required    []string `yaml:"required"`
	Recommended []string `yaml:"recommended"`
}

// Syllabus represents a top-level syllabus (e.g., KSSM Matematik Tingkatan 1).
type Syllabus struct {
	ID       string    `yaml:"id"`
	Name     string    `yaml:"name"`
	Country  string    `yaml:"country"`
	Board    string    `yaml:"board"`
	Level    string    `yaml:"level"`
	Subjects []Subject `yaml:"subjects"`
}

// Subject represents a subject within a syllabus (e.g., Algebra).
type Subject struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	NameEN      string   `yaml:"name_en"`
	SyllabusID  string   `yaml:"syllabus_id"`
	GradeID     string   `yaml:"grade_id"`
	CountryID   string   `yaml:"country_id"`
	Language    string   `yaml:"language"`
	Description string   `yaml:"description"`
	Topics      []string `yaml:"topics"`
}

// Assessment groups quiz questions for a topic.
type Assessment struct {
	TopicID    string               `yaml:"topic_id"`
	Questions  []AssessmentQuestion `yaml:"questions"`
	Provenance string               `yaml:"provenance"`
}

// AssessmentQuestion represents a single assessment item from OSS.
type AssessmentQuestion struct {
	ID                string                 `yaml:"id"`
	Text              string                 `yaml:"text"`
	Difficulty        string                 `yaml:"difficulty"`
	LearningObjective string                 `yaml:"learning_objective"`
	Answer            AssessmentAnswer       `yaml:"answer"`
	Marks             int                    `yaml:"marks"`
	Rubric            []AssessmentRubricItem `yaml:"rubric"`
	Hints             []AssessmentHint       `yaml:"hints"`
	Distractors       []AssessmentDistractor `yaml:"distractors"`
}

// AssessmentAnswer describes the expected answer format.
type AssessmentAnswer struct {
	Type    string `yaml:"type"`
	Value   string `yaml:"value"`
	Working string `yaml:"working"`
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

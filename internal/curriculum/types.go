package curriculum

// Topic represents a curriculum topic loaded from YAML.
type Topic struct {
	ID                 string              `yaml:"id"`
	Name               string              `yaml:"name"`
	SubjectID          string              `yaml:"subject_id"`
	SyllabusID         string              `yaml:"syllabus_id"`
	Difficulty         string              `yaml:"difficulty"`
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
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	TopicIDs []string `yaml:"topic_ids"`
}

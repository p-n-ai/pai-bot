package retrieval

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

var (
	formPattern    = regexp.MustCompile(`(?i)\b(?:tingkatan|form|f)[\s\-_]*([123])\b`)
	headingPattern = regexp.MustCompile(`^#{2,6}\s+`)
)

func SeedCurriculum(service *Service, loader *curriculum.Loader) error {
	if service == nil || loader == nil {
		return nil
	}

	// Step 1: register curriculum as a normal retrieval source.
	// This is the pivot that makes curriculum "one source type" instead of the
	// architectural center of the retrieval module.
	if _, err := service.UpsertSource(UpsertSourceInput{
		ID:    "source:curriculum",
		Type:  "curriculum",
		Title: "Curriculum",
		URI:   "oss://curriculum",
		Metadata: map[string]string{
			"source": "curriculum",
		},
	}); err != nil {
		return fmt.Errorf("upsert curriculum source: %w", err)
	}

	for _, topic := range loader.AllTopics() {
		subject, _ := loader.GetSubject(topic.SubjectID)
		syllabus, _ := loader.GetSyllabus(topic.SyllabusID)
		form := inferTopicForm(topic, subject)

		// Step 2: create or update the collection that scopes this topic's
		// material. Right now we use subject-level collections so search can be
		// narrowed without hard-coding curriculum logic into the service.
		collectionID := topic.SubjectID
		if collectionID == "" {
			collectionID = topic.SyllabusID
		}
		if collectionID == "" {
			collectionID = "curriculum"
		}
		collectionID = "curriculum:" + collectionID

		_, err := service.UpsertCollection(UpsertCollectionInput{
			ID:          collectionID,
			Name:        firstNonEmpty(subject.Name, subject.NameEN, syllabus.Name, "Curriculum"),
			Description: firstNonEmpty(subject.Description, syllabus.Name),
			ParentID:    "",
			Metadata: map[string]string{
				"source":      "curriculum",
				"source_id":   "source:curriculum",
				"source_type": "curriculum",
				"subject_id":  topic.SubjectID,
				"syllabus_id": topic.SyllabusID,
				"form":        form,
			},
		})
		if err != nil {
			return fmt.Errorf("upsert curriculum collection %s: %w", collectionID, err)
		}

		baseMetadata := map[string]string{
			"source":      "curriculum",
			"source_id":   "source:curriculum",
			"source_type": "curriculum",
			"topic_id":    topic.ID,
			"subject_id":  topic.SubjectID,
			"syllabus_id": topic.SyllabusID,
			"form":        form,
		}

		// Step 3: store normalized curriculum documents.
		// We keep topic cards, teaching-note sections, and assessment items as
		// separate searchable records so BM25 can rank the most relevant chunk.
		if _, err := service.UpsertDocument(UpsertDocumentInput{
			ID:           "topic:" + topic.ID,
			CollectionID: collectionID,
			Kind:         "topic_card",
			Title:        topic.Name,
			Body:         joinNonEmpty(topic.OfficialRef, topic.Difficulty, topic.Tier, strings.Join(topicObjectives(topic), "\n")),
			Tags:         topicAliases(topic),
			SourceID:     "source:curriculum",
			SourceType:   "curriculum",
			Metadata:     withKind(baseMetadata, "topic_card"),
			Source:       "curriculum",
		}); err != nil {
			return fmt.Errorf("upsert topic document %s: %w", topic.ID, err)
		}

		if notes, ok := loader.GetTeachingNotes(topic.ID); ok && strings.TrimSpace(notes) != "" {
			for i, section := range splitTeachingNoteSections(notes) {
				if _, err := service.UpsertDocument(UpsertDocumentInput{
					ID:           "note:" + topic.ID + ":" + strconv.Itoa(i),
					CollectionID: collectionID,
					Kind:         "teaching_note",
					Title:        firstNonEmpty(section.Title, topic.Name),
					Body:         section.Body,
					Tags:         topicAliases(topic),
					SourceID:     "source:curriculum",
					SourceType:   "curriculum",
					Metadata:     withKind(baseMetadata, "teaching_note"),
					Source:       "curriculum",
				}); err != nil {
					return fmt.Errorf("upsert note document %s: %w", topic.ID, err)
				}
			}
		}

		if assessment, ok := loader.GetAssessment(topic.ID); ok {
			for i, question := range assessment.Questions {
				if _, err := service.UpsertDocument(UpsertDocumentInput{
					ID:           "assessment:" + topic.ID + ":" + strconv.Itoa(i),
					CollectionID: collectionID,
					Kind:         "assessment_item",
					Title:        question.Text,
					Body:         joinNonEmpty(question.Answer.Working, joinHints(question.Hints), joinDistractors(question.Distractors)),
					Tags:         topicAliases(topic),
					SourceID:     "source:curriculum",
					SourceType:   "curriculum",
					Metadata:     withKind(baseMetadata, "assessment_item"),
					Source:       "curriculum",
				}); err != nil {
					return fmt.Errorf("upsert assessment document %s: %w", topic.ID, err)
				}
			}
		}
	}

	return nil
}

type noteSection struct {
	Title string
	Body  string
}

func splitTeachingNoteSections(markdown string) []noteSection {
	lines := strings.Split(markdown, "\n")
	var out []noteSection
	current := noteSection{Title: "Overview"}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if headingPattern.MatchString(line) {
			if strings.TrimSpace(current.Body) != "" {
				out = append(out, current)
			}
			current = noteSection{Title: headingPattern.ReplaceAllString(line, "")}
			continue
		}
		if current.Body != "" {
			current.Body += "\n"
		}
		current.Body += line
	}
	if strings.TrimSpace(current.Body) != "" {
		out = append(out, current)
	}
	if len(out) == 0 && strings.TrimSpace(markdown) != "" {
		return []noteSection{{Title: "Overview", Body: markdown}}
	}
	return out
}

func inferTopicForm(topic curriculum.Topic, subject curriculum.Subject) string {
	for _, text := range []string{subject.GradeID, subject.Name, topic.SubjectID, topic.SyllabusID} {
		match := formPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(text)))
		if len(match) == 2 {
			return match[1]
		}
	}
	return ""
}

func topicAliases(topic curriculum.Topic) []string {
	return []string{
		topic.Name,
		strings.ReplaceAll(topic.Name, "(", " "),
		strings.ReplaceAll(topic.Name, ")", " "),
	}
}

func topicObjectives(topic curriculum.Topic) []string {
	objectives := make([]string, 0, len(topic.LearningObjectives))
	for _, objective := range topic.LearningObjectives {
		objectives = append(objectives, objective.Text)
	}
	return objectives
}

func withKind(metadata map[string]string, kind string) map[string]string {
	out := cloneMetadata(metadata)
	out["kind"] = kind
	return out
}

func joinNonEmpty(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " ")
}

func firstNonEmpty(parts ...string) string {
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			return strings.TrimSpace(part)
		}
	}
	return ""
}

func joinHints(hints []curriculum.AssessmentHint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		parts = append(parts, hint.Text)
	}
	return strings.Join(parts, " ")
}

func joinDistractors(distractors []curriculum.AssessmentDistractor) string {
	parts := make([]string, 0, len(distractors))
	for _, distractor := range distractors {
		parts = append(parts, distractor.Value, distractor.Feedback)
	}
	return strings.Join(parts, " ")
}

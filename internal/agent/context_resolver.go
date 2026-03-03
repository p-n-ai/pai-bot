package agent

import (
	"log/slog"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

// ContextResolver resolves curriculum context for a user message.
// It returns a matched topic and optional teaching notes.
// Returning nil topic and empty notes means "no curriculum match".
type ContextResolver interface {
	Resolve(text string) (*curriculum.Topic, string)
}

// NoopContextResolver is a safe default that never injects curriculum context.
type NoopContextResolver struct{}

// Resolve always returns no match.
func (NoopContextResolver) Resolve(string) (*curriculum.Topic, string) {
	return nil, ""
}

// CurriculumContextResolver resolves topic context from loaded curriculum content.
type CurriculumContextResolver struct {
	loader *curriculum.Loader
	topics []curriculum.Topic
}

// NewCurriculumContextResolver builds a resolver from a curriculum loader.
func NewCurriculumContextResolver(loader *curriculum.Loader) *CurriculumContextResolver {
	if loader == nil {
		return &CurriculumContextResolver{}
	}
	return &CurriculumContextResolver{
		loader: loader,
		topics: loader.AllTopics(),
	}
}

// Resolve matches topic from text and returns teaching notes when available.
func (r *CurriculumContextResolver) Resolve(text string) (*curriculum.Topic, string) {
	if r == nil || r.loader == nil || len(r.topics) == 0 {
		return nil, ""
	}

	topicID, ok := DetectTopic(text, r.topics)
	if !ok {
		return nil, ""
	}

	for i := range r.topics {
		if r.topics[i].ID != topicID {
			continue
		}

		topic := r.topics[i]
		notes, hasNotes := r.loader.GetTeachingNotes(topic.ID)

		slog.Info("curriculum topic matched",
			"topic_id", topic.ID,
			"topic_name", topic.Name,
			"has_teaching_notes", hasNotes,
		)

		if !hasNotes {
			notes = ""
		}
		return &topic, notes
	}

	return nil, ""
}

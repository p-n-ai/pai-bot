package agent

import (
	"log/slog"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

// ContextResolver resolves curriculum context for a user message.
// It returns a matched topic and optional teaching notes.
// Returning nil topic and empty notes means "no curriculum match".
type ContextResolver interface {
	Resolve(text string) (*curriculum.Topic, string)
}

// ContextQuery captures retrieval-time priors without exposing retriever internals.
type ContextQuery struct {
	Text                string
	UserID              string
	ConversationTopicID string
}

// ContextQueryResolver is an optional richer resolver interface for callers that
// can supply metadata such as user form and active topic.
type ContextQueryResolver interface {
	ResolveQuery(query ContextQuery) (*curriculum.Topic, string)
}

// NoopContextResolver is a safe default that never injects curriculum context.
type NoopContextResolver struct{}

// Resolve always returns no match.
func (NoopContextResolver) Resolve(string) (*curriculum.Topic, string) {
	return nil, ""
}

// CurriculumContextResolver resolves topic context from loaded curriculum content.
type CurriculumContextResolver struct {
	retriever *curriculumRetriever
}

type CurriculumContextResolverOption func(*curriculumRetrieverConfig)

func WithResolverStore(store ConversationStore) CurriculumContextResolverOption {
	return func(cfg *curriculumRetrieverConfig) {
		cfg.store = store
	}
}

func WithResolverRetrievalService(service *retrieval.Service) CurriculumContextResolverOption {
	return func(cfg *curriculumRetrieverConfig) {
		cfg.service = service
	}
}

func WithResolverTracker(tracker progress.Tracker) CurriculumContextResolverOption {
	return func(cfg *curriculumRetrieverConfig) {
		cfg.tracker = tracker
	}
}

func WithResolverPrereqGraph(graph *curriculum.PrereqGraph) CurriculumContextResolverOption {
	return func(cfg *curriculumRetrieverConfig) {
		cfg.prereqGraph = graph
	}
}

// NewCurriculumContextResolver builds a resolver from a curriculum loader.
func NewCurriculumContextResolver(loader *curriculum.Loader, options ...CurriculumContextResolverOption) *CurriculumContextResolver {
	if loader == nil {
		return &CurriculumContextResolver{}
	}

	cfg := curriculumRetrieverConfig{}
	for _, option := range options {
		option(&cfg)
	}

	return &CurriculumContextResolver{retriever: newCurriculumRetriever(loader, cfg)}
}

// Resolve matches topic from text and returns teaching notes when available.
func (r *CurriculumContextResolver) Resolve(text string) (*curriculum.Topic, string) {
	return r.ResolveQuery(ContextQuery{Text: text})
}

// ResolveQuery matches topic from text using lexical retrieval plus metadata priors.
func (r *CurriculumContextResolver) ResolveQuery(query ContextQuery) (*curriculum.Topic, string) {
	if r == nil || r.retriever == nil {
		return nil, ""
	}

	// Agent-side flow:
	//  1. pass the raw user text plus priors (user, active topic)
	//  2. let the retriever decide whether there is a safe curriculum match
	//  3. if no safe match exists, inject nothing into the tutor prompt
	result := r.retriever.Resolve(query)
	if result.Topic == nil {
		return nil, ""
	}
	slog.Info("curriculum topic matched",
		"topic_id", result.Topic.ID,
		"topic_name", result.Topic.Name,
		"score", result.Score,
		"confidence", result.Confidence,
		"note_chars", len(result.Notes),
	)
	return result.Topic, result.Notes
}

func resolveContext(resolver ContextResolver, query ContextQuery) (*curriculum.Topic, string) {
	if richer, ok := resolver.(ContextQueryResolver); ok {
		return richer.ResolveQuery(query)
	}
	return resolver.Resolve(query.Text)
}

func (e *Engine) resolveCurriculumContext(userID, conversationTopicID, text string) (*curriculum.Topic, string) {
	return resolveContext(e.contextResolver, ContextQuery{
		Text:                text,
		UserID:              userID,
		ConversationTopicID: conversationTopicID,
	})
}

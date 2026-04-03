package retrieval_test

import (
	"errors"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

func TestMemoryService_DocumentCRUDAndSearch(t *testing.T) {
	// Happy path:
	// 1. create a collection
	// 2. store one document
	// 3. verify BM25 search can retrieve it
	// 4. verify list/filter paths see the same record
	service := retrieval.NewService()

	collection, err := service.UpsertCollection(retrieval.UpsertCollectionInput{
		ID:   "math-f1",
		Name: "Mathematics Form 1",
		Metadata: map[string]string{
			"form": "1",
		},
	})
	if err != nil {
		t.Fatalf("UpsertCollection() error = %v", err)
	}
	if collection.ID != "math-f1" {
		t.Fatalf("collection.ID = %q, want math-f1", collection.ID)
	}

	document, err := service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "topic:F1-02",
		CollectionID: "math-f1",
		Kind:         "topic_card",
		Title:        "Linear Equations",
		Body:         "Solve linear equations in one variable.",
		Tags:         []string{"algebra", "equations"},
		Metadata: map[string]string{
			"topic_id": "F1-02",
			"form":     "1",
		},
	})
	if err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}
	if document.ID != "topic:F1-02" {
		t.Fatalf("document.ID = %q, want topic:F1-02", document.ID)
	}

	hits, err := service.Search(retrieval.SearchRequest{
		Query: "teach me linear equations",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("len(hits) = %d, want 1", len(hits))
	}
	if hits[0].Document.ID != "topic:F1-02" {
		t.Fatalf("hits[0].Document.ID = %q, want topic:F1-02", hits[0].Document.ID)
	}

	listed := service.ListDocuments(retrieval.ListDocumentsOptions{
		Metadata: map[string]string{"topic_id": "F1-02"},
	})
	if len(listed) != 1 {
		t.Fatalf("len(listed) = %d, want 1", len(listed))
	}
}

func TestMemoryService_SearchRespectsActivation(t *testing.T) {
	// Visibility guard:
	// inactive documents and documents inside inactive collections must disappear
	// from search results without requiring callers to remember extra checks.
	service := retrieval.NewService()

	_, _ = service.UpsertCollection(retrieval.UpsertCollectionInput{ID: "g1", Name: "Collection 1"})
	_, _ = service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "c1",
		CollectionID: "g1",
		Kind:         "note",
		Title:        "Balance Method",
		Body:         "Subtract on both sides.",
	})

	if _, err := service.SetDocumentActive("c1", false); err != nil {
		t.Fatalf("SetDocumentActive() error = %v", err)
	}
	hits, err := service.Search(retrieval.SearchRequest{Query: "balance method"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("len(hits) = %d, want 0 when document inactive", len(hits))
	}

	if _, err := service.SetDocumentActive("c1", true); err != nil {
		t.Fatalf("SetDocumentActive() error = %v", err)
	}
	if _, err := service.SetCollectionActive("g1", false); err != nil {
		t.Fatalf("SetCollectionActive() error = %v", err)
	}
	hits, err = service.Search(retrieval.SearchRequest{Query: "balance method"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("len(hits) = %d, want 0 when collection inactive", len(hits))
	}
}

func TestMemoryService_SearchSupportsMetadataFilters(t *testing.T) {
	// Metadata filters are part of the contract, not post-processing sugar.
	// This test proves the service can narrow lexical results to a specific topic.
	service := retrieval.NewService()

	_, _ = service.UpsertCollection(retrieval.UpsertCollectionInput{ID: "math-f1", Name: "Math F1"})
	_, _ = service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "f1-linear",
		CollectionID: "math-f1",
		Kind:         "topic_card",
		Title:        "Linear Equations",
		Body:         "Solve linear equations in one variable.",
		Metadata: map[string]string{
			"form":  "1",
			"topic": "linear",
		},
	})
	_, _ = service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "f1-fraction",
		CollectionID: "math-f1",
		Kind:         "topic_card",
		Title:        "Fractions",
		Body:         "Add and simplify fractions.",
		Metadata: map[string]string{
			"form":  "1",
			"topic": "fractions",
		},
	})

	hits, err := service.Search(retrieval.SearchRequest{
		Query:    "equations",
		Metadata: map[string]string{"topic": "linear"},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 1 || hits[0].Document.ID != "f1-linear" {
		t.Fatalf("hits = %#v, want only f1-linear", hits)
	}
}

func TestMemoryService_SearchSkipsInactiveMetadataMatch(t *testing.T) {
	// Edge case:
	// the inactive document is intentionally the stronger lexical match.
	// Search must still return the weaker active record unless the caller opts
	// into include_inactive.
	service := retrieval.NewService()

	_, _ = service.UpsertCollection(retrieval.UpsertCollectionInput{ID: "math-f1", Name: "Math F1"})
	_, _ = service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "linear-active",
		CollectionID: "math-f1",
		Kind:         "teaching_note",
		Title:        "Linear Equations",
		Body:         "Use balance method to isolate x.",
		Metadata: map[string]string{
			"topic_id": "F1-02",
			"form":     "1",
		},
	})
	_, _ = service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "linear-inactive",
		CollectionID: "math-f1",
		Kind:         "teaching_note",
		Title:        "Linear Equations Balance Method",
		Body:         "Balance method balance method balance method. Subtract on both sides.",
		Metadata: map[string]string{
			"topic_id": "F1-02",
			"form":     "1",
		},
	})

	if _, err := service.SetDocumentActive("linear-inactive", false); err != nil {
		t.Fatalf("SetDocumentActive() error = %v", err)
	}

	hits, err := service.Search(retrieval.SearchRequest{
		Query:    "balance method subtract both sides",
		Metadata: map[string]string{"topic_id": "F1-02"},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("len(hits) = %d, want 1 active hit", len(hits))
	}
	if hits[0].Document.ID != "linear-active" {
		t.Fatalf("hits[0].Document.ID = %q, want linear-active", hits[0].Document.ID)
	}

	hits, err = service.Search(retrieval.SearchRequest{
		Query:           "balance method subtract both sides",
		Metadata:        map[string]string{"topic_id": "F1-02"},
		IncludeInactive: true,
	})
	if err != nil {
		t.Fatalf("Search() with inactive error = %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("len(hits) = %d, want 2 with inactive included", len(hits))
	}
}

func TestMemoryService_DeleteCollectionRejectsNonEmptyCollection(t *testing.T) {
	// Safety rule:
	// collection deletion is explicit and conservative. If documents still live
	// under the collection, reject the delete instead of cascading.
	service := retrieval.NewService()

	_, _ = service.UpsertCollection(retrieval.UpsertCollectionInput{ID: "math-f1", Name: "Math F1"})
	_, _ = service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "linear-active",
		CollectionID: "math-f1",
		Kind:         "teaching_note",
		Title:        "Linear Equations",
		Body:         "Use balance method to isolate x.",
	})

	err := service.DeleteCollection("math-f1")
	if err == nil {
		t.Fatal("DeleteCollection() error = nil, want invalid argument")
	}
	if !errors.Is(err, retrieval.ErrInvalidArgument) {
		t.Fatalf("DeleteCollection() error = %v, want ErrInvalidArgument", err)
	}

	if _, err := service.GetCollection("math-f1"); err != nil {
		t.Fatalf("GetCollection() after failed delete error = %v", err)
	}
	if _, err := service.GetDocument("linear-active"); err != nil {
		t.Fatalf("GetDocument() after failed delete error = %v", err)
	}
}

func TestMemoryService_SourceCRUDAndSearchFilters(t *testing.T) {
	// Generic-retrieval path:
	// 1. register a non-curriculum source
	// 2. attach a document to it
	// 3. filter search by source type
	// 4. ensure deactivating the source hides its documents
	// 5. ensure source deletion is blocked while documents still reference it
	service := retrieval.NewService()

	source, err := service.UpsertSource(retrieval.UpsertSourceInput{
		ID:    "source:youtube:demo",
		Type:  "youtube",
		Title: "Quadratic Seminar",
		URI:   "https://youtube.example/watch?v=demo",
	})
	if err != nil {
		t.Fatalf("UpsertSource() error = %v", err)
	}
	if source.ID != "source:youtube:demo" {
		t.Fatalf("source.ID = %q, want source:youtube:demo", source.ID)
	}

	_, _ = service.UpsertCollection(retrieval.UpsertCollectionInput{ID: "seminars", Name: "Seminars"})
	_, _ = service.UpsertDocument(retrieval.UpsertDocumentInput{
		ID:           "doc:youtube:demo",
		CollectionID: "seminars",
		Kind:         "transcript_chunk",
		Title:        "Quadratic Equations Seminar",
		Body:         "Complete the square and solve quadratic equations.",
		SourceID:     "source:youtube:demo",
		SourceType:   "youtube",
		Metadata: map[string]string{
			"topic": "quadratic",
		},
	})

	sources, err := service.ListSources(retrieval.ListSourcesRequest{Types: []string{"youtube"}})
	if err != nil {
		t.Fatalf("ListSources() error = %v", err)
	}
	if len(sources) != 1 || sources[0].ID != "source:youtube:demo" {
		t.Fatalf("sources = %#v, want only source:youtube:demo", sources)
	}

	hits, err := service.Search(retrieval.SearchRequest{
		Query:       "complete the square",
		SourceTypes: []string{"youtube"},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 1 || hits[0].Document.ID != "doc:youtube:demo" {
		t.Fatalf("hits = %#v, want only doc:youtube:demo", hits)
	}

	if _, err := service.SetSourceActive("source:youtube:demo", false); err != nil {
		t.Fatalf("SetSourceActive() error = %v", err)
	}
	hits, err = service.Search(retrieval.SearchRequest{
		Query:       "complete the square",
		SourceTypes: []string{"youtube"},
	})
	if err != nil {
		t.Fatalf("Search() after source deactivate error = %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("len(hits) = %d, want 0 when source inactive", len(hits))
	}

	err = service.DeleteSource("source:youtube:demo")
	if err == nil {
		t.Fatal("DeleteSource() error = nil, want invalid argument")
	}
	if !errors.Is(err, retrieval.ErrInvalidArgument) {
		t.Fatalf("DeleteSource() error = %v, want ErrInvalidArgument", err)
	}
}

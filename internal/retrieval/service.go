package retrieval

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

var (
	ErrNotFound        = errors.New("retrieval item not found")
	ErrInvalidArgument = errors.New("retrieval invalid argument")
	searchFieldWeights = map[string]float64{
		"title":      3.2,
		"tags":       2.4,
		"collection": 1.4,
		"metadata":   1.2,
		"body":       1.0,
	}
	searchStopWords = map[string]struct{}{
		"and": {}, "the": {}, "for": {}, "with": {}, "that": {}, "this": {}, "what": {}, "how": {}, "from": {},
		"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "am": {}, "do": {}, "did": {}, "does": {},
		"you": {}, "your": {}, "me": {}, "my": {}, "we": {}, "our": {}, "they": {}, "their": {}, "a": {},
		"an": {}, "to": {}, "of": {}, "in": {}, "on": {}, "at": {}, "by": {}, "or": {}, "if": {}, "it": {},
	}
	searchStemOverrides = map[string]string{
		"equation": "equation", "equations": "equation",
		"inequality": "inequality", "inequalities": "inequality",
		"variable": "variable", "variables": "variable",
		"expression": "expression", "expressions": "expression",
		"graphs": "graph", "lines": "line",
	}
)

type Collection struct {
	ID          string            `json:"id"`
	ParentID    string            `json:"parent_id,omitempty"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Active      bool              `json:"active"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type Document struct {
	ID           string            `json:"id"`
	CollectionID string            `json:"collection_id,omitempty"`
	Kind         string            `json:"kind"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	Tags         []string          `json:"tags,omitempty"`
	SourceID     string            `json:"source_id,omitempty"`
	SourceType   string            `json:"source_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Source       string            `json:"source,omitempty"`
	Active       bool              `json:"active"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type CreateCollectionRequest struct {
	ID          string            `json:"id"`
	ParentID    string            `json:"parent_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Active      *bool             `json:"active,omitempty"`
}

type UpdateCollectionRequest struct {
	ParentID    *string           `json:"parent_id,omitempty"`
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Active      *bool             `json:"active,omitempty"`
}

type UpsertCollectionInput = CreateCollectionRequest

type ListCollectionsOptions struct {
	ParentID        string
	IncludeInactive bool
}

type ListCollectionsRequest = ListCollectionsOptions

type CreateDocumentRequest struct {
	ID           string            `json:"id"`
	CollectionID string            `json:"collection_id"`
	Kind         string            `json:"kind"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	Tags         []string          `json:"tags"`
	SourceID     string            `json:"source_id,omitempty"`
	SourceType   string            `json:"source_type,omitempty"`
	Metadata     map[string]string `json:"metadata"`
	Source       string            `json:"source"`
	Active       *bool             `json:"active,omitempty"`
}

type UpdateDocumentRequest struct {
	CollectionID *string           `json:"collection_id,omitempty"`
	Kind         *string           `json:"kind,omitempty"`
	Title        *string           `json:"title,omitempty"`
	Body         *string           `json:"body,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	SourceID     *string           `json:"source_id,omitempty"`
	SourceType   *string           `json:"source_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Source       *string           `json:"source,omitempty"`
	Active       *bool             `json:"active,omitempty"`
}

type UpsertDocumentInput = CreateDocumentRequest

type ListDocumentsOptions struct {
	CollectionID    string
	IncludeInactive bool
	Metadata        map[string]string
}

type ListDocumentsRequest struct {
	CollectionIDs   []string          `json:"collection_ids,omitempty"`
	Kinds           []string          `json:"kinds,omitempty"`
	SourceIDs       []string          `json:"source_ids,omitempty"`
	SourceTypes     []string          `json:"source_types,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	IncludeInactive bool              `json:"include_inactive,omitempty"`
}

type SearchRequest struct {
	Query           string            `json:"query"`
	Limit           int               `json:"limit,omitempty"`
	CollectionID    string            `json:"collection_id,omitempty"`
	CollectionIDs   []string          `json:"collection_ids,omitempty"`
	Kinds           []string          `json:"kinds,omitempty"`
	SourceIDs       []string          `json:"source_ids,omitempty"`
	SourceTypes     []string          `json:"source_types,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	IncludeInactive bool              `json:"include_inactive,omitempty"`
}

type SearchResult struct {
	Document        Document `json:"document"`
	Score           float64  `json:"score"`
	MatchedTerms    int      `json:"matched_terms"`
	HighSignalTerms int      `json:"high_signal_terms"`
	Excerpt         string   `json:"excerpt"`
}

type SearchHit = SearchResult

type Service struct {
	mu          sync.RWMutex
	sources     map[string]Source
	collections map[string]Collection
	documents   map[string]Document
	docs        []indexedDoc
	docFreq     map[string]int
	avgFieldLen map[string]float64
}

type indexedDoc struct {
	documentID string
	terms      map[string]map[string]int
	lengths    map[string]int
}

func NewService() *Service {
	return &Service{
		sources:     make(map[string]Source),
		collections: make(map[string]Collection),
		documents:   make(map[string]Document),
		docFreq:     make(map[string]int),
		avgFieldLen: make(map[string]float64),
	}
}

// NewMemoryService returns the default in-process retrieval store.
// It keeps sources, collections, documents, and the BM25 index in memory, so:
//  1. reads and writes are local map operations
//  2. the index is rebuilt after each mutation
//  3. data disappears on process restart
//
// This is the current default implementation; a future persistent store should
// keep the same service surface.
func NewMemoryService() *Service {
	return NewService()
}

func (s *Service) CreateCollection(req CreateCollectionRequest) (Collection, error) {
	return s.UpsertCollection(req)
}

func (s *Service) UpsertCollection(req UpsertCollectionInput) (Collection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(req.Name) == "" {
		return Collection{}, fmt.Errorf("%w: collection name is required", ErrInvalidArgument)
	}
	if req.ParentID != "" {
		if _, ok := s.collections[req.ParentID]; !ok {
			return Collection{}, fmt.Errorf("%w: unknown parent collection %q", ErrInvalidArgument, req.ParentID)
		}
	}

	now := time.Now()
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = fmt.Sprintf("collection-%d", now.UnixNano())
	}
	collection, exists := s.collections[id]
	if !exists {
		collection = Collection{ID: id, CreatedAt: now}
	}
	collection.ParentID = strings.TrimSpace(req.ParentID)
	collection.Name = strings.TrimSpace(req.Name)
	collection.Description = strings.TrimSpace(req.Description)
	collection.Metadata = cloneMetadata(req.Metadata)
	collection.UpdatedAt = now
	if req.Active != nil {
		collection.Active = *req.Active
	} else if !exists {
		collection.Active = true
	}
	s.collections[id] = collection
	s.rebuildIndexLocked()
	return cloneCollection(collection), nil
}

func (s *Service) UpdateCollection(id string, req UpdateCollectionRequest) (Collection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection, ok := s.collections[id]
	if !ok {
		return Collection{}, ErrNotFound
	}
	if req.ParentID != nil {
		collection.ParentID = strings.TrimSpace(*req.ParentID)
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return Collection{}, fmt.Errorf("%w: collection name is required", ErrInvalidArgument)
		}
		collection.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		collection.Description = strings.TrimSpace(*req.Description)
	}
	if req.Metadata != nil {
		collection.Metadata = cloneMetadata(req.Metadata)
	}
	if req.Active != nil {
		collection.Active = *req.Active
	}
	collection.UpdatedAt = time.Now()
	s.collections[id] = collection
	s.rebuildIndexLocked()
	return cloneCollection(collection), nil
}

func (s *Service) DeleteCollection(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.collections[id]; !ok {
		return ErrNotFound
	}
	for _, collection := range s.collections {
		if collection.ParentID == id {
			return fmt.Errorf("%w: collection %q has child collections", ErrInvalidArgument, id)
		}
	}
	for _, document := range s.documents {
		if document.CollectionID == id {
			return fmt.Errorf("%w: collection %q is not empty", ErrInvalidArgument, id)
		}
	}
	delete(s.collections, id)
	s.rebuildIndexLocked()
	return nil
}

func (s *Service) GetCollection(id string) (Collection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collection, ok := s.collections[id]
	if !ok {
		return Collection{}, ErrNotFound
	}
	return cloneCollection(collection), nil
}

func (s *Service) ListCollections(opts ListCollectionsOptions) []Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Collection, 0, len(s.collections))
	for _, collection := range s.collections {
		if opts.ParentID != "" && collection.ParentID != opts.ParentID {
			continue
		}
		if !opts.IncludeInactive && !collection.Active {
			continue
		}
		out = append(out, cloneCollection(collection))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].ID < out[j].ID
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) SetCollectionActive(id string, active bool) (Collection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection, ok := s.collections[id]
	if !ok {
		return Collection{}, ErrNotFound
	}
	collection.Active = active
	collection.UpdatedAt = time.Now()
	s.collections[id] = collection
	s.rebuildIndexLocked()
	return cloneCollection(collection), nil
}

func (s *Service) CreateDocument(req CreateDocumentRequest) (Document, error) {
	return s.UpsertDocument(req)
}

func (s *Service) UpsertDocument(req UpsertDocumentInput) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Step 2 in the generic model: once the caller knows the source and
	// collection, it writes a normalized document here. The service owns the
	// indexing details and rebuilds the BM25 state after each mutation.
	if strings.TrimSpace(req.Kind) == "" || strings.TrimSpace(req.Title) == "" {
		return Document{}, fmt.Errorf("%w: document kind and title are required", ErrInvalidArgument)
	}
	if req.CollectionID != "" {
		if _, ok := s.collections[req.CollectionID]; !ok {
			return Document{}, fmt.Errorf("%w: unknown collection %q", ErrInvalidArgument, req.CollectionID)
		}
	}

	now := time.Now()
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = fmt.Sprintf("document-%d", now.UnixNano())
	}
	document, exists := s.documents[id]
	if !exists {
		document = Document{ID: id, CreatedAt: now}
	}
	document.CollectionID = strings.TrimSpace(req.CollectionID)
	document.Kind = strings.TrimSpace(req.Kind)
	document.Title = strings.TrimSpace(req.Title)
	document.Body = strings.TrimSpace(req.Body)
	document.Tags = cloneStrings(req.Tags)
	document.SourceID = strings.TrimSpace(req.SourceID)
	document.SourceType = strings.TrimSpace(req.SourceType)
	document.Metadata = cloneMetadata(req.Metadata)
	document.Source = strings.TrimSpace(req.Source)
	document.UpdatedAt = now
	if req.Active != nil {
		document.Active = *req.Active
	} else if !exists {
		document.Active = true
	}
	s.documents[id] = document
	s.rebuildIndexLocked()
	return cloneDocument(document), nil
}

func (s *Service) UpdateDocument(id string, req UpdateDocumentRequest) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	document, ok := s.documents[id]
	if !ok {
		return Document{}, ErrNotFound
	}
	if req.CollectionID != nil {
		if *req.CollectionID != "" {
			if _, ok := s.collections[*req.CollectionID]; !ok {
				return Document{}, fmt.Errorf("%w: unknown collection %q", ErrInvalidArgument, *req.CollectionID)
			}
		}
		document.CollectionID = strings.TrimSpace(*req.CollectionID)
	}
	if req.Kind != nil {
		document.Kind = strings.TrimSpace(*req.Kind)
	}
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return Document{}, fmt.Errorf("%w: document title is required", ErrInvalidArgument)
		}
		document.Title = strings.TrimSpace(*req.Title)
	}
	if req.Body != nil {
		document.Body = strings.TrimSpace(*req.Body)
	}
	if req.Tags != nil {
		document.Tags = cloneStrings(req.Tags)
	}
	if req.SourceID != nil {
		document.SourceID = strings.TrimSpace(*req.SourceID)
	}
	if req.SourceType != nil {
		document.SourceType = strings.TrimSpace(*req.SourceType)
	}
	if req.Metadata != nil {
		document.Metadata = cloneMetadata(req.Metadata)
	}
	if req.Source != nil {
		document.Source = strings.TrimSpace(*req.Source)
	}
	if req.Active != nil {
		document.Active = *req.Active
	}
	document.UpdatedAt = time.Now()
	s.documents[id] = document
	s.rebuildIndexLocked()
	return cloneDocument(document), nil
}

func (s *Service) GetDocument(id string) (Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	document, ok := s.documents[id]
	if !ok {
		return Document{}, ErrNotFound
	}
	return cloneDocument(document), nil
}

func (s *Service) ListDocuments(opts ListDocumentsOptions) []Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Document, 0, len(s.documents))
	for _, document := range s.documents {
		if opts.CollectionID != "" && document.CollectionID != opts.CollectionID {
			continue
		}
		if !opts.IncludeInactive && !document.Active {
			continue
		}
		if !matchesMetadata(document.Metadata, opts.Metadata) {
			continue
		}
		if document.CollectionID != "" && !opts.IncludeInactive {
			collection, ok := s.collections[document.CollectionID]
			if ok && !collection.Active {
				continue
			}
		}
		if document.SourceID != "" && !opts.IncludeInactive {
			source, ok := s.sources[document.SourceID]
			if ok && !source.Active {
				continue
			}
		}
		out = append(out, cloneDocument(document))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title == out[j].Title {
			return out[i].ID < out[j].ID
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func (s *Service) ListDocument(req ListDocumentsRequest) ([]Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collectionIDs := makeSet(req.CollectionIDs)
	kinds := makeSet(req.Kinds)
	sourceIDs := makeSet(req.SourceIDs)
	sourceTypes := makeSet(req.SourceTypes)
	out := make([]Document, 0, len(s.documents))
	for _, document := range s.documents {
		if len(collectionIDs) > 0 {
			if _, ok := collectionIDs[document.CollectionID]; !ok {
				continue
			}
		}
		if len(kinds) > 0 {
			if _, ok := kinds[document.Kind]; !ok {
				continue
			}
		}
		if len(sourceIDs) > 0 {
			if _, ok := sourceIDs[document.SourceID]; !ok {
				continue
			}
		}
		if len(sourceTypes) > 0 {
			if _, ok := sourceTypes[document.SourceType]; !ok {
				continue
			}
		}
		if !req.IncludeInactive && !document.Active {
			continue
		}
		if !matchesMetadata(document.Metadata, req.Metadata) {
			continue
		}
		if document.CollectionID != "" && !req.IncludeInactive {
			collection, ok := s.collections[document.CollectionID]
			if ok && !collection.Active {
				continue
			}
		}
		if document.SourceID != "" && !req.IncludeInactive {
			source, ok := s.sources[document.SourceID]
			if ok && !source.Active {
				continue
			}
		}
		out = append(out, cloneDocument(document))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title == out[j].Title {
			return out[i].ID < out[j].ID
		}
		return out[i].Title < out[j].Title
	})
	return out, nil
}

func (s *Service) DeleteDocument(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[id]; !ok {
		return ErrNotFound
	}
	delete(s.documents, id)
	s.rebuildIndexLocked()
	return nil
}

func (s *Service) SetDocumentActive(id string, active bool) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	document, ok := s.documents[id]
	if !ok {
		return Document{}, ErrNotFound
	}
	document.Active = active
	document.UpdatedAt = time.Now()
	s.documents[id] = document
	s.rebuildIndexLocked()
	return cloneDocument(document), nil
}

func (s *Service) Search(req SearchRequest) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Search flow:
	//  1. tokenize the query
	//  2. apply deterministic filters first (collection, kind, source, metadata)
	//  3. drop inactive documents, inactive collections, and inactive sources
	//  4. score surviving docs with BM25
	//  5. sort and trim to the caller's limit
	terms := tokenizeSearchText(req.Query)
	if len(terms) == 0 {
		return nil, nil
	}
	collectionIDs := makeSet(req.CollectionIDs)
	if req.CollectionID != "" {
		collectionIDs[req.CollectionID] = struct{}{}
	}
	kinds := makeSet(req.Kinds)
	sourceIDs := makeSet(req.SourceIDs)
	sourceTypes := makeSet(req.SourceTypes)
	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	var results []SearchResult
	for _, doc := range s.docs {
		document, ok := s.documents[doc.documentID]
		if !ok {
			continue
		}
		if !req.IncludeInactive && !document.Active {
			continue
		}
		if len(collectionIDs) > 0 {
			if _, ok := collectionIDs[document.CollectionID]; !ok {
				continue
			}
		}
		if len(kinds) > 0 {
			if _, ok := kinds[document.Kind]; !ok {
				continue
			}
		}
		if len(sourceIDs) > 0 {
			if _, ok := sourceIDs[document.SourceID]; !ok {
				continue
			}
		}
		if len(sourceTypes) > 0 {
			if _, ok := sourceTypes[document.SourceType]; !ok {
				continue
			}
		}
		if !matchesMetadata(document.Metadata, req.Metadata) {
			continue
		}
		if document.CollectionID != "" && !req.IncludeInactive {
			collection, ok := s.collections[document.CollectionID]
			if ok && !collection.Active {
				continue
			}
		}
		if document.SourceID != "" && !req.IncludeInactive {
			source, ok := s.sources[document.SourceID]
			if ok && !source.Active {
				continue
			}
		}

		score, matchedTerms, highSignalTerms := s.bm25Score(doc, terms)
		if score <= 0 {
			continue
		}
		results = append(results, SearchResult{
			Document:        cloneDocument(document),
			Score:           score,
			MatchedTerms:    matchedTerms,
			HighSignalTerms: highSignalTerms,
			Excerpt:         excerptForDocument(document),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Document.ID < results[j].Document.ID
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *Service) rebuildIndexLocked() {
	s.docs = nil
	s.docFreq = make(map[string]int)
	s.avgFieldLen = make(map[string]float64)
	fieldTotals := make(map[string]int)

	// Index flow:
	//  1. flatten each stored document into weighted search fields
	//  2. build per-field term frequencies
	//  3. update document frequencies
	//  4. cache average field lengths for BM25 normalization
	for _, document := range s.documents {
		collectionName := ""
		if collection, ok := s.collections[document.CollectionID]; ok {
			collectionName = collection.Name
		}
		sourceTitle := ""
		if source, ok := s.sources[document.SourceID]; ok {
			sourceTitle = source.Title
		}
		fields := map[string]string{
			"title":      document.Title,
			"tags":       strings.Join(document.Tags, " "),
			"collection": collectionName,
			"metadata":   flattenMetadata(document.Metadata) + " " + document.SourceID + " " + document.SourceType + " " + sourceTitle,
			"body":       document.Body,
		}
		doc := indexedDoc{
			documentID: document.ID,
			terms:      make(map[string]map[string]int),
			lengths:    make(map[string]int),
		}
		seen := map[string]struct{}{}
		for field, text := range fields {
			tokens := tokenizeSearchText(text)
			if len(tokens) == 0 {
				continue
			}
			tf := make(map[string]int, len(tokens))
			for _, token := range tokens {
				tf[token]++
				seen[token] = struct{}{}
			}
			doc.terms[field] = tf
			doc.lengths[field] = len(tokens)
			fieldTotals[field] += len(tokens)
		}
		if len(doc.terms) == 0 {
			continue
		}
		for term := range seen {
			s.docFreq[term]++
		}
		s.docs = append(s.docs, doc)
	}

	if len(s.docs) == 0 {
		return
	}
	for field, total := range fieldTotals {
		s.avgFieldLen[field] = float64(total) / float64(len(s.docs))
	}
}

func (s *Service) bm25Score(doc indexedDoc, terms []string) (float64, int, int) {
	score := 0.0
	matchedTerms := 0
	highSignalTerms := 0
	for _, term := range terms {
		df := s.docFreq[term]
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (float64(len(s.docs)-df)+0.5)/(float64(df)+0.5))
		termMatched := false
		termHighSignal := false
		for field, weight := range searchFieldWeights {
			tf := doc.terms[field][term]
			if tf == 0 {
				continue
			}
			termMatched = true
			if field == "title" || field == "tags" || field == "collection" {
				termHighSignal = true
			}
			fieldLen := float64(doc.lengths[field])
			avgLen := s.avgFieldLen[field]
			if avgLen <= 0 {
				avgLen = 1
			}
			norm := float64(tf) + bm25K1*(1-bm25B+bm25B*(fieldLen/avgLen))
			score += weight * idf * ((float64(tf) * (bm25K1 + 1)) / norm)
		}
		if termMatched {
			matchedTerms++
		}
		if termHighSignal {
			highSignalTerms++
		}
	}
	return score, matchedTerms, highSignalTerms
}

func tokenizeSearchText(text string) []string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return nil
	}
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return ' '
	}, lower)
	raw := strings.Fields(clean)
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, token := range raw {
		if len(token) < 2 {
			continue
		}
		token = normalizeSearchToken(token)
		if _, ok := searchStopWords[token]; ok || token == "" {
			continue
		}
		out = append(out, token)
	}
	return out
}

func normalizeSearchToken(token string) string {
	if stem, ok := searchStemOverrides[token]; ok {
		return stem
	}
	switch {
	case strings.HasSuffix(token, "ies") && len(token) > 4:
		return token[:len(token)-3] + "y"
	case strings.HasSuffix(token, "es") && len(token) > 4:
		return token[:len(token)-2]
	case strings.HasSuffix(token, "s") && len(token) > 4:
		return token[:len(token)-1]
	default:
		return token
	}
}

func flattenMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		parts = append(parts, key, metadata[key])
	}
	return strings.Join(parts, " ")
}

func excerptForDocument(document Document) string {
	text := strings.TrimSpace(document.Body)
	if text == "" {
		text = strings.TrimSpace(document.Title)
	}
	if len(text) <= 240 {
		return text
	}
	return text[:240] + "..."
}

func matchesMetadata(have, want map[string]string) bool {
	if len(want) == 0 {
		return true
	}
	for key, expected := range want {
		if strings.TrimSpace(have[key]) != strings.TrimSpace(expected) {
			return false
		}
	}
	return true
}

func cloneMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return append([]string(nil), in...)
}

func cloneCollection(collection Collection) Collection {
	collection.Metadata = cloneMetadata(collection.Metadata)
	return collection
}

func cloneDocument(document Document) Document {
	document.Metadata = cloneMetadata(document.Metadata)
	document.Tags = cloneStrings(document.Tags)
	return document
}

func makeSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

package retrieval

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Source struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	URI         string            `json:"uri,omitempty"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type CreateSourceRequest struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	URI         string            `json:"uri,omitempty"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Active      *bool             `json:"active,omitempty"`
}

type UpdateSourceRequest struct {
	Type        *string           `json:"type,omitempty"`
	URI         *string           `json:"uri,omitempty"`
	Title       *string           `json:"title,omitempty"`
	Description *string           `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Active      *bool             `json:"active,omitempty"`
}

type UpsertSourceInput = CreateSourceRequest

type ListSourcesRequest struct {
	Types           []string          `json:"types,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	IncludeInactive bool              `json:"include_inactive,omitempty"`
}

func (s *Service) CreateSource(req CreateSourceRequest) (Source, error) {
	return s.UpsertSource(req)
}

func (s *Service) UpsertSource(req UpsertSourceInput) (Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Step 1: register the origin of knowledge once.
	// Documents can then point at this source via SourceID/SourceType instead of
	// each caller inventing its own provenance model.
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Title) == "" {
		return Source{}, fmt.Errorf("%w: source type and title are required", ErrInvalidArgument)
	}

	now := time.Now()
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = fmt.Sprintf("source-%d", now.UnixNano())
	}
	source, exists := s.sources[id]
	if !exists {
		source = Source{ID: id, CreatedAt: now}
	}
	source.Type = strings.TrimSpace(req.Type)
	source.URI = strings.TrimSpace(req.URI)
	source.Title = strings.TrimSpace(req.Title)
	source.Description = strings.TrimSpace(req.Description)
	source.Metadata = cloneMetadata(req.Metadata)
	source.UpdatedAt = now
	if req.Active != nil {
		source.Active = *req.Active
	} else if !exists {
		source.Active = true
	}

	s.sources[id] = source
	s.rebuildIndexLocked()
	return cloneSource(source), nil
}

func (s *Service) UpdateSource(id string, req UpdateSourceRequest) (Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	source, ok := s.sources[id]
	if !ok {
		return Source{}, ErrNotFound
	}
	if req.Type != nil {
		source.Type = strings.TrimSpace(*req.Type)
	}
	if req.URI != nil {
		source.URI = strings.TrimSpace(*req.URI)
	}
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return Source{}, fmt.Errorf("%w: source title is required", ErrInvalidArgument)
		}
		source.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		source.Description = strings.TrimSpace(*req.Description)
	}
	if req.Metadata != nil {
		source.Metadata = cloneMetadata(req.Metadata)
	}
	if req.Active != nil {
		source.Active = *req.Active
	}
	source.UpdatedAt = time.Now()
	s.sources[id] = source
	s.rebuildIndexLocked()
	return cloneSource(source), nil
}

func (s *Service) GetSource(id string) (Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	source, ok := s.sources[id]
	if !ok {
		return Source{}, ErrNotFound
	}
	return cloneSource(source), nil
}

func (s *Service) ListSources(req ListSourcesRequest) ([]Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := makeSet(req.Types)
	out := make([]Source, 0, len(s.sources))
	for _, source := range s.sources {
		if len(types) > 0 {
			if _, ok := types[source.Type]; !ok {
				continue
			}
		}
		if !req.IncludeInactive && !source.Active {
			continue
		}
		if !matchesMetadata(source.Metadata, req.Metadata) {
			continue
		}
		out = append(out, cloneSource(source))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title == out[j].Title {
			return out[i].ID < out[j].ID
		}
		return out[i].Title < out[j].Title
	})
	return out, nil
}

func (s *Service) DeleteSource(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sources[id]; !ok {
		return ErrNotFound
	}
	for _, document := range s.documents {
		if document.SourceID == id {
			return fmt.Errorf("%w: source %q is still referenced", ErrInvalidArgument, id)
		}
	}
	delete(s.sources, id)
	s.rebuildIndexLocked()
	return nil
}

func (s *Service) SetSourceActive(id string, active bool) (Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	source, ok := s.sources[id]
	if !ok {
		return Source{}, ErrNotFound
	}
	source.Active = active
	source.UpdatedAt = time.Now()
	s.sources[id] = source
	s.rebuildIndexLocked()
	return cloneSource(source), nil
}

func cloneSource(source Source) Source {
	source.Metadata = cloneMetadata(source.Metadata)
	return source
}

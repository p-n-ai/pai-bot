// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTeacherResourceNotFound = errors.New("teacher resource not found")
	ErrTeacherResourceConflict = errors.New("teacher resource already exists")
	ErrTeacherResourceScope    = errors.New("teacher resource class scope denied")
	ErrGraphUnavailable        = errors.New("pgGraph 1.0.0 is unavailable")
)

const maxTeacherResourceChunks = 5000

type TeacherResource struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Title      string    `json:"title"`
	SourceType string    `json:"source_type"`
	MediaType  string    `json:"media_type"`
	ByteSize   int       `json:"byte_size"`
	ChunkCount int       `json:"chunk_count"`
	Active     bool      `json:"active"`
	ClassIDs   []string  `json:"class_ids"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TeacherUploadInput struct {
	TenantID   string
	UploaderID string
	Filename   string
	Title      string
	MediaType  string
	ClassIDs   []string
	Data       []byte
}

type TeacherEvidenceRequest struct {
	TenantID string
	ClassIDs []string
	Query    string
	Limit    int
}

type TeacherEvidence struct {
	ID           string   `json:"id"`
	SourceTitle  string   `json:"source_title"`
	Filename     string   `json:"filename"`
	LocatorType  string   `json:"locator_type"`
	LocatorStart int      `json:"locator_start"`
	LocatorEnd   int      `json:"locator_end"`
	Excerpt      string   `json:"excerpt"`
	SourceType   string   `json:"source_type"`
	Score        float64  `json:"score"`
	ClassIDs     []string `json:"class_ids"`
}

type TeacherResourceService struct {
	pool               *pgxpool.Pool
	embedder           Embedder
	embeddingModel     string
	allowGraphFallback bool
	graphDepth         int
	graphFrontier      int
}

type TeacherResourceOptions struct {
	Embedder           Embedder
	EmbeddingModel     string
	AllowGraphFallback bool
	GraphDepth         int
	GraphFrontier      int
}

func NewTeacherResourceService(pool *pgxpool.Pool, opts TeacherResourceOptions) (*TeacherResourceService, error) {
	if pool == nil {
		return nil, errors.New("teacher resource database is required")
	}
	if strings.TrimSpace(opts.EmbeddingModel) == "" {
		opts.EmbeddingModel = DefaultEmbeddingModel
	}
	if opts.GraphDepth <= 0 || opts.GraphDepth > 3 {
		opts.GraphDepth = 1
	}
	if opts.GraphFrontier <= 0 || opts.GraphFrontier > 200 {
		opts.GraphFrontier = 40
	}
	return &TeacherResourceService{
		pool: pool, embedder: opts.Embedder, embeddingModel: opts.EmbeddingModel,
		allowGraphFallback: opts.AllowGraphFallback, graphDepth: opts.GraphDepth,
		graphFrontier: opts.GraphFrontier,
	}, nil
}

func (s *TeacherResourceService) VerifyGraph(ctx context.Context) error {
	var version string
	err := s.pool.QueryRow(ctx, `SELECT extversion FROM pg_extension WHERE extname = 'graph'`).Scan(&version)
	if err != nil || version != "1.0.0" {
		if s.allowGraphFallback {
			return nil
		}
		return fmt.Errorf("%w: expected graph 1.0.0", ErrGraphUnavailable)
	}
	var enabled bool
	if err := s.pool.QueryRow(ctx, `SELECT graph.test_enabled()`).Scan(&enabled); err != nil || !enabled {
		if s.allowGraphFallback {
			return nil
		}
		return fmt.Errorf("%w: extension is disabled", ErrGraphUnavailable)
	}
	return nil
}

func (s *TeacherResourceService) Upload(ctx context.Context, input TeacherUploadInput) (TeacherResource, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.UploaderID = strings.TrimSpace(input.UploaderID)
	input.Filename = strings.TrimSpace(input.Filename)
	input.Title = strings.TrimSpace(input.Title)
	input.ClassIDs = uniqueSorted(input.ClassIDs)
	if input.TenantID == "" || input.UploaderID == "" {
		return TeacherResource{}, fmt.Errorf("%w: authenticated tenant and uploader are required", ErrTeacherResourceScope)
	}
	if input.Filename == "" || len(input.ClassIDs) == 0 {
		return TeacherResource{}, fmt.Errorf("%w: filename and at least one class are required", ErrInvalidArgument)
	}
	sourceType, units, err := ExtractTeacherResource(input.Filename, input.Data)
	if err != nil {
		return TeacherResource{}, err
	}
	chunks := ChunkTeacherResource(units)
	if len(chunks) == 0 {
		return TeacherResource{}, ErrEmptyFile
	}
	if len(chunks) > maxTeacherResourceChunks {
		return TeacherResource{}, fmt.Errorf("%w: extracted document exceeds %d chunks", ErrInvalidArgument, maxTeacherResourceChunks)
	}
	if input.Title == "" {
		input.Title = strings.TrimSuffix(input.Filename, filepathExtension(input.Filename))
	}
	var vectors [][]float32
	if s.embedder != nil {
		texts := make([]string, len(chunks))
		for i := range chunks {
			texts[i] = chunks[i].Content
		}
		vectors, _ = embedBatches(ctx, s.embedder, texts, 64)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TeacherResource{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := verifyClasses(ctx, tx, input.TenantID, input.ClassIDs); err != nil {
		return TeacherResource{}, err
	}
	digest := sha256.Sum256(input.Data)
	var resource TeacherResource
	err = tx.QueryRow(ctx, `
		INSERT INTO teacher_resources
			(tenant_id, uploader_id, filename, title, source_type, media_type, byte_size, sha256, original_file)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, filename, title, source_type, media_type, byte_size, active, created_at, updated_at`,
		input.TenantID, input.UploaderID, input.Filename, input.Title, sourceType,
		input.MediaType, len(input.Data), digest[:], input.Data,
	).Scan(&resource.ID, &resource.Filename, &resource.Title, &resource.SourceType,
		&resource.MediaType, &resource.ByteSize, &resource.Active, &resource.CreatedAt, &resource.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "teacher_resources_tenant_id_sha256_key") {
			return TeacherResource{}, ErrTeacherResourceConflict
		}
		return TeacherResource{}, err
	}
	resource.ClassIDs = input.ClassIDs
	resource.ChunkCount = len(chunks)
	for _, classID := range input.ClassIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO teacher_resource_classes (resource_id, tenant_id, class_id)
			VALUES ($1::uuid, $2::uuid, $3::uuid)`, resource.ID, input.TenantID, classID); err != nil {
			return TeacherResource{}, err
		}
	}

	chunkIDs := make([]string, len(chunks))
	for i, chunk := range chunks {
		if err := tx.QueryRow(ctx, `
			INSERT INTO teacher_resource_chunks
				(resource_id, tenant_id, ordinal, locator_type, locator_start, locator_end, content)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
			RETURNING id::text`, resource.ID, input.TenantID, chunk.Ordinal, chunk.LocatorType,
			chunk.LocatorStart, chunk.LocatorEnd, chunk.Content).Scan(&chunkIDs[i]); err != nil {
			return TeacherResource{}, err
		}
		if i > 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO teacher_resource_chunk_edges
					(tenant_id, resource_id, source_chunk_id, target_chunk_id, edge_type)
				VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'adjacent')`,
				input.TenantID, resource.ID, chunkIDs[i-1], chunkIDs[i]); err != nil {
				return TeacherResource{}, err
			}
		}
	}
	for _, edge := range RelatedTeacherChunkEdges(chunks) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO teacher_resource_chunk_edges
				(tenant_id, resource_id, source_chunk_id, target_chunk_id, edge_type)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5)`,
			input.TenantID, resource.ID, chunkIDs[edge.SourceOrdinal],
			chunkIDs[edge.TargetOrdinal], edge.Type); err != nil {
			return TeacherResource{}, err
		}
	}
	if len(vectors) == len(chunks) {
		for i, vector := range vectors {
			if _, err := tx.Exec(ctx, `
					INSERT INTO teacher_resource_embeddings
						(chunk_id, resource_id, tenant_id, model, dimensions, embedding)
					VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::vector)`,
				chunkIDs[i], resource.ID, input.TenantID, s.embeddingModel,
				DefaultEmbeddingDimensions, vectorLiteral(vector)); err != nil {
				return TeacherResource{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return TeacherResource{}, err
	}
	_, _ = s.pool.Exec(ctx, `SELECT * FROM graph.apply_sync()`)
	return resource, nil
}

func (s *TeacherResourceService) List(ctx context.Context, tenantID string, classIDs []string, includeInactive bool) ([]TeacherResource, error) {
	classIDs = uniqueSorted(classIDs)
	if strings.TrimSpace(tenantID) == "" || len(classIDs) == 0 {
		return nil, ErrTeacherResourceScope
	}
	if err := verifyClasses(ctx, s.pool, tenantID, classIDs); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.filename, r.title, r.source_type, r.media_type, r.byte_size,
		       r.active, r.created_at, r.updated_at,
		       array_agg(DISTINCT all_rc.class_id::text ORDER BY all_rc.class_id::text),
		       count(DISTINCT c.id)
		FROM teacher_resources r
		JOIN teacher_resource_classes requested
		  ON requested.resource_id = r.id AND requested.tenant_id = r.tenant_id
		 AND requested.class_id = ANY($2::uuid[])
		JOIN teacher_resource_classes all_rc
		  ON all_rc.resource_id = r.id AND all_rc.tenant_id = r.tenant_id
		JOIN teacher_resource_chunks c
		  ON c.resource_id = r.id AND c.tenant_id = r.tenant_id
		WHERE r.tenant_id = $1::uuid AND ($3 OR r.active)
		GROUP BY r.id
		ORDER BY r.created_at DESC, r.id`, tenantID, classIDs, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var resources []TeacherResource
	for rows.Next() {
		var item TeacherResource
		if err := rows.Scan(&item.ID, &item.Filename, &item.Title, &item.SourceType,
			&item.MediaType, &item.ByteSize, &item.Active, &item.CreatedAt,
			&item.UpdatedAt, &item.ClassIDs, &item.ChunkCount); err != nil {
			return nil, err
		}
		resources = append(resources, item)
	}
	return resources, rows.Err()
}

func (s *TeacherResourceService) SetActive(ctx context.Context, tenantID, resourceID string, classIDs []string, active bool) error {
	if err := s.ensureScoped(ctx, tenantID, resourceID, classIDs); err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE teacher_resources SET active = $3, updated_at = NOW()
		WHERE id = $1::uuid AND tenant_id = $2::uuid`, resourceID, tenantID, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTeacherResourceNotFound
	}
	_, _ = s.pool.Exec(ctx, `SELECT * FROM graph.apply_sync()`)
	return nil
}

func (s *TeacherResourceService) Delete(ctx context.Context, tenantID, resourceID string, classIDs []string) error {
	if err := s.ensureScoped(ctx, tenantID, resourceID, classIDs); err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM teacher_resources WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		resourceID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTeacherResourceNotFound
	}
	_, _ = s.pool.Exec(ctx, `SELECT * FROM graph.apply_sync()`)
	return nil
}

func (s *TeacherResourceService) ensureScoped(ctx context.Context, tenantID, resourceID string, classIDs []string) error {
	classIDs = uniqueSorted(classIDs)
	if tenantID == "" || resourceID == "" || len(classIDs) == 0 {
		return ErrTeacherResourceScope
	}
	if err := verifyClasses(ctx, s.pool, tenantID, classIDs); err != nil {
		return err
	}
	var allowed bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM teacher_resource_classes
			WHERE resource_id = $1::uuid AND tenant_id = $2::uuid
			  AND class_id = ANY($3::uuid[])
		)`, resourceID, tenantID, classIDs).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrTeacherResourceNotFound
	}
	return nil
}

func verifyClasses(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenantID string, classIDs []string) error {
	var count int
	err := q.QueryRow(ctx, `
		SELECT count(*) FROM groups
		WHERE tenant_id = $1::uuid AND type = 'class' AND id = ANY($2::uuid[])`,
		tenantID, classIDs).Scan(&count)
	if err != nil {
		return err
	}
	if count != len(classIDs) {
		return fmt.Errorf("%w: every selected class must belong to the authenticated tenant", ErrTeacherResourceScope)
	}
	return nil
}

type rankedChunk struct {
	id    string
	score float64
}

func (s *TeacherResourceService) Search(ctx context.Context, req TeacherEvidenceRequest) ([]TeacherEvidence, error) {
	req.ClassIDs = uniqueSorted(req.ClassIDs)
	req.Query = strings.TrimSpace(req.Query)
	if req.TenantID == "" || len(req.ClassIDs) == 0 || req.Query == "" {
		return nil, ErrTeacherResourceScope
	}
	if err := verifyClasses(ctx, s.pool, req.TenantID, req.ClassIDs); err != nil {
		return nil, err
	}
	if req.Limit <= 0 || req.Limit > 20 {
		req.Limit = 8
	}
	seedLimit := req.Limit * 4
	scores := map[string]float64{}
	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text
		FROM teacher_resource_chunks c
		JOIN teacher_resources r ON r.id = c.resource_id AND r.tenant_id = c.tenant_id
		WHERE c.tenant_id = $1::uuid AND r.active
		  AND EXISTS (
		    SELECT 1 FROM teacher_resource_classes rc
		    WHERE rc.resource_id = c.resource_id AND rc.tenant_id = c.tenant_id
		      AND rc.class_id = ANY($2::uuid[])
		  )
		  AND c.search_vector @@ websearch_to_tsquery('simple', $3)
		ORDER BY ts_rank_cd(c.search_vector, websearch_to_tsquery('simple', $3)) DESC, c.id
		LIMIT $4`, req.TenantID, req.ClassIDs, req.Query, seedLimit)
	if err != nil {
		return nil, err
	}
	rank := 1
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		scores[id] += 1 / float64(60+rank)
		rank++
	}
	rows.Close()

	if s.embedder != nil {
		vectors, embedErr := s.embedder.Embed(ctx, []string{req.Query})
		if embedErr == nil && len(vectors) == 1 {
			denseRows, denseErr := s.pool.Query(ctx, `
				SELECT c.id::text
				FROM teacher_resource_embeddings e
				JOIN teacher_resource_chunks c
				  ON c.id = e.chunk_id AND c.tenant_id = e.tenant_id
				JOIN teacher_resources r
				  ON r.id = c.resource_id AND r.tenant_id = c.tenant_id
				WHERE c.tenant_id = $1::uuid AND r.active
				  AND EXISTS (
				    SELECT 1 FROM teacher_resource_classes rc
				    WHERE rc.resource_id = c.resource_id AND rc.tenant_id = c.tenant_id
				      AND rc.class_id = ANY($2::uuid[])
				  )
				ORDER BY e.embedding <=> $3::vector, c.id
				LIMIT $4`, req.TenantID, req.ClassIDs, vectorLiteral(vectors[0]), seedLimit)
			if denseErr == nil {
				rank = 1
				for denseRows.Next() {
					var id string
					_ = denseRows.Scan(&id)
					scores[id] += 1 / float64(60+rank)
					rank++
				}
				denseRows.Close()
			}
		}
	}
	seeds := make([]rankedChunk, 0, len(scores))
	for id, score := range scores {
		seeds = append(seeds, rankedChunk{id: id, score: score})
	}
	sort.Slice(seeds, func(i, j int) bool {
		if seeds[i].score == seeds[j].score {
			return seeds[i].id < seeds[j].id
		}
		return seeds[i].score > seeds[j].score
	})
	if len(seeds) > req.Limit {
		seeds = seeds[:req.Limit]
	}
	if len(seeds) == 0 {
		return []TeacherEvidence{}, nil
	}
	expanded := make(map[string]float64, len(seeds))
	for _, seed := range seeds {
		expanded[seed.id] = seed.score
		neighbors, graphErr := s.expandGraph(ctx, req.TenantID, seed.id)
		if graphErr != nil {
			if !s.allowGraphFallback {
				return nil, graphErr
			}
			neighbors, _ = s.expandFallback(ctx, req.TenantID, seed.id)
		}
		for id, depth := range neighbors {
			score := seed.score / float64(depth+1)
			if score > expanded[id] {
				expanded[id] = score
			}
		}
	}
	ids := make([]string, 0, len(expanded))
	for id := range expanded {
		ids = append(ids, id)
	}
	evidenceRows, err := s.pool.Query(ctx, `
		SELECT c.id::text, r.title, r.filename, c.locator_type, c.locator_start,
		       c.locator_end, left(c.content, 500), r.source_type,
		       array_agg(DISTINCT rc.class_id::text ORDER BY rc.class_id::text)
		FROM teacher_resource_chunks c
		JOIN teacher_resources r ON r.id = c.resource_id AND r.tenant_id = c.tenant_id
		JOIN teacher_resource_classes rc ON rc.resource_id = r.id AND rc.tenant_id = r.tenant_id
		WHERE c.tenant_id = $1::uuid AND c.id = ANY($2::uuid[]) AND r.active
		  AND EXISTS (
		    SELECT 1 FROM teacher_resource_classes allowed
		    WHERE allowed.resource_id = c.resource_id AND allowed.tenant_id = c.tenant_id
		      AND allowed.class_id = ANY($3::uuid[])
		  )
		GROUP BY c.id, r.id`, req.TenantID, ids, req.ClassIDs)
	if err != nil {
		return nil, err
	}
	defer evidenceRows.Close()
	var evidence []TeacherEvidence
	for evidenceRows.Next() {
		var item TeacherEvidence
		if err := evidenceRows.Scan(&item.ID, &item.SourceTitle, &item.Filename,
			&item.LocatorType, &item.LocatorStart, &item.LocatorEnd, &item.Excerpt,
			&item.SourceType, &item.ClassIDs); err != nil {
			return nil, err
		}
		item.Score = expanded[item.ID]
		evidence = append(evidence, item)
	}
	sort.Slice(evidence, func(i, j int) bool {
		if evidence[i].Score == evidence[j].Score {
			return evidence[i].ID < evidence[j].ID
		}
		return evidence[i].Score > evidence[j].Score
	})
	if len(evidence) > req.Limit {
		evidence = evidence[:req.Limit]
	}
	return evidence, evidenceRows.Err()
}

func (s *TeacherResourceService) expandGraph(ctx context.Context, tenantID, seedID string) (map[string]int, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		SELECT set_config('graph.tenant_setting', 'app.pai_tenant_id', true),
		       set_config('app.pai_tenant_id', $1, true)`, tenantID); err != nil {
		return nil, fmt.Errorf("%w: set tenant scope: %v", ErrGraphUnavailable, err)
	}
	if _, err := tx.Exec(ctx, `SELECT * FROM graph.apply_sync()`); err != nil {
		return nil, fmt.Errorf("%w: apply sync: %v", ErrGraphUnavailable, err)
	}
	rows, err := tx.Query(ctx, `
		SELECT node_id, depth
		FROM graph.traverse(
		  'public.teacher_resource_chunks'::regclass, $1,
		  max_depth := $2,
		  edge_types := ARRAY['teacher_resource_chunk'],
		  direction := 'any',
		  node_tables := ARRAY['public.teacher_resource_chunks'::regclass::oid],
		  tenant := NULL,
		  include_start := false,
		  hydrate := false,
		  max_rows := $3,
		  max_nodes := $3,
		  max_frontier := $3
		)`, seedID, s.graphDepth, s.graphFrontier)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGraphUnavailable, err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var depth int
		if err := rows.Scan(&id, &depth); err != nil {
			return nil, err
		}
		out[id] = depth
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *TeacherResourceService) expandFallback(ctx context.Context, tenantID, seedID string) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
		WITH RECURSIVE neighbors(id, depth, path) AS (
		  SELECT $1::uuid, 0, ARRAY[$1::uuid]
		  UNION ALL
		  SELECT CASE WHEN e.source_chunk_id = n.id THEN e.target_chunk_id ELSE e.source_chunk_id END,
		         n.depth + 1,
		         n.path || CASE WHEN e.source_chunk_id = n.id THEN e.target_chunk_id ELSE e.source_chunk_id END
		  FROM neighbors n
		  JOIN teacher_resource_chunk_edges e
		    ON e.tenant_id = $2::uuid
		   AND (e.source_chunk_id = n.id OR e.target_chunk_id = n.id)
		  WHERE n.depth < $3
		    AND NOT (CASE WHEN e.source_chunk_id = n.id THEN e.target_chunk_id ELSE e.source_chunk_id END = ANY(n.path))
		)
		SELECT id::text, min(depth) FROM neighbors WHERE depth > 0
		GROUP BY id ORDER BY min(depth), id LIMIT $4`,
		seedID, tenantID, s.graphDepth, s.graphFrontier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var depth int
		if err := rows.Scan(&id, &depth); err != nil {
			return nil, err
		}
		out[id] = depth
	}
	return out, rows.Err()
}

func vectorLiteral(vector []float32) string {
	parts := make([]string, len(vector))
	for i, value := range vector {
		parts[i] = strconv.FormatFloat(float64(value), 'g', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func embedBatches(ctx context.Context, embedder Embedder, texts []string, batchSize int) ([][]float32, error) {
	if batchSize <= 0 {
		batchSize = 64
	}
	vectors := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += batchSize {
		end := min(start+batchSize, len(texts))
		batch, err := embedder.Embed(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		if len(batch) != end-start {
			return nil, fmt.Errorf("embedder returned %d vectors for batch of %d", len(batch), end-start)
		}
		vectors = append(vectors, batch...)
	}
	return vectors, nil
}

func uniqueSorted(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func filepathExtension(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 {
		return ""
	}
	return name[index:]
}

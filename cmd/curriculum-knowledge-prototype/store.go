package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbExecQuerier interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type contentStore struct {
	db dbExecQuerier
}

func newContentStore(db dbExecQuerier) contentStore {
	return contentStore{db: db}
}

func schemaSQL() string {
	return `
CREATE SCHEMA IF NOT EXISTS retrieval_prototype;

CREATE TABLE IF NOT EXISTS retrieval_prototype.curriculum_content (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    curriculum_source TEXT NOT NULL DEFAULT '',
    kind          TEXT NOT NULL,
    title         TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL,
    metadata      JSONB NOT NULL DEFAULT '{}',
    search_text   TEXT NOT NULL,
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('simple', search_text)
    ) STORED,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS curriculum_content_kind_idx
    ON retrieval_prototype.curriculum_content (kind);

CREATE INDEX IF NOT EXISTS curriculum_content_metadata_gin_idx
    ON retrieval_prototype.curriculum_content USING GIN (metadata);

CREATE INDEX IF NOT EXISTS curriculum_content_search_vector_idx
    ON retrieval_prototype.curriculum_content USING GIN (search_vector);
`
}

func dropSQL() string {
	return `DROP SCHEMA IF EXISTS retrieval_prototype CASCADE;`
}

func (s contentStore) applySchema(ctx context.Context) error {
	if _, err := s.db.Exec(ctx, schemaSQL()); err != nil {
		return fmt.Errorf("apply retrieval schema: %w", err)
	}
	return nil
}

func (s contentStore) drop(ctx context.Context) error {
	if _, err := s.db.Exec(ctx, dropSQL()); err != nil {
		return fmt.Errorf("drop retrieval schema: %w", err)
	}
	return nil
}

func (s contentStore) replaceContent(ctx context.Context, rows []CurriculumContent) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace content transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `TRUNCATE retrieval_prototype.curriculum_content`); err != nil {
		return fmt.Errorf("clear curriculum content: %w", err)
	}

	for _, row := range rows {
		metadata, err := json.Marshal(row.Metadata)
		if err != nil {
			return fmt.Errorf("marshal content metadata: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO retrieval_prototype.curriculum_content (
				curriculum_source, kind, title, body, metadata, search_text
			)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		`, row.CurriculumSource, row.Kind, row.Title, row.Body, string(metadata), row.SearchText); err != nil {
			return fmt.Errorf("insert curriculum content: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace content transaction: %w", err)
	}
	return nil
}

func (s contentStore) searchText(ctx context.Context, query string, limit int) ([]CurriculumContentHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []CurriculumContentHit{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.db.Query(ctx, `
		SELECT
			id,
			curriculum_source,
			kind,
			title,
			body,
			metadata,
			search_text,
			ts_rank_cd(search_vector, websearch_to_tsquery('simple', $1)) AS score
		FROM retrieval_prototype.curriculum_content
		WHERE search_vector @@ websearch_to_tsquery('simple', $1)
		ORDER BY score DESC
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search curriculum content: %w", err)
	}
	defer rows.Close()

	hits := []CurriculumContentHit{}
	for rows.Next() {
		var (
			row         CurriculumContent
			metadataRaw []byte
			score       float64
		)
		if err := rows.Scan(
			&row.ID,
			&row.CurriculumSource,
			&row.Kind,
			&row.Title,
			&row.Body,
			&metadataRaw,
			&row.SearchText,
			&score,
		); err != nil {
			return nil, fmt.Errorf("scan curriculum content hit: %w", err)
		}
		if len(metadataRaw) > 0 {
			if err := json.Unmarshal(metadataRaw, &row.Metadata); err != nil {
				return nil, fmt.Errorf("parse curriculum metadata: %w", err)
			}
		}
		hits = append(hits, CurriculumContentHit{Content: row, Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate curriculum content hits: %w", err)
	}
	return hits, nil
}

type CurriculumContentHit struct {
	Content CurriculumContent
	Score   float64
}

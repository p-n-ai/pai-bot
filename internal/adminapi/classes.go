package adminapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type JoinClassView struct {
	ClassID         string `json:"class_id"`
	ClassName       string `json:"class_name"`
	ClassSlug       string `json:"class_slug"`
	SchoolName      string `json:"school_name"`
	CurriculumLabel string `json:"curriculum_label"`
}

func (s *Service) GetJoinClass(slug string) (JoinClassView, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	normalizedSlug := normalizeSlug(slug)
	if normalizedSlug == "" {
		return JoinClassView{}, fmt.Errorf("%w: class slug is required", ErrInvalidArgument)
	}

	type row struct {
		classID       string
		className     string
		classSlug     string
		schoolName    string
		curriculumLbl string
	}

	var result row
	err := s.pool.QueryRow(ctx, `
		SELECT
			c.id::text,
			c.name,
			c.slug,
			COALESCE(NULLIF(t.config->'onboarding'->>'school_name', ''), t.name) AS school_name,
			COALESCE(NULLIF(t.config->'onboarding'->'curriculum'->>'label', ''), c.syllabus_id) AS curriculum_label
		FROM classes c
		JOIN tenants t ON t.id = c.tenant_id
		WHERE c.slug = $1
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT 1
	`, normalizedSlug).Scan(
		&result.classID,
		&result.className,
		&result.classSlug,
		&result.schoolName,
		&result.curriculumLbl,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return JoinClassView{}, ErrNotFound
		}
		return JoinClassView{}, fmt.Errorf("lookup join class: %w", err)
	}

	return JoinClassView{
		ClassID:         result.classID,
		ClassName:       strings.TrimSpace(result.className),
		ClassSlug:       strings.TrimSpace(result.classSlug),
		SchoolName:      strings.TrimSpace(result.schoolName),
		CurriculumLabel: strings.TrimSpace(result.curriculumLbl),
	}, nil
}

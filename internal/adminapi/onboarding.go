package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	onboardingSaveStatusSaved = "saved"
)

type OnboardingCurriculum struct {
	SyllabusID string `json:"syllabus_id"`
	Label      string `json:"label"`
}

type OnboardingFirstClass struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type OnboardingBotSetup struct {
	Preset string `json:"preset"`
}

type OnboardingState struct {
	SchoolName   string               `json:"school_name,omitempty"`
	Curriculum   OnboardingCurriculum `json:"curriculum"`
	FirstClass   OnboardingFirstClass `json:"first_class"`
	BotSetup     OnboardingBotSetup   `json:"bot_setup"`
	JoinLink     string               `json:"join_link"`
	SaveStatus   string               `json:"save_status"`
	ConfiguredAt time.Time            `json:"configured_at"`
}

type OnboardingView struct {
	TenantID   string           `json:"tenant_id"`
	TenantName string           `json:"tenant_name"`
	Onboarding *OnboardingState `json:"onboarding,omitempty"`
}

type SubmitOnboardingRequest struct {
	SchoolName string               `json:"school_name,omitempty"`
	Curriculum OnboardingCurriculum `json:"curriculum"`
	FirstClass OnboardingFirstClass `json:"first_class"`
	BotSetup   OnboardingBotSetup   `json:"bot_setup"`
}

type SubmitOnboardingResult struct {
	ClassID    string `json:"class_id"`
	SchoolName string `json:"school_name"`
	ClassName  string `json:"class_name"`
	JoinLink   string `json:"join_link"`
	SaveStatus string `json:"save_status"`
}

type tenantConfigEnvelope struct {
	Onboarding *OnboardingState `json:"onboarding,omitempty"`
}

func (s *Service) GetOnboarding() (OnboardingView, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if strings.TrimSpace(s.tenantID) == "" {
		return OnboardingView{}, fmt.Errorf("%w: tenant-scoped admin context is required", ErrInvalidArgument)
	}

	var (
		tenantName string
		rawConfig  []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT name, COALESCE(config, '{}'::jsonb)
		FROM tenants
		WHERE id = $1::uuid
	`, s.tenantID).Scan(&tenantName, &rawConfig)
	if err != nil {
		if err == pgx.ErrNoRows {
			return OnboardingView{}, ErrNotFound
		}
		return OnboardingView{}, fmt.Errorf("query onboarding config: %w", err)
	}

	view := OnboardingView{
		TenantID:   s.tenantID,
		TenantName: strings.TrimSpace(tenantName),
	}
	if len(rawConfig) == 0 {
		return view, nil
	}

	var envelope tenantConfigEnvelope
	if err := json.Unmarshal(rawConfig, &envelope); err != nil {
		return OnboardingView{}, fmt.Errorf("decode onboarding config: %w", err)
	}
	view.Onboarding = envelope.Onboarding
	return view, nil
}

func (s *Service) SubmitOnboarding(req SubmitOnboardingRequest, joinBaseURL string) (SubmitOnboardingResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if strings.TrimSpace(s.tenantID) == "" {
		return SubmitOnboardingResult{}, fmt.Errorf("%w: tenant-scoped admin context is required", ErrInvalidArgument)
	}

	normalized, err := normalizeOnboardingSubmit(req)
	if err != nil {
		return SubmitOnboardingResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SubmitOnboardingResult{}, fmt.Errorf("begin onboarding transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		tenantName string
		rawConfig  []byte
	)
	if err := tx.QueryRow(ctx, `
		SELECT name, COALESCE(config, '{}'::jsonb)
		FROM tenants
		WHERE id = $1::uuid
		FOR UPDATE
	`, s.tenantID).Scan(&tenantName, &rawConfig); err != nil {
		if err == pgx.ErrNoRows {
			return SubmitOnboardingResult{}, ErrNotFound
		}
		return SubmitOnboardingResult{}, fmt.Errorf("load tenant for onboarding: %w", err)
	}

	var existing tenantConfigEnvelope
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &existing); err != nil {
			return SubmitOnboardingResult{}, fmt.Errorf("decode existing onboarding config: %w", err)
		}
	}

	schoolName := normalized.SchoolName
	if schoolName == "" {
		schoolName = strings.TrimSpace(tenantName)
	}
	if strings.TrimSpace(normalized.SchoolName) != "" {
		var duplicateCount int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM tenants
			WHERE LOWER(name) = LOWER($1)
			  AND id <> $2::uuid
		`, normalized.SchoolName, s.tenantID).Scan(&duplicateCount); err != nil {
			return SubmitOnboardingResult{}, fmt.Errorf("check duplicate tenant name: %w", err)
		}
		if duplicateCount > 0 {
			return SubmitOnboardingResult{}, fmt.Errorf("%w: school name already exists", ErrInvalidArgument)
		}
	}

	classRecord, err := upsertOnboardingClass(ctx, tx, s.tenantID, existing.Onboarding, normalized)
	if err != nil {
		return SubmitOnboardingResult{}, err
	}

	joinLink := buildOnboardingJoinLink(joinBaseURL, classRecord.Slug)
	onboardingState := OnboardingState{
		SchoolName:   schoolName,
		Curriculum:   normalized.Curriculum,
		FirstClass:   OnboardingFirstClass(classRecord),
		BotSetup:     normalized.BotSetup,
		JoinLink:     joinLink,
		SaveStatus:   onboardingSaveStatusSaved,
		ConfiguredAt: time.Now().UTC(),
	}
	onboardingJSON, err := json.Marshal(onboardingState)
	if err != nil {
		return SubmitOnboardingResult{}, fmt.Errorf("encode onboarding config: %w", err)
	}

	if strings.TrimSpace(normalized.SchoolName) != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE tenants
			SET name = $2,
			    config = COALESCE(config, '{}'::jsonb) || jsonb_build_object('onboarding', $3::jsonb)
			WHERE id = $1::uuid
		`, s.tenantID, normalized.SchoolName, onboardingJSON); err != nil {
			return SubmitOnboardingResult{}, fmt.Errorf("update tenant onboarding config: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE tenants
			SET config = COALESCE(config, '{}'::jsonb) || jsonb_build_object('onboarding', $2::jsonb)
			WHERE id = $1::uuid
		`, s.tenantID, onboardingJSON); err != nil {
			return SubmitOnboardingResult{}, fmt.Errorf("update tenant onboarding config: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SubmitOnboardingResult{}, fmt.Errorf("commit onboarding transaction: %w", err)
	}

	return SubmitOnboardingResult{
		ClassID:    classRecord.ID,
		SchoolName: schoolName,
		ClassName:  classRecord.Name,
		JoinLink:   joinLink,
		SaveStatus: onboardingSaveStatusSaved,
	}, nil
}

type onboardingClassRecord struct {
	ID   string
	Name string
	Slug string
}

func upsertOnboardingClass(ctx context.Context, tx pgx.Tx, tenantID string, existing *OnboardingState, req SubmitOnboardingRequest) (onboardingClassRecord, error) {
	classID := strings.TrimSpace(req.FirstClass.ID)
	if classID == "" && existing != nil {
		classID = strings.TrimSpace(existing.FirstClass.ID)
	}

	var record onboardingClassRecord
	if classID != "" {
		err := tx.QueryRow(ctx, `
			UPDATE classes
			SET name = $3,
			    slug = $4,
			    syllabus_id = $5,
			    updated_at = NOW()
			WHERE id = $1::uuid
			  AND tenant_id = $2::uuid
			RETURNING id::text, name, slug
		`, classID, tenantID, req.FirstClass.Name, req.FirstClass.Slug, req.Curriculum.SyllabusID).Scan(
			&record.ID,
			&record.Name,
			&record.Slug,
		)
		if err == nil {
			return record, nil
		}
		if err != pgx.ErrNoRows {
			return onboardingClassRecord{}, fmt.Errorf("update onboarding class: %w", err)
		}
	}

	err := tx.QueryRow(ctx, `
		INSERT INTO classes (tenant_id, name, slug, syllabus_id)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (slug) DO UPDATE
		SET name = EXCLUDED.name,
		    syllabus_id = EXCLUDED.syllabus_id,
		    updated_at = NOW()
		WHERE classes.tenant_id = EXCLUDED.tenant_id
		RETURNING id::text, name, slug
	`, tenantID, req.FirstClass.Name, req.FirstClass.Slug, req.Curriculum.SyllabusID).Scan(
		&record.ID,
		&record.Name,
		&record.Slug,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return onboardingClassRecord{}, fmt.Errorf("%w: class slug already exists", ErrInvalidArgument)
		}
		return onboardingClassRecord{}, fmt.Errorf("upsert onboarding class: %w", err)
	}

	return record, nil
}

func normalizeOnboardingSubmit(req SubmitOnboardingRequest) (SubmitOnboardingRequest, error) {
	normalized := SubmitOnboardingRequest{
		SchoolName: strings.TrimSpace(req.SchoolName),
		Curriculum: OnboardingCurriculum{
			SyllabusID: strings.TrimSpace(req.Curriculum.SyllabusID),
			Label:      strings.TrimSpace(req.Curriculum.Label),
		},
		FirstClass: OnboardingFirstClass{
			Name: strings.TrimSpace(req.FirstClass.Name),
			Slug: normalizeSlug(req.FirstClass.Slug),
		},
		BotSetup: OnboardingBotSetup{
			Preset: strings.TrimSpace(req.BotSetup.Preset),
		},
	}

	if normalized.Curriculum.SyllabusID == "" || normalized.Curriculum.Label == "" {
		return SubmitOnboardingRequest{}, fmt.Errorf("%w: curriculum selection is required", ErrInvalidArgument)
	}
	if normalized.FirstClass.Name == "" {
		return SubmitOnboardingRequest{}, fmt.Errorf("%w: first class name is required", ErrInvalidArgument)
	}
	if normalized.FirstClass.Slug == "" {
		normalized.FirstClass.Slug = normalizeSlug(normalized.FirstClass.Name)
	}
	if normalized.FirstClass.Slug == "" {
		return SubmitOnboardingRequest{}, fmt.Errorf("%w: first class slug is required", ErrInvalidArgument)
	}
	if normalized.BotSetup.Preset == "" {
		return SubmitOnboardingRequest{}, fmt.Errorf("%w: bot setup preset is required", ErrInvalidArgument)
	}

	return normalized, nil
}

func normalizeSlug(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func buildOnboardingJoinLink(baseURL, classSlug string) string {
	path := "/join/" + strings.TrimSpace(classSlug)
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBase == "" {
		return path
	}
	return trimmedBase + path
}

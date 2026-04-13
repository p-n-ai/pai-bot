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
	defer tx.Rollback(ctx)

	var tenantName string
	if err := tx.QueryRow(ctx, `
		SELECT name
		FROM tenants
		WHERE id = $1::uuid
		FOR UPDATE
	`, s.tenantID).Scan(&tenantName); err != nil {
		if err == pgx.ErrNoRows {
			return SubmitOnboardingResult{}, ErrNotFound
		}
		return SubmitOnboardingResult{}, fmt.Errorf("load tenant for onboarding: %w", err)
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

	joinLink := buildOnboardingJoinLink(joinBaseURL, normalized.FirstClass.Slug)
	onboardingState := OnboardingState{
		SchoolName:   schoolName,
		Curriculum:   normalized.Curriculum,
		FirstClass:   normalized.FirstClass,
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
		SchoolName: schoolName,
		ClassName:  normalized.FirstClass.Name,
		JoinLink:   joinLink,
		SaveStatus: onboardingSaveStatusSaved,
	}, nil
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

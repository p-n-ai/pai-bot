package adminapi

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFormFromClassID(t *testing.T) {
	tests := []struct {
		classID string
		want    string
	}{
		{classID: "form-1-algebra", want: "Form 1"},
		{classID: "form-2-algebra", want: "Form 2"},
		{classID: "form-3-algebra", want: "Form 3"},
		{classID: "all-students", want: ""},
	}

	for _, tt := range tests {
		if got := formFromClassID(tt.classID); got != tt.want {
			t.Fatalf("formFromClassID(%q) = %q, want %q", tt.classID, got, tt.want)
		}
	}
}

func TestComputeStreakSummary(t *testing.T) {
	base := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	dates := []time.Time{
		base,
		base.Add(-24 * time.Hour),
		base.Add(-48 * time.Hour),
		base.Add(-24 * time.Hour * 5),
		base.Add(-24 * time.Hour * 6),
	}

	current, longest := computeStreakSummary(dates)
	if current != 3 {
		t.Fatalf("current = %d, want 3", current)
	}
	if longest != 3 {
		t.Fatalf("longest = %d, want 3", longest)
	}
}

func TestBuildParentEncouragement(t *testing.T) {
	tests := []struct {
		name   string
		streak StreakSummary
		stats  WeeklyStats
		rows   []ProgressItem
		want   string
	}{
		{
			name:   "strong streak celebrates consistency",
			streak: StreakSummary{Current: 5},
			stats:  WeeklyStats{NeedsReviewCount: 1},
			rows: []ProgressItem{
				{TopicID: "linear-equations", MasteryScore: 0.8},
				{TopicID: "inequalities", MasteryScore: 0.4},
			},
			want: "5-day streak",
		},
		{
			name:   "needs review nudges reset",
			streak: StreakSummary{Current: 1},
			stats:  WeeklyStats{NeedsReviewCount: 3},
			rows: []ProgressItem{
				{TopicID: "functions", MasteryScore: 0.3},
			},
			want: "Functions",
		},
		{
			name:   "empty progress falls back",
			streak: StreakSummary{},
			stats:  WeeklyStats{},
			rows:   nil,
			want:   "fresh study sprint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildParentEncouragement("Alya", tt.streak, tt.rows, tt.stats)
			if got.Headline == "" || got.Text == "" {
				t.Fatalf("encouragement = %#v, want non-empty headline and text", got)
			}
			if !strings.Contains(got.Headline+" "+got.Text, tt.want) {
				t.Fatalf("encouragement = %#v, want substring %q", got, tt.want)
			}
		})
	}
}

func TestSplitProviderModel(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantProvider string
		wantModel    string
	}{
		{name: "provider and model", raw: "openai:gpt-4o-mini", wantProvider: "openai", wantModel: "gpt-4o-mini"},
		{name: "model only", raw: "gpt-4o-mini", wantProvider: "unknown", wantModel: "gpt-4o-mini"},
		{name: "empty", raw: "", wantProvider: "unknown", wantModel: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProvider, gotModel := splitProviderModel(tt.raw)
			if gotProvider != tt.wantProvider || gotModel != tt.wantModel {
				t.Fatalf("splitProviderModel(%q) = (%q, %q), want (%q, %q)", tt.raw, gotProvider, gotModel, tt.wantProvider, tt.wantModel)
			}
		})
	}
}

func TestTenantPredicate(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		column   string
		position int
		want     string
		wantArg  any
	}{
		{
			name:     "tenant scoped service filters by tenant",
			service:  Service{tenantID: "tenant-1"},
			column:   "u.tenant_id",
			position: 1,
			want:     "($1::uuid IS NULL OR u.tenant_id = $1::uuid)",
			wantArg:  "tenant-1",
		},
		{
			name:     "platform service bypasses tenant filter",
			service:  Service{allTenants: true},
			column:   "messages.tenant_id",
			position: 2,
			want:     "($2::uuid IS NULL OR messages.tenant_id = $2::uuid)",
			wantArg:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.service.tenantPredicate(tt.column, tt.position); got != tt.want {
				t.Fatalf("tenantPredicate(%q, %d) = %q, want %q", tt.column, tt.position, got, tt.want)
			}
			if got := tt.service.tenantArg(); got != tt.wantArg {
				t.Fatalf("tenantArg() = %#v, want %#v", got, tt.wantArg)
			}
		})
	}
}

func TestComputeRetentionSeries(t *testing.T) {
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	series := computeRetentionSeries([]retentionCohortSample{
		{
			CohortDate: base,
			CohortSize: 10,
			Day1Users:  8,
			Day7Users:  6,
			Day14Users: 4,
		},
		{
			CohortDate: base.Add(24 * time.Hour),
			CohortSize: 5,
			Day1Users:  3,
			Day7Users:  2,
			Day14Users: 1,
		},
	})

	if len(series) != 2 {
		t.Fatalf("retention series len = %d, want 2", len(series))
	}
	if series[0].Day1Rate != 0.8 || series[0].Day7Rate != 0.6 || series[0].Day14Rate != 0.4 {
		t.Fatalf("series[0] = %#v, want 0.8/0.6/0.4", series[0])
	}
	if series[1].Day1Rate != 0.6 || series[1].Day7Rate != 0.4 || series[1].Day14Rate != 0.2 {
		t.Fatalf("series[1] = %#v, want 0.6/0.4/0.2", series[1])
	}
}

func TestBuildNudgeRateSummary(t *testing.T) {
	got := buildNudgeRateSummary(40, 11)
	if got.NudgesSent != 40 || got.ResponsesWithin24Hours != 11 {
		t.Fatalf("nudge summary = %#v, want nudges=40 responses=11", got)
	}
	if got.ResponseRate != 0.275 {
		t.Fatalf("response rate = %v, want 0.275", got.ResponseRate)
	}
}

func TestFinalizeAIUsageSummary(t *testing.T) {
	summary := AIUsageSummary{
		TotalMessages:     6,
		TotalInputTokens:  180,
		TotalOutputTokens: 120,
		Providers: []AIProviderUsage{
			{Provider: "openai", Model: "gpt-4o-mini", Messages: 4, InputTokens: 120, OutputTokens: 80, TotalTokens: 200},
			{Provider: "anthropic", Model: "claude-3-5-haiku", Messages: 2, InputTokens: 60, OutputTokens: 40, TotalTokens: 100},
		},
		DailyUsage: []AIDailyUsagePoint{
			{Date: "2026-03-10", Messages: 2, Tokens: 75},
			{Date: "2026-03-11", Messages: 4, Tokens: 225},
		},
	}

	finalizeAIUsageSummary(&summary, 3)

	if summary.PerStudentAverageTokens == nil {
		t.Fatal("PerStudentAverageTokens = nil, want computed value")
	}
	if *summary.PerStudentAverageTokens != 100 {
		t.Fatalf("PerStudentAverageTokens = %v, want 100", *summary.PerStudentAverageTokens)
	}
	if got := summary.DailyUsage[0].CostUSD; got != nil {
		t.Fatalf("DailyUsage[0].CostUSD = %v, want nil when cost attribution is absent", *got)
	}
	if len(summary.ProviderCosts) != 0 {
		t.Fatalf("ProviderCosts len = %d, want 0 when cost attribution is absent", len(summary.ProviderCosts))
	}
}

func TestFinalizeAIUsageSummarySkipsPerStudentAverageWithoutLearners(t *testing.T) {
	summary := AIUsageSummary{
		TotalInputTokens:  20,
		TotalOutputTokens: 10,
	}

	finalizeAIUsageSummary(&summary, 0)

	if summary.PerStudentAverageTokens != nil {
		t.Fatalf("PerStudentAverageTokens = %v, want nil without active learners", *summary.PerStudentAverageTokens)
	}
}

func TestApplyTokenBudgetWindow(t *testing.T) {
	summary := AIUsageSummary{
		TotalInputTokens:  180,
		TotalOutputTokens: 120,
	}
	window := &tokenBudgetWindow{
		BudgetTokens: 800,
		UsedTokens:   300,
		PeriodStart:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:    time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
	}

	applyTokenBudgetWindow(&summary, window)

	if summary.BudgetLimitTokens == nil || *summary.BudgetLimitTokens != 800 {
		t.Fatalf("BudgetLimitTokens = %v, want 800", summary.BudgetLimitTokens)
	}
	if summary.BudgetUsedTokens == nil || *summary.BudgetUsedTokens != 300 {
		t.Fatalf("BudgetUsedTokens = %v, want 300", summary.BudgetUsedTokens)
	}
	if summary.BudgetRemainingTokens == nil || *summary.BudgetRemainingTokens != 500 {
		t.Fatalf("BudgetRemainingTokens = %v, want 500", summary.BudgetRemainingTokens)
	}
	if summary.BudgetPeriodStart != "2026-03-01" {
		t.Fatalf("BudgetPeriodStart = %q, want 2026-03-01", summary.BudgetPeriodStart)
	}
	if summary.BudgetPeriodEnd != "2026-03-31" {
		t.Fatalf("BudgetPeriodEnd = %q, want 2026-03-31", summary.BudgetPeriodEnd)
	}
}

func TestApplyTokenBudgetWindowClampsRemainingToZero(t *testing.T) {
	summary := AIUsageSummary{}
	window := &tokenBudgetWindow{
		BudgetTokens: 100,
		UsedTokens:   160,
		PeriodStart:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:    time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
	}

	applyTokenBudgetWindow(&summary, window)

	if summary.BudgetRemainingTokens == nil || *summary.BudgetRemainingTokens != 0 {
		t.Fatalf("BudgetRemainingTokens = %v, want 0", summary.BudgetRemainingTokens)
	}
}

func TestNormalizeTokenBudgetWindowRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     UpsertTokenBudgetWindowRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: UpsertTokenBudgetWindowRequest{
				BudgetTokens: 250000,
				PeriodStart:  "2026-04-01",
				PeriodEnd:    "2026-04-30",
			},
		},
		{
			name: "budget must be positive",
			req: UpsertTokenBudgetWindowRequest{
				BudgetTokens: 0,
				PeriodStart:  "2026-04-01",
				PeriodEnd:    "2026-04-30",
			},
			wantErr: true,
		},
		{
			name: "start date must parse",
			req: UpsertTokenBudgetWindowRequest{
				BudgetTokens: 500,
				PeriodStart:  "04/01/2026",
				PeriodEnd:    "2026-04-30",
			},
			wantErr: true,
		},
		{
			name: "end date must parse",
			req: UpsertTokenBudgetWindowRequest{
				BudgetTokens: 500,
				PeriodStart:  "2026-04-01",
				PeriodEnd:    "04/30/2026",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTokenBudgetWindowRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeTokenBudgetWindowRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidArgument) {
					t.Fatalf("normalizeTokenBudgetWindowRequest() error = %v, want ErrInvalidArgument", err)
				}
				return
			}
			if got.BudgetTokens != tt.req.BudgetTokens {
				t.Fatalf("BudgetTokens = %d, want %d", got.BudgetTokens, tt.req.BudgetTokens)
			}
			if got.PeriodStart.Format(time.RFC3339) != "2026-04-01T00:00:00Z" {
				t.Fatalf("PeriodStart = %s, want 2026-04-01T00:00:00Z", got.PeriodStart.Format(time.RFC3339))
			}
			if got.PeriodEnd.Format(time.RFC3339) != "2026-04-30T23:59:59Z" {
				t.Fatalf("PeriodEnd = %s, want 2026-04-30T23:59:59Z", got.PeriodEnd.Format(time.RFC3339))
			}
		})
	}
}

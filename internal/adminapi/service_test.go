package adminapi

import (
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

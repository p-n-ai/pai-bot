package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/adminapi"
	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/auth"
)

func TestHealthEndpoints(t *testing.T) {
	mux := newMux(stubAdminAPI{}, &chatGatewayStub{})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "healthz returns 200",
			path:       "/healthz",
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		{
			name:       "readyz returns 200",
			path:       "/readyz",
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ready"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestAdminClassProgressEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/classes/form-1-algebra/progress", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Students []struct {
			ID string `json:"id"`
		} `json:"students"`
		TopicIDs []string `json:"topic_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Students) != 3 {
		t.Fatalf("students = %d, want 3", len(payload.Students))
	}
	if len(payload.TopicIDs) != 4 {
		t.Fatalf("topic_ids = %d, want 4", len(payload.TopicIDs))
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("access-control-allow-origin = %q, want http://localhost:3000", got)
	}
}

func TestAdminStudentDetailEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_1", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Student struct {
			ID string `json:"id"`
		} `json:"student"`
		Progress []struct {
			TopicID string `json:"topic_id"`
		} `json:"progress"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Student.ID != "stu_1" {
		t.Fatalf("student.id = %q, want stu_1", payload.Student.ID)
	}
	if len(payload.Progress) != 4 {
		t.Fatalf("progress = %d, want 4", len(payload.Progress))
	}
}

func TestAdminStudentConversationsEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_2/conversations", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload []struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("conversations = %d, want 2", len(payload))
	}
	if payload[0].Role != "student" {
		t.Fatalf("first role = %q, want student", payload[0].Role)
	}
}

func TestAdminParentSummaryEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/parents/parent-1", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueParentToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Parent struct {
			ID string `json:"id"`
		} `json:"parent"`
		Child struct {
			ID string `json:"id"`
		} `json:"child"`
		WeeklyStats struct {
			DaysActive        int `json:"days_active"`
			MessagesExchanged int `json:"messages_exchanged"`
		} `json:"weekly_stats"`
		Mastery []struct {
			TopicID string `json:"topic_id"`
		} `json:"mastery"`
		Encouragement struct {
			Text string `json:"text"`
		} `json:"encouragement"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Parent.ID != "parent-1" {
		t.Fatalf("parent.id = %q, want parent-1", payload.Parent.ID)
	}
	if payload.Child.ID != "stu_1" {
		t.Fatalf("child.id = %q, want stu_1", payload.Child.ID)
	}
	if payload.WeeklyStats.DaysActive != 5 || payload.WeeklyStats.MessagesExchanged != 18 {
		t.Fatalf("weekly stats = %#v, want days_active=5 messages_exchanged=18", payload.WeeklyStats)
	}
	if len(payload.Mastery) != 4 {
		t.Fatalf("mastery rows = %d, want 4", len(payload.Mastery))
	}
	if payload.Encouragement.Text == "" {
		t.Fatal("encouragement text is empty")
	}
}

func TestAdminParentSummaryEndpointNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/parents/missing", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAdminAIUsageEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/ai/usage", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		TotalMessages     int `json:"total_messages"`
		TotalInputTokens  int `json:"total_input_tokens"`
		TotalOutputTokens int `json:"total_output_tokens"`
		Providers         []struct {
			Provider     string `json:"provider"`
			Model        string `json:"model"`
			Messages     int    `json:"messages"`
			InputTokens  int    `json:"input_tokens"`
			OutputTokens int    `json:"output_tokens"`
			TotalTokens  int    `json:"total_tokens"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.TotalMessages != 6 {
		t.Fatalf("total_messages = %d, want 6", payload.TotalMessages)
	}
	if payload.TotalInputTokens != 226 || payload.TotalOutputTokens != 187 {
		t.Fatalf("token totals = %#v, want input=226 output=187", payload)
	}
	if len(payload.Providers) != 2 {
		t.Fatalf("providers = %d, want 2", len(payload.Providers))
	}
	if payload.Providers[0].Provider == "" || payload.Providers[0].Model == "" {
		t.Fatalf("first provider summary = %#v, want populated fields", payload.Providers[0])
	}
}

func TestAdminMetricsEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		WindowDays       int `json:"window_days"`
		DailyActiveUsers []struct {
			Date  string `json:"date"`
			Users int    `json:"users"`
		} `json:"daily_active_users"`
		Retention []struct {
			CohortDate string  `json:"cohort_date"`
			CohortSize int     `json:"cohort_size"`
			Day1Rate   float64 `json:"day_1_rate"`
			Day7Rate   float64 `json:"day_7_rate"`
			Day14Rate  float64 `json:"day_14_rate"`
		} `json:"retention"`
		NudgeRate struct {
			NudgesSent             int     `json:"nudges_sent"`
			ResponsesWithin24Hours int     `json:"responses_within_24h"`
			ResponseRate           float64 `json:"response_rate"`
		} `json:"nudge_rate"`
		ABComparison any `json:"ab_comparison"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.WindowDays != 14 {
		t.Fatalf("window_days = %d, want 14", payload.WindowDays)
	}
	if len(payload.DailyActiveUsers) != 2 {
		t.Fatalf("daily_active_users = %d, want 2", len(payload.DailyActiveUsers))
	}
	if len(payload.Retention) != 1 {
		t.Fatalf("retention rows = %d, want 1", len(payload.Retention))
	}
	if payload.NudgeRate.NudgesSent != 14 || payload.NudgeRate.ResponsesWithin24Hours != 5 {
		t.Fatalf("nudge rate = %#v, want nudges=14 responses=5", payload.NudgeRate)
	}
	if payload.ABComparison != nil {
		t.Fatalf("ab_comparison = %#v, want nil", payload.ABComparison)
	}
}

func TestAdminTokenBudgetWindowEndpoint(t *testing.T) {
	admin := &budgetAdminStub{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/ai/budget-window", strings.NewReader(`{"budget_tokens":250000,"period_start":"2026-04-01","period_end":"2026-04-30"}`))
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandler(admin, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if admin.req.BudgetTokens != 250000 || admin.req.PeriodStart != "2026-04-01" || admin.req.PeriodEnd != "2026-04-30" {
		t.Fatalf("request = %#v, want parsed token budget window payload", admin.req)
	}
}

func TestAdminTokenBudgetWindowEndpointRejectsTeacherRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/ai/budget-window", strings.NewReader(`{"budget_tokens":250000,"period_start":"2026-04-01","period_end":"2026-04-30"}`))
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAdminAPIOptionsPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/admin/students/stu_1", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("expected access-control-allow-headers to be set")
	}
}

func TestAdminStudentNudgeEndpoint(t *testing.T) {
	gateway := chatGatewayStub{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/students/tg_student/nudge", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &gateway).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if len(gateway.messages) != 1 {
		t.Fatalf("messages sent = %d, want 1", len(gateway.messages))
	}
	if gateway.messages[0].Channel != "telegram" {
		t.Fatalf("channel = %q, want telegram", gateway.messages[0].Channel)
	}
}

func TestAdminStudentNudgeEndpointRequiresTelegramChatID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/students/stu_1/nudge", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAdminStudentDetailNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/students/missing", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAdminEndpointsRequireAuthentication(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_1", nil)
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminEndpointsEnforceRoles(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_1", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueStudentToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestParentEndpointRejectsTeacherRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/parents/parent-1", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAdminAIUsageEndpointRejectsStudentRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/ai/usage", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueStudentToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAdminMetricsEndpointRejectsStudentRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueStudentToken(t))
	rec := httptest.NewRecorder()

	newHandler(stubAdminAPI{}, &chatGatewayStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthLoginEndpoint(t *testing.T) {
	authSvc := &stubAuthService{
		loginResp: auth.Session{
			Token:     "session-123",
			ExpiresAt: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
			User: auth.UserSession{
				UserID:   "teacher-1",
				TenantID: "tenant-abc",
				Role:     auth.RoleTeacher,
				Name:     "Teacher One",
				Email:    "teacher@example.com",
			},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"teacher@example.com","password":"secret-123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if authSvc.loginReq.Email != "teacher@example.com" {
		t.Fatalf("login email = %q, want teacher@example.com", authSvc.loginReq.Email)
	}

	var payload struct {
		ExpiresAt time.Time `json:"expires_at"`
		User      struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.ExpiresAt.IsZero() {
		t.Fatalf("expiry payload = %#v", payload)
	}
	if payload.User.UserID != "teacher-1" || payload.User.Role != string(auth.RoleTeacher) {
		t.Fatalf("user payload = %#v", payload.User)
	}
	assertAuthCookies(t, rec.Result().Cookies(), "session-123")
}

func TestAuthSessionEndpointReturnsUserAndRefreshesCookie(t *testing.T) {
	authSvc := &stubAuthService{
		sessionResp: auth.Session{
			Token:     "session-next",
			ExpiresAt: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
			User: auth.UserSession{
				UserID:   "teacher-1",
				TenantID: "tenant-abc",
				Role:     auth.RoleTeacher,
				Name:     "Teacher One",
				Email:    "teacher@example.com",
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-old"})
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if authSvc.sessionToken != "session-old" {
		t.Fatalf("session token = %q, want session-old", authSvc.sessionToken)
	}

	var payload struct {
		ExpiresAt time.Time `json:"expires_at"`
		User      struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !payload.ExpiresAt.Equal(authSvc.sessionResp.ExpiresAt) {
		t.Fatalf("expires_at = %v, want %v", payload.ExpiresAt, authSvc.sessionResp.ExpiresAt)
	}
	if payload.User.UserID != "teacher-1" || payload.User.Role != string(auth.RoleTeacher) {
		t.Fatalf("user payload = %#v", payload.User)
	}
	assertAuthCookies(t, rec.Result().Cookies(), "session-next")
}

func TestAuthAcceptInviteEndpoint(t *testing.T) {
	authSvc := &stubAuthService{
		acceptResp: auth.Session{
			Token:     "session-accept",
			ExpiresAt: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
			User: auth.UserSession{
				UserID:   "parent-1",
				TenantID: "tenant-abc",
				Role:     auth.RoleParent,
				Name:     "Parent One",
				Email:    "parent@example.com",
			},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/invitations/accept", strings.NewReader(`{"token":"invite-token","name":"Parent One","password":"strong-pass-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if authSvc.acceptReq.Token != "invite-token" {
		t.Fatalf("accept token = %q, want invite-token", authSvc.acceptReq.Token)
	}
	assertAuthCookies(t, rec.Result().Cookies(), "session-accept")
}

func TestAuthSwitchTenantEndpoint(t *testing.T) {
	authSvc := &stubAuthService{
		switchResp: auth.Session{
			Token:     "session-switched",
			ExpiresAt: time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC),
			User: auth.UserSession{
				UserID:   "teacher-2",
				TenantID: "tenant-b",
				Role:     auth.RoleTeacher,
				Name:     "Teacher One",
				Email:    "teacher@example.com",
			},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/switch-tenant", strings.NewReader(`{"tenant_id":"tenant-b","password":"secret-123"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "refresh-old"})
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if authSvc.switchToken != "refresh-old" {
		t.Fatalf("switch session token = %q, want refresh-old", authSvc.switchToken)
	}
	if authSvc.switchTenantID != "tenant-b" {
		t.Fatalf("switch tenant id = %q, want tenant-b", authSvc.switchTenantID)
	}
	if authSvc.switchPassword != "secret-123" {
		t.Fatalf("switch password = %q, want secret-123", authSvc.switchPassword)
	}
	assertAuthCookies(t, rec.Result().Cookies(), "session-switched")
}

func TestAuthLogoutEndpoint(t *testing.T) {
	authSvc := &stubAuthService{}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "refresh-old"})
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if authSvc.logoutToken != "refresh-old" {
		t.Fatalf("logout token = %q, want refresh-old", authSvc.logoutToken)
	}
	assertExpiredAuthCookies(t, rec.Result().Cookies())
}

func TestAuthLogoutEndpointClearsCookiesForInvalidTokens(t *testing.T) {
	authSvc := &stubAuthService{logoutErr: auth.ErrInvalidCredentials}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "refresh-stale"})
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if authSvc.logoutToken != "refresh-stale" {
		t.Fatalf("logout token = %q, want refresh-stale", authSvc.logoutToken)
	}
	assertExpiredAuthCookies(t, rec.Result().Cookies())
}

func TestAuthLogoutEndpointClearsCookiesWhenTokenMissing(t *testing.T) {
	authSvc := &stubAuthService{}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if authSvc.logoutToken != "" {
		t.Fatalf("logout token = %q, want empty when no session token is present", authSvc.logoutToken)
	}
	assertExpiredAuthCookies(t, rec.Result().Cookies())
}

func TestAuthLogoutClearsSessionForFollowupRequests(t *testing.T) {
	authSvc := &stubAuthService{
		sessionErr: auth.ErrInvalidCredentials,
	}
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-old"})
	logoutRec := httptest.NewRecorder()

	handler := newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour)
	handler.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", logoutRec.Code, http.StatusNoContent)
	}
	expiredCookie := findCookie(logoutRec.Result().Cookies(), auth.SessionCookieName)
	if expiredCookie == nil {
		t.Fatal("logout response missing expired session cookie")
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionReq.AddCookie(expiredCookie)
	sessionRec := httptest.NewRecorder()
	handler.ServeHTTP(sessionRec, sessionReq)

	if sessionRec.Code != http.StatusUnauthorized {
		t.Fatalf("session status = %d, want %d", sessionRec.Code, http.StatusUnauthorized)
	}
	if authSvc.sessionToken != "" {
		t.Fatalf("session token = %q, want empty because expired cookie should not authenticate", authSvc.sessionToken)
	}
}

func TestAuthLogoutEndpointRejectsGET(t *testing.T) {
	authSvc := &stubAuthService{}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "refresh-old"})
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if authSvc.logoutToken != "" {
		t.Fatalf("logout token = %q, want empty because GET must not hit logout handler", authSvc.logoutToken)
	}
}

func TestAuthGoogleStartEndpointRedirectsToProvider(t *testing.T) {
	authSvc := &stubAuthService{
		googleStartURL: "https://accounts.google.com/o/oauth2/v2/auth?state=abc",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/start?next=%2Fdashboard", nil)
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTemporaryRedirect)
	}
	if got := rec.Header().Get("Location"); got != authSvc.googleStartURL {
		t.Fatalf("location = %q, want %q", got, authSvc.googleStartURL)
	}
	if authSvc.googleStartReq.NextPath != "/dashboard" {
		t.Fatalf("next_path = %q, want /dashboard", authSvc.googleStartReq.NextPath)
	}
	if authSvc.googleStartReq.RedirectURL != "http://example.com/api/auth/google/callback" {
		t.Fatalf("redirect_url = %q, want request-derived callback", authSvc.googleStartReq.RedirectURL)
	}
}

func TestAuthGoogleLinkStartEndpointRequiresSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google/link/start?next=%2Fdashboard", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, &stubAuthService{}, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthGoogleLinkStartEndpointRequiresAllowedOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google/link/start?next=%2Fdashboard", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, &stubAuthService{}, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthGoogleLinkStartEndpointReturnsGoogleURL(t *testing.T) {
	authSvc := &stubAuthService{
		googleLinkURL: "https://accounts.google.com/o/oauth2/v2/auth?state=link-123",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google/link/start?next=%2Fdashboard", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if authSvc.googleLinkReq.UserID != "user-123" {
		t.Fatalf("google link user_id = %q, want user-123", authSvc.googleLinkReq.UserID)
	}
	if authSvc.googleLinkReq.NextPath != "/dashboard" {
		t.Fatalf("google link next_path = %q, want /dashboard", authSvc.googleLinkReq.NextPath)
	}
	if authSvc.googleLinkReq.RedirectURL != "http://example.com/api/auth/google/callback" {
		t.Fatalf("google link redirect_url = %q, want request-derived callback", authSvc.googleLinkReq.RedirectURL)
	}

	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.URL != authSvc.googleLinkURL {
		t.Fatalf("payload url = %q, want %q", payload.URL, authSvc.googleLinkURL)
	}
}

func TestAuthGoogleCallbackEndpointSetsCookiesAndRedirects(t *testing.T) {
	authSvc := &stubAuthService{
		googleCBResp: auth.GoogleCallbackResult{
			RedirectPath: "http://localhost:3000/dashboard",
			Linked:       true,
			Session: &auth.Session{
				Token:     "google-session",
				ExpiresAt: time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
				User: auth.UserSession{
					UserID:   "teacher-1",
					TenantID: "tenant-abc",
					Role:     auth.RoleTeacher,
					Name:     "Teacher One",
					Email:    "teacher@example.com",
				},
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?state=state-123&code=code-123", nil)
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "http://localhost:3000/dashboard?auth_provider=google&identity_linked=google" {
		t.Fatalf("location = %q, want google success redirect", got)
	}
	if authSvc.googleCBReq.State != "state-123" || authSvc.googleCBReq.Code != "code-123" {
		t.Fatalf("callback request = %#v", authSvc.googleCBReq)
	}
	if authSvc.googleCBReq.RedirectURL != "http://example.com/api/auth/google/callback" {
		t.Fatalf("callback redirect_url = %q, want request-derived callback", authSvc.googleCBReq.RedirectURL)
	}
	assertAuthCookies(t, rec.Result().Cookies(), "google-session")
}

func TestGoogleCallbackURLUsesForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/start", nil)
	req.Host = "internal:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "admin.pandai.org")

	if got := googleCallbackURL(req); got != "https://admin.pandai.org/api/auth/google/callback" {
		t.Fatalf("googleCallbackURL() = %q, want forwarded public callback", got)
	}
}

func TestAuthGoogleCallbackEndpointPromotesLoginRedirectToDefaultRoute(t *testing.T) {
	authSvc := &stubAuthService{
		googleCBResp: auth.GoogleCallbackResult{
			RedirectPath: "http://localhost:3000/login",
			Linked:       true,
			Session: &auth.Session{
				Token:     "google-session",
				ExpiresAt: time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
				User: auth.UserSession{
					UserID:   "teacher-1",
					TenantID: "tenant-abc",
					Role:     auth.RoleTeacher,
					Name:     "Teacher One",
					Email:    "teacher@example.com",
				},
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?state=state-123&code=code-123", nil)
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "http://localhost:3000/dashboard?auth_provider=google&identity_linked=google" {
		t.Fatalf("location = %q, want promoted dashboard redirect", got)
	}
	assertAuthCookies(t, rec.Result().Cookies(), "google-session")
}

func TestAuthGoogleCallbackEndpointKeepsSafeWorkspaceRedirect(t *testing.T) {
	authSvc := &stubAuthService{
		googleCBResp: auth.GoogleCallbackResult{
			RedirectPath: "http://localhost:3000/parents/parent-1?tab=progress",
			Linked:       true,
			Session: &auth.Session{
				Token:     "google-session",
				ExpiresAt: time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
				User: auth.UserSession{
					UserID:   "parent-1",
					TenantID: "tenant-abc",
					Role:     auth.RoleParent,
					Name:     "Parent One",
					Email:    "parent@example.com",
				},
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?state=state-123&code=code-123", nil)
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "http://localhost:3000/parents/parent-1?auth_provider=google&identity_linked=google&tab=progress" {
		t.Fatalf("location = %q, want safe workspace redirect", got)
	}
	assertAuthCookies(t, rec.Result().Cookies(), "google-session")
}

func TestAuthGoogleCallbackEndpointRedirectsErrorsToFrontend(t *testing.T) {
	authSvc := &stubAuthService{
		googleCBResp: auth.GoogleCallbackResult{
			RedirectPath: "http://localhost:3000/dashboard",
		},
		googleCBErr: auth.ErrAuthFlowInvalid,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?state=state-123&code=code-123", nil)
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "http://localhost:3000/dashboard?auth_error=flow_invalid" {
		t.Fatalf("location = %q, want frontend error redirect", got)
	}
}

func TestAuthIdentitiesEndpointReturnsLinkedProviders(t *testing.T) {
	linkedAt := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	authSvc := &stubAuthService{
		identitiesResp: []auth.LinkedIdentity{
			{
				Provider: "google",
				Email:    "teacher@gmail.com",
				LinkedAt: &linkedAt,
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/identities", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if authSvc.identitiesUserID != "user-123" {
		t.Fatalf("identities user_id = %q, want user-123", authSvc.identitiesUserID)
	}
	var payload struct {
		Identities []auth.LinkedIdentity `json:"identities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Identities) != 1 || payload.Identities[0].Provider != "google" {
		t.Fatalf("payload identities = %#v, want one google identity", payload.Identities)
	}
}

func TestAdminInviteEndpoint(t *testing.T) {
	authSvc := &stubAuthService{
		inviteResp: auth.InviteRecord{
			Email:       "newteacher@example.com",
			Role:        auth.RoleTeacher,
			Token:       "invite-token-123",
			ExpiresAt:   time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
			InvitedByID: "user-123",
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/invites", strings.NewReader(`{"email":"newteacher@example.com","role":"teacher"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if authSvc.inviteReq.Email != "newteacher@example.com" {
		t.Fatalf("invite email = %q, want newteacher@example.com", authSvc.inviteReq.Email)
	}
	if authSvc.inviteReq.InvitedByUserID != "user-123" {
		t.Fatalf("invited_by = %q, want user-123", authSvc.inviteReq.InvitedByUserID)
	}
}

func TestAdminInviteEndpointRequiresAdminRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/invites", strings.NewReader(`{"email":"newteacher@example.com","role":"teacher"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, &stubAuthService{}, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAdminInviteEndpointValidatesBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/invites", strings.NewReader(`{"email":"","role":"teacher"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, &stubAuthService{}, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthLoginEndpointMapsInvalidCredentialsToUnauthorized(t *testing.T) {
	authSvc := &stubAuthService{loginErr: auth.ErrInvalidCredentials}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"teacher@example.com","password":"bad-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthLoginEndpointMapsTenantRequiredToBadRequest(t *testing.T) {
	authSvc := &stubAuthService{loginErr: auth.NewTenantRequiredError([]auth.TenantOption{
		{TenantID: "tenant-a", TenantSlug: "school-a", TenantName: "School A"},
		{TenantID: "tenant-b", TenantSlug: "school-b", TenantName: "School B"},
	})}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"teacher@example.com","password":"secret-123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, authSvc, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var payload struct {
		Error   string              `json:"error"`
		Tenants []auth.TenantOption `json:"tenants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Error == "" {
		t.Fatal("expected error message in payload")
	}
	if len(payload.Tenants) != 2 {
		t.Fatalf("tenant choices = %d, want 2", len(payload.Tenants))
	}
}

func TestAuthEndpointsValidateJSONBody(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "login missing password", path: "/api/auth/login", body: `{"email":"teacher@example.com"}`},
		{name: "accept invite missing token", path: "/api/auth/invitations/accept", body: `{"name":"Teacher","password":"secret-123"}`},
		{name: "switch tenant missing token", path: "/api/auth/switch-tenant", body: `{"tenant_id":"tenant-b","password":"secret-123"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			newHandlerWithServices(stubAdminAPI{}, &chatGatewayStub{}, &stubAuthService{}, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func assertAuthCookies(t *testing.T, cookies []*http.Cookie, wantSession string) {
	t.Helper()

	session := findCookie(cookies, auth.SessionCookieName)
	if session == nil || session.Value != wantSession || !session.HttpOnly {
		t.Fatalf("session cookie = %#v, want value %q and HttpOnly", session, wantSession)
	}
	authCookieCount := 0
	for _, cookie := range cookies {
		if cookie.Name == auth.SessionCookieName {
			authCookieCount++
		}
	}
	if authCookieCount != 1 {
		t.Fatalf("auth cookie count = %d, want 1", authCookieCount)
	}
}

func assertExpiredAuthCookies(t *testing.T, cookies []*http.Cookie) {
	t.Helper()

	cookie := findCookie(cookies, auth.SessionCookieName)
	if cookie == nil || cookie.MaxAge != -1 {
		t.Fatalf("cookie %q = %#v, want expired cookie", auth.SessionCookieName, cookie)
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

type stubAdminAPI struct{}

func (stubAdminAPI) GetClassProgress(_ string) (adminapi.ClassProgress, error) {
	return adminapi.ClassProgress{
		Students: []adminapi.ClassStudent{
			{ID: "stu_1", Name: "Alya Sofea", Topics: map[string]float64{"linear-equations": 0.86}},
			{ID: "stu_2", Name: "Hakim Firdaus", Topics: map[string]float64{"linear-equations": 0.38}},
			{ID: "stu_3", Name: "Mei Lin", Topics: map[string]float64{"linear-equations": 0.92}},
		},
		TopicIDs: []string{"linear-equations", "algebraic-expressions", "inequalities", "functions"},
	}, nil
}

func (stubAdminAPI) GetStudentDetail(studentID string) (adminapi.StudentDetail, error) {
	if studentID == "missing" {
		return adminapi.StudentDetail{}, adminapi.ErrNotFound
	}
	next := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	last := time.Date(2026, 3, 9, 11, 20, 0, 0, time.UTC)
	externalID := "stu_1"
	if studentID == "tg_student" {
		externalID = "123456789"
	}
	return adminapi.StudentDetail{
		Student: adminapi.Student{
			ID:         studentID,
			Name:       "Alya Sofea",
			ExternalID: externalID,
			Channel:    "telegram",
			Form:       "Form 1",
			CreatedAt:  time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC),
		},
		Progress: []adminapi.ProgressItem{
			{TopicID: "linear-equations", MasteryScore: 0.86, EaseFactor: 2.5, IntervalDays: 6, NextReviewAt: &next, LastStudiedAt: &last},
			{TopicID: "algebraic-expressions", MasteryScore: 0.62, EaseFactor: 2.2, IntervalDays: 4, NextReviewAt: &next, LastStudiedAt: &last},
			{TopicID: "inequalities", MasteryScore: 0.44, EaseFactor: 1.9, IntervalDays: 2, NextReviewAt: &next, LastStudiedAt: &last},
			{TopicID: "functions", MasteryScore: 0.30, EaseFactor: 1.8, IntervalDays: 1, NextReviewAt: &next, LastStudiedAt: &last},
		},
		Streak: adminapi.StreakSummary{Current: 5, Longest: 9, TotalXP: 1240},
	}, nil
}

func (stubAdminAPI) GetStudentConversations(studentID string) ([]adminapi.StudentConversation, error) {
	if studentID == "missing" {
		return nil, adminapi.ErrNotFound
	}
	return []adminapi.StudentConversation{
		{ID: "msg_1", Timestamp: time.Date(2026, 3, 9, 11, 20, 0, 0, time.UTC), Role: "student", Text: "Question"},
		{ID: "msg_2", Timestamp: time.Date(2026, 3, 9, 11, 20, 12, 0, time.UTC), Role: "assistant", Text: "Answer"},
	}, nil
}

func (stubAdminAPI) GetParentSummary(parentID string) (adminapi.ParentSummary, error) {
	if parentID == "missing" {
		return adminapi.ParentSummary{}, adminapi.ErrNotFound
	}

	next := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	last := time.Date(2026, 3, 9, 11, 20, 0, 0, time.UTC)

	return adminapi.ParentSummary{
		Parent: adminapi.Parent{
			ID:        "parent-1",
			Name:      "Farah Parent",
			Email:     "parent@example.com",
			ChildIDs:  []string{"stu_1"},
			CreatedAt: time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC),
		},
		Child: adminapi.Student{
			ID:         "stu_1",
			Name:       "Alya Sofea",
			ExternalID: "stu_1",
			Channel:    "telegram",
			Form:       "Form 1",
			CreatedAt:  time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC),
		},
		Streak: adminapi.StreakSummary{Current: 5, Longest: 9, TotalXP: 1240},
		WeeklyStats: adminapi.WeeklyStats{
			DaysActive:        5,
			MessagesExchanged: 18,
			QuizzesCompleted:  3,
			NeedsReviewCount:  2,
		},
		Mastery: []adminapi.ProgressItem{
			{TopicID: "linear-equations", MasteryScore: 0.86, EaseFactor: 2.5, IntervalDays: 6, NextReviewAt: &next, LastStudiedAt: &last},
			{TopicID: "algebraic-expressions", MasteryScore: 0.62, EaseFactor: 2.2, IntervalDays: 4, NextReviewAt: &next, LastStudiedAt: &last},
			{TopicID: "inequalities", MasteryScore: 0.44, EaseFactor: 1.9, IntervalDays: 2, NextReviewAt: &next, LastStudiedAt: &last},
			{TopicID: "functions", MasteryScore: 0.30, EaseFactor: 1.8, IntervalDays: 1, NextReviewAt: &next, LastStudiedAt: &last},
		},
		Encouragement: adminapi.EncouragementSuggestion{
			Headline: "Alya is keeping the habit alive.",
			Text:     "Celebrate the 5-day streak and encourage one short review on inequalities this week.",
		},
	}, nil
}

func (stubAdminAPI) GetAIUsage() (adminapi.AIUsageSummary, error) {
	return adminapi.AIUsageSummary{
		TotalMessages:     6,
		TotalInputTokens:  226,
		TotalOutputTokens: 187,
		Providers: []adminapi.AIProviderUsage{
			{Provider: "openai", Model: "gpt-4o-mini", Messages: 4, InputTokens: 168, OutputTokens: 126, TotalTokens: 294},
			{Provider: "anthropic", Model: "claude-3-5-haiku", Messages: 2, InputTokens: 58, OutputTokens: 61, TotalTokens: 119},
		},
	}, nil
}

func (stubAdminAPI) UpsertTenantTokenBudgetWindow(_ adminapi.UpsertTokenBudgetWindowRequest) (adminapi.AIUsageSummary, error) {
	return adminapi.AIUsageSummary{
		BudgetLimitTokens:     int64Ptr(250000),
		BudgetUsedTokens:      int64Ptr(0),
		BudgetRemainingTokens: int64Ptr(250000),
		BudgetPeriodStart:     "2026-04-01",
		BudgetPeriodEnd:       "2026-04-30",
	}, nil
}

func (stubAdminAPI) GetMetrics() (adminapi.MetricsSummary, error) {
	return adminapi.MetricsSummary{
		WindowDays: 14,
		DailyActiveUsers: []adminapi.DailyActiveUsersPoint{
			{Date: "2026-03-10", Users: 17},
			{Date: "2026-03-11", Users: 19},
		},
		Retention: []adminapi.RetentionPoint{
			{CohortDate: "2026-03-01", CohortSize: 10, Day1Rate: 0.8, Day7Rate: 0.6, Day14Rate: 0.4},
		},
		NudgeRate: adminapi.NudgeRateSummary{
			NudgesSent:             14,
			ResponsesWithin24Hours: 5,
			ResponseRate:           5.0 / 14.0,
		},
		AIUsage: adminapi.AIUsageSummary{
			TotalMessages:     6,
			TotalInputTokens:  226,
			TotalOutputTokens: 187,
			Providers: []adminapi.AIProviderUsage{
				{Provider: "openai", Model: "gpt-4o-mini", Messages: 4, InputTokens: 168, OutputTokens: 126, TotalTokens: 294},
				{Provider: "anthropic", Model: "claude-3-5-haiku", Messages: 2, InputTokens: 58, OutputTokens: 61, TotalTokens: 119},
			},
		},
	}, nil
}

var _ adminDataSource = stubAdminAPI{}

type recordingAdminProvider struct {
	tenantIDs []string
}

func (p *recordingAdminProvider) ForRequest(r *http.Request) (adminDataSource, error) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return nil, errors.New("missing auth claims")
	}

	p.tenantIDs = append(p.tenantIDs, claims.TenantID)
	return stubAdminAPI{}, nil
}

type chatGatewayStub struct {
	messages []outboundMessage
}

type budgetAdminStub struct {
	stubAdminAPI
	req adminapi.UpsertTokenBudgetWindowRequest
}

func (s *budgetAdminStub) UpsertTenantTokenBudgetWindow(req adminapi.UpsertTokenBudgetWindowRequest) (adminapi.AIUsageSummary, error) {
	s.req = req
	return adminapi.AIUsageSummary{
		BudgetLimitTokens:     int64Ptr(req.BudgetTokens),
		BudgetUsedTokens:      int64Ptr(0),
		BudgetRemainingTokens: int64Ptr(req.BudgetTokens),
		BudgetPeriodStart:     req.PeriodStart,
		BudgetPeriodEnd:       req.PeriodEnd,
	}, nil
}

func (c *chatGatewayStub) Send(_ context.Context, msg outboundMessage) error {
	c.messages = append(c.messages, msg)
	return nil
}

type stubAuthService struct {
	loginReq         auth.LoginRequest
	loginResp        auth.Session
	loginErr         error
	inviteReq        auth.IssueInviteRequest
	inviteResp       auth.InviteRecord
	inviteErr        error
	acceptReq        auth.AcceptInviteRequest
	acceptResp       auth.Session
	acceptErr        error
	sessionToken     string
	sessionResp      auth.Session
	sessionErr       error
	switchToken      string
	switchTenantID   string
	switchPassword   string
	switchResp       auth.Session
	switchErr        error
	googleStartReq   auth.StartGoogleFlowRequest
	googleStartURL   string
	googleStartErr   error
	googleLinkReq    auth.StartGoogleFlowRequest
	googleLinkURL    string
	googleLinkErr    error
	googleCBReq      auth.GoogleCallbackRequest
	googleCBResp     auth.GoogleCallbackResult
	googleCBErr      error
	identitiesUserID string
	identitiesResp   []auth.LinkedIdentity
	identitiesErr    error
	logoutToken      string
	logoutErr        error
}

func (s *stubAuthService) Login(_ context.Context, req auth.LoginRequest) (auth.Session, error) {
	s.loginReq = req
	return s.loginResp, s.loginErr
}

func (s *stubAuthService) AcceptInvite(_ context.Context, req auth.AcceptInviteRequest) (auth.Session, error) {
	s.acceptReq = req
	return s.acceptResp, s.acceptErr
}

func (s *stubAuthService) IssueInvite(_ context.Context, req auth.IssueInviteRequest) (auth.InviteRecord, error) {
	s.inviteReq = req
	return s.inviteResp, s.inviteErr
}

func (s *stubAuthService) Session(_ context.Context, sessionToken string) (auth.Session, error) {
	s.sessionToken = sessionToken
	return s.sessionResp, s.sessionErr
}

func (s *stubAuthService) SwitchTenant(_ context.Context, sessionToken, tenantID, password string) (auth.Session, error) {
	s.switchToken = sessionToken
	s.switchTenantID = tenantID
	s.switchPassword = password
	return s.switchResp, s.switchErr
}

func (s *stubAuthService) StartGoogleLogin(_ context.Context, req auth.StartGoogleFlowRequest) (string, error) {
	s.googleStartReq = req
	return s.googleStartURL, s.googleStartErr
}

func (s *stubAuthService) StartGoogleLink(_ context.Context, req auth.StartGoogleFlowRequest) (string, error) {
	s.googleLinkReq = req
	return s.googleLinkURL, s.googleLinkErr
}

func (s *stubAuthService) CompleteGoogleCallback(_ context.Context, req auth.GoogleCallbackRequest) (auth.GoogleCallbackResult, error) {
	s.googleCBReq = req
	return s.googleCBResp, s.googleCBErr
}

func (s *stubAuthService) ListLinkedIdentities(_ context.Context, userID string) ([]auth.LinkedIdentity, error) {
	s.identitiesUserID = userID
	return s.identitiesResp, s.identitiesErr
}

func int64Ptr(v int64) *int64 {
	return &v
}

func (s *stubAuthService) Logout(_ context.Context, sessionToken string) error {
	s.logoutToken = sessionToken
	return s.logoutErr
}

func TestWriteAuthError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "invalid credentials", err: auth.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized},
		{name: "invalid invite", err: auth.ErrInvalidInvite, wantStatus: http.StatusUnauthorized},
		{name: "expired invite", err: auth.ErrInviteExpired, wantStatus: http.StatusUnauthorized},
		{name: "not implemented", err: auth.ErrNotImplemented, wantStatus: http.StatusNotImplemented},
		{name: "provider unavailable", err: auth.ErrProviderNotConfigured, wantStatus: http.StatusNotImplemented},
		{name: "identity already linked", err: auth.ErrIdentityAlreadyLinked, wantStatus: http.StatusConflict},
		{name: "domain not allowed", err: auth.ErrGoogleDomainNotAllowed, wantStatus: http.StatusForbidden},
		{name: "link required", err: auth.ErrIdentityLinkRequired, wantStatus: http.StatusBadRequest},
		{name: "flow invalid", err: auth.ErrAuthFlowInvalid, wantStatus: http.StatusBadRequest},
		{name: "tenant required", err: auth.NewTenantRequiredError([]auth.TenantOption{{TenantID: "tenant-a", TenantSlug: "school-a", TenantName: "School A"}}), wantStatus: http.StatusBadRequest},
		{name: "validation", err: errors.New("bad request"), wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeAuthError(rec, tt.err)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestTelegramInlineKeyboardContext(t *testing.T) {
	tests := []struct {
		name string
		conv agent.Conversation
		want struct {
			intensity bool
			active    bool
			paused    bool
		}
	}{
		{
			name: "quiz intensity pending",
			conv: agent.Conversation{UserID: "u-intensity", State: "quiz_intensity"},
			want: struct {
				intensity bool
				active    bool
				paused    bool
			}{intensity: true},
		},
		{
			name: "quiz active",
			conv: agent.Conversation{
				UserID: "u-active",
				State:  "quiz_active",
				QuizState: &agent.ConversationQuizState{
					RunState: "active",
				},
			},
			want: struct {
				intensity bool
				active    bool
				paused    bool
			}{active: true},
		},
		{
			name: "quiz paused outside quiz state",
			conv: agent.Conversation{
				UserID: "u-paused",
				State:  "teaching",
				QuizState: &agent.ConversationQuizState{
					RunState: "paused",
				},
			},
			want: struct {
				intensity bool
				active    bool
				paused    bool
			}{paused: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := agent.NewMemoryStore()
			if _, err := store.CreateConversation(tt.conv); err != nil {
				t.Fatalf("CreateConversation() error = %v", err)
			}

			got := telegramInlineKeyboardContext(store, tt.conv.UserID)
			if got.QuizIntensityPending != tt.want.intensity || got.QuizActive != tt.want.active || got.QuizPaused != tt.want.paused {
				t.Fatalf("telegramInlineKeyboardContext() = %#v, want intensity=%v active=%v paused=%v", got, tt.want.intensity, tt.want.active, tt.want.paused)
			}
		})
	}
}

func TestAdminEndpointsUseTenantFromClaims(t *testing.T) {
	provider := &recordingAdminProvider{}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/classes/all-students/progress", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTokenWithTenant(t, auth.RoleTeacher, "teacher-1", "tenant-second"))
	rec := httptest.NewRecorder()

	newHandlerWithServicesAndAdminProvider(provider, &chatGatewayStub{}, &stubAuthService{}, "change-me-in-production", time.Hour).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(provider.tenantIDs) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(provider.tenantIDs))
	}
	if provider.tenantIDs[0] != "tenant-second" {
		t.Fatalf("tenant_id = %q, want tenant-second", provider.tenantIDs[0])
	}
}

func TestPlatformAdminRequestsUseGlobalAdminSource(t *testing.T) {
	provider := tenantAdminDataSourceProvider{
		newForTenant: func(tenantID string) adminDataSource {
			t.Fatalf("newForTenant(%q) should not be called for platform admin", tenantID)
			return nil
		},
		newForPlatform: func() adminDataSource {
			return stubAdminAPI{}
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/classes/all-students/progress", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.TokenClaims{
		Subject: "platform-user",
		Role:    auth.RolePlatformAdmin,
	}))

	admin, err := provider.ForRequest(req)
	if err != nil {
		t.Fatalf("ForRequest() error = %v", err)
	}
	if _, ok := admin.(stubAdminAPI); !ok {
		t.Fatalf("admin source = %T, want stubAdminAPI", admin)
	}
}

func mustIssueAdminToken(t *testing.T) string {
	t.Helper()
	return mustIssueToken(t, auth.RoleAdmin)
}

func mustIssueTeacherToken(t *testing.T) string {
	t.Helper()
	return mustIssueToken(t, auth.RoleTeacher)
}

func mustIssueParentToken(t *testing.T) string {
	t.Helper()
	return mustIssueTokenWithSubject(t, auth.RoleParent, "parent-1")
}

func mustIssueStudentToken(t *testing.T) string {
	t.Helper()
	return mustIssueToken(t, auth.RoleStudent)
}

func mustIssueToken(t *testing.T, role auth.Role) string {
	t.Helper()
	return mustIssueTokenWithTenant(t, role, "user-123", "tenant-abc")
}

func mustIssueTokenWithSubject(t *testing.T, role auth.Role, subject string) string {
	t.Helper()
	return mustIssueTokenWithTenant(t, role, subject, "tenant-abc")
}

func mustIssueTokenWithTenant(t *testing.T, role auth.Role, subject, tenantID string) string {
	t.Helper()

	manager := auth.NewTokenManager("change-me-in-production", time.Hour)
	now := time.Now().UTC()
	token, err := manager.Issue(auth.TokenClaims{
		Subject:  subject,
		TenantID: tenantID,
		Role:     role,
	}, now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	return token
}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

var _ adminDataSource = stubAdminAPI{}

type chatGatewayStub struct {
	messages []outboundMessage
}

func (c *chatGatewayStub) Send(_ context.Context, msg outboundMessage) error {
	c.messages = append(c.messages, msg)
	return nil
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

func mustIssueAdminToken(t *testing.T) string {
	t.Helper()
	return mustIssueToken(t, auth.RoleAdmin)
}

func mustIssueTeacherToken(t *testing.T) string {
	t.Helper()
	return mustIssueToken(t, auth.RoleTeacher)
}

func mustIssueStudentToken(t *testing.T) string {
	t.Helper()
	return mustIssueToken(t, auth.RoleStudent)
}

func mustIssueToken(t *testing.T, role auth.Role) string {
	t.Helper()

	manager := auth.NewTokenManager("change-me-in-production", time.Hour)
	now := time.Now().UTC()
	token, err := manager.Issue(auth.TokenClaims{
		Subject:  "user-123",
		TenantID: "tenant-abc",
		Role:     role,
	}, now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	return token
}

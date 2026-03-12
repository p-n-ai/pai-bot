package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestHealthEndpoints(t *testing.T) {
	mux := newMux()

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

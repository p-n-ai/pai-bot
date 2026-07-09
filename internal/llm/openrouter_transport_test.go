package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenRouterTeam/go-sdk/models/sdkerrors"
)

func TestOpenRouterTransportRequiresExactEventStreamMediaType(t *testing.T) {
	cases := []struct {
		contentType  string
		wantAPIError bool
	}{
		{contentType: "text/event-stream-evil", wantAPIError: true},
		{contentType: "text/event-stream; charset=utf-8", wantAPIError: false},
	}
	for _, tc := range cases {
		t.Run(tc.contentType, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				_, _ = io.WriteString(w, strings.Repeat("x", openRouterMaxErrorBody+1))
			}))
			t.Cleanup(srv.Close)
			model := Model{ID: "openai/gpt-test", API: APIOpenRouterChat, Provider: "openrouter", BaseURL: srv.URL + "/v1"}
			_, err := StreamOpenRouterChat(context.Background(), model, Context{Messages: []Message{UserText("hi")}}, &StreamOptions{APIKey: "test"}).Result()
			var apiErr *sdkerrors.APIError
			if errors.As(err, &apiErr) != tc.wantAPIError {
				t.Fatalf("APIError = %v, want %v: %v", apiErr != nil, tc.wantAPIError, err)
			}
			if apiErr != nil && len(apiErr.Body) > openRouterMaxErrorBody {
				t.Fatalf("body length = %d", len(apiErr.Body))
			}
		})
	}
}

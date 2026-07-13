package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTopMuxRoutesFocusedPageBeforeAPIFallback(t *testing.T) {
	pageHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.PathValue("publicID"); got != "page-1" {
			t.Fatalf("publicID = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	fallback := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	handler := NewTopMux(TopMuxOptions{APIHandler: fallback, FocusedPageHandler: pageHandler})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/a/page-1", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}

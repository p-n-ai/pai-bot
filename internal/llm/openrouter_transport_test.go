package llm

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type openRouterRoundTripFunc func(*http.Request) (*http.Response, error)

func (f openRouterRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestOpenRouterTransportRequiresExactEventStreamMediaType(t *testing.T) {
	cases := []struct {
		contentType string
		wantBytes   int
	}{
		{contentType: "text/event-stream-evil", wantBytes: openRouterMaxErrorBody},
		{contentType: "text/event-stream; charset=utf-8", wantBytes: openRouterMaxErrorBody + 1},
	}
	for _, tc := range cases {
		t.Run(tc.contentType, func(t *testing.T) {
			transport := openRouterTransport{base: openRouterRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{tc.contentType}},
					Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", openRouterMaxErrorBody+1))),
				}, nil
			})}
			resp, err := transport.RoundTrip(&http.Request{})
			if err != nil {
				t.Fatalf("RoundTrip: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if len(body) != tc.wantBytes {
				t.Fatalf("body length = %d, want %d", len(body), tc.wantBytes)
			}
		})
	}
}

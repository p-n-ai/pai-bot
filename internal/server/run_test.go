// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunServesHealthBeforeSwapAndBuiltHandlerAfterSwap(t *testing.T) {
	addr := freeLocalAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	allowBuild := make(chan struct{})
	built := make(chan struct{})
	started := make(chan struct{})
	runErr := make(chan error, 1)
	var startCalls atomic.Int32

	go func() {
		runErr <- Run(ctx, Options{
			Addr:            addr,
			ReadTimeout:     time.Second,
			WriteTimeout:    time.Second,
			IdleTimeout:     time.Second,
			ShutdownTimeout: time.Second,
			BuildHandler: func(context.Context) (http.Handler, func(context.Context) error, error) {
				<-allowBuild
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("full handler"))
				})
				return h, func(context.Context) error {
					startCalls.Add(1)
					close(started)
					return nil
				}, nil
			},
		})
	}()

	if body := getEventually(t, "http://"+addr+"/healthz"); body != `{"status":"ok"}` {
		t.Fatalf("health body before swap = %q, want health JSON", body)
	}
	getStatusEventually(t, "http://"+addr+"/readyz", http.StatusServiceUnavailable)

	close(allowBuild)
	go func() {
		<-started
		close(built)
	}()
	select {
	case <-built:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for post-swap start hook")
	}

	if body := getEventually(t, "http://"+addr+"/anything"); body != "full handler" {
		t.Fatalf("body after swap = %q, want full handler", body)
	}
	if got := startCalls.Load(); got != 1 {
		t.Fatalf("start hook calls = %d, want 1", got)
	}

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run shutdown")
	}
}

func TestRunRejectsUnsafeOptions(t *testing.T) {
	valid := Options{
		Addr:            "127.0.0.1:8080",
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
		BuildHandler: func(context.Context) (http.Handler, func(context.Context) error, error) {
			return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), nil, nil
		},
	}

	tests := []struct {
		name   string
		mutate func(*Options)
	}{
		{name: "empty address", mutate: func(opts *Options) { opts.Addr = "" }},
		{name: "zero read timeout", mutate: func(opts *Options) { opts.ReadTimeout = 0 }},
		{name: "negative write timeout", mutate: func(opts *Options) { opts.WriteTimeout = -time.Second }},
		{name: "zero idle timeout", mutate: func(opts *Options) { opts.IdleTimeout = 0 }},
		{name: "zero shutdown timeout", mutate: func(opts *Options) { opts.ShutdownTimeout = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := valid
			tt.mutate(&opts)
			if err := Run(t.Context(), opts); err == nil {
				t.Fatal("Run() succeeded with unsafe options")
			}
		})
	}
}

func TestBootstrapHealthOnlyServesInternalLiveness(t *testing.T) {
	tests := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK},
		{method: http.MethodGet, path: "/readyz", wantStatus: http.StatusServiceUnavailable},
		{method: http.MethodGet, path: "/health", wantStatus: http.StatusNotFound},
		{method: http.MethodGet, path: "/health/api", wantStatus: http.StatusNotFound},
		{method: http.MethodGet, path: "/health/status", wantStatus: http.StatusNotFound},
		{method: http.MethodGet, path: "/health/ai", wantStatus: http.StatusNotFound},
		{method: http.MethodPost, path: "/healthz", wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			handleBootstrapHealth(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestRunCancelsPostSwapWorkWhenServerExits(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close reserved listener: %v", err)
		}
	})

	worker, err := NewFocusedPageCleanupWorker(&blockingFocusedPageCleaner{started: make(chan struct{}, 1)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	workerDone := make(chan struct{})
	started := make(chan struct{})
	runErr := Run(context.Background(), Options{
		Addr:            listener.Addr().String(),
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
		BuildHandler: func(context.Context) (http.Handler, func(context.Context) error, error) {
			return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), func(ctx context.Context) error {
				close(started)
				go func() {
					worker.run(ctx, make(chan time.Time))
					close(workerDone)
				}()
				return nil
			}, nil
		},
	})
	if runErr == nil {
		t.Fatal("Run succeeded despite an unavailable listen address")
	}
	select {
	case <-started:
	default:
		t.Fatal("post-swap work was not started")
	}
	select {
	case <-workerDone:
	case <-time.After(time.Second):
		t.Fatal("post-swap worker outlived the server")
	}
}

func freeLocalAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

func getEventually(t *testing.T, url string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				t.Fatalf("read body: %v", readErr)
			}
			if resp.StatusCode == http.StatusOK {
				return strings.TrimSpace(string(body))
			}
			lastErr = fmt.Errorf("status %d body %q", resp.StatusCode, string(body))
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("GET %s did not succeed: %v", url, lastErr)
	return ""
}

func getStatusEventually(t *testing.T, url string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == want {
				return
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("GET %s did not return %d: %v", url, want, lastErr)
}

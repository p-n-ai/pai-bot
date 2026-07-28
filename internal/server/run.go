// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

type Options struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	BuildHandler    func(context.Context) (http.Handler, func(context.Context) error, error)
}

func Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := validateOptions(opts); err != nil {
		return err
	}
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	var handler atomic.Pointer[http.Handler]
	initialHandler := http.Handler(http.HandlerFunc(handleBootstrapHealth))
	handler.Store(&initialHandler)

	srv := &http.Server{
		Addr: opts.Addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			(*handler.Load()).ServeHTTP(w, r)
		}),
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		IdleTimeout:  opts.IdleTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	fullHandler, startAfterSwap, err := opts.BuildHandler(runCtx)
	if err != nil {
		return shutdownAfterStartupError(srv, opts.ShutdownTimeout, fmt.Errorf("build handler: %w", err))
	}
	if fullHandler == nil {
		return shutdownAfterStartupError(srv, opts.ShutdownTimeout, errors.New("build handler returned nil handler"))
	}

	handler.Store(&fullHandler)
	slog.Info("full handler active")

	if startAfterSwap != nil {
		if err := startAfterSwap(runCtx); err != nil {
			return shutdownAfterStartupError(srv, opts.ShutdownTimeout, fmt.Errorf("start after swap: %w", err))
		}
	}

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), opts.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := <-serveErr; err != nil {
		return err
	}
	return nil
}

func validateOptions(opts Options) error {
	if opts.Addr == "" {
		return errors.New("server address is required")
	}
	if opts.ReadTimeout <= 0 {
		return errors.New("read timeout must be positive")
	}
	if opts.WriteTimeout <= 0 {
		return errors.New("write timeout must be positive")
	}
	if opts.IdleTimeout <= 0 {
		return errors.New("idle timeout must be positive")
	}
	if opts.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}
	if opts.BuildHandler == nil {
		return errors.New("build handler is required")
	}
	return nil
}

func handleBootstrapHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
		handleHealthz(w, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/readyz" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"starting"}`))
		return
	}
	http.NotFound(w, r)
}

func shutdownAfterStartupError(srv *http.Server, timeout time.Duration, runErr error) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return errors.Join(runErr, fmt.Errorf("shutdown: %w", err))
	}
	return runErr
}

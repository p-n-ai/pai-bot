package main

import "testing"

func TestParseCLIFlags(t *testing.T) {
	cfg, err := parseCLIFlags("ingest", []string{
		"--query",
		"pola jujukan",
		"--cache",
		"/tmp/retrieval-cache",
	})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if cfg.Query != "pola jujukan" {
		t.Fatalf("query = %q", cfg.Query)
	}
	if cfg.CacheDir != "/tmp/retrieval-cache" {
		t.Fatalf("cache dir = %q", cfg.CacheDir)
	}
}

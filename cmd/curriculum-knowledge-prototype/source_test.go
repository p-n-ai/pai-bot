package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractZipFindsArchiveRoot(t *testing.T) {
	archive := newZip(t, map[string]string{
		"oss-main/curricula/malaysia/malaysia-kssm/syllabus.yaml": `
id: malaysia-kssm
name: KSSM
subjects: []
`,
	})

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "source.zip")
	if err := os.WriteFile(archivePath, archive, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	extractDir := filepath.Join(dir, "extract")
	if err := extractZip(archivePath, extractDir); err != nil {
		t.Fatalf("extract source: %v", err)
	}
	root, err := singleExtractedRoot(extractDir)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "curricula", "malaysia", "malaysia-kssm", "syllabus.yaml")); err != nil {
		t.Fatalf("expected extracted syllabus: %v", err)
	}
}

func TestArchiveURLForGitHubSource(t *testing.T) {
	got, err := archiveURLForSource("https://github.com/p-n-ai/oss")
	if err != nil {
		t.Fatalf("archive url: %v", err)
	}
	if !strings.Contains(got, "https://codeload.github.com/p-n-ai/oss/zip/refs/heads/main") {
		t.Fatalf("unexpected archive url: %s", got)
	}
}

func TestParseGitHubSourceKeepsTreeSubpath(t *testing.T) {
	got, err := parseGitHubSource("https://github.com/p-n-ai/oss/tree/main/curricula/malaysia")
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	if !strings.Contains(got.ArchiveURL, "https://codeload.github.com/p-n-ai/oss/zip/refs/heads/main") {
		t.Fatalf("unexpected archive url: %s", got.ArchiveURL)
	}
	if got.Subpath != filepath.Join("curricula", "malaysia") {
		t.Fatalf("subpath = %q", got.Subpath)
	}
}

func TestArchiveURLRejectsNonGitHubHTTPSource(t *testing.T) {
	if _, err := archiveURLForSource("https://example.com/curriculum.zip"); err == nil {
		t.Fatal("expected non-github http source to fail")
	}
}

func TestArchiveURLRejectsLocalPath(t *testing.T) {
	if _, err := archiveURLForSource("/tmp/curriculum"); err == nil {
		t.Fatal("expected local path source to fail")
	}
}

func TestExtractZipRejectsPathTraversal(t *testing.T) {
	archive := newZip(t, map[string]string{
		"../escape.txt": "nope",
	})

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "source.zip")
	if err := os.WriteFile(archivePath, archive, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	if err := extractZip(archivePath, filepath.Join(dir, "extract")); err == nil {
		t.Fatal("expected unsafe archive path to fail")
	}
}

func newZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip file: %v", err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write zip file: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

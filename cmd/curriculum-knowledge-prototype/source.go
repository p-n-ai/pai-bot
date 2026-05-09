package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func materializeSource(ctx context.Context, source string, cacheDir string) (string, func(), error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", func() {}, fmt.Errorf("curriculum source is empty")
	}

	sourceRef, err := parseGitHubSource(source)
	if err != nil {
		return "", func() {}, err
	}

	workDir, err := os.MkdirTemp(cacheDir, "curriculum-source-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create source cache: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(workDir) }

	archivePath := filepath.Join(workDir, "source.zip")
	if err := downloadArchive(ctx, sourceRef.ArchiveURL, archivePath); err != nil {
		cleanup()
		return "", func() {}, err
	}

	extractDir := filepath.Join(workDir, "extract")
	if err := extractZip(archivePath, extractDir); err != nil {
		cleanup()
		return "", func() {}, err
	}

	root, err := singleExtractedRoot(extractDir)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	if sourceRef.Subpath != "" {
		root = filepath.Join(root, sourceRef.Subpath)
		if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
			cleanup()
			return "", func() {}, fmt.Errorf("github source path not found: %s", sourceRef.Subpath)
		}
	}
	return root, cleanup, nil
}

func archiveURLForSource(source string) (string, error) {
	sourceRef, err := parseGitHubSource(source)
	if err != nil {
		return "", err
	}
	return sourceRef.ArchiveURL, nil
}

type githubSource struct {
	ArchiveURL string
	Subpath    string
}

func parseGitHubSource(source string) (githubSource, error) {
	parsed, err := url.Parse(source)
	if err != nil || parsed.Scheme == "" {
		return githubSource{}, fmt.Errorf("source must be a GitHub repository URL")
	}
	if parsed.Host != "github.com" {
		return githubSource{}, fmt.Errorf("github source must use github.com")
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return githubSource{}, fmt.Errorf("github source must include owner and repo")
	}

	branch := "main"
	subpath := ""
	if len(parts) >= 4 && parts[2] == "tree" {
		branch = parts[3]
		if len(parts) > 4 {
			subpath = filepath.Join(parts[4:]...)
		}
	}
	return githubSource{
		ArchiveURL: fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/heads/%s", parts[0], parts[1], branch),
		Subpath:    subpath,
	}, nil
}

func downloadArchive(ctx context.Context, sourceURL string, dst string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fmt.Errorf("create source request: %w", err)
	}

	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download source archive: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download source archive: status %d", resp.StatusCode)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return fmt.Errorf("write archive file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close archive file: %w", err)
	}
	return nil
}

func extractZip(src string, dst string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open source archive: %w", err)
	}
	defer func() { _ = reader.Close() }()

	for _, file := range reader.File {
		cleanName := filepath.Clean(file.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			return fmt.Errorf("unsafe archive path")
		}

		target := filepath.Join(dst, cleanName)
		if !isPathInside(dst, target) {
			return fmt.Errorf("unsafe archive path")
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create archive directory: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create archive parent: %w", err)
		}

		if err := extractZipFile(file, target); err != nil {
			return err
		}
	}
	return nil
}

func isPathInside(root string, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func extractZipFile(file *zip.File, target string) error {
	in, err := file.Open()
	if err != nil {
		return fmt.Errorf("open archive member: %w", err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create extracted file: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("extract archive member: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close extracted file: %w", err)
	}
	return nil
}

func singleExtractedRoot(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("read extracted source: %w", err)
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(root, entries[0].Name()), nil
	}
	return root, nil
}

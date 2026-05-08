package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/platform/database"
)

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCLI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("command is required: ingest, search, or drop")
	}

	command := args[0]
	args = args[1:]

	switch command {
	case "ingest":
		cfg, err := parseCLIFlags(command, args)
		if err != nil {
			return err
		}
		return runPostgresPrototype(cfg.Query, cfg.CacheDir)
	case "search":
		cfg, err := parseCLIFlags(command, args)
		if err != nil {
			return err
		}
		return runPostgresSearch(cfg.Query)
	case "drop":
		return runPostgresDrop()
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

type cliConfig struct {
	Query    string
	CacheDir string
}

func parseCLIFlags(command string, args []string) (cliConfig, error) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	query := flags.String("query", "linear equations", "query to run")
	cacheDir := flags.String("cache", filepath.Join("tmp", "curriculum-knowledge-prototype"), "temporary source cache")
	if err := flags.Parse(args); err != nil {
		return cliConfig{}, err
	}
	return cliConfig{
		Query:    *query,
		CacheDir: *cacheDir,
	}, nil
}

func runPostgresPrototype(query string, cacheDir string) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		return fmt.Errorf("database url is required for postgres prototype")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := database.New(ctx, databaseURL, 4, 0)
	if err != nil {
		return err
	}
	defer db.Close()

	store := newContentStore(db.Pool)
	source := os.Getenv("CURRICULUM_SOURCE")
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("curriculum source is required for ingest")
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	root, cleanup, err := materializeSource(ctx, source, cacheDir)
	if err != nil {
		return err
	}
	defer cleanup()

	rows, err := buildCurriculumContentFromSource(root, source)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("no curriculum content found")
	}

	if err := store.applySchema(ctx); err != nil {
		return err
	}
	if err := store.replaceContent(ctx, rows); err != nil {
		return err
	}

	hits, err := store.searchText(ctx, query, 5)
	if err != nil {
		return err
	}

	fmt.Println("postgres prototype: PASS")
	fmt.Printf("content_rows: %d\n", len(rows))
	fmt.Printf("query: %q\n", query)
	fmt.Println("top_hits:")
	for _, hit := range hits {
		fmt.Printf("- score=%.4f kind=%s title=%q\n", hit.Score, hit.Content.Kind, hit.Content.Title)
	}
	return nil
}

func runPostgresSearch(query string) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		return fmt.Errorf("database url is required for postgres prototype")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.New(ctx, databaseURL, 4, 0)
	if err != nil {
		return err
	}
	defer db.Close()

	store := newContentStore(db.Pool)
	hits, err := store.searchText(ctx, query, 5)
	if err != nil {
		return err
	}

	fmt.Printf("query: %q\n", query)
	fmt.Println("top_hits:")
	for _, hit := range hits {
		fmt.Printf("- id=%d score=%.4f kind=%s title=%q\n", hit.Content.ID, hit.Score, hit.Content.Kind, hit.Content.Title)
	}
	return nil
}

func runPostgresDrop() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		return fmt.Errorf("database url is required for postgres prototype")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.New(ctx, databaseURL, 4, 0)
	if err != nil {
		return err
	}
	defer db.Close()

	store := newContentStore(db.Pool)
	if err := store.drop(ctx); err != nil {
		return err
	}
	fmt.Println("dropped retrieval_prototype schema")
	return nil
}

package main

import (
	"strings"
	"testing"
)

func TestSchemaSQLIsOneContentTable(t *testing.T) {
	sql := schemaSQL()

	if got := strings.Count(sql, "CREATE TABLE"); got != 1 {
		t.Fatalf("schema should create one table, got %d", got)
	}
	forbidden := []string{
		"source_url",
		"source_path",
		"external_id",
		"tenant_id",
		"owner",
		"curriculum_sources",
		"curriculum_chunks",
	}
	for _, term := range forbidden {
		if strings.Contains(sql, term) {
			t.Fatalf("schema should not contain %q", term)
		}
	}

	required := []string{
		"curriculum_content",
		"curriculum_source",
		"body",
		"search_text",
		"search_vector",
	}
	for _, term := range required {
		if !strings.Contains(sql, term) {
			t.Fatalf("schema should contain %q", term)
		}
	}
}

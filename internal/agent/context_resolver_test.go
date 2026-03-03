package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestCurriculumContextResolver_Resolve_MatchedTopicWithTeachingNotes(t *testing.T) {
	loader := createCurriculumLoaderForResolverTest(t, true)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.Resolve("Please teach me linear equations")
	if topic == nil {
		t.Fatal("expected matched topic, got nil")
	}
	if topic.ID != "F1-02" {
		t.Fatalf("topic.ID = %q, want F1-02", topic.ID)
	}
	if notes == "" {
		t.Fatal("expected teaching notes, got empty")
	}
	if !contains(notes, "subtract 5 on both sides") {
		t.Fatalf("teaching notes missing expected content, got: %s", notes)
	}
}

func TestCurriculumContextResolver_Resolve_NoMatch(t *testing.T) {
	loader := createCurriculumLoaderForResolverTest(t, true)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.Resolve("what is your favorite color")
	if topic != nil {
		t.Fatalf("expected no topic match, got %q", topic.ID)
	}
	if notes != "" {
		t.Fatalf("expected empty notes, got: %s", notes)
	}
}

func TestCurriculumContextResolver_Resolve_MatchWithoutTeachingNotes(t *testing.T) {
	loader := createCurriculumLoaderForResolverTest(t, false)
	resolver := agent.NewCurriculumContextResolver(loader)

	topic, notes := resolver.Resolve("help me with linear equations")
	if topic == nil {
		t.Fatal("expected matched topic, got nil")
	}
	if topic.ID != "F1-02" {
		t.Fatalf("topic.ID = %q, want F1-02", topic.ID)
	}
	if notes != "" {
		t.Fatalf("expected empty notes when missing teaching notes file, got: %s", notes)
	}
}

func createCurriculumLoaderForResolverTest(t *testing.T, withNotes bool) *curriculum.Loader {
	t.Helper()

	dir := t.TempDir()
	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	yamlPath := filepath.Join(topicsDir, "01-linear-equations.yaml")
	yamlData := `id: F1-02
name: Linear Equations
subject_id: math
syllabus_id: kssm-f1
difficulty: beginner
learning_objectives:
  - id: LO1
    text: Solve linear equations in one variable
    bloom: apply
`
	if err := os.WriteFile(yamlPath, []byte(yamlData), 0o644); err != nil {
		t.Fatalf("WriteFile(yaml) error = %v", err)
	}

	if withNotes {
		notesPath := filepath.Join(topicsDir, "01-linear-equations.teaching.md")
		notesData := "# Linear Equations Teaching Notes\nUse balance method and subtract 5 on both sides."
		if err := os.WriteFile(notesPath, []byte(notesData), 0o644); err != nil {
			t.Fatalf("WriteFile(notes) error = %v", err)
		}
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

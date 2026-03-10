package analyticsxlsx_test

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/analyticsxlsx"
	"github.com/xuri/excelize/v2"
)

func TestWriteWorkbookCreatesExpectedSheetsAndEscapesFormulaContent(t *testing.T) {
	exports := analyticsxlsx.BuildExampleExports()
	exports["conversation_messages"].Rows[0]["content"] = "=HYPERLINK(\"https://example.com\",\"boom\")"

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "analytics-example.xlsx")

	err := analyticsxlsx.WriteWorkbook(
		exports,
		outputPath,
		7,
		time.Date(2026, 3, 10, 1, 2, 3, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("WriteWorkbook() error = %v", err)
	}

	workbook, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("excelize.OpenFile() error = %v", err)
	}
	t.Cleanup(func() {
		_ = workbook.Close()
	})

	wantSheets := []string{
		"Overview",
		"Daily Activity",
		"Engagement",
		"Latency",
		"Usage",
		"Ratings",
		"Conversations",
		"Messages",
	}
	if got := workbook.GetSheetList(); !equalStrings(got, wantSheets) {
		t.Fatalf("GetSheetList() = %v, want %v", got, wantSheets)
	}

	assertCellValue(t, workbook, "Overview", "A1", "P&AI Analytics Report")
	assertCellValue(t, workbook, "Overview", "A5", "Latest DAU")
	assertCellValue(t, workbook, "Overview", "B5", "18")
	assertCellValue(t, workbook, "Overview", "A8", "Returning rate (%)")
	assertCellValue(t, workbook, "Overview", "B8", "66.67")
	assertCellValue(t, workbook, "Conversations", "A4", "Conversation Id")
	assertCellValue(t, workbook, "Conversations", "B5", "excel-demo-user")
	assertCellValue(t, workbook, "Conversations", "H5", "Short rolling summary")
	assertCellValue(t, workbook, "Conversations", "I5", "6")

	formula, err := workbook.GetCellFormula("Messages", "D5")
	if err != nil {
		t.Fatalf("GetCellFormula() error = %v", err)
	}
	if formula != "" {
		t.Fatalf("GetCellFormula(Messages!D5) = %q, want empty", formula)
	}

	assertCellValue(t, workbook, "Messages", "A4", "Conversation Id")
	assertCellValue(t, workbook, "Messages", "C5", "user")
	assertCellValue(t, workbook, "Messages", "D5", "=HYPERLINK(\"https://example.com\",\"boom\")")
}

func TestLoadExportsReadsCSVFiles(t *testing.T) {
	inputDir := t.TempDir()

	writeCSV(t, filepath.Join(inputDir, "dau.csv"), []string{"day", "dau"}, [][]string{
		{"2026-03-10", "18"},
		{"2026-03-09", "14"},
	})
	writeCSV(t, filepath.Join(inputDir, "messages_per_session.csv"), []string{"sessions", "avg_messages_per_session", "max_messages_in_session"}, [][]string{
		{"12", "8.25", "16"},
	})
	writeCSV(t, filepath.Join(inputDir, "latency.csv"), []string{"samples", "avg_latency_ms", "p95_latency_ms"}, [][]string{
		{"44", "1825.40", "2630.80"},
	})
	writeCSV(t, filepath.Join(inputDir, "tokens_by_model.csv"), []string{"model", "responses", "input_tokens", "output_tokens", "total_tokens"}, [][]string{
		{"gpt-4.1-mini", "30", "1200", "800", "2000"},
	})
	writeCSV(t, filepath.Join(inputDir, "returning_users.csv"), []string{"active_users", "returning_users", "returning_rate_percent"}, [][]string{
		{"9", "6", "66.67"},
	})
	writeCSV(t, filepath.Join(inputDir, "ratings_summary.csv"), []string{"ratings_submitted", "unique_rated_messages", "avg_rating"}, [][]string{
		{"5", "5", "4.4"},
	})
	writeCSV(t, filepath.Join(inputDir, "ratings_by_source.csv"), []string{"source", "submissions", "avg_rating"}, [][]string{
		{"inline_prompt", "3", "4.67"},
		{"delayed_prompt", "2", "4.00"},
	})
	writeCSV(t, filepath.Join(inputDir, "conversations.csv"), []string{
		"conversation_id",
		"user_id",
		"started_at",
		"ended_at",
		"state",
		"topic_id",
		"message_count",
		"summary",
		"compacted_at",
		"metadata",
	}, [][]string{{
		"conv-1",
		"excel-demo-user",
		"2026-03-10 01:00:00+00",
		"",
		"teaching",
		"algebra",
		"4",
		"Short rolling summary",
		"6",
		"{\"summary\":\"Short rolling summary\",\"compacted_at\":6}",
	}})
	writeCSV(t, filepath.Join(inputDir, "conversation_messages.csv"), []string{
		"conversation_id",
		"message_id",
		"role",
		"content",
		"model",
		"input_tokens",
		"output_tokens",
		"created_at",
	}, [][]string{{
		"conv-1",
		"msg-1",
		"user",
		"Hi",
		"",
		"",
		"",
		"2026-03-10 01:00:00+00",
	}})

	exports, err := analyticsxlsx.LoadExports(inputDir)
	if err != nil {
		t.Fatalf("LoadExports() error = %v", err)
	}

	if got := exports["dau"].Rows[0]["day"]; got != "2026-03-10" {
		t.Fatalf("exports[dau][0][day] = %q, want 2026-03-10", got)
	}
	if got := exports["returning_users"].Rows[0]["returning_rate_percent"]; got != "66.67" {
		t.Fatalf("exports[returning_users][0][returning_rate_percent] = %q, want 66.67", got)
	}
	if got := len(exports["ratings_by_source"].Rows); got != 2 {
		t.Fatalf("len(exports[ratings_by_source]) = %d, want 2", got)
	}
	if got := exports["conversations"].Rows[0]["conversation_id"]; got != "conv-1" {
		t.Fatalf("exports[conversations][0][conversation_id] = %q, want conv-1", got)
	}
	if got := exports["conversation_messages"].Rows[0]["message_id"]; got != "msg-1" {
		t.Fatalf("exports[conversation_messages][0][message_id] = %q, want msg-1", got)
	}
}

func TestRunWritesExampleWorkbook(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "report.xlsx")

	var stdout strings.Builder
	err := analyticsxlsx.Run([]string{
		"--example",
		"--output",
		outputPath,
		"--days",
		"14",
	}, &stdout)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", outputPath, err)
	}
	if !strings.Contains(stdout.String(), "Wrote workbook to ") {
		t.Fatalf("stdout = %q, want workbook path message", stdout.String())
	}
}

func assertCellValue(t *testing.T, workbook *excelize.File, sheet string, cell string, want string) {
	t.Helper()

	got, err := workbook.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatalf("GetCellValue(%s, %s) error = %v", sheet, cell, err)
	}
	if got != want {
		t.Fatalf("GetCellValue(%s, %s) = %q, want %q", sheet, cell, got, want)
	}
}

func writeCSV(t *testing.T, path string, header []string, rows [][]string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", path, err)
	}
	t.Cleanup(func() {
		_ = file.Close()
	})

	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		t.Fatalf("writer.Write(header) error = %v", err)
	}
	if err := writer.WriteAll(rows); err != nil {
		t.Fatalf("writer.WriteAll(rows) error = %v", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatalf("writer.Error() = %v", err)
	}
}

func equalStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

package analyticsxlsx

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Dataset struct {
	Headers []string
	Rows    []map[string]string
}

type ExportSet map[string]Dataset

type exportSpec struct {
	filename string
	sheet    string
}

var exportSpecs = map[string]exportSpec{
	"dau":                   {filename: "dau.csv", sheet: "Daily Activity"},
	"messages_per_session":  {filename: "messages_per_session.csv", sheet: "Engagement"},
	"latency":               {filename: "latency.csv", sheet: "Latency"},
	"tokens_by_model":       {filename: "tokens_by_model.csv", sheet: "Usage"},
	"returning_users":       {filename: "returning_users.csv", sheet: "Engagement"},
	"ratings_summary":       {filename: "ratings_summary.csv", sheet: "Ratings"},
	"ratings_by_source":     {filename: "ratings_by_source.csv", sheet: "Ratings"},
	"conversations":         {filename: "conversations.csv", sheet: "Conversations"},
	"conversation_messages": {filename: "conversation_messages.csv", sheet: "Messages"},
}

var exportOrder = []string{
	"dau",
	"messages_per_session",
	"latency",
	"tokens_by_model",
	"returning_users",
	"ratings_summary",
	"ratings_by_source",
	"conversations",
	"conversation_messages",
}

var sheetOrder = []string{
	"Overview",
	"Daily Activity",
	"Engagement",
	"Latency",
	"Usage",
	"Ratings",
	"Conversations",
	"Messages",
}

var colors = struct {
	Navy   string
	Blue   string
	Ice    string
	Border string
	Muted  string
	White  string
}{
	Navy:   "17324D",
	Blue:   "2A628F",
	Ice:    "F5F9FC",
	Border: "D7E2EA",
	Muted:  "5B6B7A",
	White:  "FFFFFF",
}

var titleCaser = cases.Title(language.English)

type workbookStyles struct {
	title       int
	subtitle    int
	header      int
	data        int
	dataNumber  int
	dataDecimal int
	metricLabel int
}

func BuildExampleExports() ExportSet {
	return ExportSet{
		"dau": {
			Headers: []string{"day", "dau"},
			Rows: []map[string]string{
				{"day": "2026-03-10", "dau": "18"},
				{"day": "2026-03-09", "dau": "14"},
				{"day": "2026-03-08", "dau": "11"},
				{"day": "2026-03-07", "dau": "9"},
				{"day": "2026-03-06", "dau": "7"},
				{"day": "2026-03-05", "dau": "6"},
				{"day": "2026-03-04", "dau": "5"},
			},
		},
		"messages_per_session": {
			Headers: []string{"sessions", "avg_messages_per_session", "max_messages_in_session"},
			Rows: []map[string]string{
				{
					"sessions":                 "23",
					"avg_messages_per_session": "8.75",
					"max_messages_in_session":  "17",
				},
			},
		},
		"latency": {
			Headers: []string{"samples", "avg_latency_ms", "p95_latency_ms"},
			Rows: []map[string]string{
				{"samples": "91", "avg_latency_ms": "1840.52", "p95_latency_ms": "2898.33"},
			},
		},
		"tokens_by_model": {
			Headers: []string{"model", "responses", "input_tokens", "output_tokens", "total_tokens"},
			Rows: []map[string]string{
				{
					"model":         "gpt-4.1-mini",
					"responses":     "54",
					"input_tokens":  "11800",
					"output_tokens": "7600",
					"total_tokens":  "19400",
				},
				{
					"model":         "claude-3.5-haiku",
					"responses":     "22",
					"input_tokens":  "4800",
					"output_tokens": "2900",
					"total_tokens":  "7700",
				},
				{
					"model":         "gemini-2.0-flash",
					"responses":     "15",
					"input_tokens":  "3200",
					"output_tokens": "2100",
					"total_tokens":  "5300",
				},
			},
		},
		"returning_users": {
			Headers: []string{"active_users", "returning_users", "returning_rate_percent"},
			Rows: []map[string]string{
				{"active_users": "15", "returning_users": "10", "returning_rate_percent": "66.67"},
			},
		},
		"ratings_summary": {
			Headers: []string{"ratings_submitted", "unique_rated_messages", "avg_rating"},
			Rows: []map[string]string{
				{"ratings_submitted": "12", "unique_rated_messages": "12", "avg_rating": "4.58"},
			},
		},
		"ratings_by_source": {
			Headers: []string{"source", "submissions", "avg_rating"},
			Rows: []map[string]string{
				{"source": "inline_prompt", "submissions": "8", "avg_rating": "4.75"},
				{"source": "delayed_prompt", "submissions": "4", "avg_rating": "4.25"},
			},
		},
		"conversations": {
			Headers: []string{
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
			},
			Rows: []map[string]string{
				{
					"conversation_id": "conv-demo-1",
					"user_id":         "excel-demo-user",
					"started_at":      "2026-03-10 01:00:00+00",
					"ended_at":        "",
					"state":           "teaching",
					"topic_id":        "algebra.linear-equations",
					"message_count":   "4",
					"summary":         "Short rolling summary",
					"compacted_at":    "6",
					"metadata":        "{\"summary\":\"Short rolling summary\",\"compacted_at\":6}",
				},
			},
		},
		"conversation_messages": {
			Headers: []string{
				"conversation_id",
				"message_id",
				"role",
				"content",
				"model",
				"input_tokens",
				"output_tokens",
				"created_at",
			},
			Rows: []map[string]string{
				{
					"conversation_id": "conv-demo-1",
					"message_id":      "msg-demo-1",
					"role":            "user",
					"content":         "Hi, I am testing the terminal chat.",
					"model":           "",
					"input_tokens":    "",
					"output_tokens":   "",
					"created_at":      "2026-03-10 01:00:00+00",
				},
				{
					"conversation_id": "conv-demo-1",
					"message_id":      "msg-demo-2",
					"role":            "assistant",
					"content":         "Hello. I can help with algebra today.",
					"model":           "gpt-4.1-mini",
					"input_tokens":    "120",
					"output_tokens":   "85",
					"created_at":      "2026-03-10 01:00:04+00",
				},
			},
		},
	}
}

func LoadExports(inputDir string) (ExportSet, error) {
	exports := make(ExportSet, len(exportOrder))

	for _, name := range exportOrder {
		spec := exportSpecs[name]
		filePath := filepath.Join(inputDir, spec.filename)
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", filePath, err)
		}

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		closeErr := file.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filePath, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s: %w", filePath, closeErr)
		}
		if len(records) == 0 {
			return nil, fmt.Errorf("missing analytics export header: %s", filePath)
		}

		headers := append([]string(nil), records[0]...)
		rows := make([]map[string]string, 0, max(len(records)-1, 0))
		for _, record := range records[1:] {
			row := make(map[string]string, len(headers))
			for index, header := range headers {
				if index < len(record) {
					row[header] = record[index]
				} else {
					row[header] = ""
				}
			}
			rows = append(rows, row)
		}
		exports[name] = Dataset{Headers: headers, Rows: rows}
	}

	return exports, nil
}

func Run(args []string, stdout io.Writer) error {
	flagSet := flag.NewFlagSet("analytics-xlsx", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	output := flagSet.String("output", "", "Path to the .xlsx file to create.")
	days := flagSet.Int("days", 7, "Reporting window in days.")
	generatedAt := flagSet.String("generated-at", "", "Override generated-at timestamp (RFC3339).")
	example := flagSet.Bool("example", false, "Write a sample workbook.")
	inputDir := flagSet.String("input-dir", "", "Directory containing analytics CSV exports.")

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		return errors.New("--output is required")
	}
	if *example == (*inputDir != "") {
		return errors.New("choose exactly one of --example or --input-dir")
	}

	generatedAtTime := time.Now().UTC()
	if *generatedAt != "" {
		parsed, err := time.Parse(time.RFC3339, *generatedAt)
		if err != nil {
			return fmt.Errorf("parse --generated-at: %w", err)
		}
		generatedAtTime = parsed
	}

	var (
		exports ExportSet
		err     error
	)
	if *example {
		exports = BuildExampleExports()
	} else {
		exports, err = LoadExports(*inputDir)
		if err != nil {
			return err
		}
	}

	if err := WriteWorkbook(exports, *output, *days, generatedAtTime); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Wrote workbook to %s\n", *output)
	return nil
}

func WriteWorkbook(exports ExportSet, outputPath string, days int, generatedAt time.Time) error {
	for _, name := range exportOrder {
		if _, ok := exports[name]; !ok {
			return fmt.Errorf("missing export dataset: %s", name)
		}
	}

	file := excelize.NewFile()
	defer func() {
		_ = file.Close()
	}()

	if err := file.SetSheetName(file.GetSheetName(0), "Overview"); err != nil {
		return err
	}
	for _, sheet := range sheetOrder[1:] {
		if _, err := file.NewSheet(sheet); err != nil {
			return err
		}
	}

	styles, err := newWorkbookStyles(file)
	if err != nil {
		return err
	}

	if err := populateOverview(file, styles, exports, days, generatedAt); err != nil {
		return err
	}
	if err := populateDailyActivity(file, styles, exports["dau"]); err != nil {
		return err
	}
	if err := populateEngagement(file, styles, exports["messages_per_session"], exports["returning_users"]); err != nil {
		return err
	}
	if err := populateLatency(file, styles, exports["latency"]); err != nil {
		return err
	}
	if err := populateUsage(file, styles, exports["tokens_by_model"]); err != nil {
		return err
	}
	if err := populateRatings(file, styles, exports["ratings_summary"], exports["ratings_by_source"]); err != nil {
		return err
	}
	if err := populateConversations(file, styles, exports["conversations"]); err != nil {
		return err
	}
	if err := populateMessages(file, styles, exports["conversation_messages"]); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create workbook output directory: %w", err)
	}

	return file.SaveAs(outputPath)
}

func populateOverview(file *excelize.File, styles workbookStyles, exports ExportSet, days int, generatedAt time.Time) error {
	sheet := "Overview"
	if err := styleSheetTitle(file, styles, sheet, "P&AI Analytics Report",
		fmt.Sprintf("Window: last %d day(s) | Generated at %s", days, generatedAt.UTC().Format("2006-01-02 15:04:05 UTC")), 6); err != nil {
		return err
	}

	metricRows := []struct {
		Label string
		Value string
	}{
		{Label: "Latest DAU", Value: firstValue(exports["dau"], "dau")},
		{Label: "Avg messages/session", Value: firstValue(exports["messages_per_session"], "avg_messages_per_session")},
		{Label: "Avg latency (ms)", Value: firstValue(exports["latency"], "avg_latency_ms")},
		{Label: "Returning rate (%)", Value: firstValue(exports["returning_users"], "returning_rate_percent")},
		{Label: "Average rating", Value: firstValue(exports["ratings_summary"], "avg_rating")},
		{Label: "Ratings submitted", Value: firstValue(exports["ratings_summary"], "ratings_submitted")},
	}

	if err := file.SetCellValue(sheet, "A4", "Metric"); err != nil {
		return err
	}
	if err := file.SetCellValue(sheet, "B4", "Value"); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheet, "A4", "B4", styles.header); err != nil {
		return err
	}

	for index, metric := range metricRows {
		row := index + 5
		if err := file.SetCellValue(sheet, fmt.Sprintf("A%d", row), metric.Label); err != nil {
			return err
		}
		if err := file.SetCellValue(sheet, fmt.Sprintf("B%d", row), coerceCellValue("value", metric.Value)); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styles.metricLabel); err != nil {
			return err
		}
		styleID := styles.dataDecimal
		if isIntegerString(metric.Value) {
			styleID = styles.dataNumber
		}
		if err := file.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styleID); err != nil {
			return err
		}
	}

	if err := file.SetCellValue(sheet, "D4", "Section"); err != nil {
		return err
	}
	if err := file.SetCellValue(sheet, "E4", "Rows"); err != nil {
		return err
	}
	if err := file.SetCellValue(sheet, "F4", "Primary sheet"); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheet, "D4", "F4", styles.header); err != nil {
		return err
	}

	for index, name := range exportOrder {
		row := index + 5
		if err := file.SetCellValue(sheet, fmt.Sprintf("D%d", row), labelize(name)); err != nil {
			return err
		}
		if err := file.SetCellValue(sheet, fmt.Sprintf("E%d", row), len(exports[name].Rows)); err != nil {
			return err
		}
		if err := file.SetCellValue(sheet, fmt.Sprintf("F%d", row), exportSpecs[name].sheet); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("F%d", row), styles.data); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), styles.dataNumber); err != nil {
			return err
		}
	}

	if err := file.SetColWidth(sheet, "A", "A", 24); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "B", "B", 16); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "D", "D", 24); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "E", "E", 10); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "F", "F", 18); err != nil {
		return err
	}
	return freezeAt(file, sheet, "A4")
}

func populateDailyActivity(file *excelize.File, styles workbookStyles, dataset Dataset) error {
	sheet := "Daily Activity"
	if err := styleSheetTitle(file, styles, sheet, "Daily Activity", "Unique active learners by day.", 6); err != nil {
		return err
	}
	endRow, err := writeDataset(file, styles, sheet, dataset, 4, "")
	if err != nil {
		return err
	}
	if err := addTable(file, sheet, "daily_activity_table", 4, endRow, len(dataset.Headers)); err != nil {
		return err
	}
	if len(dataset.Rows) >= 2 {
		if err := file.AddChart(sheet, "D4", &excelize.Chart{
			Type: excelize.Line,
			Series: []excelize.ChartSeries{
				{
					Name:       fmt.Sprintf("%s!$B$4", sheet),
					Categories: fmt.Sprintf("%s!$A$5:$A$%d", sheet, endRow),
					Values:     fmt.Sprintf("%s!$B$5:$B$%d", sheet, endRow),
				},
			},
			Title:     []excelize.RichTextRun{{Text: "DAU Trend"}},
			XAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "Day"}}},
			YAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "DAU"}}},
			Dimension: excelize.ChartDimension{Width: 560, Height: 280},
		}); err != nil {
			return err
		}
	}
	return freezeAt(file, sheet, "A4")
}

func populateEngagement(file *excelize.File, styles workbookStyles, messages Dataset, returning Dataset) error {
	sheet := "Engagement"
	if err := styleSheetTitle(file, styles, sheet, "Engagement", "Session depth and repeat usage.", 7); err != nil {
		return err
	}
	endRow, err := writeLabeledSingleRowSection(file, styles, sheet, "Messages Per Session", messages, 4)
	if err != nil {
		return err
	}
	if _, err := writeLabeledSingleRowSection(file, styles, sheet, "Returning Users", returning, endRow+3); err != nil {
		return err
	}
	if err := file.SetCellValue(sheet, "F4", "Review Notes"); err != nil {
		return err
	}
	if err := file.SetCellValue(sheet, "F5", "Use this sheet to compare session depth against repeat usage week over week."); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheet, "F4", "F4", styles.metricLabel); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "F", "F", 36); err != nil {
		return err
	}
	return freezeAt(file, sheet, "A4")
}

func populateLatency(file *excelize.File, styles workbookStyles, dataset Dataset) error {
	sheet := "Latency"
	if err := styleSheetTitle(file, styles, sheet, "Latency", "Assistant response time after the preceding learner message.", 6); err != nil {
		return err
	}
	if _, err := writeLabeledSingleRowSection(file, styles, sheet, "Latency Summary", dataset, 4); err != nil {
		return err
	}
	return freezeAt(file, sheet, "A4")
}

func populateUsage(file *excelize.File, styles workbookStyles, dataset Dataset) error {
	sheet := "Usage"
	if err := styleSheetTitle(file, styles, sheet, "Usage", "Token consumption and response distribution by model.", 8); err != nil {
		return err
	}
	endRow, err := writeDataset(file, styles, sheet, dataset, 4, "")
	if err != nil {
		return err
	}
	if err := addTable(file, sheet, "usage_table", 4, endRow, len(dataset.Headers)); err != nil {
		return err
	}
	if len(dataset.Rows) >= 1 {
		if err := file.AddChart(sheet, "G4", &excelize.Chart{
			Type: excelize.Bar,
			Series: []excelize.ChartSeries{
				{
					Name:       fmt.Sprintf("%s!$E$4", sheet),
					Categories: fmt.Sprintf("%s!$A$5:$A$%d", sheet, endRow),
					Values:     fmt.Sprintf("%s!$E$5:$E$%d", sheet, endRow),
				},
			},
			Title:     []excelize.RichTextRun{{Text: "Total Tokens by Model"}},
			XAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "Total Tokens"}}},
			YAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "Model"}}},
			Dimension: excelize.ChartDimension{Width: 520, Height: 240},
		}); err != nil {
			return err
		}
	}
	return freezeAt(file, sheet, "A4")
}

func populateRatings(file *excelize.File, styles workbookStyles, summary Dataset, bySource Dataset) error {
	sheet := "Ratings"
	if err := styleSheetTitle(file, styles, sheet, "Ratings", "Feedback volume, quality, and source breakdown.", 8); err != nil {
		return err
	}
	summaryEnd, err := writeLabeledSingleRowSection(file, styles, sheet, "Ratings Summary", summary, 4)
	if err != nil {
		return err
	}
	sourceStart := summaryEnd + 3
	endRow, err := writeDataset(file, styles, sheet, bySource, sourceStart, "Ratings by Source")
	if err != nil {
		return err
	}
	if err := addTable(file, sheet, "ratings_by_source_table", sourceStart, endRow, len(bySource.Headers)); err != nil {
		return err
	}
	if len(bySource.Rows) >= 1 {
		if err := file.AddChart(sheet, "E4", &excelize.Chart{
			Type: excelize.Bar,
			Series: []excelize.ChartSeries{
				{
					Name:       fmt.Sprintf("%s!$C$%d", sheet, sourceStart),
					Categories: fmt.Sprintf("%s!$A$%d:$A$%d", sheet, sourceStart+1, endRow),
					Values:     fmt.Sprintf("%s!$C$%d:$C$%d", sheet, sourceStart+1, endRow),
				},
			},
			Title:     []excelize.RichTextRun{{Text: "Average Rating by Source"}},
			XAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "Average Rating"}}},
			YAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "Source"}}},
			Dimension: excelize.ChartDimension{Width: 440, Height: 240},
		}); err != nil {
			return err
		}
	}
	return freezeAt(file, sheet, "A4")
}

func populateConversations(file *excelize.File, styles workbookStyles, dataset Dataset) error {
	sheet := "Conversations"
	if err := styleSheetTitle(file, styles, sheet, "Conversations", "Recent conversation list for manual review, including persisted AI state.", 12); err != nil {
		return err
	}
	endRow, err := writeDataset(file, styles, sheet, dataset, 4, "")
	if err != nil {
		return err
	}
	if err := addTable(file, sheet, "conversations_table", 4, endRow, len(dataset.Headers)); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "H", "H", 36); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "J", "J", 56); err != nil {
		return err
	}
	return freezeAt(file, sheet, "A4")
}

func populateMessages(file *excelize.File, styles workbookStyles, dataset Dataset) error {
	sheet := "Messages"
	if err := styleSheetTitle(file, styles, sheet, "Messages", "Recent conversation transcript for QA review.", 10); err != nil {
		return err
	}
	endRow, err := writeDataset(file, styles, sheet, dataset, 4, "")
	if err != nil {
		return err
	}
	if err := addTable(file, sheet, "conversation_messages_table", 4, endRow, len(dataset.Headers)); err != nil {
		return err
	}
	if err := file.SetColWidth(sheet, "D", "D", 56); err != nil {
		return err
	}
	return freezeAt(file, sheet, "A4")
}

func writeLabeledSingleRowSection(file *excelize.File, styles workbookStyles, sheet string, title string, dataset Dataset, startRow int) (int, error) {
	cell, err := excelize.CoordinatesToCellName(1, startRow)
	if err != nil {
		return 0, err
	}
	if err := file.SetCellValue(sheet, cell, title); err != nil {
		return 0, err
	}
	if err := file.SetCellStyle(sheet, cell, cell, styles.metricLabel); err != nil {
		return 0, err
	}
	return writeDataset(file, styles, sheet, dataset, startRow+1, "")
}

func writeDataset(file *excelize.File, styles workbookStyles, sheet string, dataset Dataset, startRow int, title string) (int, error) {
	if title != "" {
		cell, err := excelize.CoordinatesToCellName(1, startRow-1)
		if err != nil {
			return 0, err
		}
		if err := file.SetCellValue(sheet, cell, title); err != nil {
			return 0, err
		}
		if err := file.SetCellStyle(sheet, cell, cell, styles.metricLabel); err != nil {
			return 0, err
		}
	}

	if len(dataset.Headers) == 0 {
		cell, err := excelize.CoordinatesToCellName(1, startRow)
		if err != nil {
			return 0, err
		}
		if err := file.SetCellValue(sheet, cell, "No data"); err != nil {
			return 0, err
		}
		return startRow, nil
	}

	for colIndex, header := range dataset.Headers {
		cell, err := excelize.CoordinatesToCellName(colIndex+1, startRow)
		if err != nil {
			return 0, err
		}
		if err := file.SetCellValue(sheet, cell, labelize(header)); err != nil {
			return 0, err
		}
	}
	endHeader, err := excelize.CoordinatesToCellName(len(dataset.Headers), startRow)
	if err != nil {
		return 0, err
	}
	if err := file.SetCellStyle(sheet, "A"+strconv.Itoa(startRow), endHeader, styles.header); err != nil {
		return 0, err
	}

	if len(dataset.Rows) == 0 {
		return startRow, nil
	}

	for rowOffset, row := range dataset.Rows {
		rowNumber := startRow + 1 + rowOffset
		for colIndex, header := range dataset.Headers {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowNumber)
			if err != nil {
				return 0, err
			}
			value := coerceCellValue(header, row[header])
			if err := setCell(file, sheet, cell, value); err != nil {
				return 0, err
			}
			styleID := styles.data
			switch value.(type) {
			case int64:
				styleID = styles.dataNumber
			case float64:
				styleID = styles.dataDecimal
			}
			if err := file.SetCellStyle(sheet, cell, cell, styleID); err != nil {
				return 0, err
			}
		}
	}

	return startRow + len(dataset.Rows), nil
}

func styleSheetTitle(file *excelize.File, styles workbookStyles, sheet string, title string, subtitle string, span int) error {
	endCell, err := excelize.CoordinatesToCellName(span, 1)
	if err != nil {
		return err
	}
	if err := file.MergeCell(sheet, "A1", endCell); err != nil {
		return err
	}
	if err := file.SetCellValue(sheet, "A1", title); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheet, "A1", endCell, styles.title); err != nil {
		return err
	}

	subtitleEnd, err := excelize.CoordinatesToCellName(span, 2)
	if err != nil {
		return err
	}
	if err := file.MergeCell(sheet, "A2", subtitleEnd); err != nil {
		return err
	}
	if err := file.SetCellValue(sheet, "A2", subtitle); err != nil {
		return err
	}
	return file.SetCellStyle(sheet, "A2", subtitleEnd, styles.subtitle)
}

func addTable(file *excelize.File, sheet string, name string, startRow int, endRow int, columnCount int) error {
	if endRow <= startRow || columnCount == 0 {
		return nil
	}
	endColumn, err := excelize.ColumnNumberToName(columnCount)
	if err != nil {
		return err
	}
	showRowStripes := true
	return file.AddTable(sheet, &excelize.Table{
		Range:          fmt.Sprintf("A%d:%s%d", startRow, endColumn, endRow),
		Name:           name,
		StyleName:      "TableStyleMedium2",
		ShowRowStripes: &showRowStripes,
	})
}

func freezeAt(file *excelize.File, sheet string, topLeft string) error {
	row, err := strconv.Atoi(strings.TrimLeft(topLeft, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	if err != nil {
		return err
	}
	return file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      row - 1,
		TopLeftCell: topLeft,
		ActivePane:  "bottomLeft",
		Selection: []excelize.Selection{
			{SQRef: topLeft, ActiveCell: topLeft, Pane: "bottomLeft"},
		},
	})
}

func newWorkbookStyles(file *excelize.File) (workbookStyles, error) {
	title, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: colors.White, Size: 18},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colors.Navy}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return workbookStyles{}, err
	}
	subtitle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Color: colors.Muted},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return workbookStyles{}, err
	}
	header, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: colors.White},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colors.Blue}},
		Border: []excelize.Border{
			{Type: "left", Color: colors.Border, Style: 1},
			{Type: "right", Color: colors.Border, Style: 1},
			{Type: "top", Color: colors.Border, Style: 1},
			{Type: "bottom", Color: colors.Border, Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return workbookStyles{}, err
	}
	data, err := file.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: colors.Border, Style: 1},
			{Type: "right", Color: colors.Border, Style: 1},
			{Type: "top", Color: colors.Border, Style: 1},
			{Type: "bottom", Color: colors.Border, Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return workbookStyles{}, err
	}
	dataNumber, err := file.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: colors.Border, Style: 1},
			{Type: "right", Color: colors.Border, Style: 1},
			{Type: "top", Color: colors.Border, Style: 1},
			{Type: "bottom", Color: colors.Border, Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    3,
	})
	if err != nil {
		return workbookStyles{}, err
	}
	dataDecimal, err := file.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: colors.Border, Style: 1},
			{Type: "right", Color: colors.Border, Style: 1},
			{Type: "top", Color: colors.Border, Style: 1},
			{Type: "bottom", Color: colors.Border, Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    4,
	})
	if err != nil {
		return workbookStyles{}, err
	}
	metricLabel, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: colors.Navy},
		Border: []excelize.Border{
			{Type: "left", Color: colors.Border, Style: 1},
			{Type: "right", Color: colors.Border, Style: 1},
			{Type: "top", Color: colors.Border, Style: 1},
			{Type: "bottom", Color: colors.Border, Style: 1},
		},
	})
	if err != nil {
		return workbookStyles{}, err
	}

	return workbookStyles{
		title:       title,
		subtitle:    subtitle,
		header:      header,
		data:        data,
		dataNumber:  dataNumber,
		dataDecimal: dataDecimal,
		metricLabel: metricLabel,
	}, nil
}

func firstValue(dataset Dataset, key string) string {
	if len(dataset.Rows) == 0 {
		return "0"
	}
	return dataset.Rows[0][key]
}

func coerceCellValue(header string, raw string) interface{} {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if keepAsText(header) {
		return value
	}
	if isIntegerString(value) {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed
		}
	}
	if isDecimalString(value) {
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed
		}
	}
	return value
}

func keepAsText(header string) bool {
	switch header {
	case "day", "conversation_id", "message_id", "role", "content", "model", "source",
		"user_id", "started_at", "ended_at", "created_at", "state", "topic_id", "summary", "metadata":
		return true
	default:
		return false
	}
}

func isIntegerString(value string) bool {
	if value == "" {
		return false
	}
	if value[0] == '-' {
		value = value[1:]
	}
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isDecimalString(value string) bool {
	if value == "" {
		return false
	}
	if value[0] == '-' {
		value = value[1:]
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	return isIntegerString(parts[0]) && isIntegerString(parts[1])
}

func labelize(header string) string {
	return titleCaser.String(strings.ReplaceAll(header, "_", " "))
}

func setCell(file *excelize.File, sheet string, cell string, value interface{}) error {
	if text, ok := value.(string); ok {
		return file.SetCellStr(sheet, cell, text)
	}
	return file.SetCellValue(sheet, cell, value)
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

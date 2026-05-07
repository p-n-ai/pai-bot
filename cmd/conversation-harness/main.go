// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/terminalchat"
)

const (
	defaultFixturePath    = "internal/agent/testdata/ai_quality_conversations.yaml"
	fallbackMessagePhrase = "masalah teknikal"
	ratingPromptPhrase    = "rating 1-5"
	ratingThanksPhrase    = "terima kasih atas rating anda"
	ratingRetryPhrase     = "rating perlu 1 hingga 5"
)

type fixtureFile struct {
	Version       int                `yaml:"version"`
	Provider      string             `yaml:"provider"`
	Conversations []conversationSpec `yaml:"conversations"`
}

type conversationSpec struct {
	ID     string         `yaml:"id"`
	Title  string         `yaml:"title"`
	Tags   []string       `yaml:"tags"`
	Turns  []turnSpec     `yaml:"turns"`
	Checks behaviorChecks `yaml:"checks"`
}

type turnSpec struct {
	User string `yaml:"user"`
}

type behaviorChecks struct {
	RequireNonEmptyReplies    bool     `yaml:"require_non_empty_replies"`
	ForbidFallbackMessage     bool     `yaml:"forbid_fallback_message"`
	ForbidMarkdownAndLatex    bool     `yaml:"forbid_markdown_and_latex"`
	RequireStructuredSolving  bool     `yaml:"require_structured_solving"`
	RequireConceptConnection  bool     `yaml:"require_concept_connection"`
	ExpectedLanguage          string   `yaml:"expected_language"`
	RequireRatingPrompt       bool     `yaml:"require_rating_prompt"`
	RequireRatingThanks       bool     `yaml:"require_rating_thanks"`
	RequireRatingRetry        bool     `yaml:"require_rating_retry"`
	RequireResponsePhrases    []string `yaml:"require_response_phrases"`
	ForbidResponsePhrases     []string `yaml:"forbid_response_phrases"`
	ForbidFinalAnswerOnTurn   []int    `yaml:"forbid_final_answer_on_turn"`
	ForbidSectionLabelsOnTurn []int    `yaml:"forbid_section_labels_on_turn"`
	MaxResponseLines          int      `yaml:"max_response_lines"`
	MaxResponseChars          int      `yaml:"max_response_chars"`
}

type caseResult struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags,omitempty"`
	Passed   bool     `json:"passed"`
	Turns    int      `json:"turns"`
	Failures []string `json:"failures,omitempty"`
}

func main() {
	var fixturePath string
	var caseID string
	var tag string
	var maxCases int
	var timeout time.Duration
	var showResponses bool
	var jsonl bool
	var memory bool
	var mockResponse string
	var progressSideEffects bool
	var verbose bool

	flag.StringVar(&fixturePath, "fixture", defaultFixturePath, "YAML conversation fixture")
	flag.StringVar(&caseID, "case", "", "run one conversation id")
	flag.StringVar(&tag, "tag", "", "run conversations with tag")
	flag.IntVar(&maxCases, "max-cases", 0, "maximum number of selected conversations to run")
	flag.DurationVar(&timeout, "timeout", 45*time.Second, "timeout per turn")
	flag.BoolVar(&showResponses, "show-responses", false, "print user and assistant turns")
	flag.BoolVar(&jsonl, "jsonl", false, "print one JSON result per conversation")
	flag.BoolVar(&memory, "memory", true, "use in-memory state instead of PostgreSQL")
	flag.StringVar(&mockResponse, "mock-response", "", "use a deterministic mock AI response instead of configured providers")
	flag.BoolVar(&progressSideEffects, "progress", false, "enable mastery/progress side effects during harness runs")
	flag.BoolVar(&verbose, "verbose", false, "show diagnostic warnings from curriculum loading and background checks")
	flag.Parse()

	logLevel := slog.LevelError
	if verbose {
		logLevel = slog.LevelWarn
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	fixture, err := loadFixture(fixturePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load fixture: %v\n", err)
		os.Exit(1)
	}
	conversations := selectConversations(fixture.Conversations, caseID, tag, maxCases)
	if len(conversations) == 0 {
		fmt.Fprintln(os.Stderr, "no conversations selected")
		os.Exit(1)
	}

	engine, cleanup, err := buildEngine(memory, mockResponse, progressSideEffects)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build harness: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	results := make([]caseResult, 0, len(conversations))
	for _, conv := range conversations {
		result := runConversation(engine, conv, timeout, showResponses)
		results = append(results, result)
		if jsonl {
			_ = json.NewEncoder(os.Stdout).Encode(result)
		}
	}
	if !jsonl {
		printSummary(results)
	}
	if failedCount(results) > 0 {
		os.Exit(1)
	}
}

func loadFixture(path string) (fixtureFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return fixtureFile{}, err
	}
	var fixture fixtureFile
	if err := yaml.Unmarshal(b, &fixture); err != nil {
		return fixtureFile{}, err
	}
	if fixture.Version != 1 {
		return fixtureFile{}, fmt.Errorf("version = %d, want 1", fixture.Version)
	}
	return fixture, nil
}

func buildEngine(memory bool, mockResponse string, progressSideEffects bool) (*agent.Engine, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	router := ai.NewRouter()
	if mockResponse != "" {
		router.Register("mock", ai.NewMockProvider(mockResponse))
	} else {
		if !cfg.HasAIProvider() {
			return nil, nil, fmt.Errorf("at least one AI provider must be configured")
		}
		router = airouter.Setup(cfg)
		if !router.HasProvider() {
			return nil, nil, fmt.Errorf("no AI providers configured")
		}
	}

	loader, err := curriculum.NewLoader(cfg.CurriculumPath)
	if err != nil {
		slog.Warn("curriculum not loaded", "path", cfg.CurriculumPath, "error", err)
	}
	state, cleanup, err := terminalchat.BuildState(context.Background(), cfg, terminalchat.StateOptions{
		Memory:  memory,
		Channel: "harness",
	}, terminalchat.StateDeps{})
	if err != nil {
		return nil, nil, err
	}

	var goalStore agent.GoalStore
	var challengeStore agent.ChallengeStore
	if memory {
		goalStore = agent.NewMemoryGoalStore()
		challengeStore = agent.NewMemoryChallengeStore()
	} else {
		goalStore = agent.NewPostgresGoalStoreForChannel(state.DB.Pool, state.TenantID, "harness")
		challengeStore = agent.NewPostgresChallengeStoreForChannel(state.DB.Pool, state.TenantID, "harness")
	}

	engineCfg := agent.EngineConfig{
		AIRouter:             router,
		Store:                state.Store,
		EventLogger:          state.EventLogger,
		CurriculumLoader:     loader,
		DisableMultiLanguage: cfg.Runtime.DisableMultiLanguage,
		RatingPromptEvery:    cfg.Runtime.RatingPromptEvery,
		Goals:                goalStore,
		Challenges:           challengeStore,
		DevMode:              cfg.Runtime.DevMode,
	}
	if progressSideEffects {
		engineCfg.Tracker = state.Tracker
	}
	engine := agent.NewEngine(engineCfg)
	return engine, cleanup, nil
}

func selectConversations(conversations []conversationSpec, caseID, tag string, maxCases int) []conversationSpec {
	caseID = strings.TrimSpace(caseID)
	tag = strings.TrimSpace(tag)
	selected := make([]conversationSpec, 0, len(conversations))
	for _, conv := range conversations {
		if caseID != "" && conv.ID != caseID {
			continue
		}
		if tag != "" && !hasTag(conv.Tags, tag) {
			continue
		}
		selected = append(selected, conv)
		if maxCases > 0 && len(selected) >= maxCases {
			break
		}
	}
	return selected
}

func runConversation(engine *agent.Engine, conv conversationSpec, timeout time.Duration, showResponses bool) caseResult {
	userID := "harness-" + strings.ToLower(conv.ID) + "-" + fmt.Sprint(time.Now().UnixNano())
	responses := make([]string, 0, len(conv.Turns))
	failures := []string{}

	for i, turn := range conv.Turns {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		resp, err := engine.ProcessMessage(ctx, chat.InboundMessage{
			Channel: "harness",
			UserID:  userID,
			Text:    turn.User,
		})
		cancel()
		if err != nil {
			failures = append(failures, fmt.Sprintf("turn %d: ProcessMessage error: %v", i+1, err))
			continue
		}
		responses = append(responses, resp)
		if showResponses {
			fmt.Printf("\n[%s turn %d]\nUser: %s\nAssistant: %s\n", conv.ID, i+1, turn.User, resp)
		}
		failures = append(failures, checkTurn(i+1, resp, conv.Checks)...)
	}
	failures = append(failures, checkConversation(conv.Checks, responses)...)

	return caseResult{
		ID:       conv.ID,
		Title:    conv.Title,
		Tags:     conv.Tags,
		Passed:   len(failures) == 0,
		Turns:    len(conv.Turns),
		Failures: failures,
	}
}

func checkTurn(turn int, resp string, checks behaviorChecks) []string {
	var failures []string
	if checks.RequireNonEmptyReplies && strings.TrimSpace(resp) == "" {
		failures = append(failures, fmt.Sprintf("turn %d: empty response", turn))
	}
	if checks.ForbidFallbackMessage && containsFold(resp, fallbackMessagePhrase) {
		failures = append(failures, fmt.Sprintf("turn %d: fallback phrase %q", turn, fallbackMessagePhrase))
	}
	if checks.ForbidMarkdownAndLatex {
		for _, token := range []string{"**", "```", "\\(", "\\)", "\\[", "\\]", "$$"} {
			if strings.Contains(resp, token) {
				failures = append(failures, fmt.Sprintf("turn %d: forbidden formatting token %q", turn, token))
			}
		}
	}
	for _, phrase := range checks.ForbidResponsePhrases {
		if containsFold(resp, phrase) {
			failures = append(failures, fmt.Sprintf("turn %d: forbidden phrase %q", turn, phrase))
		}
	}
	if containsInt(checks.ForbidFinalAnswerOnTurn, turn) && looksLikeFinalAnswer(resp) {
		failures = append(failures, fmt.Sprintf("turn %d: looks like final answer was revealed", turn))
	}
	if containsInt(checks.ForbidSectionLabelsOnTurn, turn) && containsSectionLabel(resp) {
		failures = append(failures, fmt.Sprintf("turn %d: contains tutoring section label despite naturalness check", turn))
	}
	if checks.MaxResponseLines > 0 && responseLineCount(resp) > checks.MaxResponseLines {
		failures = append(failures, fmt.Sprintf("turn %d: response has %d lines, max %d", turn, responseLineCount(resp), checks.MaxResponseLines))
	}
	if checks.MaxResponseChars > 0 && len(resp) > checks.MaxResponseChars {
		failures = append(failures, fmt.Sprintf("turn %d: response has %d chars, max %d", turn, len(resp), checks.MaxResponseChars))
	}
	return failures
}

func checkConversation(checks behaviorChecks, responses []string) []string {
	var failures []string
	combined := strings.ToLower(strings.Join(responses, "\n"))
	for _, phrase := range checks.RequireResponsePhrases {
		if !strings.Contains(combined, strings.ToLower(phrase)) {
			failures = append(failures, fmt.Sprintf("missing required phrase %q", phrase))
		}
	}
	if checks.ExpectedLanguage != "" {
		failures = append(failures, checkLanguage(checks.ExpectedLanguage, combined)...)
	}
	if checks.RequireStructuredSolving {
		if !containsAny(combined, []string{"plan", "rancang", "langkah", "step"}) {
			failures = append(failures, "missing plan/step marker")
		}
		if !containsAny(combined, []string{"verify", "semak", "check", "substitute", "gantikan"}) {
			failures = append(failures, "missing verify/check marker")
		}
	}
	if checks.RequireConceptConnection &&
		!containsAny(combined, []string{"konsep", "concept", "persamaan linear", "linear equation", "inverse operation", "operasi songsang", "bila guna", "when to use"}) {
		failures = append(failures, "missing concept connection marker")
	}
	if checks.RequireRatingPrompt && !strings.Contains(combined, ratingPromptPhrase) {
		failures = append(failures, "missing rating prompt")
	}
	if checks.RequireRatingThanks && !strings.Contains(combined, ratingThanksPhrase) {
		failures = append(failures, "missing rating thank-you response")
	}
	if checks.RequireRatingRetry && !strings.Contains(combined, ratingRetryPhrase) {
		failures = append(failures, "missing rating retry response")
	}
	return failures
}

func checkLanguage(expected, combined string) []string {
	bmMarkers := []string{"langkah", "jawapan", "kenapa", "boleh", "kita", "persamaan", "semak", "contoh", "tingkatan", "matematik", "anda"}
	enMarkers := []string{"step", "answer", "equation", "because", "check", "solution", "math", "you", "let"}
	switch expected {
	case "bm_or_mixed":
		if !containsAny(combined, bmMarkers) {
			return []string{"missing BM or mixed language markers"}
		}
	case "en_or_mixed":
		if !containsAny(combined, enMarkers) {
			return []string{"missing EN or mixed language markers"}
		}
	}
	return nil
}

func printSummary(results []caseResult) {
	total := len(results)
	failed := failedCount(results)
	fmt.Printf("conversation harness: %d passed, %d failed, %d total\n", total-failed, failed, total)
	for _, result := range results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		fmt.Printf("%s %s %s\n", status, result.ID, result.Title)
		for _, failure := range result.Failures {
			fmt.Printf("  - %s\n", failure)
		}
	}
}

func failedCount(results []caseResult) int {
	count := 0
	for _, result := range results {
		if !result.Passed {
			count++
		}
	}
	return count
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func containsAny(s string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func containsFold(s, needle string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(needle))
}

func containsInt(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func looksLikeFinalAnswer(resp string) bool {
	normalized := strings.ToLower(resp)
	return strings.Contains(normalized, "x =") ||
		strings.Contains(normalized, "x=") ||
		strings.Contains(normalized, "jawapan akhir") ||
		strings.Contains(normalized, "final answer") ||
		strings.Contains(normalized, "answer is")
}

func containsSectionLabel(resp string) bool {
	for _, label := range []string{
		"Faham/Understand:",
		"Selesaikan/Solve:",
		"Semak/Verify:",
		"Konsep/Connect:",
		"Faham:",
		"Understand:",
		"Semak:",
		"Verify:",
		"Konsep:",
		"Concept:",
	} {
		if strings.Contains(resp, label) {
			return true
		}
	}
	return false
}

func responseLineCount(resp string) int {
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return 0
	}
	count := 0
	for _, line := range strings.Split(resp, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

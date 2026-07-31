// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
	"github.com/p-n-ai/pai-bot/internal/terminalchat"
)

var evidenceCitationCheckPattern = regexp.MustCompile(`\[S[0-9]+\]`)

const (
	defaultFixturePath    = "internal/agent/testdata/ai_quality_conversations.yaml"
	fallbackMessagePhrase = "masalah teknikal"
)

type fixtureFile struct {
	Version       int                `yaml:"version"`
	Provider      string             `yaml:"provider"`
	Characters    []characterSpec    `yaml:"characters"`
	Conversations []conversationSpec `yaml:"conversations"`
}

type characterSpec struct {
	ID        string `yaml:"id"`
	FirstName string `yaml:"first_name"`
	Username  string `yaml:"username"`
	Language  string `yaml:"language"`
}

type conversationSpec struct {
	ID          string         `yaml:"id"`
	Title       string         `yaml:"title"`
	CharacterID string         `yaml:"character"`
	Character   characterSpec  `yaml:"-"`
	Tags        []string       `yaml:"tags"`
	Evidence    []evidenceSpec `yaml:"evidence"`
	Turns       []turnSpec     `yaml:"turns"`
	Checks      behaviorChecks `yaml:"checks"`
}

type turnSpec struct {
	User   string         `yaml:"user"`
	Checks behaviorChecks `yaml:"checks"`
}

type evidenceSpec struct {
	ID           string `yaml:"id"`
	Origin       string `yaml:"origin"`
	Title        string `yaml:"title"`
	Filename     string `yaml:"filename"`
	LocatorType  string `yaml:"locator_type"`
	LocatorStart int    `yaml:"locator_start"`
	LocatorEnd   int    `yaml:"locator_end"`
	Locator      string `yaml:"locator"`
	Excerpt      string `yaml:"excerpt"`
}

type behaviorChecks struct {
	RequireNonEmptyReplies    bool     `yaml:"require_non_empty_replies"`
	ForbidFallbackMessage     bool     `yaml:"forbid_fallback_message"`
	ForbidMarkdownAndLatex    bool     `yaml:"forbid_markdown_and_latex"`
	RequireStructuredSolving  bool     `yaml:"require_structured_solving"`
	RequireConceptConnection  bool     `yaml:"require_concept_connection"`
	ExpectedLanguage          string   `yaml:"expected_language"`
	RequireResponsePhrases    []string `yaml:"require_response_phrases"`
	ForbidResponsePhrases     []string `yaml:"forbid_response_phrases"`
	ForbidFinalAnswerOnTurn   []int    `yaml:"forbid_final_answer_on_turn"`
	ForbidSectionLabelsOnTurn []int    `yaml:"forbid_section_labels_on_turn"`
	MaxResponseLines          int      `yaml:"max_response_lines"`
	MaxResponseChars          int      `yaml:"max_response_chars"`
	RequireEvidenceCitation   bool     `yaml:"require_evidence_citation"`
}

type caseResult struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags,omitempty"`
	Passed   bool     `json:"passed"`
	Turns    int      `json:"turns"`
	Failures []string `json:"failures,omitempty"`
}

type requestDumpRecord struct {
	Sequence    int                  `json:"sequence"`
	Provider    string               `json:"provider"`
	Request     ai.CompletionRequest `json:"request"`
	StartedAt   time.Time            `json:"started_at"`
	CompletedAt time.Time            `json:"completed_at"`
	Error       string               `json:"error,omitempty"`
}

type requestDumper struct {
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
	nextSeq int
	err     error
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
	var dumpRequestsPath string
	var requestOnly bool

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
	flag.StringVar(&dumpRequestsPath, "dump-requests", "", "write mock AI completion requests as JSONL to this path")
	flag.BoolVar(&requestOnly, "request-only", false, "skip behavior scoring; useful with --dump-requests")
	flag.Parse()
	if err := validateRequestOnlyMode(requestOnly, dumpRequestsPath); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

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

	var dumper *requestDumper
	if dumpRequestsPath != "" {
		if mockResponse == "" {
			mockResponse = "mock tutor response"
		}
		dumper, err = newRequestDumper(dumpRequestsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open request dump: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := dumper.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "close request dump: %v\n", err)
				os.Exit(1)
			}
		}()
	}

	engine, cleanup, err := buildEngine(memory, mockResponse, progressSideEffects, traceFuncForDumper(dumper), newHarnessEvidenceRetriever(conversations))
	if err != nil {
		fmt.Fprintf(os.Stderr, "build harness: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	results := make([]caseResult, 0, len(conversations))
	for _, conv := range conversations {
		result := runConversation(engine, conv, timeout, showResponses, !requestOnly)
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

func newRequestDumper(path string) (*requestDumper, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &requestDumper{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

func traceFuncForDumper(dumper *requestDumper) func(ai.CompletionTrace) {
	if dumper == nil {
		return nil
	}
	return dumper.Record
}

func (d *requestDumper) Record(trace ai.CompletionTrace) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		return
	}
	d.nextSeq++
	d.err = d.encoder.Encode(requestDumpRecord{
		Sequence:    d.nextSeq,
		Provider:    trace.Provider,
		Request:     trace.Request,
		StartedAt:   trace.StartedAt,
		CompletedAt: trace.CompletedAt,
		Error:       trace.Error,
	})
}

func (d *requestDumper) Close() error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		_ = d.file.Close()
		return d.err
	}
	return d.file.Close()
}

func validateRequestOnlyMode(requestOnly bool, dumpRequestsPath string) error {
	if requestOnly && strings.TrimSpace(dumpRequestsPath) == "" {
		return fmt.Errorf("--request-only requires --dump-requests")
	}
	return nil
}

func loadFixture(path string) (fixtureFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return fixtureFile{}, err
	}
	defer func() { _ = file.Close() }()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var fixture fixtureFile
	if err := decoder.Decode(&fixture); err != nil {
		return fixtureFile{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fixtureFile{}, fmt.Errorf("fixture must contain exactly one YAML document")
		}
		return fixtureFile{}, err
	}
	if err := normalizeFixture(&fixture); err != nil {
		return fixtureFile{}, err
	}
	return fixture, nil
}

func normalizeFixture(fixture *fixtureFile) error {
	if fixture.Version != 1 {
		return fmt.Errorf("version = %d, want 1", fixture.Version)
	}

	charactersByID := make(map[string]characterSpec, len(fixture.Characters))
	for i := range fixture.Characters {
		character := &fixture.Characters[i]
		character.ID = strings.TrimSpace(character.ID)
		character.FirstName = strings.TrimSpace(character.FirstName)
		character.Username = strings.TrimSpace(character.Username)
		character.Language = strings.TrimSpace(character.Language)
		if character.ID == "" {
			return fmt.Errorf("characters[%d].id is required", i)
		}
		if character.Language != "" {
			normalized := i18n.NormalizeLocale(character.Language)
			if normalized == "" {
				return fmt.Errorf("character %q has unsupported language %q", character.ID, character.Language)
			}
			character.Language = normalized
		}
		if _, exists := charactersByID[character.ID]; exists {
			return fmt.Errorf("duplicate character id %q", character.ID)
		}
		charactersByID[character.ID] = *character
	}

	conversationIDs := make(map[string]struct{}, len(fixture.Conversations))
	for i := range fixture.Conversations {
		conversation := &fixture.Conversations[i]
		conversation.ID = strings.TrimSpace(conversation.ID)
		conversation.Title = strings.TrimSpace(conversation.Title)
		conversation.CharacterID = strings.TrimSpace(conversation.CharacterID)

		if conversation.ID == "" {
			return fmt.Errorf("conversations[%d].id is required", i)
		}
		if _, exists := conversationIDs[conversation.ID]; exists {
			return fmt.Errorf("duplicate conversation id %q", conversation.ID)
		}
		conversationIDs[conversation.ID] = struct{}{}
		if conversation.Title == "" {
			return fmt.Errorf("conversation %q title is required", conversation.ID)
		}
		if conversation.CharacterID != "" {
			character, exists := charactersByID[conversation.CharacterID]
			if !exists {
				return fmt.Errorf("conversation %q references unknown character %q", conversation.ID, conversation.CharacterID)
			}
			conversation.Character = character
		}
		if len(conversation.Turns) == 0 {
			return fmt.Errorf("conversation %q requires at least one turn", conversation.ID)
		}
		if err := normalizeBehaviorChecks(&conversation.Checks, len(conversation.Turns), fmt.Sprintf("conversation %q checks", conversation.ID)); err != nil {
			return err
		}
		for j := range conversation.Turns {
			turn := &conversation.Turns[j]
			turn.User = strings.TrimSpace(turn.User)
			if turn.User == "" {
				return fmt.Errorf("conversation %q turn %d user text is required", conversation.ID, j+1)
			}
			if err := normalizeBehaviorChecks(&turn.Checks, len(conversation.Turns), fmt.Sprintf("conversation %q turn %d checks", conversation.ID, j+1)); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeBehaviorChecks(checks *behaviorChecks, turnCount int, location string) error {
	checks.ExpectedLanguage = strings.TrimSpace(checks.ExpectedLanguage)
	normalizeStrings(checks.RequireResponsePhrases)
	normalizeStrings(checks.ForbidResponsePhrases)
	switch checks.ExpectedLanguage {
	case "", "bm_or_mixed", "en_or_mixed":
	default:
		return fmt.Errorf("%s expected_language %q is unsupported", location, checks.ExpectedLanguage)
	}
	if checks.MaxResponseLines < 0 {
		return fmt.Errorf("%s max_response_lines must be non-negative", location)
	}
	if checks.MaxResponseChars < 0 {
		return fmt.Errorf("%s max_response_chars must be non-negative", location)
	}
	if err := validateTurnReferences(checks.ForbidFinalAnswerOnTurn, turnCount, location, "forbid_final_answer_on_turn"); err != nil {
		return err
	}
	if err := validateTurnReferences(checks.ForbidSectionLabelsOnTurn, turnCount, location, "forbid_section_labels_on_turn"); err != nil {
		return err
	}
	return nil
}

func validateTurnReferences(turns []int, turnCount int, location, field string) error {
	for _, turn := range turns {
		if turn < 1 || turn > turnCount {
			return fmt.Errorf("%s %s contains turn %d outside 1..%d", location, field, turn, turnCount)
		}
	}
	return nil
}

func normalizeStrings(values []string) {
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
}

func buildEngine(memory bool, mockResponse string, progressSideEffects bool, traceFunc func(ai.CompletionTrace), evidenceRetriever retrieval.TutorEvidenceRetriever) (*agent.Engine, func(), error) {
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
		router = airouter.Setup(cfg.AI)
		if !router.HasProvider() {
			return nil, nil, fmt.Errorf("no AI providers configured")
		}
	}
	if traceFunc != nil {
		router.SetTraceFunc(traceFunc)
	}

	loader, err := curriculum.NewLoader(cfg.CurriculumPath)
	if err != nil {
		slog.Warn("curriculum not loaded", "path", cfg.CurriculumPath, "error", err)
	}
	state, cleanup, err := terminalchat.BuildState(context.Background(), cfg.Database, terminalchat.StateOptions{
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
		Goals:                goalStore,
		Challenges:           challengeStore,
		DevMode:              cfg.Runtime.DevMode,
		FeatureFlags:         func() featureflags.Features { return cfg.FeatureFlags },
		EvidenceRetriever:    evidenceRetriever,
	}
	if progressSideEffects {
		engineCfg.Tracker = state.Tracker
		if loader != nil {
			curriculumProgress, ok := state.Tracker.(curriculum.ProgressStore)
			if !ok {
				cleanup()
				return nil, nil, fmt.Errorf("initialize curriculum runtime: progress tracker does not support mastery evidence")
			}
			curriculumRuntime, runtimeErr := curriculum.NewEngine(curriculum.EngineConfig{
				Loader:   loader,
				Progress: curriculumProgress,
			})
			if runtimeErr != nil {
				cleanup()
				return nil, nil, fmt.Errorf("initialize curriculum runtime: %w", runtimeErr)
			}
			engineCfg.CurriculumRuntime = curriculumRuntime
		}
	}
	engine := agent.NewEngine(engineCfg)
	return engine, cleanup, nil
}

type harnessEvidenceRetriever struct {
	byCase map[string]map[string][]retrieval.TutorEvidence
}

func newHarnessEvidenceRetriever(conversations []conversationSpec) retrieval.TutorEvidenceRetriever {
	byCase := make(map[string]map[string][]retrieval.TutorEvidence)
	for _, conversation := range conversations {
		var items []retrieval.TutorEvidence
		for _, item := range conversation.Evidence {
			items = append(items, retrieval.TutorEvidence{
				ID: item.ID, Origin: item.Origin, SourceTitle: item.Title, Filename: item.Filename,
				LocatorType: item.LocatorType, LocatorStart: item.LocatorStart,
				LocatorEnd: item.LocatorEnd, Locator: item.Locator, Excerpt: item.Excerpt,
			})
		}
		byQuery := make(map[string][]retrieval.TutorEvidence)
		for _, turn := range conversation.Turns {
			if len(items) > 0 {
				byQuery[strings.TrimSpace(turn.User)] = items
			}
		}
		if len(byQuery) > 0 {
			byCase[strings.ToLower(conversation.ID)] = byQuery
		}
	}
	if len(byCase) == 0 {
		return nil
	}
	return &harnessEvidenceRetriever{byCase: byCase}
}

func (r *harnessEvidenceRetriever) Retrieve(_ context.Context, request retrieval.TutorEvidenceRequest) ([]retrieval.TutorEvidence, error) {
	byQuery := r.byCase[harnessCaseIDFromLearnerID(request.LearnerID)]
	if items := byQuery[strings.TrimSpace(request.Query)]; len(items) > 0 {
		return append([]retrieval.TutorEvidence(nil), items...), nil
	}
	for query, items := range byQuery {
		if strings.Contains(request.Query, query) {
			return append([]retrieval.TutorEvidence(nil), items...), nil
		}
	}
	return nil, nil
}

func harnessCaseIDFromLearnerID(learnerID string) string {
	const prefix = "harness-"
	if !strings.HasPrefix(learnerID, prefix) {
		return ""
	}
	scoped := strings.TrimPrefix(learnerID, prefix)
	separator := strings.LastIndexByte(scoped, '-')
	if separator <= 0 || separator == len(scoped)-1 {
		return ""
	}
	return scoped[:separator]
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

type conversationProcessor interface {
	ProcessMessage(context.Context, chat.InboundMessage) (string, error)
}

func runConversation(engine conversationProcessor, conv conversationSpec, timeout time.Duration, showResponses bool, runChecks bool) caseResult {
	userID := "harness-" + strings.ToLower(conv.ID) + "-" + fmt.Sprint(time.Now().UnixNano())
	responses := make([]string, 0, len(conv.Turns))
	failures := []string{}

	for i, turn := range conv.Turns {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		resp, err := engine.ProcessMessage(ctx, chat.InboundMessage{
			Channel:    "harness",
			UserID:     userID,
			DeliveryID: fmt.Sprintf("harness:%s:%d", userID, i+1),
			Text:       turn.User,
			FirstName:  conv.Character.FirstName,
			Username:   conv.Character.Username,
			Language:   conv.Character.Language,
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
		if runChecks {
			failures = append(failures, checkTurn(i+1, resp, turn.Checks)...)
			for _, failure := range checkConversation(turn.Checks, responses[len(responses)-1:]) {
				failures = append(failures, fmt.Sprintf("turn %d: %s", i+1, failure))
			}
			failures = append(failures, checkTurn(i+1, resp, conv.Checks)...)
		}
	}
	if runChecks {
		failures = append(failures, checkConversation(conv.Checks, responses)...)
	}

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
	if checks.RequireEvidenceCitation && !evidenceCitationCheckPattern.MatchString(resp) {
		failures = append(failures, fmt.Sprintf("turn %d: response has no supplied evidence citation label", turn))
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

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
	"github.com/p-n-ai/pai-bot/internal/conversationharness"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
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
	ID                string         `yaml:"id"`
	Title             string         `yaml:"title"`
	Tags              []string       `yaml:"tags"`
	CharacterID       string         `yaml:"character"`
	Evidence          []evidenceSpec `yaml:"evidence"`
	Turns             []turnSpec     `yaml:"turns"`
	Checks            behaviorChecks `yaml:"checks"`
	resolvedCharacter characterSpec
}

type turnSpec struct {
	User         string                       `yaml:"user"`
	Delivery     conversationharness.Delivery `yaml:"delivery"`
	After        durationSpec                 `yaml:"after"`
	ExpectStatus conversationharness.Status   `yaml:"expect_status"`
	Checks       behaviorChecks               `yaml:"checks"`
}

type durationSpec struct {
	time.Duration
}

func (d *durationSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("duration must be a string")
	}
	if strings.TrimSpace(node.Value) == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", node.Value, err)
	}
	if parsed < 0 {
		return fmt.Errorf("duration must not be negative")
	}
	d.Duration = parsed
	return nil
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
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Tags        []string            `json:"tags,omitempty"`
	Passed      bool                `json:"passed"`
	Turns       int                 `json:"turns"`
	Delivered   int                 `json:"delivered"`
	Interrupted int                 `json:"interrupted,omitempty"`
	Outcomes    []turnOutcomeResult `json:"outcomes"`
	Failures    []string            `json:"failures,omitempty"`
}

type turnOutcomeResult struct {
	Turn       int                          `json:"turn"`
	Delivery   conversationharness.Delivery `json:"delivery"`
	Status     conversationharness.Status   `json:"status"`
	DurationMS int64                        `json:"duration_ms"`
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
		result := runConversation(engine.ProcessMessage, conv, timeout, showResponses, !requestOnly)
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
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var fixture fixtureFile
	if err := decoder.Decode(&fixture); err != nil {
		return fixtureFile{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fixtureFile{}, fmt.Errorf("multiple YAML documents are not supported")
		}
		return fixtureFile{}, err
	}
	if err := validateFixture(&fixture); err != nil {
		return fixtureFile{}, err
	}
	return fixture, nil
}

func validateFixture(fixture *fixtureFile) error {
	if fixture == nil {
		return fmt.Errorf("fixture is required")
	}
	if fixture.Version != 1 && fixture.Version != 2 {
		return fmt.Errorf("version = %d, want 1 or 2", fixture.Version)
	}
	characters := make(map[string]characterSpec, len(fixture.Characters))
	for index, character := range fixture.Characters {
		character.ID = strings.TrimSpace(character.ID)
		character.FirstName = strings.TrimSpace(character.FirstName)
		character.Username = strings.TrimSpace(character.Username)
		character.Language = strings.TrimSpace(character.Language)
		if character.ID == "" {
			return fmt.Errorf("character %d: id is required", index+1)
		}
		if _, exists := characters[character.ID]; exists {
			return fmt.Errorf("duplicate character id %q", character.ID)
		}
		fixture.Characters[index] = character
		characters[character.ID] = character
	}

	conversationIDs := make(map[string]struct{}, len(fixture.Conversations))
	for conversationIndex := range fixture.Conversations {
		conversation := &fixture.Conversations[conversationIndex]
		conversation.ID = strings.TrimSpace(conversation.ID)
		if conversation.ID == "" {
			return fmt.Errorf("conversation %d: id is required", conversationIndex+1)
		}
		if _, exists := conversationIDs[conversation.ID]; exists {
			return fmt.Errorf("duplicate conversation id %q", conversation.ID)
		}
		conversationIDs[conversation.ID] = struct{}{}
		if strings.TrimSpace(conversation.Title) == "" {
			return fmt.Errorf("conversation %s: title is required", conversation.ID)
		}
		if len(conversation.Turns) == 0 {
			return fmt.Errorf("conversation %s: at least one turn is required", conversation.ID)
		}
		if err := validateBehaviorChecks(
			fmt.Sprintf("conversation %s checks", conversation.ID),
			conversation.Checks,
			len(conversation.Turns),
		); err != nil {
			return err
		}
		conversation.CharacterID = strings.TrimSpace(conversation.CharacterID)
		if conversation.CharacterID != "" {
			character, exists := characters[conversation.CharacterID]
			if !exists {
				return fmt.Errorf("conversation %s: unknown character %q", conversation.ID, conversation.CharacterID)
			}
			conversation.resolvedCharacter = character
		}
		for turnIndex, turn := range conversation.Turns {
			if strings.TrimSpace(turn.User) == "" {
				return fmt.Errorf("conversation %s turn %d: user text is required", conversation.ID, turnIndex+1)
			}
			switch normalizedDelivery(turn.Delivery) {
			case conversationharness.DeliveryWait, conversationharness.DeliveryQueue, conversationharness.DeliveryInterrupt:
			default:
				return fmt.Errorf(
					"conversation %s turn %d: unsupported delivery %q",
					conversation.ID,
					turnIndex+1,
					turn.Delivery,
				)
			}
			switch turn.ExpectStatus {
			case "", conversationharness.StatusDelivered, conversationharness.StatusInterrupted, conversationharness.StatusFailed:
			default:
				return fmt.Errorf(
					"conversation %s turn %d: unsupported expect_status %q",
					conversation.ID,
					turnIndex+1,
					turn.ExpectStatus,
				)
			}
			if err := validateBehaviorChecks(
				fmt.Sprintf("conversation %s turn %d checks", conversation.ID, turnIndex+1),
				turn.Checks,
				len(conversation.Turns),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBehaviorChecks(label string, checks behaviorChecks, turnCount int) error {
	switch checks.ExpectedLanguage {
	case "", "bm_or_mixed", "en_or_mixed":
	default:
		return fmt.Errorf("%s: unsupported expected_language %q", label, checks.ExpectedLanguage)
	}
	if checks.MaxResponseLines < 0 {
		return fmt.Errorf("%s: max_response_lines must not be negative", label)
	}
	if checks.MaxResponseChars < 0 {
		return fmt.Errorf("%s: max_response_chars must not be negative", label)
	}
	for _, entry := range []struct {
		name  string
		turns []int
	}{
		{name: "forbid_final_answer_on_turn", turns: checks.ForbidFinalAnswerOnTurn},
		{name: "forbid_section_labels_on_turn", turns: checks.ForbidSectionLabelsOnTurn},
	} {
		for _, turn := range entry.turns {
			if turn < 1 || turn > turnCount {
				return fmt.Errorf("%s: %s contains out-of-range turn %d", label, entry.name, turn)
			}
		}
	}
	return nil
}

func normalizedDelivery(delivery conversationharness.Delivery) conversationharness.Delivery {
	if delivery == "" {
		return conversationharness.DeliveryWait
	}
	return delivery
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

func runConversation(
	process conversationharness.Processor,
	conv conversationSpec,
	timeout time.Duration,
	showResponses bool,
	runChecks bool,
) caseResult {
	userID := "harness-" + strings.ToLower(conv.ID) + "-" + fmt.Sprint(time.Now().UnixNano())
	responses := make([]string, 0, len(conv.Turns))
	failures := []string{}
	turns := make([]conversationharness.Turn, 0, len(conv.Turns))
	for _, turn := range conv.Turns {
		turns = append(turns, conversationharness.Turn{
			Delivery: normalizedDelivery(turn.Delivery),
			After:    turn.After.Duration,
			Timeout:  timeout,
			Message: chat.InboundMessage{
				Channel:   "harness",
				UserID:    userID,
				Text:      turn.User,
				FirstName: conv.resolvedCharacter.FirstName,
				Username:  conv.resolvedCharacter.Username,
				Language:  conv.resolvedCharacter.Language,
			},
		})
	}
	outcomes, err := conversationharness.Run(
		context.Background(),
		process,
		turns,
	)
	if err != nil {
		failures = append(failures, fmt.Sprintf("conversation runner error: %v", err))
	}

	delivered := 0
	interrupted := 0
	turnOutcomes := make([]turnOutcomeResult, 0, len(outcomes))
	for i, outcome := range outcomes {
		turn := conv.Turns[i]
		turnOutcomes = append(turnOutcomes, turnOutcomeResult{
			Turn:       i + 1,
			Delivery:   normalizedDelivery(turn.Delivery),
			Status:     outcome.Status,
			DurationMS: outcome.Duration.Milliseconds(),
		})
		expectedStatus := turn.ExpectStatus
		if expectedStatus == "" {
			expectedStatus = conversationharness.StatusDelivered
		}
		if outcome.Status != expectedStatus {
			failures = append(failures, fmt.Sprintf(
				"turn %d: status %q, want %q",
				i+1,
				outcome.Status,
				expectedStatus,
			))
		}
		switch outcome.Status {
		case conversationharness.StatusInterrupted:
			interrupted++
		case conversationharness.StatusFailed:
			if expectedStatus != conversationharness.StatusFailed {
				failures = append(failures, fmt.Sprintf("turn %d: ProcessMessage error: %v", i+1, outcome.Err))
			}
		case conversationharness.StatusDelivered:
			delivered++
			responses = append(responses, outcome.Response)
		}
		if showResponses {
			fmt.Printf(
				"\n[%s turn %d, %s]\n%s: %s\n",
				conv.ID,
				i+1,
				outcome.Status,
				characterLabel(conv.resolvedCharacter),
				turn.User,
			)
			if outcome.Status == conversationharness.StatusDelivered {
				fmt.Printf("Assistant: %s\n", outcome.Response)
			}
		}
		if outcome.Status != conversationharness.StatusDelivered || !runChecks {
			continue
		}
		failures = append(failures, checkTurn(i+1, outcome.Response, conv.Checks)...)
		failures = append(failures, checkTurn(i+1, outcome.Response, turn.Checks)...)
		failures = append(failures, checkConversation(turn.Checks, []string{outcome.Response})...)
	}
	if runChecks {
		failures = append(failures, checkConversation(conv.Checks, responses)...)
	}

	return caseResult{
		ID:          conv.ID,
		Title:       conv.Title,
		Tags:        conv.Tags,
		Passed:      len(failures) == 0,
		Turns:       len(conv.Turns),
		Delivered:   delivered,
		Interrupted: interrupted,
		Outcomes:    turnOutcomes,
		Failures:    failures,
	}
}

func characterLabel(character characterSpec) string {
	if character.FirstName != "" {
		return character.FirstName
	}
	if character.Username != "" {
		return character.Username
	}
	return "User"
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

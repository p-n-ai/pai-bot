//go:build integration
// +build integration

package agent_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

const (
	liveFixturePath       = "testdata/openai_live_conversations.yaml"
	fallbackMessagePhrase = "masalah teknikal"
	ratingPromptPhrase    = "rating 1-5"
	ratingThanksPhrase    = "terima kasih atas rating anda"
	ratingRetryPhrase     = "rating perlu 1 hingga 5"
)

// LiveConversationSpec describes a single scripted live conversation.
type LiveConversationSpec struct {
	ID     string         `yaml:"id"`
	Title  string         `yaml:"title"`
	Tags   []string       `yaml:"tags"`
	Turns  []LiveTurnSpec `yaml:"turns"`
	Checks BehaviorChecks `yaml:"checks"`
}

// LiveTurnSpec describes one user message turn.
type LiveTurnSpec struct {
	User string `yaml:"user"`
}

// BehaviorChecks controls assertions for a conversation.
type BehaviorChecks struct {
	RequireNonEmptyReplies      bool   `yaml:"require_non_empty_replies"`
	ForbidFallbackMessage       bool   `yaml:"forbid_fallback_message"`
	ForbidMarkdownAndLatex      bool   `yaml:"forbid_markdown_and_latex"`
	RequireContinuityOnFollowup bool   `yaml:"require_continuity_on_followup"`
	RequireStructuredSolving    bool   `yaml:"require_structured_solving"`
	RequireConceptConnection    bool   `yaml:"require_concept_connection"`
	ExpectedLanguage            string `yaml:"expected_language"`
	RequireRatingPrompt         bool   `yaml:"require_rating_prompt"`
	RequireRatingThanks         bool   `yaml:"require_rating_thanks"`
	RequireRatingRetry          bool   `yaml:"require_rating_retry"`
}

type liveFixture struct {
	Version       int                    `yaml:"version"`
	Provider      string                 `yaml:"provider"`
	Conversations []LiveConversationSpec `yaml:"conversations"`
}

type trackingProvider struct {
	inner    ai.Provider
	requests []ai.CompletionRequest
}

func (p *trackingProvider) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	p.requests = append(p.requests, cloneCompletionRequest(req))
	return p.inner.Complete(ctx, req)
}

func (p *trackingProvider) StreamComplete(ctx context.Context, req ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return p.inner.StreamComplete(ctx, req)
}

func (p *trackingProvider) Models() []ai.ModelInfo {
	return p.inner.Models()
}

func (p *trackingProvider) HealthCheck(ctx context.Context) error {
	return p.inner.HealthCheck(ctx)
}

func cloneCompletionRequest(req ai.CompletionRequest) ai.CompletionRequest {
	cloned := req
	cloned.Messages = make([]ai.Message, len(req.Messages))
	for i, msg := range req.Messages {
		cloned.Messages[i] = ai.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			ImageURLs: append([]string(nil), msg.ImageURLs...),
		}
	}
	return cloned
}

func TestOpenAILiveConversations(t *testing.T) {
	if runningInCI() {
		t.Skip("skipping live OpenAI integration tests on CI")
	}

	fixture, err := loadLiveFixture(filepath.FromSlash(liveFixturePath))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	if err := validateLiveFixture(fixture); err != nil {
		t.Fatalf("invalid fixture: %v", err)
	}

	apiKey := strings.TrimSpace(os.Getenv("LEARN_AI_OPENAI_API_KEY"))
	if apiKey == "" {
		t.Skip("LEARN_AI_OPENAI_API_KEY is not set; skipping live OpenAI integration tests")
	}

	timeout := time.Duration(envInt("LEARN_AI_LIVE_TIMEOUT_SECONDS", 45)) * time.Second
	if timeout <= 0 {
		timeout = 45 * time.Second
	}

	maxCases := envInt("LEARN_AI_LIVE_MAX_CASES", 30)
	if maxCases <= 0 || maxCases > len(fixture.Conversations) {
		maxCases = len(fixture.Conversations)
	}

	conversations := fixture.Conversations[:maxCases]
	t.Logf("running %d/%d live conversations (timeout=%s)", len(conversations), len(fixture.Conversations), timeout)

	for _, conv := range conversations {
		conv := conv
		t.Run(conv.ID, func(t *testing.T) {
			provider := ai.NewOpenAIProvider(apiKey, ai.WithHTTPClient(&http.Client{Timeout: timeout}))
			tracker := &trackingProvider{inner: provider}

			router := ai.NewRouterWithConfig(ai.RouterConfig{
				RetryBackoff:            []time.Duration{250 * time.Millisecond, 500 * time.Millisecond},
				BreakerFailureThreshold: 2,
				BreakerCooldown:         2 * time.Second,
			})
			router.Register("openai", tracker)

			engine := agent.NewEngine(agent.EngineConfig{
				AIRouter: router,
				Store:    agent.NewMemoryStore(),
			})

			responses := make([]string, 0, len(conv.Turns))
			userID := "live-openai-" + strings.ToLower(conv.ID)

			for i, turn := range conv.Turns {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				resp, err := engine.ProcessMessage(ctx, chat.InboundMessage{
					Channel: "telegram",
					UserID:  userID,
					Text:    turn.User,
				})
				cancel()
				if err != nil {
					t.Fatalf("turn %d ProcessMessage error: %v", i+1, err)
				}

				responses = append(responses, resp)

				if conv.Checks.RequireNonEmptyReplies && strings.TrimSpace(resp) == "" {
					t.Fatalf("turn %d returned empty response", i+1)
				}
				if conv.Checks.ForbidFallbackMessage && containsFold(resp, fallbackMessagePhrase) {
					t.Fatalf("turn %d returned fallback phrase: %q", i+1, resp)
				}
				if conv.Checks.ForbidMarkdownAndLatex {
					assertNoMarkdownOrLatex(t, i+1, resp)
				}
			}

			if conv.Checks.RequireContinuityOnFollowup {
				assertContinuityOnFollowup(t, conv, responses, tracker.requests)
			}
			if conv.Checks.ExpectedLanguage != "" {
				assertExpectedLanguage(t, conv.Checks.ExpectedLanguage, responses)
			}
			if conv.Checks.RequireStructuredSolving {
				assertStructuredSolving(t, responses)
			}
			if conv.Checks.RequireConceptConnection {
				assertConceptConnection(t, responses)
			}
			if conv.Checks.RequireRatingPrompt {
				assertAnyContainsFold(t, responses, ratingPromptPhrase, "expected rating prompt in one assistant reply")
			}
			if conv.Checks.RequireRatingThanks {
				assertAnyContainsFold(t, responses, ratingThanksPhrase, "expected rating thank-you response")
				if len(tracker.requests) != len(conv.Turns)-1 {
					t.Fatalf("expected last turn to skip AI call after rating; provider calls=%d turns=%d", len(tracker.requests), len(conv.Turns))
				}
			}
			if conv.Checks.RequireRatingRetry {
				assertAnyContainsFold(t, responses, ratingRetryPhrase, "expected invalid rating retry guidance")
				if len(tracker.requests) != len(conv.Turns)-1 {
					t.Fatalf("expected last turn to skip AI call after invalid rating; provider calls=%d turns=%d", len(tracker.requests), len(conv.Turns))
				}
			}

			t.Logf("conversation=%s turns=%d provider_calls=%d", conv.ID, len(conv.Turns), len(tracker.requests))
		})
	}
}

func loadLiveFixture(path string) (liveFixture, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return liveFixture{}, fmt.Errorf("read %s: %w", path, err)
	}
	var fixture liveFixture
	if err := yaml.Unmarshal(b, &fixture); err != nil {
		return liveFixture{}, fmt.Errorf("unmarshal YAML: %w", err)
	}
	return fixture, nil
}

func validateLiveFixture(fixture liveFixture) error {
	if fixture.Version != 1 {
		return fmt.Errorf("version = %d, want 1", fixture.Version)
	}
	if !strings.EqualFold(strings.TrimSpace(fixture.Provider), "openai") {
		return fmt.Errorf("provider = %q, want openai", fixture.Provider)
	}
	if len(fixture.Conversations) != 30 {
		return fmt.Errorf("conversations = %d, want exactly 30", len(fixture.Conversations))
	}

	seen := map[string]struct{}{}
	for i, conv := range fixture.Conversations {
		if strings.TrimSpace(conv.ID) == "" {
			return fmt.Errorf("conversation[%d] has empty id", i)
		}
		if _, ok := seen[conv.ID]; ok {
			return fmt.Errorf("duplicate conversation id: %s", conv.ID)
		}
		seen[conv.ID] = struct{}{}
		if strings.TrimSpace(conv.Title) == "" {
			return fmt.Errorf("conversation %s has empty title", conv.ID)
		}
		if len(conv.Turns) < 2 || len(conv.Turns) > 10 {
			return fmt.Errorf("conversation %s has %d turns, want 2..10", conv.ID, len(conv.Turns))
		}
		for turnIdx, turn := range conv.Turns {
			if strings.TrimSpace(turn.User) == "" {
				return fmt.Errorf("conversation %s turn %d has empty user text", conv.ID, turnIdx+1)
			}
		}
	}

	return nil
}

func assertNoMarkdownOrLatex(t *testing.T, turn int, resp string) {
	t.Helper()
	forbiddenTokens := []string{"**", "```", "\\(", "\\)", "\\[", "\\]", "$$"}
	for _, token := range forbiddenTokens {
		if strings.Contains(resp, token) {
			t.Fatalf("turn %d response contains forbidden token %q: %q", turn, token, resp)
		}
	}
}

func assertContinuityOnFollowup(t *testing.T, conv LiveConversationSpec, responses []string, requests []ai.CompletionRequest) {
	t.Helper()
	if len(conv.Turns) < 2 || len(responses) < 2 {
		t.Fatalf("conversation %s requires at least 2 turns for continuity check", conv.ID)
	}
	if len(requests) < 2 {
		t.Fatalf("conversation %s requires at least 2 provider requests for continuity check, got %d", conv.ID, len(requests))
	}

	lastReq := requests[len(requests)-1]
	firstUser := conv.Turns[0].User
	prevAssistant := responses[len(responses)-2]
	lastUser := conv.Turns[len(conv.Turns)-1].User

	if !requestContains(lastReq.Messages, firstUser) {
		t.Fatalf("continuity check failed: last request missing first user turn %q", firstUser)
	}
	if !requestContains(lastReq.Messages, prevAssistant) {
		t.Fatalf("continuity check failed: last request missing previous assistant response")
	}
	if !requestContains(lastReq.Messages, lastUser) {
		t.Fatalf("continuity check failed: last request missing final user follow-up %q", lastUser)
	}
}

func requestContains(messages []ai.Message, want string) bool {
	want = strings.TrimSpace(strings.ToLower(want))
	for _, msg := range messages {
		content := strings.TrimSpace(strings.ToLower(msg.Content))
		if strings.Contains(content, want) {
			return true
		}
	}
	return false
}

func assertExpectedLanguage(t *testing.T, expected string, responses []string) {
	t.Helper()
	combined := strings.ToLower(strings.Join(responses, "\n"))

	bmMarkers := []string{"langkah", "jawapan", "kenapa", "boleh", "kita", "persamaan", "semak", "contoh", "tingkatan", "matematik", "anda"}
	enMarkers := []string{"step", "answer", "equation", "because", "check", "solution", "math", "you", "let"}

	switch expected {
	case "bm_or_mixed":
		if !containsAny(combined, bmMarkers) {
			t.Fatalf("expected BM or mixed language markers; got: %q", truncateForLog(combined, 500))
		}
	case "en_or_mixed":
		if !containsAny(combined, enMarkers) {
			t.Fatalf("expected EN or mixed language markers; got: %q", truncateForLog(combined, 500))
		}
	default:
		t.Fatalf("unsupported expected_language: %q", expected)
	}
}

func assertStructuredSolving(t *testing.T, responses []string) {
	t.Helper()
	combined := strings.ToLower(strings.Join(responses, "\n"))

	planMarkers := []string{"plan", "rancang", "langkah", "step"}
	verifyMarkers := []string{"verify", "semak", "check", "substitute", "gantikan"}

	if !containsAny(combined, planMarkers) {
		t.Fatalf("missing plan/step marker in response set")
	}
	if !containsAny(combined, verifyMarkers) {
		t.Fatalf("missing verify/check marker in response set")
	}
}

func assertConceptConnection(t *testing.T, responses []string) {
	t.Helper()
	combined := strings.ToLower(strings.Join(responses, "\n"))
	conceptMarkers := []string{"konsep", "concept", "persamaan linear", "linear equation", "inverse operation", "operasi songsang", "bila guna", "when to use"}
	if !containsAny(combined, conceptMarkers) {
		t.Fatalf("missing concept connection marker in response set")
	}
}

func assertAnyContainsFold(t *testing.T, responses []string, needle string, msg string) {
	t.Helper()
	for _, resp := range responses {
		if containsFold(resp, needle) {
			return
		}
	}
	t.Fatal(msg)
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

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func runningInCI() bool {
	ci := strings.TrimSpace(os.Getenv("CI"))
	if ci != "" && ci != "0" && !strings.EqualFold(ci, "false") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")), "true")
}

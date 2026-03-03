package agent

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

const (
	defaultCompactThreshold      = 20
	defaultCompactTokenThreshold = 20000 // ~20k tokens triggers compaction
	defaultKeepRecent            = 6
)

// EngineConfig holds dependencies for the agent engine.
type EngineConfig struct {
	AIRouter              *ai.Router
	Store                 ConversationStore
	EventLogger           EventLogger
	CompactThreshold      int // messages before compaction triggers (default 20)
	CompactTokenThreshold int // estimated tokens before compaction triggers (default 3000)
	KeepRecent            int // recent messages to keep after compaction (default 6)
}

// Engine is the core conversation processor.
type Engine struct {
	aiRouter              *ai.Router
	store                 ConversationStore
	eventLogger           EventLogger
	compactThreshold      int
	compactTokenThreshold int
	keepRecent            int
}

// NewEngine creates a new agent engine.
func NewEngine(cfg EngineConfig) *Engine {
	store := cfg.Store
	if store == nil {
		store = NewMemoryStore()
	}
	threshold := cfg.CompactThreshold
	if threshold == 0 {
		threshold = defaultCompactThreshold
	}
	tokenThreshold := cfg.CompactTokenThreshold
	if tokenThreshold == 0 {
		tokenThreshold = defaultCompactTokenThreshold
	}
	keepRecent := cfg.KeepRecent
	if keepRecent == 0 {
		keepRecent = defaultKeepRecent
	}
	eventLogger := cfg.EventLogger
	if eventLogger == nil {
		eventLogger = NopEventLogger{}
	}
	return &Engine{
		aiRouter:              cfg.AIRouter,
		store:                 store,
		eventLogger:           eventLogger,
		compactThreshold:      threshold,
		compactTokenThreshold: tokenThreshold,
		keepRecent:            keepRecent,
	}
}

// ProcessMessage handles an incoming message and returns a response.
func (e *Engine) ProcessMessage(ctx context.Context, msg chat.InboundMessage) (string, error) {
	slog.Info("processing message",
		"channel", msg.Channel,
		"user_id", msg.UserID,
		"text_len", len(msg.Text),
	)

	// Handle commands
	if strings.HasPrefix(msg.Text, "/") {
		return e.handleCommand(ctx, msg)
	}

	// Get or create active conversation.
	conv, err := e.getOrCreateConversation(msg.UserID)
	if err != nil {
		slog.Error("failed to get conversation", "error", err)
		return "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar.", nil
	}
	if conv.State == "onboarding" {
		return e.handleOnboardingSelection(ctx, msg, conv), nil
	}

	// Build user content — include replied message as context if present.
	userContent := msg.Text
	if msg.HasImage {
		if userContent == "" {
			userContent = "Please analyze the attached image and help me solve it step by step."
		}
		userContent = "[Student attached an image]\nAnalyze the image content first, then answer the student's request.\n\n" + userContent
	}
	if msg.ReplyToText != "" {
		userContent = fmt.Sprintf("[Replying to: \"%s\"]\n\n%s", msg.ReplyToText, userContent)
	}

	// Record user message.
	if err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: userContent,
	}); err != nil {
		slog.Error("failed to store user message", "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "message_sent",
		Data: map[string]any{
			"channel":   msg.Channel,
			"text_len":  len(msg.Text),
			"has_reply": msg.ReplyToText != "",
			"has_image": msg.HasImage,
			"source":    "chat",
		},
	})

	// Refresh conversation to get latest messages.
	conv, _ = e.store.GetConversation(conv.ID)

	// Compact if needed (summarize older messages).
	e.maybeCompact(ctx, conv)

	// Build messages: system prompt + (optional summary) + recent messages.
	systemPrompt := e.buildSystemPrompt(msg)
	messages := []ai.Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, e.buildContextMessages(conv)...)
	if msg.HasImage && msg.ImageDataURL == "" {
		return "Saya terima gambar anda, tapi gagal memproses fail gambar itu. Cuba hantar semula gambar yang lebih jelas.", nil
	}
	if msg.ImageDataURL != "" {
		messages = append(messages, ai.Message{
			Role:      "user",
			Content:   "Attached image from the student. Analyze this image directly and answer based on what you see. If unreadable, say exactly what is unclear and how to retake it.",
			ImageURLs: []string{msg.ImageDataURL},
		})
	}

	reqModel := ""
	if msg.ImageDataURL != "" {
		// Prefer a vision-capable model for image understanding.
		reqModel = "gpt-4o"
	}

	// Call AI
	resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
		Messages:  messages,
		Model:     reqModel,
		Task:      ai.TaskTeaching,
		MaxTokens: 1024,
	})
	if err != nil {
		slog.Error("AI completion failed", "error", err)
		return "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar.", nil
	}

	// Telegram does not render LaTeX blocks; keep equations plain.
	plainContent := normalizeEquationFormatting(resp.Content)

	// Record assistant response with token metadata.
	if err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:         "assistant",
		Content:      plainContent,
		Model:        resp.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	}); err != nil {
		slog.Error("failed to store assistant message", "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "ai_response",
		Data: map[string]any{
			"channel":       msg.Channel,
			"model":         resp.Model,
			"input_tokens":  resp.InputTokens,
			"output_tokens": resp.OutputTokens,
			"text_len":      len(resp.Content),
			"has_image":     msg.HasImage,
		},
	})

	return plainContent, nil
}

// buildContextMessages returns the conversation messages for the AI prompt.
// If a summary exists, it prepends it and only includes messages after compaction point.
func (e *Engine) buildContextMessages(conv *Conversation) []ai.Message {
	var messages []ai.Message

	if conv.Summary != "" {
		messages = append(messages, ai.Message{
			Role:    "user",
			Content: "Previous conversation summary:\n" + conv.Summary,
		})
		messages = append(messages, ai.Message{
			Role:    "assistant",
			Content: "Understood, I'll continue based on our previous conversation.",
		})
		// Only include messages after the compaction point.
		for _, m := range conv.Messages[conv.CompactedAt:] {
			messages = append(messages, ai.Message{Role: m.Role, Content: m.Content})
		}
	} else {
		for _, m := range conv.Messages {
			messages = append(messages, ai.Message{Role: m.Role, Content: m.Content})
		}
	}

	return messages
}

// estimateTokens gives a rough token count for messages (1 token ≈ 4 chars).
func estimateTokens(messages []StoredMessage) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) / 4
	}
	return total
}

// maybeCompact checks if the conversation needs compaction and summarizes if so.
// Triggers when message count OR estimated token count exceeds thresholds.
// Only considers messages since the last compaction to avoid re-compressing.
func (e *Engine) maybeCompact(ctx context.Context, conv *Conversation) {
	uncompacted := conv.Messages[conv.CompactedAt:]
	messagesSinceCompact := len(uncompacted)
	tokensSinceCompact := estimateTokens(uncompacted)

	if messagesSinceCompact <= e.compactThreshold && tokensSinceCompact <= e.compactTokenThreshold {
		return
	}

	// Summarize everything except the most recent messages.
	compactUpTo := len(conv.Messages) - e.keepRecent
	if compactUpTo <= conv.CompactedAt {
		return
	}

	toSummarize := conv.Messages[conv.CompactedAt:compactUpTo]

	// Build the summarization prompt.
	var content strings.Builder
	if conv.Summary != "" {
		content.WriteString("Previous summary:\n")
		content.WriteString(conv.Summary)
		content.WriteString("\n\nNew messages to incorporate:\n")
	}
	for _, m := range toSummarize {
		role := "Student"
		if m.Role == "assistant" {
			role = "Tutor"
		}
		content.WriteString(fmt.Sprintf("%s: %s\n", role, m.Content))
	}

	resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "system", Content: `Summarize this tutoring conversation concisely. Capture:
- Topics discussed and key concepts
- What the student understood or struggled with
- Any examples or problems worked through
Keep the summary under 150 words. Write in the same language used in the conversation.`},
			{Role: "user", Content: content.String()},
		},
		Task:      ai.TaskAnalysis,
		MaxTokens: 256,
	})
	if err != nil {
		slog.Warn("compaction failed, continuing without summary", "error", err)
		return
	}

	if err := e.store.SetSummary(conv.ID, resp.Content, compactUpTo); err != nil {
		slog.Warn("failed to save summary", "error", err)
		return
	}

	// Update the in-memory conv so buildContextMessages uses the new summary.
	conv.Summary = resp.Content
	conv.CompactedAt = compactUpTo

	slog.Info("conversation compacted",
		"conversation_id", conv.ID,
		"compacted_messages", compactUpTo,
		"remaining_messages", len(conv.Messages)-compactUpTo,
	)
}

func (e *Engine) getOrCreateConversation(userID string) (*Conversation, error) {
	conv, found := e.store.GetActiveConversation(userID)
	if found {
		return conv, nil
	}
	return e.createConversation(userID, "teaching")
}

func (e *Engine) createConversation(userID, state string) (*Conversation, error) {
	id, err := e.store.CreateConversation(Conversation{
		UserID: userID,
		State:  state,
	})
	if err != nil {
		return nil, err
	}
	conv, err := e.store.GetConversation(id)
	if err != nil {
		return nil, err
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         userID,
		EventType:      "session_started",
		Data: map[string]any{
			"state": conv.State,
		},
	})
	return conv, nil
}

func (e *Engine) logEventAsync(event Event) {
	go func() {
		if err := e.eventLogger.LogEvent(event); err != nil {
			slog.Warn("failed to log event",
				"event_type", event.EventType,
				"conversation_id", event.ConversationID,
				"user_id", event.UserID,
				"error", err,
			)
		}
	}()
}

func (e *Engine) handleCommand(_ context.Context, msg chat.InboundMessage) (string, error) {
	cmd := strings.Split(msg.Text, " ")[0]

	switch cmd {
	case "/start":
		e.endActiveConversation(msg.UserID)
		return e.handleStart(msg.UserID, msg)
	case "/clear":
		e.endActiveConversation(msg.UserID)
		return "Sejarah perbualan telah dikosongkan. Hantar soalan baru untuk mula semula.", nil
	default:
		return fmt.Sprintf("Arahan tidak diketahui: %s\nGuna /start untuk bermula atau /clear untuk reset perbualan.", cmd), nil
	}
}

func (e *Engine) endActiveConversation(userID string) {
	if conv, found := e.store.GetActiveConversation(userID); found {
		if err := e.store.EndConversation(conv.ID); err != nil {
			slog.Error("failed to end conversation", "error", err)
		}
	}
}

func (e *Engine) handleStart(userID string, msg chat.InboundMessage) (string, error) {
	// Explicitly create an onboarding conversation on /start. In Postgres-backed
	// deployments this also guarantees the user record exists before first question.
	if _, err := e.createConversation(userID, "onboarding"); err != nil {
		slog.Error("failed to create onboarding conversation", "user_id", userID, "error", err)
		return "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar.", nil
	}

	name := msg.FirstName
	if name == "" {
		name = msg.Username
	}
	if name == "" {
		name = "pelajar"
	}

	return fmt.Sprintf(`Hai %s!

Saya P&AI Bot — tutor matematik peribadi anda!

Saya boleh membantu anda dengan KSSM Matematik:
- Tingkatan 1
- Tingkatan 2
- Tingkatan 3

Tingkatan berapa anda sekarang?
Balas dengan: 1, 2, atau 3.`, name), nil
}

func (e *Engine) handleOnboardingSelection(ctx context.Context, msg chat.InboundMessage, conv *Conversation) string {
	if err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: msg.Text,
	}); err != nil {
		slog.Error("failed to store onboarding user message", "error", err)
	}

	form, ok := parseFormSelection(msg.Text)
	if !ok {
		form, ok = e.classifyFormSelectionWithAI(ctx, msg.Text)
	}
	if !ok {
		response := "Saya belum pasti tingkatan anda. Boleh jawab bebas (contoh: saya tingkatan 2 / form two), atau balas terus 1, 2, atau 3."
		if err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store onboarding assistant message", "error", err)
		}
		return response
	}

	if err := e.store.UpdateConversationState(conv.ID, "teaching"); err != nil {
		slog.Error("failed to update conversation state", "conversation_id", conv.ID, "error", err)
		return "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar."
	}

	response := fmt.Sprintf("Bagus, anda Tingkatan %d. Sekarang hantar topik atau soalan matematik yang anda mahu belajar.", form)
	if err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store onboarding assistant message", "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "onboarding_completed",
		Data: map[string]any{
			"selected_form": form,
		},
	})
	return response
}

func (e *Engine) classifyFormSelectionWithAI(ctx context.Context, answer string) (int, bool) {
	resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
		Messages: []ai.Message{
			{
				Role: "system",
				Content: `Classify the student's form level from their answer.
Return exactly one token only: 1, 2, 3, or unknown.
No extra words.`,
			},
			{
				Role:    "user",
				Content: answer,
			},
		},
		Task:      ai.TaskAnalysis,
		MaxTokens: 8,
	})
	if err != nil {
		slog.Warn("onboarding form classification failed", "error", err)
		return 0, false
	}

	switch strings.TrimSpace(strings.ToLower(resp.Content)) {
	case "1":
		return 1, true
	case "2":
		return 2, true
	case "3":
		return 3, true
	default:
		return 0, false
	}
}

var formSelectionPattern = regexp.MustCompile(`(?i)^\s*(tingkatan|form|f)?\s*([123])\s*$`)

var singleDigitPattern = regexp.MustCompile(`^\s*([123])\s*$`)

func parseFormSelection(text string) (int, bool) {
	trimmed := strings.TrimSpace(text)
	if singleDigitPattern.MatchString(trimmed) {
		return int(trimmed[0] - '0'), true
	}

	m := formSelectionPattern.FindStringSubmatch(trimmed)
	if len(m) >= 3 {
		return int(m[2][0] - '0'), true
	}

	lower := strings.ToLower(trimmed)
	hasContext := strings.Contains(lower, "tingkatan") || strings.Contains(lower, "form")

	if hasContext {
		if hasAnyToken(lower, []string{"1", "satu", "one", "first", "pertama"}) {
			return 1, true
		}
		if hasAnyToken(lower, []string{"2", "dua", "two", "second", "kedua"}) {
			return 2, true
		}
		if hasAnyToken(lower, []string{"3", "tiga", "three", "third", "ketiga"}) {
			return 3, true
		}
	}

	switch lower {
	case "satu", "one", "first", "pertama":
		return 1, true
	case "dua", "two", "second", "kedua":
		return 2, true
	case "tiga", "three", "third", "ketiga":
		return 3, true
	}
	return 0, false
}

func hasAnyToken(text string, tokens []string) bool {
	for _, token := range tokens {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func (e *Engine) buildSystemPrompt(_ chat.InboundMessage) string {
	return `You are P&AI Bot, a friendly and encouraging mathematics tutor for Malaysian secondary school students.

CURRICULUM: KSSM Matematik (Form 1, 2, 3) — focus on Algebra topics.

LANGUAGE: Respond in the same language the student uses. Most students use Bahasa Melayu or English. Mix both if the student does.

TEACHING STYLE:
- Start with what the student knows, build from there
- Use simple, relatable examples (Malaysian context: ringgit, kopitiam, school scenarios)
- Break complex problems into small steps
- Celebrate small wins ("Bagus!", "Betul!")
- If the student is stuck, give a hint before the answer
- Use mathematical notation where needed
- Write equations in plain text (example: 6x = 30, x = 5). Do not use LaTeX delimiters like \[ \], \( \), or $$.
- Keep responses concise — this is a chat, not a textbook

RULES:
- Never give answers without explanation
- Always check if the student understood before moving on
- If unsure of the student's level, ask a diagnostic question
- If an image is attached, analyze the image content first before answering.
- Only say you cannot identify an image when it is genuinely unreadable; in that case ask for a clearer retake.
- If the student asks a follow-up about an earlier image but did not reply to that image (or reattach it), ask them to reply directly to the image message.
- Be patient and never condescending`
}

func normalizeEquationFormatting(content string) string {
	replacer := strings.NewReplacer(
		`\\[`, "",
		`\\]`, "",
		`\\(`, "",
		`\\)`, "",
		`$$`, "",
		`\[`, "",
		`\]`, "",
		`\(`, "",
		`\)`, "",
		`\times`, "x",
		`\cdot`, "*",
		`\div`, "/",
	)
	return replacer.Replace(content)
}

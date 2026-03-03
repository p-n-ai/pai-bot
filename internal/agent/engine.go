package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
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
	CurriculumLoader      *curriculum.Loader
	ContextResolver       ContextResolver
	CompactThreshold      int // messages before compaction triggers (default 20)
	CompactTokenThreshold int // estimated tokens before compaction triggers (default 3000)
	KeepRecent            int // recent messages to keep after compaction (default 6)
}

// Engine is the core conversation processor.
type Engine struct {
	aiRouter              *ai.Router
	store                 ConversationStore
	eventLogger           EventLogger
	contextResolver       ContextResolver
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

	contextResolver := cfg.ContextResolver
	if contextResolver == nil {
		if cfg.CurriculumLoader != nil {
			contextResolver = NewCurriculumContextResolver(cfg.CurriculumLoader)
		} else {
			contextResolver = NoopContextResolver{}
		}
	}
	return &Engine{
		aiRouter:              cfg.AIRouter,
		store:                 store,
		eventLogger:           eventLogger,
		contextResolver:       contextResolver,
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

	matchedTopic, teachingNotes := e.contextResolver.Resolve(msg.Text)

	// Build messages: system prompt + (optional summary) + recent messages.
	systemPrompt := e.buildSystemPrompt(msg, matchedTopic, teachingNotes)
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
		fmt.Fprintf(&content, "%s: %s\n", role, m.Content)
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
	id, err := e.store.CreateConversation(Conversation{
		UserID: userID,
		State:  "teaching",
	})
	if err != nil {
		return nil, err
	}
	conv, err = e.store.GetConversation(id)
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
		return e.handleStart(msg)
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

func (e *Engine) handleStart(msg chat.InboundMessage) (string, error) {
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

Apa yang anda ingin belajar hari ini?`, name), nil
}

func (e *Engine) buildSystemPrompt(_ chat.InboundMessage, topic *curriculum.Topic, teachingNotes string) string {
	base := `You are P&AI Bot, a supportive mathematics tutor for Malaysian secondary students (KSSM Form 1-3, Algebra-first).

PRIMARY GOAL:
Help the student understand and solve the problem independently, not just get a final answer.

LANGUAGE:
Respond in the student's language (Bahasa Melayu, English, or mixed if they mix).

STRUCTURED SOLVING LOOP (follow in order):
1. Understand: Restate the student's question briefly and identify what is asked.
2. Plan: Give a short plan (1-3 steps) before calculating.
3. Solve: Show steps clearly, with plain-text equations.
4. Verify: Check the result quickly (substitute or sanity-check).
5. Connect: Link to the underlying concept and when to use it again.

TEACHING RULES:
1. Keep answers concise and chat-friendly.
2. Use simple, relatable examples (ringgit, school, daily life) when helpful.
3. If the student is stuck, give a hint first; reveal full answer after effort.
4. Ask one quick check-for-understanding question when appropriate.
5. Never be condescending.

SAFETY + ACCURACY:
1. Do not invent facts, formulas, or curriculum references.
2. If context is missing, ask a clarifying question before solving.
3. If uncertain, state what is uncertain and propose the next step.

IMAGE HANDLING:
1. If an image is attached, analyze it first, then answer.
2. If image text is unclear, state what is unclear and ask for a clearer retake.
3. If the student asks a follow-up about an earlier image but did not reply to that image (or reattach it), ask them to reply directly to the image message.

FORMAT CONSTRAINT:
Use plain-text math only (example: 6x = 30, x = 5). Do not use LaTeX delimiters like \[ \], \( \), or $$.`

	if topic == nil {
		return base
	}

	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nTOPIC CONTEXT:\n")
	fmt.Fprintf(&b, "- Matched topic ID: %s\n", topic.ID)
	fmt.Fprintf(&b, "- Matched topic name: %s\n", topic.Name)
	if topic.SyllabusID != "" {
		fmt.Fprintf(&b, "- Matched syllabus: %s\n", topic.SyllabusID)
	}
	if topic.SubjectID != "" {
		fmt.Fprintf(&b, "- Matched subject: %s\n", topic.SubjectID)
	}
	if len(topic.LearningObjectives) > 0 {
		b.WriteString("- Learning objectives:\n")
		for i, lo := range topic.LearningObjectives {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&b, "  - %s\n", lo.Text)
		}
	}
	if teachingNotes != "" {
		b.WriteString("\nTEACHING NOTES (use as guidance):\n")
		b.WriteString(truncateForPrompt(teachingNotes, 2500))
		b.WriteString("\n")
	}
	b.WriteString("\nINSTRUCTIONS FOR THIS REPLY:\n")
	b.WriteString("- Prioritize the matched topic context and teaching notes.\n")
	b.WriteString("- Include one short curriculum citation in this format: ")
	b.WriteString("\"")
	b.WriteString(topic.SyllabusID)
	b.WriteString(" > ")
	b.WriteString(topic.Name)
	b.WriteString("\".\n")
	return b.String()
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

func truncateForPrompt(text string, max int) string {
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max] + "\n...[truncated]"
}

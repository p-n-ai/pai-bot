package agent

import (
	"context"
	"fmt"
	"log/slog"
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
	CompactThreshold      int // messages before compaction triggers (default 20)
	CompactTokenThreshold int // estimated tokens before compaction triggers (default 3000)
	KeepRecent            int // recent messages to keep after compaction (default 6)
}

// Engine is the core conversation processor.
type Engine struct {
	aiRouter              *ai.Router
	store                 ConversationStore
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
	return &Engine{
		aiRouter:              cfg.AIRouter,
		store:                 store,
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
	if msg.ReplyToText != "" {
		userContent = fmt.Sprintf("[Replying to: \"%s\"]\n\n%s", msg.ReplyToText, msg.Text)
	}

	// Record user message.
	if err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: userContent,
	}); err != nil {
		slog.Error("failed to store user message", "error", err)
	}

	// Refresh conversation to get latest messages.
	conv, _ = e.store.GetConversation(conv.ID)

	// Compact if needed (summarize older messages).
	e.maybeCompact(ctx, conv)

	// Build messages: system prompt + (optional summary) + recent messages.
	systemPrompt := e.buildSystemPrompt(msg)
	messages := []ai.Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, e.buildContextMessages(conv)...)

	// Call AI
	resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
		Messages:  messages,
		Task:      ai.TaskTeaching,
		MaxTokens: 1024,
	})
	if err != nil {
		slog.Error("AI completion failed", "error", err)
		return "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar.", nil
	}

	// Record assistant response with token metadata.
	if err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:         "assistant",
		Content:      resp.Content,
		Model:        resp.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	}); err != nil {
		slog.Error("failed to store assistant message", "error", err)
	}

	return resp.Content, nil
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
	id, err := e.store.CreateConversation(Conversation{
		UserID: userID,
		State:  "teaching",
	})
	if err != nil {
		return nil, err
	}
	return e.store.GetConversation(id)
}

func (e *Engine) handleCommand(_ context.Context, msg chat.InboundMessage) (string, error) {
	cmd := strings.Split(msg.Text, " ")[0]

	switch cmd {
	case "/start":
		// End any active conversation.
		if conv, found := e.store.GetActiveConversation(msg.UserID); found {
			if err := e.store.EndConversation(conv.ID); err != nil {
				slog.Error("failed to end conversation", "error", err)
			}
		}
		return e.handleStart(msg)
	default:
		return fmt.Sprintf("Arahan tidak diketahui: %s\nGuna /start untuk bermula.", cmd), nil
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
- Keep responses concise — this is a chat, not a textbook

RULES:
- Never give answers without explanation
- Always check if the student understood before moving on
- If unsure of the student's level, ask a diagnostic question
- Be patient and never condescending`
}

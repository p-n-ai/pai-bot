package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

// EngineConfig holds dependencies for the agent engine.
type EngineConfig struct {
	AIRouter *ai.Router
}

// Engine is the core conversation processor.
type Engine struct {
	aiRouter *ai.Router
}

// NewEngine creates a new agent engine.
func NewEngine(cfg EngineConfig) *Engine {
	return &Engine{
		aiRouter: cfg.AIRouter,
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

	// Build system prompt
	systemPrompt := e.buildSystemPrompt(msg)

	// Call AI
	resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: msg.Text},
		},
		Task:      ai.TaskTeaching,
		MaxTokens: 1024,
	})
	if err != nil {
		slog.Error("AI completion failed", "error", err)
		return "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar.", nil
	}

	return resp.Content, nil
}

func (e *Engine) handleCommand(_ context.Context, msg chat.InboundMessage) (string, error) {
	cmd := strings.Split(msg.Text, " ")[0]

	switch cmd {
	case "/start":
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

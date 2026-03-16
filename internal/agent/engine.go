package agent

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

const (
	defaultCompactThreshold      = 20
	defaultCompactTokenThreshold = 20000 // ~20k tokens triggers compaction
	defaultKeepRecent            = 6
	defaultRatingPromptEvery     = 5
	ratingPromptText             = "Sebelum kita teruskan, boleh beri rating 1-5 untuk bantuan setakat ini? (1=tak membantu, 5=sangat membantu)"
	// ReviewActionCode is a control marker emitted by AI to trigger rating UI/actions.
	ReviewActionCode = "[[PAI_REVIEW]]"
	langPrefCodeEN   = "[[PAI_PREF_LANG:en]]"
	langPrefCodeMS   = "[[PAI_PREF_LANG:ms]]"
	langPrefCodeZH   = "[[PAI_PREF_LANG:zh]]"
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
	DisableMultiLanguage  bool
	RatingPromptEvery     int // ask for rating every N tutoring replies (default 5)
	Tracker               progress.Tracker
	Streaks               progress.StreakTracker
	XP                    progress.XPTracker
	Goals                 GoalStore
	DevMode               bool
}

// Engine is the core conversation processor.
type Engine struct {
	aiRouter              *ai.Router
	store                 ConversationStore
	eventLogger           EventLogger
	curriculumLoader      *curriculum.Loader
	contextResolver       ContextResolver
	compactThreshold      int
	compactTokenThreshold int
	keepRecent            int
	disableMultiLanguage  bool
	ratingPromptEvery     int
	tracker               progress.Tracker
	streaks               progress.StreakTracker
	xp                    progress.XPTracker
	goals                 GoalStore
	devMode               bool
	prereqGraph           *curriculum.PrereqGraph
	unlocks               *pendingUnlocks
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
	ratingEvery := cfg.RatingPromptEvery
	if ratingEvery <= 0 {
		ratingEvery = defaultRatingPromptEvery
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
		curriculumLoader:      cfg.CurriculumLoader,
		contextResolver:       contextResolver,
		compactThreshold:      threshold,
		compactTokenThreshold: tokenThreshold,
		keepRecent:            keepRecent,
		disableMultiLanguage:  cfg.DisableMultiLanguage,
		ratingPromptEvery:     ratingEvery,
		tracker:               cfg.Tracker,
		streaks:               cfg.Streaks,
		xp:                    cfg.XP,
		goals:                 cfg.Goals,
		devMode:               cfg.DevMode,
		prereqGraph:           buildPrereqGraph(cfg.CurriculumLoader),
		unlocks:               newPendingUnlocks(),
	}
}

// ProcessMessage handles an incoming message and returns a response.
func (e *Engine) ProcessMessage(ctx context.Context, msg chat.InboundMessage) (string, error) {
	slog.Info("processing message",
		"channel", msg.Channel,
		"user_id", msg.UserID,
		"text_len", len(msg.Text),
	)

	e.maybePersistUserProfile(msg)

	// Drain any pending topic unlock notifications from previous mastery updates.
	unlockPrefix := e.drainUnlockNotification(msg.UserID, e.messageLocale(msg, nil))

	// Handle commands
	if strings.HasPrefix(msg.Text, "/") {
		resp, err := e.handleCommand(ctx, msg)
		if err != nil {
			return resp, err
		}
		if unlockPrefix != "" {
			return unlockPrefix + "\n\n" + resp, nil
		}
		return resp, nil
	}
	// Auto-trigger onboarding for first-time users who send a normal message.
	if e.supportsAutoStartLookup() && !e.store.UserExists(msg.UserID) {
		e.logEventAsync(Event{
			UserID:    msg.UserID,
			EventType: "auto_start_triggered",
			Data: map[string]any{
				"channel": msg.Channel,
				"source":  "chat_flow",
			},
		})
		return e.handleStart(msg.UserID, msg)
	}

	// Get or create active conversation.
	conv, err := e.getOrCreateConversation(msg.UserID)
	if err != nil {
		slog.Error("failed to get conversation", "error", err)
		return i18n.S(e.messageLocale(msg, nil), i18n.MsgTechnicalIssue), nil
	}
	if strings.HasPrefix(conv.State, "onboarding") {
		return e.handleOnboardingSelection(ctx, msg, conv), nil
	}
	if conv.State == "language_selection" {
		return e.handleLanguageSelection(msg, conv), nil
	}
	if response, handled := e.maybeHandleRatingInput(msg, conv); handled {
		return response, nil
	}
	if response, handled := e.maybeHandlePendingGoal(ctx, msg, conv); handled {
		return response, nil
	}
	if response, handled := e.maybeHandleQuizTurn(ctx, msg, conv); handled {
		return response, nil
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
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
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
	replyCount := countTutoringReplies(conv.Messages) + 1
	promptRequested := shouldRequestRatingAfterReply(replyCount, e.ratingPromptEvery)

	// Build messages: system prompt + (optional summary) + recent messages.
	systemPrompt := e.buildSystemPrompt(msg, conv, matchedTopic, teachingNotes)
	messages := []ai.Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, e.buildContextMessages(conv)...)
	if msg.HasImage && msg.ImageDataURL == "" {
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgImageProcessingFailed), nil
	}
	if msg.ImageDataURL != "" {
		messages = append(messages, ai.Message{
			Role:      "user",
			Content:   "Attached image from the student. Analyze this image directly and answer based on what you see. If unreadable, say exactly what is unclear and how to retake it.",
			ImageURLs: []string{msg.ImageDataURL},
		})
	}
	if promptRequested {
		messages = append(messages, ai.Message{
			Role:    "user",
			Content: "At the end of your response, ask for a quick 1-5 rating in one short sentence and include the exact control token [[PAI_REVIEW]] once.",
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
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), nil
	}

	// Telegram does not render LaTeX blocks; keep equations plain.
	plainContent := normalizeLegacyExamReferences(normalizeEquationFormatting(resp.Content))
	finalContent := plainContent
	if promptRequested && !strings.Contains(finalContent, ReviewActionCode) {
		finalContent = strings.TrimSpace(finalContent) + "\n\n" + ReviewActionCode
	}

	// Record assistant response with token metadata.
	assistantMessageID, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:         "assistant",
		Content:      finalContent,
		Model:        resp.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	})
	if err != nil {
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
			"text_len":      len(finalContent),
			"has_image":     msg.HasImage,
		},
	})
	e.assessMasteryAsync(ctx, msg.UserID, matchedTopic, userContent, plainContent)
	e.recordActivityAsync(msg.UserID)

	if promptRequested {
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "answer_rating_requested",
			Data: map[string]any{
				"channel":                msg.Channel,
				"after_tutoring_replies": replyCount,
				"rated_message_id":       assistantMessageID,
			},
		})
	}
	responseContent := finalContent
	if promptRequested && assistantMessageID != "" {
		responseContent = injectReviewTokenWithMessageID(finalContent, assistantMessageID)
	}

	if unlockPrefix != "" {
		responseContent = unlockPrefix + "\n\n" + responseContent
	}

	return responseContent, nil
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
			cleanContent := sanitizeControlContent(m.Content)
			if cleanContent == "" {
				continue
			}
			messages = append(messages, ai.Message{Role: m.Role, Content: cleanContent})
		}
	} else {
		for _, m := range conv.Messages {
			cleanContent := sanitizeControlContent(m.Content)
			if cleanContent == "" {
				continue
			}
			messages = append(messages, ai.Message{Role: m.Role, Content: cleanContent})
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

func (e *Engine) assessMasteryAsync(ctx context.Context, userID string, topic *curriculum.Topic, userMessage, aiResponse string) {
	if e.tracker == nil || topic == nil {
		return
	}
	go func() {
		prompt := fmt.Sprintf(
			"Rate how well the student demonstrated understanding of %q in this exchange.\n\nStudent: %s\n\nTutor: %s\n\nReturn ONLY a single decimal number between 0.0 and 1.0.",
			topic.Name,
			truncateForPrompt(userMessage, 500),
			truncateForPrompt(aiResponse, 500),
		)
		resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
			Messages: []ai.Message{
				{Role: "system", Content: "You are a grading assistant. Return ONLY a single float between 0.0 and 1.0. No other text."},
				{Role: "user", Content: prompt},
			},
			Task:      ai.TaskGrading,
			MaxTokens: 8,
		})
		if err != nil {
			slog.Warn("mastery assessment AI call failed", "user_id", userID, "topic", topic.ID, "error", err)
			return
		}
		delta, err := strconv.ParseFloat(strings.TrimSpace(resp.Content), 64)
		if err != nil {
			slog.Warn("mastery assessment parse failed", "user_id", userID, "topic", topic.ID, "response", resp.Content, "error", err)
			return
		}
		syllabusID := topic.SyllabusID
		if syllabusID == "" {
			syllabusID = "default"
		}
		if err := e.tracker.UpdateMastery(userID, syllabusID, topic.ID, delta); err != nil {
			slog.Warn("mastery update failed", "user_id", userID, "topic", topic.ID, "error", err)
			return
		}
		e.syncGoalProgress(userID, syllabusID, topic.ID)
		e.checkTopicUnlocks(userID, syllabusID, topic)
	}()
}

// recordActivityAsync records streak activity and awards session XP in a goroutine.
func (e *Engine) recordActivityAsync(userID string) {
	go func() {
		now := time.Now()

		// Record streak activity.
		if e.streaks != nil {
			if err := e.streaks.RecordActivity(userID, now); err != nil {
				slog.Warn("streak record failed", "user_id", userID, "error", err)
			} else {
				// Check for milestone celebration.
				s, _ := e.streaks.GetStreak(userID)
				if progress.IsStreakMilestone(s.CurrentStreak) && e.xp != nil {
					_ = e.xp.Award(userID, progress.XPSourceStreak, progress.XPStreakMilestone, map[string]any{
						"streak_days": s.CurrentStreak,
					})
				}
			}
		}

		// Award session XP.
		if e.xp != nil {
			_ = e.xp.Award(userID, progress.XPSourceSession, progress.XPSession, nil)
		}
	}()
}

func (e *Engine) handleCommand(ctx context.Context, msg chat.InboundMessage) (string, error) {
	fields := strings.Fields(msg.Text)
	cmd := fields[0]
	locale := e.messageLocale(msg, nil)

	switch cmd {
	case "/start":
		e.endActiveConversation(msg.UserID)
		return e.handleStart(msg.UserID, msg)
	case "/clear":
		e.clearUserRuntimeState(msg.UserID)
		return i18n.S(locale, i18n.MsgHistoryCleared), nil
	case "/reset-profile":
		e.resetLearnerProfile(msg.UserID)
		onboarding, err := e.handleStart(msg.UserID, msg)
		if err != nil {
			return "", err
		}
		return i18n.S(locale, i18n.MsgProfileReset) + "\n\n" + onboarding, nil
	case "/language":
		return e.handleLanguageCommand(msg, fields[1:])
	case "/progress":
		return e.handleProgressCommand(msg)
	case "/goal":
		return e.handleGoalCommand(ctx, msg, fields[1:])
	case "/learn":
		return e.handleLearnCommand(ctx, msg, fields[1:])
	case "/dev-reset":
		if !e.devMode {
			return i18n.S(locale, i18n.MsgUnknownCommand, cmd), nil
		}
		return e.handleDevReset(msg)
	case "/dev-boost":
		if !e.devMode {
			return i18n.S(locale, i18n.MsgUnknownCommand, cmd), nil
		}
		return e.handleDevBoost(msg, fields[1:])
	default:
		return i18n.S(locale, i18n.MsgUnknownCommand, cmd), nil
	}
}

func (e *Engine) handleLanguageCommand(msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)
	if e.disableMultiLanguage {
		return i18n.S(locale, i18n.MsgMultilingualDisabled), nil
	}
	conv, err := e.getOrCreateConversation(msg.UserID)
	if err != nil {
		slog.Error("failed to get conversation for /language", "user_id", msg.UserID, "error", err)
		return i18n.S(locale, i18n.MsgTechnicalIssue), nil
	}
	locale = e.messageLocale(msg, conv)

	if len(args) == 0 {
		nextState := "language_selection"
		if strings.HasPrefix(conv.State, "onboarding") {
			nextState = "onboarding_language"
		}
		if err := e.store.UpdateConversationState(conv.ID, nextState); err != nil {
			slog.Error("failed to set language selection state", "conversation_id", conv.ID, "error", err)
			return i18n.S(locale, i18n.MsgTechnicalIssue), nil
		}
		return i18n.S(locale, i18n.MsgLanguagePrompt), nil
	}

	lang, ok := parseLanguagePreference(strings.Join(args, " "))
	if !ok {
		return i18n.S(locale, i18n.MsgLanguageInvalidFormat), nil
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: languagePreferenceControlCode(lang),
	}); err != nil {
		slog.Error("failed to store language preference marker", "conversation_id", conv.ID, "error", err)
	}
	if err := e.store.SetUserPreferredLanguage(msg.UserID, lang); err != nil {
		slog.Error("failed to persist user preferred language", "user_id", msg.UserID, "error", err)
	}
	onboardingFlow := strings.HasPrefix(conv.State, "onboarding")
	if onboardingFlow {
		if err := e.store.UpdateConversationState(conv.ID, "onboarding_form"); err != nil {
			slog.Error("failed to move onboarding to form step", "conversation_id", conv.ID, "error", err)
			return i18n.S(lang, i18n.MsgTechnicalIssue), nil
		}
	} else if conv.State == "language_selection" {
		if err := e.store.UpdateConversationState(conv.ID, "teaching"); err != nil {
			slog.Error("failed to restore conversation state after /language", "conversation_id", conv.ID, "error", err)
		}
	}

	source := "command"
	if onboardingFlow {
		source = "onboarding_command"
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "language_changed",
		Data: map[string]any{
			"preferred_language": lang,
			"source":             source,
		},
	})

	if onboardingFlow {
		return languageChangedMessage(lang) + "\n\n" + onboardingFormPrompt(lang), nil
	}
	return languageChangedMessage(lang), nil
}

func (e *Engine) handleProgressCommand(msg chat.InboundMessage) (string, error) {
	if e.tracker == nil {
		return "Progress tracking is not enabled.", nil
	}

	items, err := e.tracker.GetAllProgress(msg.UserID)
	if err != nil {
		slog.Error("failed to get progress", "user_id", msg.UserID, "error", err)
		return i18n.S(e.messageLocale(msg, nil), i18n.MsgTechnicalIssue), nil
	}

	var totalXP int
	if e.xp != nil {
		totalXP, _ = e.xp.GetTotal(msg.UserID)
	}
	var streak int
	if e.streaks != nil {
		s, _ := e.streaks.GetStreak(msg.UserID)
		streak = s.CurrentStreak
	}
	return e.appendGoalToProgressReport(msg.UserID, progress.FormatProgressReport(items, totalXP, streak)), nil
}

func (e *Engine) endActiveConversation(userID string) {
	if conv, found := e.store.GetActiveConversation(userID); found {
		if err := e.store.EndConversation(conv.ID); err != nil {
			slog.Error("failed to end conversation", "error", err)
		}
	}
}

func (e *Engine) clearUserRuntimeState(userID string) {
	e.endActiveConversation(userID)
	if err := e.store.SetUserPreferredQuizIntensity(userID, ""); err != nil {
		slog.Error("failed to clear quiz intensity preference", "user_id", userID, "error", err)
	}
}

func (e *Engine) resetLearnerProfile(userID string) {
	e.endActiveConversation(userID)
	if err := e.store.SetUserForm(userID, ""); err != nil {
		slog.Error("failed to clear learner form", "user_id", userID, "error", err)
	}
	if err := e.store.SetUserPreferredLanguage(userID, ""); err != nil {
		slog.Error("failed to clear learner language", "user_id", userID, "error", err)
	}
	if err := e.store.SetUserPreferredQuizIntensity(userID, ""); err != nil {
		slog.Error("failed to clear learner quiz intensity", "user_id", userID, "error", err)
	}
}

func (e *Engine) supportsAutoStartLookup() bool {
	// MemoryStore does not model a durable user directory; using auto-start in tests/dev
	// would hijack many normal chat-path tests. Enable this behavior for persistent stores.
	_, isMemory := e.store.(*MemoryStore)
	return !isMemory
}

func (e *Engine) handleStart(userID string, msg chat.InboundMessage) (string, error) {
	// Explicitly create an onboarding conversation on /start. In Postgres-backed
	// deployments this also guarantees the user record exists before first question.
	initialState := "onboarding_language"
	if e.disableMultiLanguage {
		initialState = "onboarding_form"
	}
	if _, err := e.createConversation(userID, initialState); err != nil {
		slog.Error("failed to create onboarding conversation", "user_id", userID, "error", err)
		return i18n.S(e.messageLocale(msg, nil), i18n.MsgTechnicalIssue), nil
	}

	locale := e.messageLocale(msg, nil)
	name := msg.FirstName
	if name == "" {
		name = msg.Username
	}
	if name == "" {
		name = i18n.S(locale, i18n.MsgDefaultStudentName)
	}

	if e.disableMultiLanguage {
		return i18n.S(locale, i18n.MsgStartOnboardingForm, name), nil
	}

	return i18n.S(locale, i18n.MsgStartOnboardingLang, name), nil
}

func (e *Engine) maybePersistUserProfile(msg chat.InboundMessage) {
	if msg.UserID == "" {
		return
	}
	name := preferredIncomingName(msg)
	if name != "" {
		if err := e.store.SetUserName(msg.UserID, name); err != nil {
			slog.Error("failed to persist user name", "user_id", msg.UserID, "error", err)
		}
	}
}

func preferredIncomingName(msg chat.InboundMessage) string {
	name := strings.TrimSpace(msg.FirstName)
	if name != "" {
		return name
	}
	return strings.TrimSpace(msg.Username)
}

func (e *Engine) handleOnboardingSelection(ctx context.Context, msg chat.InboundMessage, conv *Conversation) string {
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: msg.Text,
	}); err != nil {
		slog.Error("failed to store onboarding user message", "error", err)
	}

	if !e.disableMultiLanguage && conv.State == "onboarding_language" {
		lang, ok := parseLanguagePreference(msg.Text)
		if !ok {
			response := i18n.S(e.messageLocale(msg, conv), i18n.MsgLanguageUnclear)
			if _, err := e.store.AddMessage(conv.ID, StoredMessage{
				Role:    "assistant",
				Content: response,
			}); err != nil {
				slog.Error("failed to store onboarding assistant message", "error", err)
			}
			return response
		}

		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: languagePreferenceControlCode(lang),
		}); err != nil {
			slog.Error("failed to store language preference marker", "error", err)
		}
		if err := e.store.SetUserPreferredLanguage(msg.UserID, lang); err != nil {
			slog.Error("failed to persist user preferred language", "user_id", msg.UserID, "error", err)
		}
		if err := e.store.UpdateConversationState(conv.ID, "onboarding_form"); err != nil {
			slog.Error("failed to update conversation state", "conversation_id", conv.ID, "error", err)
			return i18n.S(lang, i18n.MsgTechnicalIssue)
		}

		response := languageChangedMessage(lang) + "\n\n" + onboardingFormPrompt(lang)
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store onboarding assistant message", "error", err)
		}
		return response
	}

	// Legacy fallback: old onboarding state behaves like form-selection step.
	form, ok := parseFormSelection(msg.Text)
	if !ok {
		form, ok = e.classifyFormSelectionWithAI(ctx, msg.Text)
	}
	if !ok {
		response := i18n.S(e.messageLocale(msg, conv), i18n.MsgOnboardingFormUnclear)
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store onboarding assistant message", "error", err)
		}
		return response
	}

	if err := e.store.UpdateConversationState(conv.ID, "teaching"); err != nil {
		slog.Error("failed to update conversation state", "conversation_id", conv.ID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
	}
	if err := e.store.SetUserForm(msg.UserID, strconv.Itoa(form)); err != nil {
		slog.Error("failed to persist user form", "user_id", msg.UserID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
	}

	lang, hasLangPref := e.preferredLanguageForConversation(conv)
	if !hasLangPref {
		lang = "ms"
	}
	response := onboardingCompletionMessage(lang, form)
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
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
			"selected_form":      form,
			"preferred_language": lang,
		},
	})
	return response
}

func (e *Engine) handleLanguageSelection(msg chat.InboundMessage, conv *Conversation) string {
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: msg.Text,
	}); err != nil {
		slog.Error("failed to store language selection message", "error", err)
	}

	lang, ok := parseLanguagePreference(msg.Text)
	if !ok {
		response := i18n.S(e.messageLocale(msg, conv), i18n.MsgLanguageUnclear)
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store language selection clarification", "error", err)
		}
		return response
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: languagePreferenceControlCode(lang),
	}); err != nil {
		slog.Error("failed to store language preference marker", "error", err)
	}
	if err := e.store.SetUserPreferredLanguage(msg.UserID, lang); err != nil {
		slog.Error("failed to persist user preferred language", "user_id", msg.UserID, "error", err)
	}
	if err := e.store.UpdateConversationState(conv.ID, "teaching"); err != nil {
		slog.Error("failed to restore teaching state after language selection", "conversation_id", conv.ID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
	}

	response := languageChangedMessage(lang)
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store language changed response", "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "language_changed",
		Data: map[string]any{
			"preferred_language": lang,
			"source":             "command_interactive",
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

func parseLanguagePreference(text string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return "", false
	}
	if strings.HasPrefix(lower, "lang:") {
		switch strings.TrimPrefix(lower, "lang:") {
		case "en":
			return "en", true
		case "ms", "bm":
			return "ms", true
		case "zh":
			return "zh", true
		}
	}

	switch {
	case strings.Contains(lower, "english"), strings.Contains(lower, "inggeris"), lower == "en":
		return "en", true
	case strings.Contains(lower, "bahasa melayu"), strings.Contains(lower, "bahasa malaysia"), lower == "bm", lower == "ms", strings.Contains(lower, "melayu"), strings.Contains(lower, "malay"):
		return "ms", true
	case strings.Contains(lower, "chinese"), strings.Contains(lower, "mandarin"), strings.Contains(lower, "cina"), strings.Contains(text, "中文"), strings.Contains(text, "华文"), strings.Contains(text, "汉语"):
		return "zh", true
	default:
		return "", false
	}
}

func languagePreferenceControlCode(lang string) string {
	switch lang {
	case "en":
		return langPrefCodeEN
	case "zh":
		return langPrefCodeZH
	default:
		return langPrefCodeMS
	}
}

func preferredLanguageFromMessages(messages []StoredMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		content := strings.TrimSpace(messages[i].Content)
		switch content {
		case langPrefCodeEN:
			return "en"
		case langPrefCodeMS:
			return "ms"
		case langPrefCodeZH:
			return "zh"
		}
	}
	return ""
}

func (e *Engine) preferredLanguageForConversation(conv *Conversation) (string, bool) {
	if e.disableMultiLanguage {
		return "ms", true
	}
	if conv != nil {
		if lang, ok := e.store.GetUserPreferredLanguage(conv.UserID); ok && lang != "" {
			return lang, true
		}
		lang := preferredLanguageFromMessages(conv.Messages)
		if lang != "" {
			return lang, true
		}
	}
	return "", false
}

func (e *Engine) messageLocale(msg chat.InboundMessage, conv *Conversation) string {
	if conv != nil {
		if lang, has := e.preferredLanguageForConversation(conv); has {
			return lang
		}
	}
	if !e.disableMultiLanguage {
		if lang, ok := e.store.GetUserPreferredLanguage(msg.UserID); ok && lang != "" {
			return lang
		}
	}
	if lang := i18n.NormalizeLocale(msg.Language); lang != "" {
		return lang
	}
	return i18n.DefaultLocale
}

func onboardingFormPrompt(lang string) string {
	return i18n.S(lang, i18n.MsgOnboardingFormPrompt)
}

func onboardingCompletionMessage(lang string, form int) string {
	return i18n.S(lang, i18n.MsgOnboardingCompleted, form)
}

func languageChangedMessage(lang string) string {
	return i18n.S(lang, i18n.MsgLanguageChanged)
}

var ratingPattern = regexp.MustCompile(`^\s*([1-5])\s*$`)
var numericPattern = regexp.MustCompile(`^\s*([0-9]+)\s*$`)
var reviewActionPattern = regexp.MustCompile(`\[\[PAI_REVIEW(?::([A-Za-z0-9-]+))?\]\]`)

func (e *Engine) maybeHandleRatingInput(msg chat.InboundMessage, conv *Conversation) (string, bool) {
	awaitingRating := isAwaitingRating(conv)
	fromInlineCallback := msg.Channel == "telegram" && msg.CallbackQueryID != ""
	if fromInlineCallback && !looksLikeRatingCallback(msg.Text) {
		return "", false
	}
	if !awaitingRating && !fromInlineCallback {
		return "", false
	}

	ratedMessageID, rating, valid, inputKind := parseRatingInput(msg.Text, fromInlineCallback, conv)
	if !valid {
		if fromInlineCallback {
			// Ignore invalid callback payloads; do not fall through to tutoring AI.
			return "", true
		}
		if !awaitingRating {
			return "", false
		}
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "answer_rating_skipped",
			Data: map[string]any{
				"channel":           msg.Channel,
				"rating_input_kind": inputKind,
				"text_len":          len(msg.Text),
			},
		})
		return "", false
	}
	if ratedMessageID != "" && e.ratingAlreadySubmitted(conv.ID, ratedMessageID) {
		return "", true
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "answer_rating_submitted",
		Data: map[string]any{
			"channel":          msg.Channel,
			"rating":           rating,
			"rated_message_id": ratedMessageID,
			"source":           map[bool]string{true: "telegram_inline_button", false: "text"}[fromInlineCallback],
			"delayed_submit":   !awaitingRating,
		},
	})
	thanksText := e.ratingThanksTextForMessage(conv, msg)
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: thanksText,
	}); err != nil {
		slog.Error("failed to store rating thanks response", "error", err)
	}

	return thanksText, true
}

func looksLikeRatingCallback(text string) bool {
	if strings.HasPrefix(strings.TrimSpace(text), "rating:") {
		return true
	}
	_, _, ok := parseRatingCallbackDataForEngine(text)
	return ok
}

func isAwaitingRating(conv *Conversation) bool {
	if conv == nil || len(conv.Messages) == 0 {
		return false
	}
	last := conv.Messages[len(conv.Messages)-1]
	if last.Role != "assistant" {
		return false
	}
	trimmed := strings.TrimSpace(last.Content)
	return reviewActionPattern.MatchString(trimmed)
}

func parseRatingResponse(text string) (int, bool, string) {
	matches := ratingPattern.FindStringSubmatch(text)
	if len(matches) == 2 {
		return int(matches[1][0] - '0'), true, "valid_rating"
	}
	if numericPattern.MatchString(text) {
		return 0, false, "numeric_out_of_range"
	}
	return 0, false, "non_rating_text"
}

func parseRatingCallbackDataForEngine(text string) (string, int, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "rating:") {
		return "", 0, false
	}

	parts := strings.Split(trimmed, ":")
	if len(parts) != 3 {
		return "", 0, false
	}

	ratedMessageID := strings.TrimSpace(parts[1])
	rating, valid, _ := parseRatingResponse(parts[2])
	if !valid {
		return "", 0, false
	}
	return ratedMessageID, rating, true
}

func latestRatingPromptMessageID(conv *Conversation) (string, bool) {
	if conv == nil {
		return "", false
	}
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		msg := conv.Messages[i]
		if msg.Role == "assistant" && reviewActionPattern.MatchString(strings.TrimSpace(msg.Content)) {
			if msg.ID == "" {
				return "", false
			}
			return msg.ID, true
		}
	}
	return "", false
}

func parseRatingInput(text string, fromInlineCallback bool, conv *Conversation) (string, int, bool, string) {
	if fromInlineCallback {
		ratedMessageID, rating, ok := parseRatingCallbackDataForEngine(text)
		if ok {
			return ratedMessageID, rating, true, "valid_rating"
		}

		// Backward compatibility for legacy callback data values "1".."5".
		rating, valid, _ := parseRatingResponse(text)
		if !valid {
			return "", 0, false, "invalid_callback_data"
		}
		ratedMessageID, _ = latestRatingPromptMessageID(conv)
		return ratedMessageID, rating, true, "valid_rating"
	}

	rating, valid, inputKind := parseRatingResponse(text)
	if !valid {
		return "", 0, false, inputKind
	}
	ratedMessageID, _ := latestRatingPromptMessageID(conv)
	return ratedMessageID, rating, true, inputKind
}

type ratingSubmissionChecker interface {
	HasRatingSubmission(conversationID, ratedMessageID string) bool
}

func (e *Engine) ratingAlreadySubmitted(conversationID, ratedMessageID string) bool {
	checker, ok := e.eventLogger.(ratingSubmissionChecker)
	if !ok || ratedMessageID == "" {
		return false
	}
	return checker.HasRatingSubmission(conversationID, ratedMessageID)
}

func (e *Engine) ratingThanksTextForMessage(conv *Conversation, msg chat.InboundMessage) string {
	if lang, hasPref := e.preferredLanguageForConversation(conv); hasPref {
		return i18n.S(lang, i18n.MsgRatingThanks)
	}
	if lang := i18n.NormalizeLocale(msg.Language); lang != "" {
		return i18n.S(lang, i18n.MsgRatingThanks)
	}
	return i18n.S(i18n.DefaultLocale, i18n.MsgRatingThanks)
}

func countTutoringReplies(messages []StoredMessage) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == "assistant" && msg.Model != "" {
			count++
		}
	}
	return count
}

func shouldRequestRatingAfterReply(replyCount, every int) bool {
	if every <= 0 {
		every = defaultRatingPromptEvery
	}
	return replyCount > 0 && replyCount%every == 0
}

func (e *Engine) buildSystemPrompt(msg chat.InboundMessage, conv *Conversation, topic *curriculum.Topic, teachingNotes string) string {
	languageBlock := `LANGUAGE:
Respond in the student's language (Bahasa Melayu, English, or mixed if they mix).
If the user writes mostly in Bahasa Melayu, respond mainly in Bahasa Melayu.
If the user writes mostly in English, respond mainly in English.`
	if lang, hasLangPref := e.preferredLanguageForConversation(conv); hasLangPref {
		langInstruction := "Preferred language setting: Bahasa Melayu. Follow this preference, unless the student's latest message is clearly in another language for that reply."
		switch lang {
		case "en":
			langInstruction = "Preferred language setting: English. Follow this preference, unless the student's latest message is clearly in another language for that reply."
		case "zh":
			langInstruction = "Preferred language setting: Chinese (Simplified). Follow this preference, unless the student's latest message is clearly in another language for that reply."
		}
		languageBlock = languageBlock + "\n" + langInstruction
	}
	base := `You are P&AI Bot, a supportive mathematics tutor for Malaysian secondary students (KSSM Form 1-3, Algebra-first).

PRIMARY GOAL:
Help the student think and solve independently.
Never shortcut their thinking by revealing the final answer too early.

` + languageBlock + `

========================================
CURRICULUM AWARENESS:
========================================

You are provided structured teaching notes and assessment schema for the current topic.

You must:
- Align explanations with the official KSSM learning objectives.
- Use terminology from the Bahasa Melayu key terms table when appropriate.
- Watch for known misconceptions listed in the teaching notes.
- If the student makes a common misconception listed in the notes, explicitly address it using the recommended strategy.
- When evaluating an attempt, think using the rubric structure (partial understanding vs full mastery).
- Keep responses aligned to Tahap Penguasaan 1-3 unless explicitly asked for extension.

EXAM TERMINOLOGY:
- Use UASA for Form 1-3 exam references. Use SPM only for upper-secondary exam references.
- Do not call Form 1-3 assessment PT3. Treat PT3 as obsolete legacy terminology and rewrite it to UASA if it appears in prior context.
- Before sending a reply, scan your draft for the token "PT3".
- If "PT3" appears anywhere in a normal tutoring reply, replace it with "UASA" (or "UASA/SPM" if contrasting lower-secondary vs upper-secondary).
- Your final tutoring reply should not contain the token "PT3".

========================================
PEDAGOGICAL CONTROL LOGIC
========================================

You must internally determine the teaching stage based on the conversation history.

STAGE A - NEW PROBLEM
If the student asks a fresh math question and has not attempted it:
- Output ONLY:
  Faham/Understand: [restate what is asked]
  Rancang/Plan: [give a short 1-3 step plan]
- End with a question asking them to execute the first step.
- Do NOT solve.
- Do NOT reveal the final answer.

STAGE B - WAITING FOR ATTEMPT
If you have already given a plan and are waiting:
- Do NOT provide the answer.
- Encourage them to try.
- Ask a small guiding question.

STAGE C - EVALUATING ATTEMPT
If the student provides a calculation or algebra step:
- Check it carefully.
- If correct:
    Praise briefly and guide to next step (do NOT jump to final answer unless they completed everything).
- If incorrect:
    Identify the specific mistake.
    Provide ONE focused hint only.
    Do NOT reveal full solution.

STAGE D - HINT ESCALATION
If the student makes repeated incorrect attempts or says "I don't know":
- Gradually increase scaffolding.
- Reveal at most ONE additional transformation step at a time.
- Still avoid revealing the final answer unless absolutely necessary.

STAGE E - FULL WRAP UP
Only give full solution (including final numerical answer) if:
- The student has completed all steps correctly, OR
- The student has made multiple genuine attempts and remains stuck.

========================================
CHEATING PROTECTION
========================================

If the student says:
- "Just give me the answer"
- "What is x?"
- "Tell me quickly"
- Any attempt to bypass thinking

You must:
- Politely refuse.
- Remind them the goal is understanding.
- Ask what the first step should be.

Never be harsh or sarcastic.

========================================
OUTPUT FORMAT
========================================
Use these exact plain-text labels in order when they are needed for each substantive tutoring reply:
Faham/Understand:
Rancang/Plan:
Selesaikan/Solve:
Semak/Verify:
Konsep/Connect:

IMPORTANT:
- In early stages (A or B), usually output only Faham/Understand and Rancang/Plan.
- Only include Selesaikan/Solve, Semak/Verify, and Konsep/Connect when they add real value for the current stage.
- Never fill Solve with full solution unless in FULL WRAP UP stage.
- The student benefits most from an explanation style where you frequently pause to confirm understanding by asking test questions.
- Those test questions should preferably use simple, explicit examples.
- When you ask a test question, do not continue the explanation until the student has answered to your satisfaction.
- Do not keep generating the explanation after the check question; actually stop and wait for the student's next reply first.
- Keep responses concise and chat-friendly.
- Avoid long walls of text.
- Use relatable Malaysian examples when helpful.
- Never be condescending.
- Do not ask for rating/feedback unless the system explicitly instructs you to include control token [[PAI_REVIEW]].

SAFETY + ACCURACY:
1. Do not invent facts, formulas, or curriculum references.
2. If context is missing, ask a clarifying question before solving.
3. If uncertain, state what is uncertain and propose the next step.

IMAGE HANDLING:
1. If an image is attached, analyze it first, then answer.
2. If image text is unclear, state what is unclear and ask for a clearer retake.
3. If the student asks a follow-up about an earlier image but did not reply to that image (or reattach it), ask them to reply directly to the image message.

FORMAT CONSTRAINT:
Use plain-text math only (example: 6x = 30, x = 5). Do not use LaTeX delimiters like \[ \], \( \), or $$.
Do not format replies using Markdown (no headings, bold, italic, code blocks, or Markdown lists). Use plain chat text with simple line breaks only.`

	// Inject adaptive explanation depth based on mastery level.
	if e.tracker != nil {
		userID := msg.UserID
		if conv != nil {
			userID = conv.UserID
		}
		var topicMastery float64
		if topic != nil {
			syllabusID := topic.SyllabusID
			if syllabusID == "" {
				syllabusID = "default"
			}
			topicMastery, _ = e.tracker.GetMastery(userID, syllabusID, topic.ID)
		}
		allProgress, _ := e.tracker.GetAllProgress(userID)
		base += adaptiveDepthBlock(topicMastery, allProgress)
	}

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

func sanitizeControlContent(content string) string {
	clean := reviewActionPattern.ReplaceAllString(content, "")
	clean = strings.TrimSpace(clean)
	switch clean {
	case langPrefCodeEN, langPrefCodeMS, langPrefCodeZH:
		return ""
	default:
		return clean
	}
}

func injectReviewTokenWithMessageID(content, messageID string) string {
	token := fmt.Sprintf("[[PAI_REVIEW:%s]]", messageID)
	if reviewActionPattern.MatchString(content) {
		return reviewActionPattern.ReplaceAllString(content, token)
	}
	return strings.TrimSpace(content) + "\n\n" + token
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
	return stripMarkdownFormatting(replacer.Replace(content))
}

func normalizeLegacyExamReferences(content string) string {
	replacer := strings.NewReplacer(
		"PT3/SPM", "UASA/SPM",
		"pt3/spm", "uasa/spm",
		"PT3-style", "UASA-style",
		"pt3-style", "uasa-style",
		"gaya PT3", "gaya UASA",
		"Gaya PT3", "Gaya UASA",
		"pelajar PT3", "pelajar UASA",
		"Pelajar PT3", "Pelajar UASA",
		" PT3 ", " UASA ",
		"(PT3)", "(UASA)",
		" PT3.", " UASA.",
		" PT3,", " UASA,",
		" PT3?", " UASA?",
		" PT3!", " UASA!",
		" PT3:", " UASA:",
	)
	return replacer.Replace(content)
}

func stripMarkdownFormatting(content string) string {
	if content == "" {
		return content
	}

	// Remove common markdown styling tokens while preserving sentence text.
	content = strings.NewReplacer(
		"```", "",
		"`", "",
		"**", "",
		"__", "",
		"~~", "",
	).Replace(content)

	lines := strings.Split(content, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Strip markdown heading prefixes.
		for strings.HasPrefix(trimmed, "#") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		}

		// Strip blockquote prefix.
		if strings.HasPrefix(trimmed, ">") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
		}

		// Strip markdown list prefixes.
		switch {
		case strings.HasPrefix(trimmed, "- "),
			strings.HasPrefix(trimmed, "* "),
			strings.HasPrefix(trimmed, "+ "):
			trimmed = strings.TrimSpace(trimmed[2:])
		default:
			trimmed = trimOrderedListPrefix(trimmed)
		}

		cleaned = append(cleaned, trimmed)
	}

	// Keep at most one consecutive blank line.
	var b strings.Builder
	lastBlank := false
	for _, line := range cleaned {
		if line == "" {
			if lastBlank {
				continue
			}
			lastBlank = true
		} else {
			lastBlank = false
		}

		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
	}

	return strings.TrimSpace(b.String())
}

func trimOrderedListPrefix(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(s) {
		return s
	}
	if (s[i] == '.' || s[i] == ')') && i+1 < len(s) && s[i+1] == ' ' {
		return strings.TrimSpace(s[i+2:])
	}
	return s
}

func truncateForPrompt(text string, max int) string {
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max] + "\n...[truncated]"
}

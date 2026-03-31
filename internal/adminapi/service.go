package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p-n-ai/pai-bot/internal/group"
)

var ErrNotFound = errors.New("admin resource not found")

type Student struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ExternalID string    `json:"external_id"`
	Channel    string    `json:"channel"`
	Form       string    `json:"form"`
	CreatedAt  time.Time `json:"created_at"`
}

type ProgressItem struct {
	TopicID       string     `json:"topic_id"`
	MasteryScore  float64    `json:"mastery_score"`
	EaseFactor    float64    `json:"ease_factor"`
	IntervalDays  int        `json:"interval_days"`
	NextReviewAt  *time.Time `json:"next_review_at"`
	LastStudiedAt *time.Time `json:"last_studied_at"`
}

type StreakSummary struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
	TotalXP int `json:"total_xp"`
}

type StudentDetail struct {
	Student  Student        `json:"student"`
	Progress []ProgressItem `json:"progress"`
	Streak   StreakSummary  `json:"streak"`
}

type StudentConversation struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"`
	Text      string    `json:"text"`
}

type Parent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	ChildIDs  []string  `json:"child_ids"`
	CreatedAt time.Time `json:"created_at"`
}

type WeeklyStats struct {
	DaysActive        int `json:"days_active"`
	MessagesExchanged int `json:"messages_exchanged"`
	QuizzesCompleted  int `json:"quizzes_completed"`
	NeedsReviewCount  int `json:"needs_review_count"`
}

type EncouragementSuggestion struct {
	Headline string `json:"headline"`
	Text     string `json:"text"`
}

type ParentSummary struct {
	Parent        Parent                  `json:"parent"`
	Child         Student                 `json:"child"`
	Streak        StreakSummary           `json:"streak"`
	WeeklyStats   WeeklyStats             `json:"weekly_stats"`
	Mastery       []ProgressItem          `json:"mastery"`
	Encouragement EncouragementSuggestion `json:"encouragement"`
}

type AIProviderUsage struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Messages     int    `json:"messages"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
}

type AIUsageSummary struct {
	TotalMessages     int               `json:"total_messages"`
	TotalInputTokens  int               `json:"total_input_tokens"`
	TotalOutputTokens int               `json:"total_output_tokens"`
	Providers         []AIProviderUsage `json:"providers"`
}

type DailyActiveUsersPoint struct {
	Date  string `json:"date"`
	Users int    `json:"users"`
}

type RetentionPoint struct {
	CohortDate string  `json:"cohort_date"`
	CohortSize int     `json:"cohort_size"`
	Day1Rate   float64 `json:"day_1_rate"`
	Day7Rate   float64 `json:"day_7_rate"`
	Day14Rate  float64 `json:"day_14_rate"`
}

type NudgeRateSummary struct {
	NudgesSent             int     `json:"nudges_sent"`
	ResponsesWithin24Hours int     `json:"responses_within_24h"`
	ResponseRate           float64 `json:"response_rate"`
}

type MetricsSummary struct {
	WindowDays       int                   `json:"window_days"`
	DailyActiveUsers []DailyActiveUsersPoint `json:"daily_active_users"`
	Retention        []RetentionPoint      `json:"retention"`
	NudgeRate        NudgeRateSummary      `json:"nudge_rate"`
	AIUsage          AIUsageSummary        `json:"ai_usage"`
	ABComparison     any                   `json:"ab_comparison"`
}

type ClassStudent struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Topics map[string]float64 `json:"topics"`
}

type ClassProgress struct {
	Students []ClassStudent `json:"students"`
	TopicIDs []string       `json:"topic_ids"`
}

type ClassListItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SyllabusID  string    `json:"syllabus_id,omitempty"`
	JoinCode    string    `json:"join_code"`
	Status      string    `json:"status"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type ClassDetail struct {
	ClassListItem
	Members []ClassMember `json:"members"`
}

type ClassMember struct {
	UserID         string    `json:"user_id"`
	MembershipRole string    `json:"membership_role"`
	JoinedAt       time.Time `json:"joined_at"`
}

type Service struct {
	pool       *pgxpool.Pool
	tenantID   string
	allTenants bool
	groupStore group.Store
}

type retentionCohortSample struct {
	CohortDate time.Time
	CohortSize int
	Day1Users  int
	Day7Users  int
	Day14Users int
}

func New(pool *pgxpool.Pool, tenantID string, groupStore group.Store) *Service {
	return &Service{pool: pool, tenantID: tenantID, groupStore: groupStore}
}

func NewPlatform(pool *pgxpool.Pool, groupStore group.Store) *Service {
	return &Service{pool: pool, allTenants: true, groupStore: groupStore}
}

func (s *Service) tenantPredicate(column string, argPos int) string {
	return fmt.Sprintf("($%d::uuid IS NULL OR %s = $%d::uuid)", argPos, column, argPos)
}

func (s *Service) tenantArg() any {
	if s.allTenants {
		return nil
	}
	return s.tenantID
}

func (s *Service) GetClassProgress(classID string) (ClassProgress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	formFilter := formFromClassID(classID)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT
			COALESCE(NULLIF(u.external_id, ''), u.id::text) AS student_id,
			u.name,
			lp.topic_id,
			lp.mastery_score
		FROM users u
		LEFT JOIN learning_progress lp
			ON lp.user_id = u.id
			AND lp.tenant_id = u.tenant_id
		WHERE %s
			AND u.role = 'student'
			AND ($2 = '' OR u.form = $2)
		ORDER BY u.created_at ASC, u.name ASC, lp.topic_id ASC
	`, s.tenantPredicate("u.tenant_id", 1)), s.tenantArg(), formFilter)
	if err != nil {
		return ClassProgress{}, fmt.Errorf("query class progress: %w", err)
	}
	defer rows.Close()

	studentsByID := make(map[string]*ClassStudent)
	var studentOrder []string
	var topicIDs []string

	for rows.Next() {
		var studentID string
		var studentName string
		var topicID *string
		var mastery *float64
		if err := rows.Scan(&studentID, &studentName, &topicID, &mastery); err != nil {
			return ClassProgress{}, fmt.Errorf("scan class progress: %w", err)
		}

		student, ok := studentsByID[studentID]
		if !ok {
			student = &ClassStudent{ID: studentID, Name: studentName, Topics: map[string]float64{}}
			studentsByID[studentID] = student
			studentOrder = append(studentOrder, studentID)
		}

		if topicID != nil && mastery != nil {
			student.Topics[*topicID] = *mastery
			if !slices.Contains(topicIDs, *topicID) {
				topicIDs = append(topicIDs, *topicID)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return ClassProgress{}, fmt.Errorf("iterate class progress: %w", err)
	}

	students := make([]ClassStudent, 0, len(studentOrder))
	for _, id := range studentOrder {
		students = append(students, *studentsByID[id])
	}

	return ClassProgress{Students: students, TopicIDs: topicIDs}, nil
}

func (s *Service) GetStudentDetail(studentID string) (StudentDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var internalUserID string
	var detail StudentDetail
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT
			u.id::text,
			COALESCE(NULLIF(u.external_id, ''), u.id::text) AS student_id,
			u.name,
			COALESCE(u.external_id, ''),
			u.channel,
			COALESCE(u.form, ''),
			u.created_at
		FROM users u
		WHERE %s
			AND u.role = 'student'
			AND COALESCE(NULLIF(u.external_id, ''), u.id::text) = $2
		LIMIT 1
	`, s.tenantPredicate("u.tenant_id", 1)), s.tenantArg(), studentID).Scan(
		&internalUserID,
		&detail.Student.ID,
		&detail.Student.Name,
		&detail.Student.ExternalID,
		&detail.Student.Channel,
		&detail.Student.Form,
		&detail.Student.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return StudentDetail{}, ErrNotFound
	}
	if err != nil {
		return StudentDetail{}, fmt.Errorf("query student detail: %w", err)
	}

	progressRows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT topic_id, mastery_score, ease_factor, interval_days, next_review_at, last_studied_at
		FROM learning_progress
		WHERE %s
			AND user_id = $2::uuid
		ORDER BY last_studied_at DESC NULLS LAST, topic_id ASC
	`, s.tenantPredicate("tenant_id", 1)), s.tenantArg(), internalUserID)
	if err != nil {
		return StudentDetail{}, fmt.Errorf("query student progress: %w", err)
	}
	defer progressRows.Close()

	for progressRows.Next() {
		var item ProgressItem
		if err := progressRows.Scan(
			&item.TopicID,
			&item.MasteryScore,
			&item.EaseFactor,
			&item.IntervalDays,
			&item.NextReviewAt,
			&item.LastStudiedAt,
		); err != nil {
			return StudentDetail{}, fmt.Errorf("scan student progress: %w", err)
		}
		detail.Progress = append(detail.Progress, item)
	}
	if err := progressRows.Err(); err != nil {
		return StudentDetail{}, fmt.Errorf("iterate student progress: %w", err)
	}

	current, longest, err := s.loadStreakSummary(ctx, internalUserID)
	if err != nil {
		return StudentDetail{}, err
	}
	totalXP, err := s.loadTotalXP(ctx, internalUserID)
	if err != nil {
		return StudentDetail{}, err
	}
	detail.Streak = StreakSummary{
		Current: current,
		Longest: longest,
		TotalXP: totalXP,
	}

	return detail, nil
}

func (s *Service) GetStudentConversations(studentID string) ([]StudentConversation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT
			m.id::text,
			m.created_at,
			CASE WHEN m.role = 'user' THEN 'student' ELSE m.role END AS role,
			m.content
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		JOIN users u ON u.id = c.user_id
		WHERE %s
			AND u.role = 'student'
			AND COALESCE(NULLIF(u.external_id, ''), u.id::text) = $2
			AND m.role IN ('user', 'assistant')
		ORDER BY m.created_at ASC
	`, s.tenantPredicate("u.tenant_id", 1)), s.tenantArg(), studentID)
	if err != nil {
		return nil, fmt.Errorf("query student conversations: %w", err)
	}
	defer rows.Close()

	var conversations []StudentConversation
	for rows.Next() {
		var item StudentConversation
		if err := rows.Scan(&item.ID, &item.Timestamp, &item.Role, &item.Text); err != nil {
			return nil, fmt.Errorf("scan student conversation: %w", err)
		}
		conversations = append(conversations, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student conversations: %w", err)
	}
	if len(conversations) == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx, fmt.Sprintf(`
			SELECT EXISTS(
				SELECT 1 FROM users
				WHERE %s
					AND role = 'student'
					AND COALESCE(NULLIF(external_id, ''), id::text) = $2
			)
		`, s.tenantPredicate("tenant_id", 1)), s.tenantArg(), studentID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check student existence: %w", err)
		}
		if !exists {
			return nil, ErrNotFound
		}
	}

	return conversations, nil
}

func (s *Service) GetParentSummary(parentID string) (ParentSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	parent, childID, err := s.loadParent(ctx, parentID)
	if err != nil {
		return ParentSummary{}, err
	}

	child, childInternalID, err := s.loadStudentByExternalID(ctx, childID)
	if err != nil {
		return ParentSummary{}, err
	}

	progress, err := s.loadStudentProgress(ctx, childInternalID)
	if err != nil {
		return ParentSummary{}, err
	}

	current, longest, err := s.loadStreakSummary(ctx, childInternalID)
	if err != nil {
		return ParentSummary{}, err
	}
	totalXP, err := s.loadTotalXP(ctx, childInternalID)
	if err != nil {
		return ParentSummary{}, err
	}

	weeklyStats, err := s.loadWeeklyStats(ctx, childInternalID)
	if err != nil {
		return ParentSummary{}, err
	}

	return ParentSummary{
		Parent:      parent,
		Child:       child,
		Streak:      StreakSummary{Current: current, Longest: longest, TotalXP: totalXP},
		WeeklyStats: weeklyStats,
		Mastery:     progress,
		Encouragement: buildParentEncouragement(
			child.Name,
			StreakSummary{Current: current, Longest: longest, TotalXP: totalXP},
			progress,
			weeklyStats,
		),
	}, nil
}

func (s *Service) GetAIUsage() (AIUsageSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT
			COALESCE(model, '') AS model,
			COUNT(*) AS message_count,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens
		FROM messages
		WHERE %s
			AND model IS NOT NULL
			AND model <> ''
		GROUP BY model
		ORDER BY COUNT(*) DESC, model ASC
	`, s.tenantPredicate("tenant_id", 1)), s.tenantArg())
	if err != nil {
		return AIUsageSummary{}, fmt.Errorf("query ai usage: %w", err)
	}
	defer rows.Close()

	var summary AIUsageSummary
	for rows.Next() {
		var item AIProviderUsage
		if err := rows.Scan(&item.Model, &item.Messages, &item.InputTokens, &item.OutputTokens); err != nil {
			return AIUsageSummary{}, fmt.Errorf("scan ai usage: %w", err)
		}
		item.Provider, item.Model = splitProviderModel(item.Model)
		item.TotalTokens = item.InputTokens + item.OutputTokens

		summary.TotalMessages += item.Messages
		summary.TotalInputTokens += item.InputTokens
		summary.TotalOutputTokens += item.OutputTokens
		summary.Providers = append(summary.Providers, item)
	}
	if err := rows.Err(); err != nil {
		return AIUsageSummary{}, fmt.Errorf("iterate ai usage: %w", err)
	}

	return summary, nil
}

func (s *Service) GetMetrics() (MetricsSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	daily, err := s.loadDailyActiveUsers(ctx, 14)
	if err != nil {
		return MetricsSummary{}, err
	}
	retention, err := s.loadRetention(ctx)
	if err != nil {
		return MetricsSummary{}, err
	}
	nudgeRate, err := s.loadNudgeRate(ctx, 14)
	if err != nil {
		return MetricsSummary{}, err
	}
	aiUsage, err := s.GetAIUsage()
	if err != nil {
		return MetricsSummary{}, err
	}

	return MetricsSummary{
		WindowDays:       14,
		DailyActiveUsers: daily,
		Retention:        retention,
		NudgeRate:        nudgeRate,
		AIUsage:          aiUsage,
		ABComparison:     nil,
	}, nil
}

func (s *Service) loadParent(ctx context.Context, parentID string) (Parent, string, error) {
	var (
		parent    Parent
		childJSON []byte
	)

	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT
			u.id::text,
			u.name,
			COALESCE(ai.identifier, ''),
			u.created_at,
			COALESCE(u.config->'children', '[]'::jsonb)
		FROM users u
		LEFT JOIN auth_identities ai
			ON ai.user_id = u.id
			AND ai.tenant_id = u.tenant_id
			AND ai.provider = 'password'
		WHERE %s
			AND u.role = 'parent'
			AND (
				u.id::text = $2
				OR COALESCE(NULLIF(u.external_id, ''), u.id::text) = $2
			)
		ORDER BY ai.created_at ASC NULLS LAST
		LIMIT 1
	`, s.tenantPredicate("u.tenant_id", 1)), s.tenantArg(), parentID).Scan(
		&parent.ID,
		&parent.Name,
		&parent.Email,
		&parent.CreatedAt,
		&childJSON,
	)
	if err == pgx.ErrNoRows {
		return Parent{}, "", ErrNotFound
	}
	if err != nil {
		return Parent{}, "", fmt.Errorf("query parent summary: %w", err)
	}

	if err := json.Unmarshal(childJSON, &parent.ChildIDs); err != nil {
		return Parent{}, "", fmt.Errorf("decode parent children: %w", err)
	}
	if len(parent.ChildIDs) == 0 {
		return Parent{}, "", ErrNotFound
	}

	return parent, parent.ChildIDs[0], nil
}

func (s *Service) loadDailyActiveUsers(ctx context.Context, days int) ([]DailyActiveUsersPoint, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		WITH day_series AS (
			SELECT generate_series(
				DATE(NOW() AT TIME ZONE 'UTC') - ($2::int - 1),
				DATE(NOW() AT TIME ZONE 'UTC'),
				INTERVAL '1 day'
			)::date AS activity_date
		),
		activity AS (
			SELECT DATE(e.created_at AT TIME ZONE 'UTC') AS activity_date, e.user_id
			FROM events e
			WHERE %s
				AND e.user_id IS NOT NULL
				AND e.created_at >= DATE(NOW() AT TIME ZONE 'UTC') - ($2::int - 1)
			UNION
			SELECT DATE(m.created_at AT TIME ZONE 'UTC') AS activity_date, c.user_id
			FROM messages m
			JOIN conversations c ON c.id = m.conversation_id
			WHERE %s
				AND m.created_at >= DATE(NOW() AT TIME ZONE 'UTC') - ($2::int - 1)
		)
		SELECT ds.activity_date, COUNT(DISTINCT a.user_id)
		FROM day_series ds
		LEFT JOIN activity a ON a.activity_date = ds.activity_date
		GROUP BY ds.activity_date
		ORDER BY ds.activity_date ASC
	`, s.tenantPredicate("e.tenant_id", 1), s.tenantPredicate("c.tenant_id", 1)), s.tenantArg(), days)
	if err != nil {
		return nil, fmt.Errorf("query daily active users: %w", err)
	}
	defer rows.Close()

	points := make([]DailyActiveUsersPoint, 0, days)
	for rows.Next() {
		var day time.Time
		var users int
		if err := rows.Scan(&day, &users); err != nil {
			return nil, fmt.Errorf("scan daily active users: %w", err)
		}
		points = append(points, DailyActiveUsersPoint{
			Date:  day.UTC().Format("2006-01-02"),
			Users: users,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily active users: %w", err)
	}
	return points, nil
}

func (s *Service) loadRetention(ctx context.Context) ([]RetentionPoint, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		WITH student_cohorts AS (
			SELECT
				u.id,
				DATE(u.created_at AT TIME ZONE 'UTC') AS cohort_date
			FROM users u
			WHERE %s
				AND u.role = 'student'
				AND DATE(u.created_at AT TIME ZONE 'UTC') <= DATE(NOW() AT TIME ZONE 'UTC') - 1
		),
		activity AS (
			SELECT DISTINCT e.user_id, DATE(e.created_at AT TIME ZONE 'UTC') AS activity_date
			FROM events e
			WHERE %s
				AND e.user_id IS NOT NULL
			UNION
			SELECT DISTINCT c.user_id, DATE(m.created_at AT TIME ZONE 'UTC') AS activity_date
			FROM messages m
			JOIN conversations c ON c.id = m.conversation_id
			WHERE %s
		)
		SELECT
			sc.cohort_date,
			COUNT(*) AS cohort_size,
			COUNT(*) FILTER (WHERE EXISTS (
				SELECT 1 FROM activity a
				WHERE a.user_id = sc.id
					AND a.activity_date = sc.cohort_date + 1
			)) AS day_1_users,
			COUNT(*) FILTER (WHERE EXISTS (
				SELECT 1 FROM activity a
				WHERE a.user_id = sc.id
					AND a.activity_date = sc.cohort_date + 7
			)) AS day_7_users,
			COUNT(*) FILTER (WHERE EXISTS (
				SELECT 1 FROM activity a
				WHERE a.user_id = sc.id
					AND a.activity_date = sc.cohort_date + 14
			)) AS day_14_users
		FROM student_cohorts sc
		GROUP BY sc.cohort_date
		ORDER BY sc.cohort_date DESC
		LIMIT 8
	`, s.tenantPredicate("u.tenant_id", 1), s.tenantPredicate("e.tenant_id", 1), s.tenantPredicate("c.tenant_id", 1)), s.tenantArg(), s.tenantArg(), s.tenantArg())
	if err != nil {
		return nil, fmt.Errorf("query retention: %w", err)
	}
	defer rows.Close()

	var samples []retentionCohortSample
	for rows.Next() {
		var sample retentionCohortSample
		if err := rows.Scan(&sample.CohortDate, &sample.CohortSize, &sample.Day1Users, &sample.Day7Users, &sample.Day14Users); err != nil {
			return nil, fmt.Errorf("scan retention: %w", err)
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate retention: %w", err)
	}

	slices.Reverse(samples)
	return computeRetentionSeries(samples), nil
}

func (s *Service) loadNudgeRate(ctx context.Context, days int) (NudgeRateSummary, error) {
	var nudgesSent int
	var responses int

	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		WITH nudges AS (
			SELECT nl.id, nl.user_id, nl.sent_at
			FROM nudge_log nl
			WHERE %s
				AND nl.sent_at >= NOW() - make_interval(days => $2::int)
		)
		SELECT
			COUNT(*) AS nudges_sent,
			COUNT(*) FILTER (WHERE EXISTS (
				SELECT 1
				FROM (
					SELECT e.created_at
					FROM events e
					WHERE %s
						AND e.user_id = nudges.user_id
						AND e.created_at > nudges.sent_at
						AND e.created_at <= nudges.sent_at + INTERVAL '24 hour'
					UNION ALL
					SELECT m.created_at
					FROM messages m
					JOIN conversations c ON c.id = m.conversation_id
					WHERE %s
						AND c.user_id = nudges.user_id
						AND m.role = 'user'
						AND m.created_at > nudges.sent_at
						AND m.created_at <= nudges.sent_at + INTERVAL '24 hour'
				) responses
			)) AS responses_within_24h
		FROM nudges
	`, s.tenantPredicate("nl.tenant_id", 1), s.tenantPredicate("e.tenant_id", 1), s.tenantPredicate("c.tenant_id", 1)), s.tenantArg(), days, s.tenantArg(), s.tenantArg()).Scan(&nudgesSent, &responses)
	if err != nil {
		return NudgeRateSummary{}, fmt.Errorf("query nudge rate: %w", err)
	}

	return buildNudgeRateSummary(nudgesSent, responses), nil
}

func (s *Service) loadStudentByExternalID(ctx context.Context, studentID string) (Student, string, error) {
	var (
		internalUserID string
		student        Student
	)

	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT
			u.id::text,
			COALESCE(NULLIF(u.external_id, ''), u.id::text) AS student_id,
			u.name,
			COALESCE(u.external_id, ''),
			u.channel,
			COALESCE(u.form, ''),
			u.created_at
		FROM users u
		WHERE %s
			AND u.role = 'student'
			AND COALESCE(NULLIF(u.external_id, ''), u.id::text) = $2
		LIMIT 1
	`, s.tenantPredicate("u.tenant_id", 1)), s.tenantArg(), studentID).Scan(
		&internalUserID,
		&student.ID,
		&student.Name,
		&student.ExternalID,
		&student.Channel,
		&student.Form,
		&student.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return Student{}, "", ErrNotFound
	}
	if err != nil {
		return Student{}, "", fmt.Errorf("query parent child detail: %w", err)
	}

	return student, internalUserID, nil
}

func (s *Service) loadStudentProgress(ctx context.Context, internalUserID string) ([]ProgressItem, error) {
	progressRows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT topic_id, mastery_score, ease_factor, interval_days, next_review_at, last_studied_at
		FROM learning_progress
		WHERE %s
			AND user_id = $2::uuid
		ORDER BY last_studied_at DESC NULLS LAST, topic_id ASC
	`, s.tenantPredicate("tenant_id", 1)), s.tenantArg(), internalUserID)
	if err != nil {
		return nil, fmt.Errorf("query student progress: %w", err)
	}
	defer progressRows.Close()

	var progress []ProgressItem
	for progressRows.Next() {
		var item ProgressItem
		if err := progressRows.Scan(
			&item.TopicID,
			&item.MasteryScore,
			&item.EaseFactor,
			&item.IntervalDays,
			&item.NextReviewAt,
			&item.LastStudiedAt,
		); err != nil {
			return nil, fmt.Errorf("scan student progress: %w", err)
		}
		progress = append(progress, item)
	}
	if err := progressRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student progress: %w", err)
	}

	return progress, nil
}

func (s *Service) loadWeeklyStats(ctx context.Context, userID string) (WeeklyStats, error) {
	var stats WeeklyStats

	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		WITH window AS (
			SELECT NOW() - INTERVAL '7 day' AS since_at
		)
		SELECT
			COALESCE((
				SELECT COUNT(*)
				FROM (
					SELECT DISTINCT DATE(m.created_at AT TIME ZONE 'UTC') AS activity_day
					FROM messages m
					JOIN conversations c ON c.id = m.conversation_id
					CROSS JOIN window
					WHERE c.user_id = $2::uuid
						AND m.created_at >= window.since_at
					UNION
					SELECT DISTINCT DATE(e.created_at AT TIME ZONE 'UTC')
					FROM events e
					CROSS JOIN window
					WHERE e.user_id = $2::uuid
						AND e.created_at >= window.since_at
				) active_days
			), 0) AS days_active,
			COALESCE((
				SELECT COUNT(*)
				FROM messages m
				JOIN conversations c ON c.id = m.conversation_id
				CROSS JOIN window
				WHERE c.user_id = $2::uuid
					AND m.created_at >= window.since_at
					AND m.role IN ('user', 'assistant')
			), 0) AS messages_exchanged,
			COALESCE((
				SELECT COUNT(*)
				FROM events e
				CROSS JOIN window
				WHERE %s
					AND e.user_id = $2::uuid
					AND e.created_at >= window.since_at
					AND e.event_type = 'quiz_completed'
			), 0) AS quizzes_completed,
			COALESCE((
				SELECT COUNT(*)
				FROM learning_progress lp
				WHERE %s
					AND lp.user_id = $2::uuid
					AND (
						lp.mastery_score < 0.6
						OR (lp.next_review_at IS NOT NULL AND lp.next_review_at <= NOW() + INTERVAL '7 day')
					)
			), 0) AS needs_review_count
	`, s.tenantPredicate("e.tenant_id", 1), s.tenantPredicate("lp.tenant_id", 1)), s.tenantArg(), userID).Scan(
		&stats.DaysActive,
		&stats.MessagesExchanged,
		&stats.QuizzesCompleted,
		&stats.NeedsReviewCount,
	)
	if err != nil {
		return WeeklyStats{}, fmt.Errorf("query weekly stats: %w", err)
	}

	return stats, nil
}

func (s *Service) loadStreakSummary(ctx context.Context, userID string) (int, int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT activity_day
		FROM (
			SELECT DISTINCT DATE(created_at AT TIME ZONE 'UTC') AS activity_day
			FROM (
				SELECT m.created_at
				FROM messages m
				JOIN conversations c ON c.id = m.conversation_id
				WHERE c.user_id = $1::uuid
				UNION
				SELECT e.created_at
				FROM events e
				WHERE e.user_id = $1::uuid
			) activity
		) days
		ORDER BY activity_day DESC
	`, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("query streak summary: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var day time.Time
		if err := rows.Scan(&day); err != nil {
			return 0, 0, fmt.Errorf("scan streak summary: %w", err)
		}
		dates = append(dates, day)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("iterate streak summary: %w", err)
	}

	current, longest := computeStreakSummary(dates)
	return current, longest, nil
}

func (s *Service) loadTotalXP(ctx context.Context, userID string) (int, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT event_type, COUNT(*)
		FROM events
		WHERE %s
			AND user_id = $2::uuid
		GROUP BY event_type
	`, s.tenantPredicate("tenant_id", 1)), s.tenantArg(), userID)
	if err != nil {
		return 0, fmt.Errorf("query xp summary: %w", err)
	}
	defer rows.Close()

	weights := map[string]int{
		"session_started":         25,
		"quiz_completed":          40,
		"study_streak_extended":   20,
		"topic_selected":          10,
		"goal_set":                15,
		"progress_viewed":         5,
		"answer_rating_submitted": 5,
		"help_requested":          5,
		"ai_response":             2,
	}

	total := 0
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return 0, fmt.Errorf("scan xp summary: %w", err)
		}
		total += weights[eventType] * count
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate xp summary: %w", err)
	}

	return total, nil
}

func formFromClassID(classID string) string {
	lower := strings.ToLower(classID)
	switch {
	case strings.Contains(lower, "form-1"):
		return "Form 1"
	case strings.Contains(lower, "form-2"):
		return "Form 2"
	case strings.Contains(lower, "form-3"):
		return "Form 3"
	default:
		return ""
	}
}

func computeRetentionSeries(samples []retentionCohortSample) []RetentionPoint {
	points := make([]RetentionPoint, 0, len(samples))
	for _, sample := range samples {
		if sample.CohortSize <= 0 {
			continue
		}
		denom := float64(sample.CohortSize)
		points = append(points, RetentionPoint{
			CohortDate: sample.CohortDate.UTC().Format("2006-01-02"),
			CohortSize: sample.CohortSize,
			Day1Rate:   float64(sample.Day1Users) / denom,
			Day7Rate:   float64(sample.Day7Users) / denom,
			Day14Rate:  float64(sample.Day14Users) / denom,
		})
	}
	return points
}

func buildNudgeRateSummary(nudgesSent, responses int) NudgeRateSummary {
	summary := NudgeRateSummary{
		NudgesSent:             nudgesSent,
		ResponsesWithin24Hours: responses,
	}
	if nudgesSent > 0 {
		summary.ResponseRate = float64(responses) / float64(nudgesSent)
	}
	return summary
}

func buildParentEncouragement(childName string, streak StreakSummary, progress []ProgressItem, stats WeeklyStats) EncouragementSuggestion {
	if len(progress) == 0 {
		return EncouragementSuggestion{
			Headline: fmt.Sprintf("%s is ready for a fresh study sprint.", childName),
			Text:     "Celebrate any small step this week and invite one short check-in to rebuild momentum together.",
		}
	}

	lowest := progress[0]
	for _, item := range progress[1:] {
		if item.MasteryScore < lowest.MasteryScore {
			lowest = item
		}
	}

	if streak.Current >= 5 {
		return EncouragementSuggestion{
			Headline: fmt.Sprintf("%s is showing strong consistency.", childName),
			Text: fmt.Sprintf(
				"Celebrate the %d-day streak, then encourage one short practice on %s to turn steady effort into stronger mastery.",
				streak.Current,
				humanizeTopicID(lowest.TopicID),
			),
		}
	}

	if stats.NeedsReviewCount > 1 {
		return EncouragementSuggestion{
			Headline: fmt.Sprintf("%s could use a gentle reset this week.", childName),
			Text: fmt.Sprintf(
				"Keep the tone light and ask for one focused review on %s. A short session now can prevent multiple topics from slipping.",
				humanizeTopicID(lowest.TopicID),
			),
		}
	}

	return EncouragementSuggestion{
		Headline: fmt.Sprintf("%s is building momentum topic by topic.", childName),
		Text: fmt.Sprintf(
			"Offer specific praise for recent study activity and suggest one more quick round on %s to lift confidence.",
			humanizeTopicID(lowest.TopicID),
		),
	}
}

func humanizeTopicID(topicID string) string {
	parts := strings.Split(topicID, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func splitProviderModel(raw string) (provider string, model string) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	if len(parts) == 1 && parts[0] != "" {
		return "unknown", parts[0]
	}
	return "unknown", ""
}

func computeStreakSummary(dates []time.Time) (int, int) {
	if len(dates) == 0 {
		return 0, 0
	}

	normalized := make([]time.Time, 0, len(dates))
	for _, date := range dates {
		normalized = append(normalized, time.Date(date.UTC().Year(), date.UTC().Month(), date.UTC().Day(), 0, 0, 0, 0, time.UTC))
	}

	current := 1
	for i := 1; i < len(normalized); i++ {
		diff := normalized[i-1].Sub(normalized[i])
		if diff == 0 {
			continue
		}
		if diff == 24*time.Hour {
			current++
			continue
		}
		break
	}

	longest := 1
	run := 1
	for i := 1; i < len(normalized); i++ {
		diff := normalized[i-1].Sub(normalized[i])
		if diff == 24*time.Hour {
			run++
		} else if diff == 0 {
			continue
		} else {
			if run > longest {
				longest = run
			}
			run = 1
		}
	}
	if run > longest {
		longest = run
	}

	return current, longest
}

// ListClasses returns all active classes (groups) for the tenant with member counts.
func (s *Service) ListClasses(ctx context.Context) ([]ClassListItem, error) {
	if s.groupStore == nil {
		return nil, fmt.Errorf("group store not configured")
	}
	if s.allTenants {
		return nil, fmt.Errorf("ListClasses requires a tenant-scoped service")
	}

	groups, err := s.groupStore.ListByTenant(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}

	items := make([]ClassListItem, 0, len(groups))
	for _, g := range groups {
		count, err := s.groupStore.MemberCount(ctx, g.ID)
		if err != nil {
			return nil, fmt.Errorf("member count for %s: %w", g.ID, err)
		}
		items = append(items, ClassListItem{
			ID:          g.ID,
			Name:        g.Name,
			SyllabusID:  g.SyllabusID,
			JoinCode:    g.JoinCode,
			Status:      g.Status,
			MemberCount: count,
			CreatedAt:   g.CreatedAt,
		})
	}
	return items, nil
}

// CreateClass creates a new class for the tenant.
func (s *Service) CreateClass(ctx context.Context, name, syllabusID, createdByUserID string) (*ClassListItem, error) {
	if s.groupStore == nil {
		return nil, fmt.Errorf("group store not configured")
	}
	if s.allTenants {
		return nil, fmt.Errorf("CreateClass requires a tenant-scoped service")
	}

	created, err := s.groupStore.Create(ctx, group.Group{
		TenantID:   s.tenantID,
		Name:       name,
		SyllabusID: syllabusID,
		CreatedBy:  createdByUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("create class: %w", err)
	}

	count, err := s.groupStore.MemberCount(ctx, created.ID)
	if err != nil {
		return nil, fmt.Errorf("member count after create: %w", err)
	}

	return &ClassListItem{
		ID:          created.ID,
		Name:        created.Name,
		SyllabusID:  created.SyllabusID,
		JoinCode:    created.JoinCode,
		Status:      created.Status,
		MemberCount: count,
		CreatedAt:   created.CreatedAt,
	}, nil
}

// GetClassDetail returns a class with its member list.
func (s *Service) GetClassDetail(ctx context.Context, classID string) (*ClassDetail, error) {
	if s.groupStore == nil {
		return nil, fmt.Errorf("group store not configured")
	}
	if s.allTenants {
		return nil, fmt.Errorf("GetClassDetail requires a tenant-scoped service")
	}

	g, err := s.groupStore.GetByID(ctx, s.tenantID, classID)
	if err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get class: %w", err)
	}

	rawMembers, err := s.groupStore.GetMembers(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	members := make([]ClassMember, 0, len(rawMembers))
	for _, m := range rawMembers {
		members = append(members, ClassMember{
			UserID:         m.UserID,
			MembershipRole: m.MembershipRole,
			JoinedAt:       m.JoinedAt,
		})
	}

	return &ClassDetail{
		ClassListItem: ClassListItem{
			ID:          g.ID,
			Name:        g.Name,
			SyllabusID:  g.SyllabusID,
			JoinCode:    g.JoinCode,
			Status:      g.Status,
			MemberCount: len(members),
			CreatedAt:   g.CreatedAt,
		},
		Members: members,
	}, nil
}

// UpdateClass renames or archives a class.
func (s *Service) UpdateClass(ctx context.Context, classID string, name *string, status *string) error {
	if s.groupStore == nil {
		return fmt.Errorf("group store not configured")
	}
	if s.allTenants {
		return fmt.Errorf("UpdateClass requires a tenant-scoped service")
	}

	if name != nil {
		if err := s.groupStore.Rename(ctx, s.tenantID, classID, *name); err != nil {
			if errors.Is(err, group.ErrGroupNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("rename class: %w", err)
		}
	}

	if status != nil && *status == "archived" {
		if err := s.groupStore.Archive(ctx, s.tenantID, classID); err != nil {
			if errors.Is(err, group.ErrGroupNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("archive class: %w", err)
		}
	}

	return nil
}

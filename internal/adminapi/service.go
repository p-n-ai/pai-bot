package adminapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

type ClassStudent struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Topics map[string]float64 `json:"topics"`
}

type ClassProgress struct {
	Students []ClassStudent `json:"students"`
	TopicIDs []string       `json:"topic_ids"`
}

type Service struct {
	pool     *pgxpool.Pool
	tenantID string
}

func New(pool *pgxpool.Pool, tenantID string) *Service {
	return &Service{pool: pool, tenantID: tenantID}
}

func (s *Service) GetClassProgress(classID string) (ClassProgress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	formFilter := formFromClassID(classID)
	rows, err := s.pool.Query(ctx, `
		SELECT
			COALESCE(NULLIF(u.external_id, ''), u.id::text) AS student_id,
			u.name,
			lp.topic_id,
			lp.mastery_score
		FROM users u
		LEFT JOIN learning_progress lp
			ON lp.user_id = u.id
			AND lp.tenant_id = u.tenant_id
		WHERE u.tenant_id = $1
			AND u.role = 'student'
			AND ($2 = '' OR u.form = $2)
		ORDER BY u.created_at ASC, u.name ASC, lp.topic_id ASC
	`, s.tenantID, formFilter)
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
	err := s.pool.QueryRow(ctx, `
		SELECT
			u.id::text,
			COALESCE(NULLIF(u.external_id, ''), u.id::text) AS student_id,
			u.name,
			COALESCE(u.external_id, ''),
			u.channel,
			COALESCE(u.form, ''),
			u.created_at
		FROM users u
		WHERE u.tenant_id = $1
			AND u.role = 'student'
			AND COALESCE(NULLIF(u.external_id, ''), u.id::text) = $2
		LIMIT 1
	`, s.tenantID, studentID).Scan(
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

	progressRows, err := s.pool.Query(ctx, `
		SELECT topic_id, mastery_score, ease_factor, interval_days, next_review_at, last_studied_at
		FROM learning_progress
		WHERE tenant_id = $1
			AND user_id = $2::uuid
		ORDER BY last_studied_at DESC NULLS LAST, topic_id ASC
	`, s.tenantID, internalUserID)
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

	rows, err := s.pool.Query(ctx, `
		SELECT
			m.id::text,
			m.created_at,
			CASE WHEN m.role = 'user' THEN 'student' ELSE m.role END AS role,
			m.content
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		JOIN users u ON u.id = c.user_id
		WHERE u.tenant_id = $1
			AND u.role = 'student'
			AND COALESCE(NULLIF(u.external_id, ''), u.id::text) = $2
			AND m.role IN ('user', 'assistant')
		ORDER BY m.created_at ASC
	`, s.tenantID, studentID)
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
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM users
				WHERE tenant_id = $1
					AND role = 'student'
					AND COALESCE(NULLIF(external_id, ''), id::text) = $2
			)
		`, s.tenantID, studentID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check student existence: %w", err)
		}
		if !exists {
			return nil, ErrNotFound
		}
	}

	return conversations, nil
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
	rows, err := s.pool.Query(ctx, `
		SELECT event_type, COUNT(*)
		FROM events
		WHERE tenant_id = $1
			AND user_id = $2::uuid
		GROUP BY event_type
	`, s.tenantID, userID)
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

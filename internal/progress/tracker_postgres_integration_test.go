// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package progress

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresTracker_RecordMasteryEvidence(t *testing.T) {
	ctx := context.Background()
	pool := startProgressPostgres(t, ctx)

	t.Run("durable exact replay", func(t *testing.T) {
		tenantID, learnerID := seedProgressLearner(t, ctx, pool, "durable-replay")
		evidence := progressTestEvidence(learnerID, "attempt-1", "topic-1", 0.8)

		first, err := NewPostgresTracker(pool, tenantID).RecordMasteryEvidence(ctx, evidence)
		if err != nil {
			t.Fatalf("first RecordMasteryEvidence() error = %v", err)
		}
		if !first.Applied {
			t.Fatal("first RecordMasteryEvidence().Applied = false, want true")
		}

		replay, err := NewPostgresTracker(pool, tenantID).RecordMasteryEvidence(ctx, evidence)
		if err != nil {
			t.Fatalf("replayed RecordMasteryEvidence() error = %v", err)
		}
		if replay.Applied {
			t.Fatal("replayed RecordMasteryEvidence().Applied = true, want false")
		}
		assertMasteryTransitionEqual(t, replay, first)
		counts, err := NewPostgresTracker(pool, tenantID).GetMasteryEvidenceCounts(ctx, learnerID)
		if err != nil {
			t.Fatalf("GetMasteryEvidenceCounts() error = %v", err)
		}
		if len(counts) != 1 || counts[0].Count != 1 || counts[0].TopicID != evidence.TopicID {
			t.Fatalf("evidence counts = %#v, want one durable attempt", counts)
		}
		assertProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence, 1, 1, 1)
	})

	t.Run("mastery snapshot keeps score and evidence coverage together", func(t *testing.T) {
		tenantID, learnerID := seedProgressLearner(t, ctx, pool, "coherent-snapshot")
		tracker := NewPostgresTracker(pool, tenantID)
		first := progressTestEvidence(learnerID, "attempt-1", "evidence-topic", 0.2)
		second := progressTestEvidence(learnerID, "attempt-2", "evidence-topic", 0.8)
		if _, err := tracker.RecordMasteryEvidence(ctx, first); err != nil {
			t.Fatalf("first RecordMasteryEvidence() error = %v", err)
		}
		transition, err := tracker.RecordMasteryEvidence(ctx, second)
		if err != nil {
			t.Fatalf("second RecordMasteryEvidence() error = %v", err)
		}
		if err := tracker.SetMasteryForLearner(learnerID, "syllabus-1", "progress-only-topic", 0.4); err != nil {
			t.Fatalf("SetMasteryForLearner() error = %v", err)
		}

		snapshot, err := tracker.GetMasterySnapshot(ctx, learnerID)
		if err != nil {
			t.Fatalf("GetMasterySnapshot() error = %v", err)
		}
		if len(snapshot) != 2 {
			t.Fatalf("snapshot = %#v, want two topics", snapshot)
		}
		byTopic := make(map[string]MasterySnapshotItem, len(snapshot))
		for _, item := range snapshot {
			byTopic[item.TopicID] = item
		}
		evidenceTopic := byTopic["evidence-topic"]
		if !evidenceTopic.MasteryKnown ||
			!closeFloat(evidenceTopic.MasteryScore, transition.MasteryAfter) ||
			evidenceTopic.EvidenceCount != 2 {
			t.Fatalf(
				"evidence topic snapshot = %#v, want score %v with 2 evidence records",
				evidenceTopic,
				transition.MasteryAfter,
			)
		}
		progressOnly := byTopic["progress-only-topic"]
		if !progressOnly.MasteryKnown ||
			!closeFloat(progressOnly.MasteryScore, 0.4) ||
			progressOnly.EvidenceCount != 0 {
			t.Fatalf("progress-only snapshot = %#v, want known 0.4 score without evidence", progressOnly)
		}
	})

	t.Run("payload conflict does not mutate progress", func(t *testing.T) {
		tenantID, learnerID := seedProgressLearner(t, ctx, pool, "payload-conflict")
		tracker := NewPostgresTracker(pool, tenantID)
		evidence := progressTestEvidence(learnerID, "attempt-1", "topic-1", 0.35)

		if _, err := tracker.RecordMasteryEvidence(ctx, evidence); err != nil {
			t.Fatalf("first RecordMasteryEvidence() error = %v", err)
		}
		before := readProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence)

		conflict := evidence
		conflict.PayloadHash = sha256.Sum256([]byte("different answer payload"))
		if _, err := tracker.RecordMasteryEvidence(ctx, conflict); !errors.Is(err, ErrMasteryEvidenceConflict) {
			t.Fatalf("conflicting RecordMasteryEvidence() error = %v, want ErrMasteryEvidenceConflict", err)
		}

		after := readProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence)
		if after != before {
			t.Fatalf("persistence after conflict = %+v, want unchanged %+v", after, before)
		}
	})

	t.Run("source revision conflict does not mutate progress", func(t *testing.T) {
		tenantID, learnerID := seedProgressLearner(t, ctx, pool, "revision-conflict")
		tracker := NewPostgresTracker(pool, tenantID)
		evidence := progressTestEvidence(learnerID, "attempt-1", "topic-1", 0.35)

		if _, err := tracker.RecordMasteryEvidence(ctx, evidence); err != nil {
			t.Fatalf("first RecordMasteryEvidence() error = %v", err)
		}
		before := readProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence)

		conflict := evidence
		conflict.SourceRevision = "sha256:changed"
		if _, err := tracker.RecordMasteryEvidence(ctx, conflict); !errors.Is(err, ErrMasteryEvidenceConflict) {
			t.Fatalf("conflicting RecordMasteryEvidence() error = %v, want ErrMasteryEvidenceConflict", err)
		}

		after := readProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence)
		if after != before {
			t.Fatalf("persistence after conflict = %+v, want unchanged %+v", after, before)
		}
	})

	t.Run("tenant mismatch is rejected without mutation", func(t *testing.T) {
		trackerTenantID, _ := seedProgressLearner(t, ctx, pool, "tenant-mismatch-tracker")
		otherTenantID, otherLearnerID := seedProgressLearner(t, ctx, pool, "tenant-mismatch-learner")
		tracker := NewPostgresTracker(pool, trackerTenantID)
		evidence := progressTestEvidence(otherLearnerID, "attempt-1", "topic-1", 0.9)

		_, err := tracker.RecordMasteryEvidence(ctx, evidence)
		if err == nil {
			t.Fatal("RecordMasteryEvidence() error = nil, want tenant mismatch")
		}
		if !strings.Contains(err.Error(), "does not belong to tracker tenant") {
			t.Fatalf("RecordMasteryEvidence() error = %v, want tenant mismatch context", err)
		}
		assertProgressPersistence(t, ctx, pool, trackerTenantID, otherLearnerID, evidence, 0, 0, 0)
		assertProgressPersistence(t, ctx, pool, otherTenantID, otherLearnerID, evidence, 0, 0, 0)
	})

	t.Run("concurrent duplicate applies once", func(t *testing.T) {
		tenantID, learnerID := seedProgressLearner(t, ctx, pool, "concurrent-duplicate")
		tracker := NewPostgresTracker(pool, tenantID)
		evidence := progressTestEvidence(learnerID, "attempt-1", "topic-1", 0.9)

		const callers = 8
		start := make(chan struct{})
		results := make(chan masteryCallResult, callers)
		var wg sync.WaitGroup
		for range callers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				result, err := tracker.RecordMasteryEvidence(ctx, evidence)
				results <- masteryCallResult{result: result, err: err}
			}()
		}
		close(start)
		wg.Wait()
		close(results)

		applied := 0
		var stable MasteryEvidenceResult
		calls := make([]masteryCallResult, 0, callers)
		for call := range results {
			if call.err != nil {
				t.Fatalf("concurrent RecordMasteryEvidence() error = %v", call.err)
			}
			calls = append(calls, call)
			if call.result.Applied {
				applied++
				stable = call.result
			}
		}
		if applied != 1 {
			t.Fatalf("concurrent applied count = %d, want 1", applied)
		}
		if stable.MasteryBefore != 0 || !closeFloat(stable.MasteryAfter, evidence.Score) {
			t.Fatalf("applied transition = %+v, want 0 -> %v", stable, evidence.Score)
		}
		for _, call := range calls {
			assertMasteryTransitionEqual(t, call.result, stable)
		}
		assertProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence, 1, 1, 1)
	})

	t.Run("concurrent distinct attempts serialize on the topic", func(t *testing.T) {
		tenantID, learnerID := seedProgressLearner(t, ctx, pool, "concurrent-distinct")
		tracker := NewPostgresTracker(pool, tenantID)
		first := progressTestEvidence(learnerID, "attempt-1", "topic-1", 0.8)
		second := progressTestEvidence(learnerID, "attempt-2", "topic-1", 0.8)

		start := make(chan struct{})
		results := make(chan masteryCallResult, 2)
		var wg sync.WaitGroup
		for _, evidence := range []MasteryEvidence{first, second} {
			evidence := evidence
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				result, err := tracker.RecordMasteryEvidence(ctx, evidence)
				results <- masteryCallResult{result: result, err: err}
			}()
		}
		close(start)
		wg.Wait()
		close(results)

		transitions := make([]MasteryEvidenceResult, 0, 2)
		for call := range results {
			if call.err != nil {
				t.Fatalf("concurrent RecordMasteryEvidence() error = %v", call.err)
			}
			if !call.result.Applied {
				t.Fatal("distinct RecordMasteryEvidence().Applied = false, want true")
			}
			transitions = append(transitions, call.result)
		}
		sort.Slice(transitions, func(i, j int) bool {
			return transitions[i].MasteryBefore < transitions[j].MasteryBefore
		})
		if len(transitions) != 2 ||
			!closeFloat(transitions[0].MasteryBefore, 0) ||
			!closeFloat(transitions[0].MasteryAfter, first.Score) ||
			!closeFloat(transitions[1].MasteryBefore, transitions[0].MasteryAfter) ||
			!closeFloat(transitions[1].MasteryAfter, first.Score) {
			t.Fatalf("serialized transitions = %+v, want chained same-score updates", transitions)
		}
		assertProgressPersistence(t, ctx, pool, tenantID, learnerID, first, 2, 1, 2)
	})

	t.Run("snapshot failure rolls back progress and attempt", func(t *testing.T) {
		tenantID, learnerID := seedProgressLearner(t, ctx, pool, "snapshot-rollback")
		tracker := NewPostgresTracker(pool, tenantID)
		evidence := progressTestEvidence(learnerID, "attempt-1", "rollback-topic", 1)

		const constraint = "mastery_snapshots_progress_test_reject_topic"
		if _, err := pool.Exec(ctx,
			`ALTER TABLE mastery_snapshots
			 ADD CONSTRAINT `+constraint+` CHECK (topic_id <> 'rollback-topic')`,
		); err != nil {
			t.Fatalf("install snapshot failure constraint: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(),
				`ALTER TABLE mastery_snapshots DROP CONSTRAINT IF EXISTS `+constraint)
		})

		if _, err := tracker.RecordMasteryEvidence(ctx, evidence); err == nil {
			t.Fatal("RecordMasteryEvidence() error = nil, want snapshot write failure")
		} else if !strings.Contains(err.Error(), "write mastery snapshot") {
			t.Fatalf("RecordMasteryEvidence() error = %v, want snapshot write context", err)
		}
		assertProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence, 0, 0, 0)
	})
}

type masteryCallResult struct {
	result MasteryEvidenceResult
	err    error
}

type progressPersistence struct {
	attempts    int
	snapshots   int
	progress    int
	repetitions int
	mastery     float64
}

func startProgressPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("pai"),
		tcpostgres.WithUsername("pai"),
		tcpostgres.WithPassword("pai"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate postgres container: %v", err)
		}
	})

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("container connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	deadline := time.Now().Add(15 * time.Second)
	for {
		if err := pool.Ping(ctx); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("postgres did not become ready before timeout")
		}
		time.Sleep(100 * time.Millisecond)
	}

	for _, name := range []string{
		"20260318100000_initial.sql",
		"20260318100300_auth_tables.sql",
		"20260318100400_auth_identity_tenant_consistency.sql",
		"20260407100000_add_groups.sql",
		"20260731100000_learning_attempts.sql",
	} {
		applyProgressMigration(t, ctx, pool, filepath.Join("..", "..", "migrations", name))
	}
	return pool
}

func applyProgressMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, path string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	upSQL, err := progressGooseUpSQL(string(content))
	if err != nil {
		t.Fatalf("parse migration %s: %v", path, err)
	}
	if _, err := pool.Exec(ctx, upSQL); err != nil {
		t.Fatalf("apply migration %s: %v", path, err)
	}
}

func progressGooseUpSQL(content string) (string, error) {
	const (
		upMarker   = "-- +goose Up"
		downMarker = "-- +goose Down"
	)
	upIndex := strings.Index(content, upMarker)
	if upIndex < 0 {
		return strings.TrimSpace(content), nil
	}
	body := content[upIndex+len(upMarker):]
	if downIndex := strings.Index(body, downMarker); downIndex >= 0 {
		body = body[:downIndex]
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("missing goose Up section")
	}
	return body, nil
}

func seedProgressLearner(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	slug string,
) (string, LearnerID) {
	t.Helper()

	var tenantID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (name, slug)
		 VALUES ($1, $2)
		 RETURNING id::text`,
		"Progress test "+slug, "progress-test-"+slug,
	).Scan(&tenantID); err != nil {
		t.Fatalf("insert tenant %q: %v", slug, err)
	}

	var learner string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, external_id, channel)
		 VALUES ($1::uuid, 'student', $2, $3, 'telegram')
		 RETURNING id::text`,
		tenantID, "Learner "+slug, "learner-"+slug,
	).Scan(&learner); err != nil {
		t.Fatalf("insert learner %q: %v", slug, err)
	}
	learnerID, err := NewLearnerID(learner)
	if err != nil {
		t.Fatalf("NewLearnerID(%q) error = %v", learner, err)
	}
	return tenantID, learnerID
}

func progressTestEvidence(
	learnerID LearnerID,
	attemptID string,
	topicID string,
	score float64,
) MasteryEvidence {
	return MasteryEvidence{
		AttemptID:      attemptID,
		LearnerID:      learnerID,
		SyllabusID:     "syllabus-1",
		TopicID:        topicID,
		SourceKind:     "oss_assessment",
		SourceID:       "question-1",
		SourceRevision: "sha256:fixture",
		Score:          score,
		PayloadHash:    sha256.Sum256([]byte(learnerID.String() + "|" + attemptID + "|" + topicID)),
	}
}

func assertMasteryTransitionEqual(t *testing.T, got, want MasteryEvidenceResult) {
	t.Helper()
	if !closeFloat(got.MasteryBefore, want.MasteryBefore) ||
		!closeFloat(got.MasteryAfter, want.MasteryAfter) {
		t.Fatalf("mastery transition = %+v, want %+v", got, want)
	}
}

func assertProgressPersistence(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID string,
	learnerID LearnerID,
	evidence MasteryEvidence,
	wantAttempts int,
	wantSnapshots int,
	wantRepetitions int,
) {
	t.Helper()
	got := readProgressPersistence(t, ctx, pool, tenantID, learnerID, evidence)
	wantProgress := 0
	if wantRepetitions > 0 {
		wantProgress = 1
	}
	if got.attempts != wantAttempts ||
		got.snapshots != wantSnapshots ||
		got.progress != wantProgress ||
		got.repetitions != wantRepetitions ||
		(wantProgress == 1 && !closeFloat(got.mastery, evidence.Score)) {
		t.Fatalf(
			"persistence = %+v, want attempts=%d snapshots=%d progress=%d repetitions=%d mastery=%v",
			got, wantAttempts, wantSnapshots, wantProgress, wantRepetitions, evidence.Score,
		)
	}
}

func readProgressPersistence(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID string,
	learnerID LearnerID,
	evidence MasteryEvidence,
) progressPersistence {
	t.Helper()

	var state progressPersistence
	if err := pool.QueryRow(ctx,
		`SELECT count(*)
		 FROM learning_attempts
		 WHERE tenant_id = $1::uuid
		   AND user_id = $2::uuid
		   AND syllabus_id = $3
		   AND topic_id = $4`,
		tenantID, learnerID.String(), evidence.SyllabusID, evidence.TopicID,
	).Scan(&state.attempts); err != nil {
		t.Fatalf("count learning attempts: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*)
		 FROM mastery_snapshots
		 WHERE tenant_id = $1::uuid
		   AND user_id = $2::uuid
		   AND topic_id = $3`,
		tenantID, learnerID.String(), evidence.TopicID,
	).Scan(&state.snapshots); err != nil {
		t.Fatalf("count mastery snapshots: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*), COALESCE(max(repetitions), 0), COALESCE(max(mastery_score), 0)
		 FROM learning_progress
		 WHERE tenant_id = $1::uuid
		   AND user_id = $2::uuid
		   AND syllabus_id = $3
		   AND topic_id = $4`,
		tenantID, learnerID.String(), evidence.SyllabusID, evidence.TopicID,
	).Scan(&state.progress, &state.repetitions, &state.mastery); err != nil {
		t.Fatalf("read learning progress: %v", err)
	}
	return state
}

func closeFloat(left, right float64) bool {
	return math.Abs(left-right) <= 1e-6
}

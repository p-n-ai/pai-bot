package seed

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSeedDemo_SucceedsAndCommits(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTx{
		queryRowValues: [][]any{
			{"11111111-1111-1111-1111-111111111111"},
			{"22222222-2222-2222-2222-222222222222"},
		},
	}
	db := &fakeBeginner{tx: tx}

	if err := seedDemo(ctx, db); err != nil {
		t.Fatalf("seedDemo() error = %v", err)
	}

	if !tx.committed {
		t.Fatal("expected transaction to commit")
	}
	if tx.rolledBack {
		t.Fatal("did not expect rollback on success")
	}
	if len(tx.execSQL) != 19 {
		t.Fatalf("expected 19 exec statements, got %d", len(tx.execSQL))
	}
	if len(tx.queryRowSQL) != 2 {
		t.Fatalf("expected 2 tenant upsert queries, got %d", len(tx.queryRowSQL))
	}
	if !strings.Contains(tx.queryRowSQL[0], "INSERT INTO tenants") {
		t.Fatalf("first tenant upsert SQL = %q, want INSERT INTO tenants", tx.queryRowSQL[0])
	}
	if !strings.Contains(tx.queryRowSQL[1], "Second Demo School") {
		t.Fatalf("second tenant upsert SQL = %q, want Second Demo School", tx.queryRowSQL[1])
	}
	if !strings.Contains(tx.execSQL[0], "INSERT INTO users") {
		t.Fatalf("first statement = %q, want INSERT INTO users", tx.execSQL[0])
	}
	if !strings.Contains(tx.execSQL[0], "'platform_admin'") {
		t.Fatalf("user seed SQL = %q, want platform_admin demo user", tx.execSQL[0])
	}
	if !strings.Contains(tx.execSQL[0], "('10000000-0000-0000-0000-000000000007', NULL, 'platform_admin'") {
		t.Fatalf("user seed SQL = %q, want platform_admin seeded without tenant_id", tx.execSQL[0])
	}
	if !strings.Contains(tx.execSQL[0], "teacher_2") {
		t.Fatalf("user seed SQL = %q, want second tenant teacher user", tx.execSQL[0])
	}
	if !strings.Contains(tx.execSQL[1], "student@example.com") {
		t.Fatalf("tenant auth identity seed SQL = %q, want student@example.com", tx.execSQL[1])
	}
	if !strings.Contains(tx.execSQL[1], "'10000000-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'password', 'second-student@example.com', 'second-student@example.com'") {
		t.Fatalf("tenant auth identity seed SQL = %q, want second tenant student identity row to use second-student@example.com", tx.execSQL[1])
	}
	if count := strings.Count(tx.execSQL[1], "teacher@example.com"); count != 4 {
		t.Fatalf("tenant auth identity seed SQL teacher@example.com count = %d, want 4", count)
	}
	if !strings.Contains(tx.execSQL[2], "platform-admin@example.com") {
		t.Fatalf("platform auth identity seed SQL = %q, want platform-admin@example.com", tx.execSQL[2])
	}
	if !strings.Contains(tx.execSQL[2], "'10000000-0000-0000-0000-000000000007', NULL, 'password', 'platform-admin@example.com'") {
		t.Fatalf("platform auth identity seed SQL = %q, want platform admin auth identity without tenant_id", tx.execSQL[2])
	}
	if !strings.Contains(tx.execSQL[2], "'platform-admin@example.com', 'platform-admin@example.com'") {
		t.Fatalf("platform auth identity seed SQL = %q, want platform admin identifier and normalized identifier to match", tx.execSQL[2])
	}
	if !strings.Contains(tx.execSQL[1], "ON CONFLICT (tenant_id, provider, identifier_normalized)") {
		t.Fatalf("auth identity seed SQL = %q, want tenant-scoped upsert strategy", tx.execSQL[1])
	}
	if !strings.Contains(tx.execSQL[3], "platform-admin@example.com") {
		t.Fatalf("platform admin identity normalization SQL = %q, want platform-admin@example.com", tx.execSQL[3])
	}
	if !strings.Contains(tx.execSQL[3], "'60000000-0000-0000-0000-000000000005'") {
		t.Fatalf("platform admin identity normalization SQL = %q, want stable auth identity id", tx.execSQL[3])
	}
	if !strings.Contains(tx.execSQL[3], "tenant_id = NULL") {
		t.Fatalf("platform admin identity normalization SQL = %q, want NULL tenant_id update path", tx.execSQL[3])
	}
	if !strings.Contains(tx.execSQL[6], "INSERT INTO token_budgets") {
		t.Fatalf("budget seed SQL = %q, want INSERT INTO token_budgets", tx.execSQL[6])
	}
	if !strings.Contains(tx.execSQL[len(tx.execSQL)-1], "INSERT INTO events") {
		t.Fatalf("last statement = %q, want INSERT INTO events", tx.execSQL[len(tx.execSQL)-1])
	}
}

func TestSeedDemo_RollsBackOnExecError(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTx{
		queryRowValues: [][]any{
			{"11111111-1111-1111-1111-111111111111"},
			{"22222222-2222-2222-2222-222222222222"},
		},
		execErrAt: 2,
		execErr:   errors.New("boom"),
	}
	db := &fakeBeginner{tx: tx}

	err := seedDemo(ctx, db)
	if err == nil {
		t.Fatal("seedDemo() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "seed statement 3") {
		t.Fatalf("seedDemo() error = %v, want seed statement index", err)
	}
	if !tx.rolledBack {
		t.Fatal("expected rollback on exec error")
	}
	if tx.committed {
		t.Fatal("did not expect commit on exec error")
	}
}

func TestSeedDemo_RollsBackOnTenantLookupError(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTx{
		queryRowErr: errors.New("tenant failure"),
	}
	db := &fakeBeginner{tx: tx}

	err := seedDemo(ctx, db)
	if err == nil {
		t.Fatal("seedDemo() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "upsert default tenant") {
		t.Fatalf("seedDemo() error = %v, want tenant upsert context", err)
	}
	if !tx.rolledBack {
		t.Fatal("expected rollback on tenant lookup error")
	}
	if tx.committed {
		t.Fatal("did not expect commit on tenant lookup error")
	}
}

func TestSeedTokenBudget_SucceedsAndCommits(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTx{
		queryRowValues: [][]any{
			{"11111111-1111-1111-1111-111111111111"},
		},
	}
	db := &fakeBeginner{tx: tx}

	err := seedTokenBudget(ctx, db, TokenBudgetSeedParams{
		TenantSlug:   "default",
		BudgetTokens: 9000,
		PeriodStart:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("seedTokenBudget() error = %v", err)
	}
	if !tx.committed {
		t.Fatal("expected token budget seed transaction to commit")
	}
	if len(tx.queryRowSQL) != 1 || !strings.Contains(tx.queryRowSQL[0], "FROM tenants") {
		t.Fatalf("tenant lookup SQL = %#v, want tenant lookup query", tx.queryRowSQL)
	}
	if len(tx.execSQL) != 1 || !strings.Contains(tx.execSQL[0], "INSERT INTO token_budgets") {
		t.Fatalf("token budget SQL = %#v, want token budget upsert", tx.execSQL)
	}
}

func TestSeedTokenBudget_ValidatesInputs(t *testing.T) {
	tests := []struct {
		name   string
		params TokenBudgetSeedParams
	}{
		{
			name: "missing tenant slug",
			params: TokenBudgetSeedParams{
				BudgetTokens: 100,
				PeriodStart:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				PeriodEnd:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "non-positive budget",
			params: TokenBudgetSeedParams{
				TenantSlug:   "default",
				BudgetTokens: 0,
				PeriodStart:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				PeriodEnd:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "invalid period",
			params: TokenBudgetSeedParams{
				TenantSlug:   "default",
				BudgetTokens: 100,
				PeriodStart:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
				PeriodEnd:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := seedTokenBudget(context.Background(), &fakeBeginner{tx: &fakeTx{}}, tt.params); err == nil {
				t.Fatal("seedTokenBudget() error = nil, want validation error")
			}
		})
	}
}

type fakeBeginner struct {
	tx       *fakeTx
	beginErr error
}

func (f *fakeBeginner) Begin(ctx context.Context) (txLike, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.tx, nil
}

type fakeTx struct {
	queryRowValues [][]any
	queryRowErr    error
	queryRowSQL    []string
	execSQL        []string
	execErrAt      int
	execErr        error
	committed      bool
	rolledBack     bool
}

func (f *fakeTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTx) Commit(ctx context.Context) error {
	f.committed = true
	return nil
}

func (f *fakeTx) Rollback(ctx context.Context) error {
	f.rolledBack = true
	return nil
}

func (f *fakeTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}

func (f *fakeTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (f *fakeTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (f *fakeTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.execSQL = append(f.execSQL, sql)
	if f.execErr != nil && len(f.execSQL)-1 == f.execErrAt {
		return pgconn.CommandTag{}, f.execErr
	}
	return pgconn.CommandTag{}, nil
}

func (f *fakeTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queryRowSQL = append(f.queryRowSQL, sql)
	values := []any{}
	if len(f.queryRowValues) > 0 {
		values = f.queryRowValues[0]
		f.queryRowValues = f.queryRowValues[1:]
	}
	return fakeRow{
		values: values,
		err:    f.queryRowErr,
	}
}

func (f *fakeTx) Conn() *pgx.Conn {
	return nil
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		default:
			return errors.New("unsupported scan dest")
		}
	}
	return nil
}

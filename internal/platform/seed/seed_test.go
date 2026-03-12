package seed

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSeedDemo_SucceedsAndCommits(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTx{
		queryRowValues: []any{"11111111-1111-1111-1111-111111111111"},
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
	if len(tx.execSQL) != 15 {
		t.Fatalf("expected 15 exec statements, got %d", len(tx.execSQL))
	}
	if !strings.Contains(tx.queryRowSQL, "INSERT INTO tenants") {
		t.Fatalf("tenant upsert SQL = %q, want INSERT INTO tenants", tx.queryRowSQL)
	}
	if !strings.Contains(tx.execSQL[0], "INSERT INTO users") {
		t.Fatalf("first statement = %q, want INSERT INTO users", tx.execSQL[0])
	}
	if !strings.Contains(tx.execSQL[len(tx.execSQL)-1], "INSERT INTO events") {
		t.Fatalf("last statement = %q, want INSERT INTO events", tx.execSQL[len(tx.execSQL)-1])
	}
}

func TestSeedDemo_RollsBackOnExecError(t *testing.T) {
	ctx := context.Background()
	tx := &fakeTx{
		queryRowValues: []any{"11111111-1111-1111-1111-111111111111"},
		execErrAt:      2,
		execErr:        errors.New("boom"),
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
	queryRowValues []any
	queryRowErr    error
	queryRowSQL    string
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
	f.queryRowSQL = sql
	return fakeRow{
		values: f.queryRowValues,
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

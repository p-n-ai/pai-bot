package tenant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type stubQuerier struct {
	rows      []stubRow
	queries   []string
	queryArgs [][]any
}

func (s *stubQuerier) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	s.queries = append(s.queries, sql)
	s.queryArgs = append(s.queryArgs, args)
	if len(s.rows) == 0 {
		return stubRow{err: errors.New("unexpected query")}
	}
	row := s.rows[0]
	s.rows = s.rows[1:]
	return row
}

type stubRow struct {
	id  string
	err error
}

func (r stubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	id, ok := dest[0].(*string)
	if !ok {
		return errors.New("destination is not *string")
	}
	*id = r.id
	return nil
}

func TestEnsureDefaultTenantForMode_SingleExisting(t *testing.T) {
	q := &stubQuerier{
		rows: []stubRow{
			{id: "tenant-existing"},
		},
	}

	tenantID, err := EnsureDefaultTenantForMode(context.Background(), "single", q)
	if err != nil {
		t.Fatalf("EnsureDefaultTenantForMode() error = %v", err)
	}
	if tenantID != "tenant-existing" {
		t.Fatalf("tenantID = %q, want tenant-existing", tenantID)
	}
	if len(q.queries) != 1 {
		t.Fatalf("query count = %d, want 1", len(q.queries))
	}
	if !strings.Contains(q.queries[0], "WHERE slug = $1") {
		t.Fatalf("lookup query = %q, want slug predicate", q.queries[0])
	}
}

func TestEnsureDefaultTenantForMode_SingleCreatesWhenMissing(t *testing.T) {
	q := &stubQuerier{
		rows: []stubRow{
			{err: pgx.ErrNoRows},
			{id: "tenant-created"},
		},
	}

	tenantID, err := EnsureDefaultTenantForMode(context.Background(), "single", q)
	if err != nil {
		t.Fatalf("EnsureDefaultTenantForMode() error = %v", err)
	}
	if tenantID != "tenant-created" {
		t.Fatalf("tenantID = %q, want tenant-created", tenantID)
	}
	if len(q.queries) != 2 {
		t.Fatalf("query count = %d, want 2", len(q.queries))
	}
	if !strings.Contains(q.queries[1], "INSERT INTO tenants") {
		t.Fatalf("second query = %q, want tenant insert", q.queries[1])
	}
}

func TestEnsureDefaultTenantForMode_MultiSkipsBootstrap(t *testing.T) {
	q := &stubQuerier{}

	tenantID, err := EnsureDefaultTenantForMode(context.Background(), "multi", q)
	if err != nil {
		t.Fatalf("EnsureDefaultTenantForMode() error = %v", err)
	}
	if tenantID != "" {
		t.Fatalf("tenantID = %q, want empty for multi mode", tenantID)
	}
	if len(q.queries) != 0 {
		t.Fatalf("query count = %d, want 0", len(q.queries))
	}
}

func TestEnsureDefaultTenantForMode_InvalidMode(t *testing.T) {
	q := &stubQuerier{}
	_, err := EnsureDefaultTenantForMode(context.Background(), "weird", q)
	if err == nil {
		t.Fatal("expected error for invalid tenant mode")
	}
}

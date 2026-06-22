// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type stubQuerier struct {
	query string
	row   stubRow
}

func (s *stubQuerier) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	s.query = sql
	return s.row
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

func TestDefaultTenantID(t *testing.T) {
	q := &stubQuerier{row: stubRow{id: "tenant-default"}}

	id, err := DefaultTenantID(context.Background(), q)
	if err != nil {
		t.Fatalf("DefaultTenantID() error = %v", err)
	}
	if id != "tenant-default" {
		t.Fatalf("id = %q, want tenant-default", id)
	}
	if !strings.Contains(q.query, "WHERE slug = 'default'") {
		t.Fatalf("query = %q, want default slug predicate", q.query)
	}
}

func TestDefaultTenantIDError(t *testing.T) {
	q := &stubQuerier{row: stubRow{err: pgx.ErrNoRows}}

	_, err := DefaultTenantID(context.Background(), q)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "lookup default tenant") {
		t.Fatalf("error = %v, want lookup context", err)
	}
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

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
	id, err := scanDefaultTenantID(stubRow{id: "tenant-default"})
	if err != nil {
		t.Fatalf("DefaultTenantID() error = %v", err)
	}
	if id != "tenant-default" {
		t.Fatalf("id = %q, want tenant-default", id)
	}
}

func TestDefaultTenantIDError(t *testing.T) {
	_, err := scanDefaultTenantID(stubRow{err: pgx.ErrNoRows})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "lookup default tenant") {
		t.Fatalf("error = %v, want lookup context", err)
	}
}

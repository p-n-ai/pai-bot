// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestCloseAllRunsInReverseOrderAndJoinsErrors(t *testing.T) {
	firstErr := errors.New("first close")
	secondErr := errors.New("second close")
	var order []string

	err := closeAll([]func() error{
		func() error {
			order = append(order, "first")
			return firstErr
		},
		func() error {
			order = append(order, "second")
			return secondErr
		},
	})

	if !reflect.DeepEqual(order, []string{"second", "first"}) {
		t.Fatalf("cleanup order = %v, want reverse acquisition order", order)
	}
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("closeAll() error = %v, want both cleanup errors", err)
	}
}

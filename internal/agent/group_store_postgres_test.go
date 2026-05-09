// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"strings"
	"testing"
)

func TestBuildWeeklyLeaderboardQueryUsesJoinedMemberSet(t *testing.T) {
	query, args := buildWeeklyLeaderboardQuery("33333333-3333-3333-3333-333333333333", 7)

	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2", len(args))
	}
	if args[0] != "33333333-3333-3333-3333-333333333333" || args[1] != 7 {
		t.Fatalf("args = %#v, want group id and limit", args)
	}
	if strings.Contains(query, "IN (SELECT user_id FROM members)") {
		t.Fatalf("query should join the member set instead of repeated IN subqueries:\n%s", query)
	}
	for _, want := range []string{
		"SELECT gm.user_id, gm.tenant_id",
		"JOIN learning_progress lp",
		"ON lp.user_id = m.user_id",
		"AND lp.tenant_id = m.tenant_id",
		"JOIN mastery_snapshots ms",
		"AND ms.tenant_id = m.tenant_id",
		"LIMIT $2",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}
}

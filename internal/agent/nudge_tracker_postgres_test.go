// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"strings"
	"testing"
)

func TestBuildNudgeCountTodayQueryUsesSargableSentAtRange(t *testing.T) {
	query, args := buildNudgeCountTodayForLearnerQuery("tenant-1", "learner-1")

	if len(args) != 3 {
		t.Fatalf("args len = %d, want 3", len(args))
	}
	if args[0] != "tenant-1" || args[1] != "learner-1" || args[2] != nudgeDayTimeZone {
		t.Fatalf("args = %#v, want tenant id, learner id, timezone", args)
	}
	if strings.Contains(query, "nl.sent_at AT TIME ZONE") {
		t.Fatalf("query should not wrap indexed sent_at in a WHERE function:\n%s", query)
	}
	for _, want := range []string{
		"nl.user_id = $2::uuid",
		"nl.sent_at >=",
		"nl.sent_at <",
		"NOW() AT TIME ZONE $3",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}
}

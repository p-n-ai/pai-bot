package agent

import (
	"strings"
	"testing"
)

func TestLockingChallengeSelectAvoidsOuterJoins(t *testing.T) {
	query := lockingChallengeSelect()
	if strings.Contains(query, "LEFT JOIN") {
		t.Fatalf("lockingChallengeSelect() = %q, want no outer joins", query)
	}
	if !strings.Contains(query, "(SELECT u.external_id FROM users u WHERE u.id = c.opponent_user_id)") {
		t.Fatalf("lockingChallengeSelect() = %q, want opponent external id subquery", query)
	}
}

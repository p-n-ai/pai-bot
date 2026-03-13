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

func TestChallengeCanPromoteToAI(t *testing.T) {
	tests := []struct {
		name      string
		challenge *Challenge
		want      bool
	}{
		{
			name: "waiting public queue without opponent",
			challenge: &Challenge{
				Source: challengeSourcePublicQueue,
				State:  challengeStateWaiting,
			},
			want: true,
		},
		{
			name: "already matched human",
			challenge: &Challenge{
				Source:     challengeSourcePublicQueue,
				State:      challengeStateWaiting,
				OpponentID: "peer",
			},
			want: false,
		},
		{
			name: "already ready",
			challenge: &Challenge{
				Source: challengeSourcePublicQueue,
				State:  challengeStateReady,
			},
			want: false,
		},
		{
			name: "private challenge",
			challenge: &Challenge{
				Source: challengeSourcePrivateCode,
				State:  challengeStateWaiting,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := challengeCanPromoteToAI(tt.challenge); got != tt.want {
				t.Fatalf("challengeCanPromoteToAI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateHumanMatchTarget(t *testing.T) {
	tests := []struct {
		name      string
		challenge *Challenge
		opponent  string
		wantErr   error
	}{
		{
			name: "valid waiting queue",
			challenge: &Challenge{
				CreatorID: "creator",
				Source:    challengeSourcePublicQueue,
				State:     challengeStateWaiting,
			},
			opponent: "peer",
		},
		{
			name: "self join rejected",
			challenge: &Challenge{
				CreatorID: "creator",
				Source:    challengeSourcePublicQueue,
				State:     challengeStateWaiting,
			},
			opponent: "creator",
			wantErr:  ErrChallengeSelfJoin,
		},
		{
			name: "private challenge rejected",
			challenge: &Challenge{
				CreatorID: "creator",
				Source:    challengeSourcePrivateCode,
				State:     challengeStateWaiting,
			},
			opponent: "peer",
			wantErr:  ErrChallengeNotFound,
		},
		{
			name: "matched challenge rejected",
			challenge: &Challenge{
				CreatorID:  "creator",
				Source:     challengeSourcePublicQueue,
				State:      challengeStateWaiting,
				OpponentID: "other",
			},
			opponent: "peer",
			wantErr:  ErrChallengeFull,
		},
		{
			name: "same opponent treated as active",
			challenge: &Challenge{
				CreatorID:  "creator",
				Source:     challengeSourcePublicQueue,
				State:      challengeStateWaiting,
				OpponentID: "peer",
			},
			opponent: "peer",
			wantErr:  ErrChallengeAlreadyActive,
		},
		{
			name: "non waiting rejected",
			challenge: &Challenge{
				CreatorID: "creator",
				Source:    challengeSourcePublicQueue,
				State:     challengeStateReady,
			},
			opponent: "peer",
			wantErr:  ErrChallengeFull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHumanMatchTarget(tt.challenge, tt.opponent)
			if err != tt.wantErr {
				t.Fatalf("validateHumanMatchTarget() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

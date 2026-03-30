package agent

import "math/rand/v2"

const (
	ABGroupA = "A"
	ABGroupB = "B"
)

func AssignABGroup() string {
	if rand.IntN(2) == 0 {
		return ABGroupA
	}
	return ABGroupB
}

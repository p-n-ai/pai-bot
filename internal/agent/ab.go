// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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

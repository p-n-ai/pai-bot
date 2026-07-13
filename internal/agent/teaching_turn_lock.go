// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import "sync"

type conversationTurnLock struct {
	mu   sync.Mutex
	refs int
}

func (e *Engine) lockTeachingTurn(conversationID string) func() {
	e.teachingTurnMu.Lock()
	lock := e.teachingTurns[conversationID]
	if lock == nil {
		lock = &conversationTurnLock{}
		e.teachingTurns[conversationID] = lock
	}
	lock.refs++
	e.teachingTurnMu.Unlock()

	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		e.teachingTurnMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(e.teachingTurns, conversationID)
		}
		e.teachingTurnMu.Unlock()
	}
}

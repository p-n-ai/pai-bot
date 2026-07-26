// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"errors"
	"fmt"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

var errLearnerNotFound = errors.New("learner not found")

func (e *Engine) getUserNameForChannel(channel, externalID string) (string, bool) {
	identity, err := NewLearnerIdentity(channel, externalID)
	if err != nil {
		return "", false
	}
	return e.getUserName(identity)
}

func (e *Engine) identityStore() (IdentityConversationStore, bool) {
	store, ok := e.store.(IdentityConversationStore)
	return store, ok
}

func (e *Engine) userExists(identity LearnerIdentity) bool {
	if store, ok := e.identityStore(); ok {
		return store.UserExistsFor(identity)
	}
	return e.store.UserExists(identity.ExternalID())
}

func (e *Engine) getUserName(identity LearnerIdentity) (string, bool) {
	if store, ok := e.identityStore(); ok {
		return store.GetUserNameFor(identity)
	}
	return e.store.GetUserName(identity.ExternalID())
}

func (e *Engine) setUserName(identity LearnerIdentity, name string) error {
	if store, ok := e.identityStore(); ok {
		return store.SetUserNameFor(identity, name)
	}
	return e.store.SetUserName(identity.ExternalID(), name)
}

func (e *Engine) getUserForm(identity LearnerIdentity) (string, bool) {
	if store, ok := e.identityStore(); ok {
		return store.GetUserFormFor(identity)
	}
	return e.store.GetUserForm(identity.ExternalID())
}

func (e *Engine) setUserForm(identity LearnerIdentity, form string) error {
	if store, ok := e.identityStore(); ok {
		return store.SetUserFormFor(identity, form)
	}
	return e.store.SetUserForm(identity.ExternalID(), form)
}

func (e *Engine) getUserPreferredLanguage(identity LearnerIdentity) (string, bool) {
	if store, ok := e.identityStore(); ok {
		return store.GetUserPreferredLanguageFor(identity)
	}
	return e.store.GetUserPreferredLanguage(identity.ExternalID())
}

func (e *Engine) setUserPreferredLanguage(identity LearnerIdentity, language string) error {
	if store, ok := e.identityStore(); ok {
		return store.SetUserPreferredLanguageFor(identity, language)
	}
	return e.store.SetUserPreferredLanguage(identity.ExternalID(), language)
}

func (e *Engine) getUserPreferredQuizIntensity(identity LearnerIdentity) (string, bool) {
	if store, ok := e.identityStore(); ok {
		return store.GetUserPreferredQuizIntensityFor(identity)
	}
	return e.store.GetUserPreferredQuizIntensity(identity.ExternalID())
}

func (e *Engine) setUserPreferredQuizIntensity(identity LearnerIdentity, intensity string) error {
	if store, ok := e.identityStore(); ok {
		return store.SetUserPreferredQuizIntensityFor(identity, intensity)
	}
	return e.store.SetUserPreferredQuizIntensity(identity.ExternalID(), intensity)
}

func (e *Engine) getUserABGroup(identity LearnerIdentity) (string, bool) {
	if store, ok := e.identityStore(); ok {
		return store.GetUserABGroupFor(identity)
	}
	return e.store.GetUserABGroup(identity.ExternalID())
}

func (e *Engine) setUserABGroup(identity LearnerIdentity, group string) error {
	if store, ok := e.identityStore(); ok {
		return store.SetUserABGroupFor(identity, group)
	}
	return e.store.SetUserABGroup(identity.ExternalID(), group)
}

func (e *Engine) resolveUserUUID(identity LearnerIdentity) (string, error) {
	if store, ok := e.identityStore(); ok {
		return store.ResolveUserUUIDFor(identity)
	}
	return e.store.ResolveUserUUID(identity.ExternalID())
}

func (e *Engine) progressLearnerID(identity LearnerIdentity) (progress.LearnerID, error) {
	userUUID, err := e.resolveUserUUID(identity)
	if err != nil {
		return progress.LearnerID{}, err
	}
	if userUUID == "" {
		return progress.LearnerID{}, fmt.Errorf("%w: %s/%s", errLearnerNotFound, identity.Channel(), identity.ExternalID())
	}
	return progress.NewLearnerID(userUUID)
}

func (e *Engine) updateMastery(identity LearnerIdentity, syllabusID, topicID string, delta float64) error {
	if tracker, ok := e.tracker.(progress.LearnerTracker); ok {
		learnerID, err := e.progressLearnerID(identity)
		if err != nil {
			return err
		}
		return tracker.UpdateMasteryForLearner(learnerID, syllabusID, topicID, delta)
	}
	return e.tracker.UpdateMastery(identity.ExternalID(), syllabusID, topicID, delta)
}

func (e *Engine) getMastery(identity LearnerIdentity, syllabusID, topicID string) (float64, error) {
	if tracker, ok := e.tracker.(progress.LearnerTracker); ok {
		learnerID, err := e.progressLearnerID(identity)
		if err != nil {
			if errors.Is(err, errLearnerNotFound) {
				return 0, nil
			}
			return 0, err
		}
		return tracker.GetMasteryForLearner(learnerID, syllabusID, topicID)
	}
	return e.tracker.GetMastery(identity.ExternalID(), syllabusID, topicID)
}

func (e *Engine) getAllProgress(identity LearnerIdentity) ([]progress.ProgressItem, error) {
	if tracker, ok := e.tracker.(progress.LearnerTracker); ok {
		learnerID, err := e.progressLearnerID(identity)
		if err != nil {
			if errors.Is(err, errLearnerNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return tracker.GetAllProgressForLearner(learnerID)
	}
	return e.tracker.GetAllProgress(identity.ExternalID())
}

func (e *Engine) getDueReviews(identity LearnerIdentity) ([]progress.ProgressItem, error) {
	if tracker, ok := e.tracker.(progress.LearnerTracker); ok {
		learnerID, err := e.progressLearnerID(identity)
		if err != nil {
			if errors.Is(err, errLearnerNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return tracker.GetDueReviewsForLearner(learnerID)
	}
	return e.tracker.GetDueReviews(identity.ExternalID())
}

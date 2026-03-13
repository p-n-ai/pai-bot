package agent

import (
	"fmt"
	"math"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

type challengeRequest struct {
	Raw   string
	Topic curriculum.Topic
	Form  string
}

func (e *Engine) resolveChallengeRequest(userID, raw string) (*challengeRequest, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, challengeEmptyStateMessage(), nil
	}
	topic, _ := e.contextResolver.Resolve(raw)
	if topic == nil {
		return nil, "I couldn't find a challenge topic from that yet.\n\nTry something like:\n- /challenge linear equations\n- /challenge private fractions", nil
	}
	form, _ := e.store.GetUserForm(userID)
	return &challengeRequest{
		Raw:   raw,
		Topic: *topic,
		Form:  strings.TrimSpace(form),
	}, "", nil
}

func (e *Engine) tryHumanChallengeMatch(userID string, request *challengeRequest) (*Challenge, error) {
	candidates, err := e.challenges.ListWaitingPublicChallenges(userID)
	if err != nil {
		return nil, err
	}
	best := e.pickHumanChallengeCandidate(request, candidates)
	if best == nil {
		return nil, nil
	}

	topic, err := e.selectHumanBattleTopic(userID, request.Topic, best.CreatorID, best)
	if err != nil {
		return nil, err
	}
	input, err := e.buildHumanChallengeInput(request, best, topic)
	if err != nil {
		return nil, err
	}
	return e.challenges.ActivateHumanMatch(best.Code, userID, input)
}

func (e *Engine) pickHumanChallengeCandidate(request *challengeRequest, candidates []*Challenge) *Challenge {
	var best *Challenge
	bestFormGap := math.MaxFloat64
	for _, candidate := range candidates {
		if candidate == nil || candidate.OpponentID != "" || candidate.Source != challengeSourcePublicQueue {
			continue
		}
		if !challengeTopicsCompatible(request.Topic, candidate) {
			continue
		}
		formGap := challengeFormGap(request.Form, candidate.Metadata.CreatorForm)
		if best == nil || formGap < bestFormGap || (formGap == bestFormGap && candidate.CreatedAt.Before(best.CreatedAt)) {
			best = candidate
			bestFormGap = formGap
		}
	}
	return best
}

func challengeTopicsCompatible(request curriculum.Topic, candidate *Challenge) bool {
	if candidate == nil {
		return false
	}
	if candidate.SubjectID == "" || candidate.SyllabusID == "" {
		return false
	}
	return request.SubjectID == candidate.SubjectID && request.SyllabusID == candidate.SyllabusID
}

func challengeFormGap(a, b string) float64 {
	ai, aok := parseFormNumber(a)
	bi, bok := parseFormNumber(b)
	if !aok || !bok {
		return 0
	}
	return math.Abs(float64(ai - bi))
}

func parseFormNumber(raw string) (int, bool) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.TrimPrefix(raw, "form ")
	if raw == "" {
		return 0, false
	}
	switch raw {
	case "1", "2", "3", "4", "5":
		return int(raw[0] - '0'), true
	default:
		return 0, false
	}
}

func (e *Engine) selectHumanBattleTopic(userID string, requested curriculum.Topic, opponentID string, candidate *Challenge) (curriculum.Topic, error) {
	candidateTopic, ok := e.curriculumLoader.GetTopic(candidate.Metadata.RequestedTopicID)
	if !ok {
		candidateTopic = requested
	}
	neighborhood := e.challengeTopicNeighborhood(requested)
	if len(neighborhood) == 0 {
		return requested, nil
	}

	best := requested
	bestNeed := -1.0
	for _, topic := range neighborhood {
		need := e.challengeNeedScore(userID, topic) + e.challengeNeedScore(opponentID, topic)
		if need > bestNeed {
			best = topic
			bestNeed = need
		}
	}
	if bestNeed >= 0 {
		return best, nil
	}
	return candidateTopic, nil
}

func (e *Engine) buildHumanChallengeInput(request *challengeRequest, candidate *Challenge, topic curriculum.Topic) (ChallengeInput, error) {
	return e.challengeInputForTopic(topic, ChallengeMetadata{
		RequestedText:      request.Raw,
		RequestedTopicID:   request.Topic.ID,
		RequestedTopicName: request.Topic.Name,
		CreatorForm:        request.Form,
		OpponentForm:       candidate.Metadata.CreatorForm,
	})
}

func (e *Engine) buildPrivateChallengeInput(userID string, request *challengeRequest) (curriculum.Topic, ChallengeInput, error) {
	topic := e.selectSoloChallengeTopic(userID, request.Topic)
	input, err := e.challengeInputForTopic(topic, ChallengeMetadata{
		RequestedText:      request.Raw,
		RequestedTopicID:   request.Topic.ID,
		RequestedTopicName: request.Topic.Name,
		CreatorForm:        request.Form,
	})
	return topic, input, err
}

func (e *Engine) activateAIChallenge(userID string, challenge *Challenge) (*Challenge, error) {
	requestedTopic, ok := e.curriculumLoader.GetTopic(challenge.Metadata.RequestedTopicID)
	if !ok {
		requestedTopic, ok = e.curriculumLoader.GetTopic(challenge.TopicID)
		if !ok {
			return challenge, nil
		}
	}
	finalTopic := e.selectSoloChallengeTopic(userID, requestedTopic)
	aiProfile := e.buildChallengeAIProfile(userID, finalTopic, defaultChallengeQuestionMax)
	input, err := e.challengeInputForTopic(finalTopic, ChallengeMetadata{
		RequestedText:      challenge.Metadata.RequestedText,
		RequestedTopicID:   requestedTopic.ID,
		RequestedTopicName: requestedTopic.Name,
		CreatorForm:        challenge.Metadata.CreatorForm,
		AIProfile:          aiProfile,
	})
	if err != nil {
		return challenge, nil
	}
	return e.challenges.ActivateAIFallback(challenge.Code, input)
}

func (e *Engine) challengeInputForTopic(topic curriculum.Topic, metadata ChallengeMetadata) (ChallengeInput, error) {
	assessment, ok := e.curriculumLoader.GetAssessment(topic.ID)
	if !ok || len(assessment.Questions) == 0 {
		return ChallengeInput{}, fmt.Errorf("assessment not available for topic %s", topic.ID)
	}
	return ChallengeInput{
		TopicID:       topic.ID,
		TopicName:     topic.Name,
		SubjectID:     topic.SubjectID,
		SyllabusID:    topic.SyllabusID,
		Questions:     trimChallengeQuestions(questionsFromAssessment(assessment), defaultChallengeQuestionMax),
		QuestionCount: defaultChallengeQuestionMax,
		Metadata:      metadata,
	}, nil
}

func (e *Engine) selectSoloChallengeTopic(userID string, requested curriculum.Topic) curriculum.Topic {
	neighborhood := e.challengeTopicNeighborhood(requested)
	if len(neighborhood) == 0 {
		return requested
	}
	best := requested
	bestNeed := -1.0
	for _, topic := range neighborhood {
		need := e.challengeNeedScore(userID, topic)
		if need > bestNeed {
			best = topic
			bestNeed = need
		}
	}
	return best
}

func (e *Engine) challengeTopicNeighborhood(requested curriculum.Topic) []curriculum.Topic {
	if e.curriculumLoader == nil {
		return nil
	}
	var neighborhood []curriculum.Topic
	for _, topic := range e.curriculumLoader.AllTopics() {
		if topic.SubjectID != requested.SubjectID || topic.SyllabusID != requested.SyllabusID {
			continue
		}
		if _, ok := e.curriculumLoader.GetAssessment(topic.ID); !ok {
			continue
		}
		neighborhood = append(neighborhood, topic)
	}
	if len(neighborhood) == 0 {
		neighborhood = append(neighborhood, requested)
	}
	return neighborhood
}

func (e *Engine) challengeNeedScore(userID string, topic curriculum.Topic) float64 {
	ability := e.estimateChallengeAbility(userID, topic)
	return 1 - ability
}

func (e *Engine) estimateChallengeAbility(userID string, topic curriculum.Topic) float64 {
	if userID == "" {
		return 0.5
	}
	if e.tracker != nil {
		if mastery, err := e.tracker.GetMastery(userID, topic.SyllabusID, topic.ID); err == nil && mastery > 0 {
			return clampChallengeAbility(mastery)
		}
		for _, nearby := range e.challengeTopicNeighborhood(topic) {
			if nearby.ID == topic.ID {
				continue
			}
			if mastery, err := e.tracker.GetMastery(userID, nearby.SyllabusID, nearby.ID); err == nil && mastery > 0 {
				return clampChallengeAbility((mastery + formAbility(e.store, userID)) / 2)
			}
		}
	}
	return clampChallengeAbility(formAbility(e.store, userID))
}

func formAbility(store ConversationStore, userID string) float64 {
	if store == nil {
		return 0.5
	}
	form, ok := store.GetUserForm(userID)
	if !ok {
		return 0.5
	}
	value, ok := parseFormNumber(form)
	if !ok {
		return 0.5
	}
	return 0.35 + float64(value-1)*0.1
}

func clampChallengeAbility(value float64) float64 {
	if value < 0.15 {
		return 0.15
	}
	if value > 0.9 {
		return 0.9
	}
	return value
}

func (e *Engine) buildChallengeAIProfile(userID string, topic curriculum.Topic, questionCount int) *ChallengeAIProfile {
	ability := e.estimateChallengeAbility(userID, topic)
	shift := stableChallengeShift(userID + ":" + topic.ID)
	planned := clampChallengeCorrectAnswers(int(math.Round(ability*float64(questionCount)))+shift, questionCount)
	label := "Adaptive AI rival"
	if form, ok := e.store.GetUserForm(userID); ok && strings.TrimSpace(form) != "" {
		label = "Adaptive AI rival (Form " + strings.TrimSpace(form) + ")"
	}
	return &ChallengeAIProfile{
		Label:          label,
		AbilityScore:   ability,
		PlannedCorrect: planned,
	}
}

func stableChallengeShift(seed string) int {
	sum := 0
	for _, r := range seed {
		sum += int(r)
	}
	switch sum % 3 {
	case 0:
		return -1
	case 1:
		return 0
	default:
		return 1
	}
}

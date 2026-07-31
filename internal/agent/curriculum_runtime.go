// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

// CurriculumRuntime is the tutor's provider-neutral curriculum policy boundary.
type CurriculumRuntime interface {
	PlanTurn(context.Context, curriculum.PlanTurnInput) (curriculum.TeachingPlan, error)
	RecordAttempt(context.Context, curriculum.AttemptInput) (curriculum.AttemptResult, error)
}

type curriculumPlanControl struct {
	Action      curriculum.TeachingAction
	GoalTopicID string
	TopicID     string
	ObjectiveID string
	QuestionID  string
	Constraints []string
}

type curriculumSourceEvidence struct {
	Label      string
	SourcePath string
	Revision   string
	Content    string
}

func curriculumAttemptID(msg chat.InboundMessage) (string, bool) {
	deliveryID := strings.TrimSpace(msg.DeliveryID)
	channel := strings.TrimSpace(msg.Channel)
	if deliveryID == "" || channel == "" {
		return "", false
	}
	digest := sha256.Sum256([]byte(deliveryID))
	return fmt.Sprintf("chat:v1:%s:%x", channel, digest), true
}

func renderPlannedCheck(check *curriculum.PlannedCheck) string {
	if check == nil || strings.TrimSpace(check.Question) == "" {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(check.Question)
	for _, option := range check.Options {
		fmt.Fprintf(&builder, "\n%s. %s", option.ID, option.Text)
	}
	return builder.String()
}

func ensurePlannedCheck(response string, check *curriculum.PlannedCheck) string {
	rendered := renderPlannedCheck(check)
	if rendered == "" {
		return response
	}
	question := strings.ToLower(strings.Join(strings.Fields(check.Question), " "))
	normalizedResponse := strings.ToLower(strings.Join(strings.Fields(response), " "))
	if question != "" && strings.Contains(normalizedResponse, question) {
		return response
	}
	if strings.TrimSpace(response) == "" {
		return rendered
	}
	return strings.TrimSpace(response) + "\n\n" + rendered
}

func curriculumPlanContextPackets(goalTopicID string, plan curriculum.TeachingPlan) []contextPacket {
	questionID := ""
	if plan.Check != nil {
		questionID = plan.Check.QuestionID
	}
	packets := []contextPacket{newContextPacket(contextPacket{
		ID:     "curriculum.plan.control",
		Kind:   contextKindCurriculumPlan,
		Trust:  contextTrustSystemOwned,
		Source: "curriculum_runtime",
		Data: curriculumPlanControl{
			Action:      plan.Action,
			GoalTopicID: goalTopicID,
			TopicID:     plan.Target.TopicID,
			ObjectiveID: plan.Target.ObjectiveID,
			QuestionID:  questionID,
			Constraints: append([]string(nil), plan.Constraints...),
		},
		RenderAs: contextRenderSystemData,
	})}

	label := 1
	appendEvidence := func(id string, evidence curriculum.EvidenceRef, content string) {
		if strings.TrimSpace(content) == "" {
			return
		}
		packets = append(packets, newContextPacket(contextPacket{
			ID:       id,
			Kind:     contextKindEvidence,
			Trust:    contextTrustExternal,
			Source:   "oss_curriculum",
			RenderAs: contextRenderQuotedData,
			Data: curriculumSourceEvidence{
				Label:      fmt.Sprintf("C%d", label),
				SourcePath: evidence.SourcePath,
				Revision:   evidence.Revision,
				Content:    content,
			},
		}))
		label++
	}

	var guidance strings.Builder
	fmt.Fprintf(&guidance, "Topic: %s\nObjective: %s", plan.Target.TopicName, plan.Target.Objective)
	for _, step := range plan.TeachingSequence {
		fmt.Fprintf(&guidance, "\nTeaching sequence: %s", step)
	}
	for _, item := range plan.Guidance.BackgroundKnowledge {
		fmt.Fprintf(&guidance, "\nBackground knowledge: %s", item)
	}
	for _, item := range plan.Guidance.CommonMisconceptions {
		fmt.Fprintf(&guidance, "\nMisconception: %s\nRemediation: %s", item.Misconception, item.Remediation)
	}
	for _, item := range plan.Guidance.EngagementHooks {
		fmt.Fprintf(&guidance, "\nEngagement hook: %s", item)
	}
	appendEvidence("curriculum.plan.guidance", plan.Guidance.Evidence, guidance.String())

	if plan.Materials.TeachingNote != nil {
		appendEvidence(
			"curriculum.plan.teaching_note",
			plan.Materials.TeachingNote.Evidence,
			plan.Materials.TeachingNote.Content,
		)
	}
	for index, material := range plan.Materials.WorkedExamples {
		example := material.Example
		content := fmt.Sprintf(
			"Worked example %s\nScenario: %s\nWorking: %s\nMisconception alert: %s",
			example.ID,
			example.Scenario,
			example.Working,
			example.MisconceptionAlert,
		)
		appendEvidence(
			fmt.Sprintf("curriculum.plan.example.%d", index+1),
			material.Evidence,
			content,
		)
	}
	if plan.Check != nil {
		evidence := curriculum.EvidenceRef{SourcePath: plan.Check.SourcePath}
		for _, candidate := range plan.Evidence {
			if candidate.QuestionID == plan.Check.QuestionID {
				evidence = candidate
				break
			}
		}
		appendEvidence("curriculum.plan.check", evidence, renderPlannedCheck(plan.Check))
	}
	return packets
}

func (e *Engine) planCurriculumTurn(
	ctx context.Context,
	msg chat.InboundMessage,
	goalTopic *curriculum.Topic,
) (*curriculum.TeachingPlan, *curriculum.Topic, error) {
	if e.curriculumRuntime == nil || goalTopic == nil {
		return nil, goalTopic, nil
	}
	identity, err := learnerIdentityForMessage(msg)
	if err != nil {
		return nil, nil, err
	}
	learnerID, err := e.progressLearnerID(identity)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve curriculum learner: %w", err)
	}
	plan, err := e.curriculumRuntime.PlanTurn(ctx, curriculum.PlanTurnInput{
		LearnerID: learnerID,
		TopicID:   goalTopic.ID,
		Locale:    e.messageLocale(msg, nil),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plan curriculum turn: %w", err)
	}
	activeTopic := goalTopic
	if plan.Target.TopicID != "" && plan.Target.TopicID != goalTopic.ID {
		if e.curriculumLoader == nil {
			return nil, nil, fmt.Errorf("curriculum plan selected topic %q without loader", plan.Target.TopicID)
		}
		selected, found := e.curriculumLoader.GetTopic(plan.Target.TopicID)
		if !found {
			return nil, nil, fmt.Errorf("curriculum plan selected missing topic %q", plan.Target.TopicID)
		}
		activeTopic = &selected
	}
	return &plan, activeTopic, nil
}

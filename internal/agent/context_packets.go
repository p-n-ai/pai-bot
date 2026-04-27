// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func appendProfilePackets(packets []contextPacket, profile learnerProfile) []contextPacket {
	if profile.Form != "" || profile.Language != "" || profile.QuizIntensity != "" || profile.ABGroup != "" {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "profile.system",
			Kind:     contextKindProfile,
			Trust:    contextTrustSystemOwned,
			Source:   "profile",
			Data:     profileSystemContext(profile),
			RenderAs: contextRenderSystemData,
		}))
	}
	if profile.Name != "" {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "profile.name",
			Kind:     contextKindProfile,
			Trust:    contextTrustLearnerProvided,
			Source:   "profile",
			Data:     profile.Name,
			RenderAs: contextRenderQuotedData,
		}))
	}
	return packets
}

func appendGoalPackets(packets []contextPacket, goals []*Goal) []contextPacket {
	if systemGoals := goalSystemContext(goals); len(systemGoals) > 0 {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "goals.system",
			Kind:     contextKindGoal,
			Trust:    contextTrustSystemOwned,
			Source:   "goals",
			Data:     systemGoals,
			RenderAs: contextRenderSystemData,
		}))
	}
	if summaries := goalSummaryContext(goals); len(summaries) > 0 {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "goals.summary",
			Kind:     contextKindGoal,
			Trust:    contextTrustLearnerProvided,
			Source:   "goals",
			Data:     summaries,
			RenderAs: contextRenderQuotedData,
		}))
	}
	return packets
}

func appendImagePackets(packets []contextPacket, imageDataURL string) []contextPacket {
	return append(packets,
		newContextPacket(contextPacket{
			ID:       "image.instruction",
			Kind:     contextKindControlInstruction,
			Trust:    contextTrustSystemOwned,
			Source:   "image",
			Data:     "Analyze the attached image directly and answer based on what you see. If unreadable, say exactly what is unclear and how to retake it.",
			RenderAs: contextRenderSystemInstruction,
		}),
		newContextPacket(contextPacket{
			ID:        "image.attachment",
			Kind:      contextKindImage,
			Trust:     contextTrustExternal,
			Source:    "image",
			Data:      imageDataURL,
			RenderAs:  contextRenderAttachment,
			TraceMode: contextTraceOmit,
		}),
	)
}

func newContextPacket(packet contextPacket) contextPacket {
	if packet.RenderAs == "" {
		packet.RenderAs = defaultRenderMode(packet.Trust)
	}
	if packet.TraceMode == "" {
		packet.TraceMode = contextTraceMetadataOnly
	}
	return packet
}

func defaultRenderMode(trust contextTrust) contextRenderMode {
	if trust == contextTrustSystemOwned {
		return contextRenderSystemData
	}
	return contextRenderQuotedData
}

func validateContextPacket(packet contextPacket) error {
	if packet.Trust == "" {
		return fmt.Errorf("context packet %q missing trust", packet.ID)
	}
	if packet.Kind == "" {
		return fmt.Errorf("context packet %q missing kind", packet.ID)
	}
	if packet.TraceMode == "" {
		return fmt.Errorf("context packet %q missing trace mode", packet.ID)
	}
	if (packet.RenderAs == contextRenderSystemInstruction || packet.RenderAs == contextRenderSystemData) && packet.Trust != contextTrustSystemOwned {
		return fmt.Errorf("context packet %q renders untrusted data as system content", packet.ID)
	}
	return nil
}

func validateContextPackets(packets []contextPacket) error {
	for _, packet := range packets {
		if err := validateContextPacket(packet); err != nil {
			return err
		}
	}
	return nil
}

func contextSources(packets []contextPacket) []contextSource {
	var sources []contextSource
	for _, packet := range packets {
		if packet.TraceMode == contextTraceOmit {
			continue
		}
		sources = append(sources, contextSource{
			Name:     packet.Source,
			Included: true,
		})
	}
	return sources
}

func profileSystemContext(profile learnerProfile) []string {
	var fields []string
	if profile.Form != "" {
		fields = append(fields, "Form: "+profile.Form)
	}
	if profile.Language != "" {
		fields = append(fields, "Preferred language: "+profile.Language)
	}
	if profile.QuizIntensity != "" {
		fields = append(fields, "Preferred quiz intensity: "+profile.QuizIntensity)
	}
	if profile.ABGroup != "" {
		fields = append(fields, "Experiment group: "+profile.ABGroup)
	}
	return fields
}

func conversationSystemContext(conv *Conversation) []string {
	if conv == nil {
		return nil
	}
	var fields []string
	if conv.State != "" {
		fields = append(fields, "Conversation state: "+conv.State)
	}
	if conv.TopicID != "" {
		fields = append(fields, "Active conversation topic ID: "+conv.TopicID)
	}
	return fields
}

type goalSystemData struct {
	TopicID        string
	TopicName      string
	SyllabusID     string
	TargetMastery  float64
	CurrentMastery float64
}

type curriculumTopicData struct {
	ID         string
	Name       string
	SyllabusID string
	SubjectID  string
}

func curriculumTopicContext(topic *curriculum.Topic) curriculumTopicData {
	if topic == nil {
		return curriculumTopicData{}
	}
	return curriculumTopicData{
		ID:         topic.ID,
		Name:       topic.Name,
		SyllabusID: topic.SyllabusID,
		SubjectID:  topic.SubjectID,
	}
}

func goalSystemContext(goals []*Goal) []goalSystemData {
	var out []goalSystemData
	for _, goal := range goals {
		if goal == nil {
			continue
		}
		out = append(out, goalSystemData{
			TopicID:        goal.TopicID,
			TopicName:      goal.TopicName,
			SyllabusID:     goal.SyllabusID,
			TargetMastery:  goal.TargetMastery,
			CurrentMastery: goal.CurrentMastery,
		})
	}
	return out
}

func goalSummaryContext(goals []*Goal) []string {
	var out []string
	for _, goal := range goals {
		if goal == nil || strings.TrimSpace(goal.Summary) == "" {
			continue
		}
		out = append(out, goal.Summary)
	}
	return out
}

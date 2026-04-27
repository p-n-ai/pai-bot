// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func appendProfilePackets(packets []ContextPacket, profile LearnerProfile) []ContextPacket {
	if profile.Form != "" || profile.Language != "" || profile.QuizIntensity != "" || profile.ABGroup != "" {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "profile.system",
			Kind:     ContextKindProfile,
			Trust:    ContextTrustSystemOwned,
			Source:   "profile",
			Data:     profileSystemContext(profile),
			RenderAs: ContextRenderSystemData,
		}))
	}
	if profile.Name != "" {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "profile.name",
			Kind:     ContextKindProfile,
			Trust:    ContextTrustLearnerProvided,
			Source:   "profile",
			Data:     profile.Name,
			RenderAs: ContextRenderQuotedData,
		}))
	}
	return packets
}

func appendGoalPackets(packets []ContextPacket, goals []*Goal) []ContextPacket {
	if systemGoals := goalSystemContext(goals); len(systemGoals) > 0 {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "goals.system",
			Kind:     ContextKindGoal,
			Trust:    ContextTrustSystemOwned,
			Source:   "goals",
			Data:     systemGoals,
			RenderAs: ContextRenderSystemData,
		}))
	}
	if summaries := goalSummaryContext(goals); len(summaries) > 0 {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "goals.summary",
			Kind:     ContextKindGoal,
			Trust:    ContextTrustLearnerProvided,
			Source:   "goals",
			Data:     summaries,
			RenderAs: ContextRenderQuotedData,
		}))
	}
	return packets
}

func appendImagePackets(packets []ContextPacket, imageDataURL string) []ContextPacket {
	return append(packets,
		newContextPacket(ContextPacket{
			ID:       "image.instruction",
			Kind:     ContextKindControlInstruction,
			Trust:    ContextTrustSystemOwned,
			Source:   "image",
			Data:     "Analyze the attached image directly and answer based on what you see. If unreadable, say exactly what is unclear and how to retake it.",
			RenderAs: ContextRenderSystemInstruction,
		}),
		newContextPacket(ContextPacket{
			ID:        "image.attachment",
			Kind:      ContextKindImage,
			Trust:     ContextTrustExternal,
			Source:    "image",
			Data:      imageDataURL,
			RenderAs:  ContextRenderAttachment,
			TraceMode: ContextTraceOmit,
		}),
	)
}

func newContextPacket(packet ContextPacket) ContextPacket {
	if packet.RenderAs == "" {
		packet.RenderAs = defaultRenderMode(packet.Trust)
	}
	if packet.TraceMode == "" {
		packet.TraceMode = ContextTraceMetadataOnly
	}
	return packet
}

func defaultRenderMode(trust ContextTrust) ContextRenderMode {
	if trust == ContextTrustSystemOwned {
		return ContextRenderSystemData
	}
	return ContextRenderQuotedData
}

func validateContextPacket(packet ContextPacket) error {
	if packet.Trust == "" {
		return fmt.Errorf("context packet %q missing trust", packet.ID)
	}
	if packet.Kind == "" {
		return fmt.Errorf("context packet %q missing kind", packet.ID)
	}
	if packet.TraceMode == "" {
		return fmt.Errorf("context packet %q missing trace mode", packet.ID)
	}
	if (packet.RenderAs == ContextRenderSystemInstruction || packet.RenderAs == ContextRenderSystemData) && packet.Trust != ContextTrustSystemOwned {
		return fmt.Errorf("context packet %q renders untrusted data as system content", packet.ID)
	}
	return nil
}

func validateContextPackets(packets []ContextPacket) error {
	for _, packet := range packets {
		if err := validateContextPacket(packet); err != nil {
			return err
		}
	}
	return nil
}

func contextSources(packets []ContextPacket) []ContextSource {
	var sources []ContextSource
	for _, packet := range packets {
		if packet.TraceMode == ContextTraceOmit {
			continue
		}
		sources = append(sources, ContextSource{
			Name:     packet.Source,
			Included: true,
		})
	}
	return sources
}

func profileSystemContext(profile LearnerProfile) []string {
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

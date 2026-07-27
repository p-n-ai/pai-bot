// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval

import (
	"context"
	"testing"
)

type membershipStub struct {
	tenantID, learnerID string
	classIDs            []string
}

func (s *membershipStub) ActiveClassIDs(_ context.Context, tenantID, learnerID string) ([]string, error) {
	s.tenantID, s.learnerID = tenantID, learnerID
	return s.classIDs, nil
}

type teacherSearchStub struct {
	request TeacherEvidenceRequest
	items   []TeacherEvidence
}

func (s *teacherSearchStub) Search(_ context.Context, request TeacherEvidenceRequest) ([]TeacherEvidence, error) {
	s.request = request
	return s.items, nil
}

func TestTutorEvidenceFusesOSSAndOnlyLearnerClasses(t *testing.T) {
	oss := NewService()
	active := true
	if _, err := oss.UpsertDocument(UpsertDocumentInput{
		ID: "oss-linear", Kind: "teaching_note", Title: "Linear equations",
		Body:     "Keep an equation balanced by applying the same operation to both sides.",
		SourceID: "source:curriculum", SourceType: "curriculum", Source: "oss/algebra.md", Active: &active,
	}); err != nil {
		t.Fatal(err)
	}
	memberships := &membershipStub{classIDs: []string{"class-a"}}
	teacher := &teacherSearchStub{items: []TeacherEvidence{{
		ID: "chunk-a", SourceTitle: "Cikgu's method", Filename: "lesson.pdf",
		LocatorType: "page", LocatorStart: 3, LocatorEnd: 3, Excerpt: "Use the balance scale phrase.",
	}}}
	service := newTutorEvidenceService(oss, teacher, memberships)

	items, err := service.Retrieve(context.Background(), TutorEvidenceRequest{
		TenantID: "tenant-a", LearnerID: "learner-a", Query: "linear equation balance", Limit: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Origin != "oss" || items[1].Origin != "teacher" {
		t.Fatalf("fused evidence = %#v", items)
	}
	if memberships.tenantID != "tenant-a" || memberships.learnerID != "learner-a" {
		t.Fatalf("membership scope = %q/%q", memberships.tenantID, memberships.learnerID)
	}
	if len(teacher.request.ClassIDs) != 1 || teacher.request.ClassIDs[0] != "class-a" ||
		teacher.request.TenantID != "tenant-a" {
		t.Fatalf("teacher search scope = %#v", teacher.request)
	}
}

func TestTutorEvidenceWithoutMembershipIsOSSOnly(t *testing.T) {
	oss := NewService()
	active := true
	_, _ = oss.UpsertDocument(UpsertDocumentInput{
		ID: "oss-fractions", Kind: "topic_card", Title: "Fractions", Body: "Equivalent fractions",
		SourceType: "curriculum", Source: "oss/fractions.yaml", Active: &active,
	})
	teacher := &teacherSearchStub{items: []TeacherEvidence{{ID: "must-not-leak"}}}
	service := newTutorEvidenceService(oss, teacher, &membershipStub{})
	items, err := service.Retrieve(context.Background(), TutorEvidenceRequest{
		TenantID: "tenant-a", LearnerID: "learner-a", Query: "fractions", Limit: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Origin != "oss" {
		t.Fatalf("evidence = %#v, want OSS only", items)
	}
	if teacher.request.Query != "" {
		t.Fatalf("teacher search ran without class membership: %#v", teacher.request)
	}
}

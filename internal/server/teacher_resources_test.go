// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

const (
	teacherTestTenantID   = "00000000-0000-4000-8000-000000000001"
	teacherTestUploaderID = "00000000-0000-4000-8000-000000000002"
	teacherTestClassOneID = "00000000-0000-4000-8000-000000000011"
	teacherTestClassTwoID = "00000000-0000-4000-8000-000000000012"
	teacherTestResourceID = "00000000-0000-4000-8000-000000000021"
	teacherTestChunkID    = "00000000-0000-4000-8000-000000000031"
)

type teacherResourceServiceStub struct {
	uploadInput retrieval.TeacherUploadInput
	uploadItem  retrieval.TeacherResource
	uploadErr   error
	listTenant  string
	listClasses []string
	listItems   []retrieval.TeacherResource
	listErr     error
	activeID    string
	activeValue bool
	activeErr   error
	deleteID    string
	deleteErr   error
	searchInput retrieval.TeacherEvidenceRequest
	searchItems []retrieval.TeacherEvidence
	searchErr   error
}

func (s *teacherResourceServiceStub) Upload(_ context.Context, input retrieval.TeacherUploadInput) (retrieval.TeacherResource, error) {
	s.uploadInput = input
	return s.uploadItem, s.uploadErr
}

func (s *teacherResourceServiceStub) List(_ context.Context, tenantID string, classIDs []string, _ bool) ([]retrieval.TeacherResource, error) {
	s.listTenant = tenantID
	s.listClasses = classIDs
	return s.listItems, s.listErr
}

func (s *teacherResourceServiceStub) SetActive(_ context.Context, _ string, resourceID string, _ []string, active bool) error {
	s.activeID = resourceID
	s.activeValue = active
	return s.activeErr
}

func (s *teacherResourceServiceStub) Delete(_ context.Context, _ string, resourceID string, _ []string) error {
	s.deleteID = resourceID
	return s.deleteErr
}

func (s *teacherResourceServiceStub) Search(_ context.Context, input retrieval.TeacherEvidenceRequest) ([]retrieval.TeacherEvidence, error) {
	s.searchInput = input
	return s.searchItems, s.searchErr
}

func TestTeacherResourceAuthenticatedMultipartUpload(t *testing.T) {
	service := &teacherResourceServiceStub{uploadItem: retrieval.TeacherResource{
		ID: teacherTestResourceID, Filename: "lesson.docx", ChunkCount: 2, ClassIDs: []string{teacherTestClassOneID},
	}}
	handler := teacherResourceTestHandler(service)
	body, contentType := teacherMultipartBody(t, "lesson.docx", []byte("document"), []string{teacherTestClassOneID})
	req := teacherRequest(http.MethodPost, "/api/admin/teacher-resources", body, teacherTestTenantID, teacherTestUploaderID)
	req.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if service.uploadInput.TenantID != teacherTestTenantID || service.uploadInput.UploaderID != teacherTestUploaderID {
		t.Fatalf("auth-derived upload input = %#v", service.uploadInput)
	}
	if service.uploadInput.Filename != "lesson.docx" || len(service.uploadInput.ClassIDs) != 1 {
		t.Fatalf("multipart upload input = %#v", service.uploadInput)
	}
	var response retrieval.TeacherResource
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.ChunkCount != 2 {
		t.Fatalf("chunk_count = %d, want 2", response.ChunkCount)
	}
}

func TestTeacherResourceUploadRequiresTenantAndUploader(t *testing.T) {
	for _, test := range []struct {
		name, tenantID, uploaderID string
	}{
		{name: "missing tenant", uploaderID: teacherTestUploaderID},
		{name: "missing uploader", tenantID: teacherTestTenantID},
		{name: "malformed tenant", tenantID: "not-a-uuid", uploaderID: teacherTestUploaderID},
		{name: "malformed uploader", tenantID: teacherTestTenantID, uploaderID: "not-a-uuid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := teacherResourceTestHandler(&teacherResourceServiceStub{})
			body, contentType := teacherMultipartBody(t, "lesson.docx", []byte("document"), []string{teacherTestClassOneID})
			req := teacherRequest(http.MethodPost, "/api/admin/teacher-resources", body, test.tenantID, test.uploaderID)
			req.Header.Set("Content-Type", contentType)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", recorder.Code)
			}
		})
	}
}

func TestTeacherResourceUploadErrors(t *testing.T) {
	t.Run("oversize", func(t *testing.T) {
		service := &teacherResourceServiceStub{}
		handler := teacherResourceTestHandler(service)
		body, contentType := teacherMultipartBody(t, "large.pdf", make([]byte, retrieval.MaxTeacherResourceBytes+1), []string{teacherTestClassOneID})
		req := teacherRequest(http.MethodPost, "/api/admin/teacher-resources", body, teacherTestTenantID, teacherTestUploaderID)
		req.Header.Set("Content-Type", contentType)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
	})
	for _, test := range []struct {
		name string
		err  error
		want int
	}{
		{name: "unsupported", err: retrieval.ErrUnsupportedFile, want: http.StatusUnsupportedMediaType},
		{name: "class scope", err: retrieval.ErrTeacherResourceScope, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &teacherResourceServiceStub{uploadErr: test.err}
			handler := teacherResourceTestHandler(service)
			body, contentType := teacherMultipartBody(t, "lesson.txt", []byte("text"), []string{teacherTestClassOneID})
			req := teacherRequest(http.MethodPost, "/api/admin/teacher-resources", body, teacherTestTenantID, teacherTestUploaderID)
			req.Header.Set("Content-Type", contentType)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != test.want {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestTeacherResourceListDeactivateDeleteAndSearch(t *testing.T) {
	service := &teacherResourceServiceStub{
		listItems:   []retrieval.TeacherResource{{ID: teacherTestResourceID, ChunkCount: 3}},
		searchItems: []retrieval.TeacherEvidence{{ID: teacherTestChunkID, Excerpt: "neighbor"}},
	}
	handler := teacherResourceTestHandler(service)

	listReq := teacherRequest(http.MethodGet, "/api/admin/teacher-resources?class_ids="+teacherTestClassOneID+","+teacherTestClassTwoID, nil, teacherTestTenantID, teacherTestUploaderID)
	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, listReq)
	if listRecorder.Code != http.StatusOK || service.listTenant != teacherTestTenantID || len(service.listClasses) != 2 {
		t.Fatalf("list status/input = %d, %q, %#v", listRecorder.Code, service.listTenant, service.listClasses)
	}
	if !strings.Contains(listRecorder.Body.String(), `"chunk_count":3`) {
		t.Fatalf("list response = %s", listRecorder.Body.String())
	}

	deactivateReq := teacherRequest(http.MethodPost, "/api/admin/teacher-resources/"+teacherTestResourceID+"/deactivate?class_id="+teacherTestClassOneID, nil, teacherTestTenantID, teacherTestUploaderID)
	deactivateRecorder := httptest.NewRecorder()
	handler.ServeHTTP(deactivateRecorder, deactivateReq)
	if deactivateRecorder.Code != http.StatusNoContent || service.activeID != teacherTestResourceID || service.activeValue {
		t.Fatalf("deactivate status/input = %d, %q, %t", deactivateRecorder.Code, service.activeID, service.activeValue)
	}

	deleteReq := teacherRequest(http.MethodDelete, "/api/admin/teacher-resources/"+teacherTestResourceID+"?class_id="+teacherTestClassOneID, nil, teacherTestTenantID, teacherTestUploaderID)
	deleteRecorder := httptest.NewRecorder()
	handler.ServeHTTP(deleteRecorder, deleteReq)
	if deleteRecorder.Code != http.StatusNoContent || service.deleteID != teacherTestResourceID {
		t.Fatalf("delete status/input = %d, %q", deleteRecorder.Code, service.deleteID)
	}

	searchReq := teacherRequest(http.MethodPost, "/api/admin/teacher-resources/search",
		strings.NewReader(`{"query":"pecahan","class_ids":["`+teacherTestClassOneID+`"],"limit":4}`), teacherTestTenantID, teacherTestUploaderID)
	searchRecorder := httptest.NewRecorder()
	handler.ServeHTTP(searchRecorder, searchReq)
	if searchRecorder.Code != http.StatusOK || service.searchInput.Query != "pecahan" ||
		service.searchInput.TenantID != teacherTestTenantID || len(service.searchInput.ClassIDs) != 1 {
		t.Fatalf("search status/input = %d, %#v", searchRecorder.Code, service.searchInput)
	}
}

func TestTeacherResourceHandlersRejectMalformedUUIDsBeforeService(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   io.Reader
		setup  func(*http.Request)
	}{
		{
			name: "upload class", method: http.MethodPost, target: "/api/admin/teacher-resources",
			setup: func(req *http.Request) {
				body, contentType := teacherMultipartBody(t, "lesson.docx", []byte("document"), []string{"not-a-uuid"})
				req.Body = io.NopCloser(body)
				req.Header.Set("Content-Type", contentType)
			},
		},
		{name: "list class", method: http.MethodGet, target: "/api/admin/teacher-resources?class_id=not-a-uuid"},
		{name: "deactivate resource", method: http.MethodPost, target: "/api/admin/teacher-resources/not-a-uuid/deactivate?class_id=" + teacherTestClassOneID},
		{name: "deactivate class", method: http.MethodPost, target: "/api/admin/teacher-resources/" + teacherTestResourceID + "/deactivate?class_id=not-a-uuid"},
		{name: "delete resource", method: http.MethodDelete, target: "/api/admin/teacher-resources/not-a-uuid?class_id=" + teacherTestClassOneID},
		{
			name: "search class", method: http.MethodPost, target: "/api/admin/teacher-resources/search",
			body: strings.NewReader(`{"query":"pecahan","class_ids":["not-a-uuid"]}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &teacherResourceServiceStub{}
			handler := teacherResourceTestHandler(service)
			req := teacherRequest(test.method, test.target, test.body, teacherTestTenantID, teacherTestUploaderID)
			if test.setup != nil {
				test.setup(req)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if service.uploadInput.TenantID != "" || service.listTenant != "" ||
				service.activeID != "" || service.deleteID != "" || service.searchInput.TenantID != "" {
				t.Fatalf("service was called: %#v", service)
			}
		})
	}
}

func teacherResourceTestHandler(service teacherResourceService) http.Handler {
	mux := http.NewServeMux()
	registerTeacherResourceRoutes(mux, service, auth.RequireRoles(auth.RoleTeacher, auth.RoleAdmin, auth.RolePlatformAdmin))
	return mux
}

func teacherRequest(method, target string, body io.Reader, tenantID, uploaderID string) *http.Request {
	req := httptest.NewRequest(method, target, body)
	return req.WithContext(auth.WithClaims(req.Context(), auth.TokenClaims{
		Subject: uploaderID, TenantID: tenantID, Role: auth.RoleTeacher,
	}))
}

func teacherMultipartBody(t *testing.T, filename string, data []byte, classIDs []string) (*bytes.Reader, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, classID := range classIDs {
		if err := writer.WriteField("class_id", classID); err != nil {
			t.Fatal(err)
		}
	}
	file, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(body.Bytes()), writer.FormDataContentType()
}

// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

const maxTeacherMultipartBytes = retrieval.MaxTeacherResourceBytes + (1 << 20)

type teacherResourceService interface {
	Upload(context.Context, retrieval.TeacherUploadInput) (retrieval.TeacherResource, error)
	List(context.Context, string, []string, bool) ([]retrieval.TeacherResource, error)
	SetActive(context.Context, string, string, []string, bool) error
	Delete(context.Context, string, string, []string) error
	Search(context.Context, retrieval.TeacherEvidenceRequest) ([]retrieval.TeacherEvidence, error)
}

func registerTeacherResourceRoutes(mux *http.ServeMux, service teacherResourceService, teacherOrAbove func(http.Handler) http.Handler) {
	mux.Handle("POST /api/admin/teacher-resources", teacherOrAbove(handleTeacherResourceUpload(service)))
	mux.Handle("GET /api/admin/teacher-resources", teacherOrAbove(handleTeacherResourceList(service)))
	mux.Handle("POST /api/admin/teacher-resources/{id}/deactivate", teacherOrAbove(handleTeacherResourceDeactivate(service)))
	mux.Handle("DELETE /api/admin/teacher-resources/{id}", teacherOrAbove(handleTeacherResourceDelete(service)))
	mux.Handle("POST /api/admin/teacher-resources/search", teacherOrAbove(handleTeacherResourceSearch(service)))
}

func handleTeacherResourceUpload(service teacherResourceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := teacherResourceClaims(w, r)
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxTeacherMultipartBytes)
		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, "multipart/form-data is required", http.StatusBadRequest)
			return
		}
		input := retrieval.TeacherUploadInput{
			TenantID: claims.TenantID, UploaderID: claims.Subject,
		}
		for {
			part, err := reader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				http.Error(w, "multipart body exceeds the allowed limit", http.StatusRequestEntityTooLarge)
				return
			}
			if err := consumeTeacherPart(part, &input); err != nil {
				writeTeacherResourceError(w, err)
				return
			}
		}
		if err := validateTeacherUUIDs("class ID", input.ClassIDs); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		resource, err := service.Upload(r.Context(), input)
		if err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resource)
	}
}

func consumeTeacherPart(part *multipart.Part, input *retrieval.TeacherUploadInput) error {
	defer func() { _ = part.Close() }()
	name := part.FormName()
	if name == "file" {
		if len(input.Data) > 0 {
			return fmt.Errorf("%w: exactly one file is allowed", retrieval.ErrInvalidArgument)
		}
		data, err := io.ReadAll(io.LimitReader(part, retrieval.MaxTeacherResourceBytes+1))
		if err != nil {
			return err
		}
		if len(data) > retrieval.MaxTeacherResourceBytes {
			return retrieval.ErrFileTooLarge
		}
		input.Data = data
		input.Filename = part.FileName()
		input.MediaType = part.Header.Get("Content-Type")
		return nil
	}
	value, err := io.ReadAll(io.LimitReader(part, 64<<10))
	if err != nil {
		return err
	}
	switch name {
	case "title":
		input.Title = string(value)
	case "class_id":
		input.ClassIDs = append(input.ClassIDs, string(value))
	case "class_ids":
		var ids []string
		if json.Unmarshal(value, &ids) == nil {
			input.ClassIDs = append(input.ClassIDs, ids...)
		} else {
			input.ClassIDs = append(input.ClassIDs, strings.Split(string(value), ",")...)
		}
	}
	return nil
}

func handleTeacherResourceList(service teacherResourceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := teacherResourceClaims(w, r)
		if !ok {
			return
		}
		classIDs := teacherClassIDs(r)
		if err := validateTeacherUUIDs("class ID", classIDs); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		resources, err := service.List(r.Context(), claims.TenantID, classIDs, r.URL.Query().Get("include_inactive") == "true")
		if err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resources)
	}
}

func handleTeacherResourceDeactivate(service teacherResourceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := teacherResourceClaims(w, r)
		if !ok {
			return
		}
		resourceID := r.PathValue("id")
		classIDs := teacherClassIDs(r)
		if err := validateTeacherUUID("resource ID", resourceID); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		if err := validateTeacherUUIDs("class ID", classIDs); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		if err := service.SetActive(r.Context(), claims.TenantID, resourceID, classIDs, false); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleTeacherResourceDelete(service teacherResourceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := teacherResourceClaims(w, r)
		if !ok {
			return
		}
		resourceID := r.PathValue("id")
		classIDs := teacherClassIDs(r)
		if err := validateTeacherUUID("resource ID", resourceID); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		if err := validateTeacherUUIDs("class ID", classIDs); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		if err := service.Delete(r.Context(), claims.TenantID, resourceID, classIDs); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleTeacherResourceSearch(service teacherResourceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := teacherResourceClaims(w, r)
		if !ok {
			return
		}
		var body struct {
			Query    string   `json:"query"`
			ClassIDs []string `json:"class_ids"`
			Limit    int      `json:"limit"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := validateTeacherUUIDs("class ID", body.ClassIDs); err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		evidence, err := service.Search(r.Context(), retrieval.TeacherEvidenceRequest{
			TenantID: claims.TenantID, ClassIDs: body.ClassIDs, Query: body.Query, Limit: body.Limit,
		})
		if err != nil {
			writeTeacherResourceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, evidence)
	}
}

func teacherResourceClaims(w http.ResponseWriter, r *http.Request) (auth.TokenClaims, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok ||
		validateTeacherUUID("authenticated tenant", claims.TenantID) != nil ||
		validateTeacherUUID("authenticated uploader", claims.Subject) != nil {
		http.Error(w, "authenticated tenant scope is required", http.StatusForbidden)
		return auth.TokenClaims{}, false
	}
	return claims, true
}

func validateTeacherUUID(label, value string) error {
	if _, err := uuid.Parse(strings.TrimSpace(value)); err != nil {
		return fmt.Errorf("%w: %s must be a UUID", retrieval.ErrInvalidArgument, label)
	}
	return nil
}

func validateTeacherUUIDs(label string, values []string) error {
	for _, value := range values {
		if err := validateTeacherUUID(label, value); err != nil {
			return err
		}
	}
	return nil
}

func teacherClassIDs(r *http.Request) []string {
	values := r.URL.Query()["class_id"]
	for _, value := range r.URL.Query()["class_ids"] {
		values = append(values, strings.Split(value, ",")...)
	}
	return values
}

func writeTeacherResourceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, retrieval.ErrFileTooLarge):
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
	case errors.Is(err, retrieval.ErrUnsupportedFile):
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
	case errors.Is(err, retrieval.ErrEncryptedFile),
		errors.Is(err, retrieval.ErrImageOnlyFile),
		errors.Is(err, retrieval.ErrEmptyFile),
		errors.Is(err, retrieval.ErrMalformedFile),
		errors.Is(err, retrieval.ErrInvalidArgument):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	case errors.Is(err, retrieval.ErrTeacherResourceConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, retrieval.ErrTeacherResourceScope):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, retrieval.ErrTeacherResourceNotFound):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	case errors.Is(err, retrieval.ErrGraphUnavailable):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

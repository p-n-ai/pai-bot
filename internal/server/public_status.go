// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"
)

type publicStatusComponent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type publicStatusResponse struct {
	Status     string                  `json:"status"`
	Components []publicStatusComponent `json:"components"`
}

func handlePublicStatus(w http.ResponseWriter, r *http.Request, opts TopMuxOptions) {
	response := publicStatusResponse{
		Status: "ok",
		Components: []publicStatusComponent{
			{ID: "application", Status: "operational"},
		},
	}

	if opts.AIHealthEnabled != nil && opts.AIHealthEnabled() && opts.AIHealthCheck != nil {
		aiStatus := "operational"
		if err := opts.AIHealthCheck(r.Context()); err != nil {
			aiStatus = "unavailable"
			response.Status = "degraded"
		}
		response.Components = append(
			response.Components,
			publicStatusComponent{ID: "ai", Status: aiStatus},
		)
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, response)
}
